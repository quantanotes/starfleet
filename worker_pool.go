package main

type WorkerPoolConfig []WorkerConfig

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
		worker.Work()
	}
}

func (wp *WorkerPool) Enlist(job *Job) {
	for {
		select {
		case <-job.Ctx.Done():
			return
		case wp.getWorker().Jobs <- job:
			return
		default:
			continue
		}
	}
}

func (wp *WorkerPool) getWorker() *Worker {
	if len(wp.workers) == 0 {
		return nil
	}

	freeWorker := wp.workers[0]
	minLoad := freeWorker.Load()
	for _, worker := range wp.workers[1:] {
		load := worker.Load()
		if load < minLoad {
			freeWorker = worker
			minLoad = load
		}
	}
	return freeWorker
}
