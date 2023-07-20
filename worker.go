package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"time"
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

type Worker struct {
	Alive      bool
	Jobs       chan *Job
	queue      chan struct{}
	host       string
	capacity   int
	client     http.Client
	heartbeatT time.Duration
	timeout    time.Duration
}

func NewWorker(config WorkerConfig) *Worker {
	config.defaults()
	return &Worker{
		Alive:      true,
		Jobs:       make(chan *Job, config.Capacity),
		queue:      make(chan struct{}, config.Capacity),
		host:       config.Host,
		capacity:   config.Capacity,
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
		<-w.Jobs
	}
}

func (w *Worker) Load() float64 {
	return float64(len(w.Jobs)) / float64(w.capacity)
}

func (w *Worker) heartbeat() {
	for range time.Tick(w.heartbeatT * time.Second) {
		w.Alive = w.ping()
	}
}

func (w *Worker) ping() bool {
	resp, err := w.client.Get(w.host)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (w *Worker) generate(job *Job) {
	w.queue <- struct{}{}

	defer func() {
		job.Done <- struct{}{}
		<-w.queue
	}()

	res, err := w.prompt(job.Payload)
	if err != nil {
		job.Err <- err
	}

	for {
		select {
		case <-job.Ctx.Done():
			return
		case <-time.After(w.timeout):
			job.Err <- fmt.Errorf("LLM timed out after %v", 10)
			return
		default:
			data := make([]byte, 1024)
			_, err := res.Body.Read(data)
			if err != nil {
				job.Err <- err
				return
			}

			select {
			case job.Output <- string(data):
				break
			default:
				return
			}
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
