package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type WorkerConfig struct {
	Host      string `json:"host"`
	Capacity  int    `json:"capacity"`
	Heartbeat int    `json:"heartbeat,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
}

func (c *WorkerConfig) defaults() {
	if c.Heartbeat <= 0 {
		c.Heartbeat = 3
	}
	if c.Timeout <= 0 {
		c.Timeout = 10
	}
}

type WorkerStats struct {
	Host     string
	Alive    bool
	Capacity int
	Queued   int
	Running  int
}

type Worker struct {
	Alive      bool
	Jobs       chan *Job
	queue      chan struct{}
	host       string
	capacity   int
	running    int32
	client     http.Client
	heartbeatT time.Duration
	timeout    time.Duration
}

func NewWorker(config WorkerConfig) *Worker {
	config.defaults()
	return &Worker{
		Alive:      true,
		Jobs:       make(chan *Job, config.Capacity*2),
		queue:      make(chan struct{}, config.Capacity),
		host:       config.Host,
		capacity:   config.Capacity,
		running:    0,
		client:     http.Client{},
		heartbeatT: time.Duration(config.Heartbeat) * time.Second,
		timeout:    time.Duration(config.Timeout) * time.Second,
	}
}

func (w *Worker) Work() {
	go w.heartbeat()
	for job := range w.Jobs {
		if !w.Alive {
			job.Err <- fmt.Errorf("LLM became unresponsive")
			job.Done <- struct{}{}
			continue
		}
		go w.generate(job)
	}
}

func (w *Worker) Load() float64 {
	return float64(len(w.Jobs)+len(w.queue)) / float64(w.capacity)
}

func (w *Worker) Stats() WorkerStats {
	return WorkerStats{
		Host:     w.host,
		Capacity: w.capacity,
		Alive:    w.Alive,
		Queued:   len(w.Jobs) + len(w.queue) - int(w.running),
		Running:  int(w.running),
	}
}

func (w *Worker) heartbeat() {
	for range time.Tick(w.heartbeatT * time.Second) {
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
	w.queue <- struct{}{}
	atomic.AddInt32(&w.running, 1)

	defer func() {
		log.Info().Str("request id", job.Id).Str("worker host", w.host).Msg("Finishing generate request with worker")
		job.Done <- struct{}{}
		<-w.queue
		atomic.AddInt32(&w.running, -1)
	}()

	res, err := w.prompt(job.Payload)
	if err != nil {
		job.Err <- err
	}

	log.Info().Str("request id", job.Id).Str("worker host", w.host).Msg("Initiated generate request with worker")

	scanner := bufio.NewScanner(res.Body)
	defer res.Body.Close()

	for scanner.Scan() {
		data := scanner.Text()
		if data == "" {
			return
		}

		select {
		case job.Output <- data:
			continue
		case <-job.Ctx.Done():
			return
		case <-time.After(w.timeout):
			job.Err <- fmt.Errorf("LLM timed out after %v", w.timeout)
			return
		}
	}
}

func (w *Worker) prompt(payload []byte) (*http.Response, error) {
	path, _ := url.JoinPath(w.host, "/generate")
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")

	return w.client.Do(req)
}
