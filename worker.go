package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type WorkerConfig struct {
	Host             string            `json:"host"`
	Capacity         int               `json:"capacity"`
	Heartbeat        int               `json:"heartbeat,omitempty"`
	Timeout          int               `json:"timeout,omitempty"`
	CheckAlive       bool              `json:"checkAlive,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	GenerateEndpoint string            `json:"generateEndpoint,omitempty"`
}

func (c *WorkerConfig) defaults() {
	if c.Heartbeat <= 0 {
		c.Heartbeat = 1
	}
	if c.Timeout <= 0 {
		c.Timeout = 10
	}
	if c.GenerateEndpoint == "" {
		c.GenerateEndpoint = "/generate"
	}
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
}

type WorkerStats struct {
	Host     string
	Alive    bool
	Capacity int
	Queued   int
	Running  int
	Requests int
	Finished int
}

type Worker struct {
	Alive bool
	Jobs  chan *Job
	host  string
	queue Queue

	capacity int
	running  int32
	requests int32
	finished int32

	heartbeat  time.Duration
	timeout    time.Duration
	checkAlive bool

	headers          map[string]string
	generateEndpoint string
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
		heartbeat:        time.Duration(config.Heartbeat) * time.Second,
		timeout:          time.Duration(config.Timeout) * time.Second,
		checkAlive:       config.CheckAlive,
		headers:          config.Headers,
		generateEndpoint: config.GenerateEndpoint,
	}
}

func (w *Worker) Work() {
	if w.checkAlive {
		go w.doHearbeat()
	}

	for job := range w.Jobs {
		if !w.Alive {
			job.Err <- fmt.Errorf("LLM became unresponsive")
			select {
			case job.Done <- struct{}{}:
			default:
			}
			continue
		}
		go w.generate(job)
	}
}

func (w *Worker) Load() float64 {
	return float64(len(w.Jobs)+w.queue.Size()+int(w.running)) / float64(w.capacity)
}

func (w *Worker) Stats() WorkerStats {
	return WorkerStats{
		Host:     w.host,
		Capacity: w.capacity,
		Alive:    w.Alive,
		Queued:   len(w.Jobs) + w.queue.Size(),
		Running:  int(w.running),
		Requests: int(w.requests),
		Finished: int(w.finished),
	}
}

func (w *Worker) doHearbeat() {
	for range time.Tick(w.heartbeat) {
		alive := w.ping()
		if w.Alive && !alive {
			log.Error().Str("host", w.host).Msg("Worker has died")
		}
		if !w.Alive && alive {
			log.Error().Str("host", w.host).Msg("Worker has been revived")
		}
		w.Alive = alive
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

	defer func() {
		log.Info().Str("request id", job.Id).Str("worker host", w.host).Msg("Finishing generate request with worker")
		select {
		case job.Done <- struct{}{}:
		default:
		}
		atomic.AddInt32(&w.running, -1)
		atomic.AddInt32(&w.finished, 1)
	}()

	select {
	case <-job.Ctx.Done():
		return
	default:
	}

	res, err := w.prompt(job.Ctx, job.Payload)
	if err != nil {
		//lint:ignore ST1005 frontend error
		job.Err <- fmt.Errorf("Error prompting LLM")
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
			return
		}
		token := string(data)

		select {
		case job.Output <- token:
			if token != "" {
				continue
			}
		case <-job.Ctx.Done():
			return
		case <-time.After(w.timeout):
			job.Err <- fmt.Errorf("LLM timed out after %v", w.timeout)
			return
		default:
			continue
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

	return http.DefaultClient.Do(req.WithContext(ctx))
}
