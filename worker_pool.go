package main

import (
	"math/rand"
	"strconv"

	"github.com/rs/zerolog/log"
)

type WorkerPoolConfig []WorkerConfig
type WorkerPoolStats []WorkerStats

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

func (wp *WorkerPool) Enlist(job *Job) {
	for {
		worker := wp.getWorker()
		select {
		case <-job.Ctx.Done():
			return
		case worker.Jobs <- job:
			log.
				Info().
				Str("request id", job.Id).
				Str("worker host", worker.host).
				Msg("allocating job to worker")
			return
		default:
			continue
		}
	}
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

		if i == 0 || load < minLoad {
			freeWorker = worker
			minLoad = load
		}
	}

	return freeWorker
}
