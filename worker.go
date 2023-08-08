package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	workerDefaultHeartbeat  = 1
	workerDefaultTimeout    = 20
	workerDefaultMaxRetries = 10
)

type WorkerConfig struct {
	Host             string            `json:"host"`
	Capacity         int               `json:"capacity"`
	Heartbeat        int               `json:"heartbeat,omitempty"`
	Timeout          int               `json:"timeout,omitempty"`
	CheckAlive       bool              `json:"checkAlive,omitempty"`
	MaxRetries       int               `json:"maxRetries,omitempty"`
	Restart          bool              `json:"restart,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	GenerateEndpoint string            `json:"generateEndpoint,omitempty"`
	OpenAI           bool              `json:"openai,omitempty"`
}

func (c *WorkerConfig) defaults() {
	if c.Heartbeat <= 0 {
		c.Heartbeat = workerDefaultHeartbeat
	}
	if c.Timeout <= 0 {
		c.Timeout = workerDefaultTimeout
	}
	if c.GenerateEndpoint == "" {
		c.GenerateEndpoint = "/generate"
	}
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
}

type WorkerStats struct {
	Host           string
	Alive          bool
	Capacity       int
	Queued         int
	Released       int
	Running        int
	Requests       int
	Finished       int
	Successes      int
	Fails          int
	AvgRequestTime int
}

type Worker struct {
	Alive bool
	Jobs  chan *Job
	host  string
	queue Queue

	capacity  int
	running   int32
	requests  int32
	finished  int32
	fails     int32
	successes int32
	failCount int32

	maxRetries int
	restart    bool

	avgReqTime   int64
	totalReqTime int64

	heartbeat      time.Duration
	isHeartbeating bool
	hbMu           sync.Mutex

	timeout    time.Duration
	checkAlive bool

	headers          map[string]string
	generateEndpoint string
	openai           bool

	config WorkerConfig
}

func NewWorker(config WorkerConfig) *Worker {
	config.defaults()
	return &Worker{
		Alive:            true,
		Jobs:             make(chan *Job, config.Capacity*2),
		host:             config.Host,
		queue:            NewQueue(config.Capacity),
		capacity:         config.Capacity,
		running:          0,
		requests:         0,
		finished:         0,
		fails:            0,
		successes:        0,
		failCount:        0,
		restart:          config.Restart,
		maxRetries:       config.MaxRetries,
		avgReqTime:       0,
		totalReqTime:     0,
		heartbeat:        time.Duration(config.Heartbeat) * time.Second,
		isHeartbeating:   false,
		hbMu:             sync.Mutex{},
		timeout:          time.Duration(config.Timeout) * time.Second,
		checkAlive:       config.CheckAlive,
		headers:          config.Headers,
		generateEndpoint: config.GenerateEndpoint,
		openai:           config.OpenAI,
		config:           config,
	}
}

func (w *Worker) Work() {
	if w.checkAlive {
		go w.doHearbeat()
	}

	for job := range w.Jobs {
		if !w.Alive {
			job.Err <- fmt.Errorf("LLM became unresponsive")
			job.Finish()
			continue
		}
		go w.generate(job)
	}
}

func (w *Worker) Revive() {
	if w.config.CheckAlive {
		w.hbMu.Lock()
		defer w.hbMu.Unlock()
		w.checkAlive = true
		w.Alive = true
		go w.doHearbeat()
	} else {
		w.Alive = true
	}
}

func (w *Worker) Load() float64 {
	return float64(len(w.Jobs)+w.queue.Stats().Size+int(w.running)) / float64(w.capacity)
}

func (w *Worker) Stats() WorkerStats {
	return WorkerStats{
		Host:           w.host,
		Capacity:       w.capacity,
		Alive:          w.Alive,
		Queued:         w.queue.Stats().Size,
		Released:       w.queue.Stats().Released,
		Running:        int(w.running),
		Requests:       int(w.requests),
		Finished:       int(w.finished),
		Successes:      int(w.successes),
		Fails:          int(w.fails),
		AvgRequestTime: int(w.avgReqTime),
	}
}

func (w *Worker) doHearbeat() {
	w.hbMu.Lock()
	w.isHeartbeating = true
	w.hbMu.Unlock()

	defer func() {
		w.hbMu.Lock()
		w.isHeartbeating = false
		w.hbMu.Unlock()
	}()

	for range time.Tick(w.heartbeat) {
		w.hbMu.Lock()
		if !w.checkAlive {
			w.hbMu.Unlock()
			return
		}

		alive := w.ping()
		if w.Alive && !alive {
			log.Error().Str("host", w.host).Msg("Worker has died")
		}
		if !w.Alive && alive {
			log.Error().Str("host", w.host).Msg("Worker has been revived")
		}

		w.Alive = alive
		w.hbMu.Unlock()
	}
}

func (w *Worker) ping() bool {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(w.host)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (w *Worker) generate(job *Job) {
	atomic.AddInt32(&w.requests, 1)
	w.queue.Wait(job.Ctx, job.Id)
	atomic.AddInt32(&w.running, 1)

	failed := false
	early := false

	reqTime := time.Now().UnixMilli()

	defer func() {
		log.Info().Str("request id", job.Id).Str("worker host", w.host).Msg("Finishing generate request with worker")

		job.Finish()

		atomic.AddInt32(&w.running, -1)
		atomic.AddInt32(&w.finished, 1)

		if failed {
			w.countFail()
		} else if !early {
			w.countSuccess()
		}

		if atomic.LoadInt32(&w.failCount) >= int32(w.maxRetries) {
			log.Warn().Str("worker host", w.host).Msgf("Worker has failed after %v retries", atomic.LoadInt32(&w.failCount))
			w.hbMu.Lock()
			w.checkAlive = false
			w.Alive = false
			w.hbMu.Unlock()
			if w.restart {
				go w.doRestart()
			}
		}

		if !w.Alive {
			atomic.StoreInt32(&w.failCount, 0)
		}

		w.calcAvgReqTime(reqTime)
	}()

	select {
	case <-job.ReqCtx.Done():
		early = true
		return
	default:
	}

	res, err := w.prompt(job.Ctx, job.Payload)
	if err != nil {
		//lint:ignore ST1005 frontend error
		job.Err <- fmt.Errorf("Error prompting LLM")
		failed = true
		return
	}
	defer res.Body.Close()

	log.Info().Str("request id", job.Id).Str("worker host", w.host).Msg("Initiated generate request with worker")

	for {
		data := make([]byte, 1024)
		if _, err := res.Body.Read(data); err == io.EOF {
			return
		} else if err != nil {
			//lint:ignore ST1005 frontend error
			job.Err <- fmt.Errorf("Error reading tokens from LLM")
			failed = true
			return
		}

		token := ""
		if w.openai {
			if token, err = w.openaiFilter(data); err == io.EOF {
				return
			} else if err != nil {
				//lint:ignore ST1005 frontend error
				job.Err <- fmt.Errorf("Error reading tokens from LLM")
				failed = true
				return
			}
		} else {
			token = string(data)
		}

		select {
		case job.Output <- token:
			if token == "" {
				return
			}
		case <-job.ReqCtx.Done():
			early = true
			return
		case <-time.After(w.timeout):
			job.Err <- fmt.Errorf("LLM timed out after %v", w.timeout)
			failed = true
			return
		}
	}
}

func (w *Worker) prompt(ctx context.Context, payload []byte) (*http.Response, error) {
	path, err := url.JoinPath(w.host, w.generateEndpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	for h, v := range w.headers {
		req.Header.Set(h, v)
	}

	client := http.Client{}
	return client.Do(req.WithContext(ctx))
}

func (w *Worker) doRestart() {

}

func (w *Worker) countSuccess() {
	atomic.AddInt32(&w.successes, 1)
	atomic.StoreInt32(&w.failCount, 0)
}

func (w *Worker) countFail() {
	atomic.AddInt32(&w.fails, 1)
	atomic.AddInt32(&w.failCount, 1)
}

func (w *Worker) calcAvgReqTime(reqTime int64) {
	reqTime = time.Now().UnixMilli() - reqTime
	atomic.AddInt64(&w.totalReqTime, reqTime)

	totalReqTime := atomic.LoadInt64(&w.totalReqTime)
	numRequests := atomic.LoadInt32(&w.finished)

	w.avgReqTime = totalReqTime / int64(numRequests)
}

type openaiResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (w *Worker) openaiFilter(data []byte) (string, error) {
	if string(data) == "[DONE]" {
		return "", io.EOF
	}

	var jsn openaiResponse
	if err := json.Unmarshal(data, &jsn); err != nil {
		return "", err
	}

	if len(jsn.Choices) > 0 {
		return jsn.Choices[0].Delta.Content, nil
	}

	return "", nil
}
