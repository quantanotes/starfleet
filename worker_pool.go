package main

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/rs/zerolog/log"
)

type (
	WorkerPoolConfig []WorkerConfig
	WorkerPoolStats  []WorkerStats
)

type WorkerPool struct {
	workers []*Worker
}

func NewWorkerPool(config WorkerPoolConfig) WorkerPool {
	workers := make([]*Worker, len(config))
	for i, wc := range config {
		workers[i] = NewWorker(wc)
	}
	return WorkerPool{
		workers: workers,
	}
}

func (wp *WorkerPool) Run() {
	for _, worker := range wp.workers {
		go worker.Work()
	}
}

func (wp *WorkerPool) Enlist(job *Job) error {
	worker := wp.getWorker()
	if worker == nil {
		return fmt.Errorf("could not connect to live LLM server")
	}
	select {
	case <-job.ReqCtx.Done():
		return nil
	case worker.Jobs <- job:
		log.
			Info().
			Str("request id", job.Id).
			Str("worker host", worker.host).
			Msg("Allocating job to worker")
		return nil
	}
}

func (wp *WorkerPool) Revive(num int) {
	if num >= len(wp.workers) {
		return
	}
	wp.workers[num].Revive()
}

func (wp *WorkerPool) Stats() WorkerPoolStats {
	stats := make(WorkerPoolStats, len(wp.workers))
	for i, w := range wp.workers {
		stats[i] = w.Stats()
		stats[i].Host = strconv.Itoa(i) // Temporary until dashboard security is implemented
	}
	return stats
}

func (wp *WorkerPool) getWorker() *Worker {
	if len(wp.workers) == 0 {
		return nil
	}

	var freeWorker *Worker
	var minLoad float64

	perm := rand.Perm(len(wp.workers))
	for i, v := range perm {
		worker := wp.workers[v]
		load := worker.Load()

		if (i == 0 || load < minLoad) && worker.Alive {
			freeWorker = worker
			minLoad = load
		}
	}

	return freeWorker
}
