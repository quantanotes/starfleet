package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
)

type QueueItem struct {
	iter  int32
	ahead int32
}

type QueueStats struct {
	Size     int
	Released int
}

type Queue struct {
	queue sync.Map
	size  int32

	pool     chan struct{}
	capacity int

	released int32
	iter     int32
}

func NewQueue(capacity int) Queue {
	return Queue{
		queue:    sync.Map{},
		pool:     make(chan struct{}, capacity),
		capacity: capacity,
		size:     0,
		released: 0,
		iter:     0,
	}
}

func (q *Queue) Wait(ctx context.Context, id string) {
	atomic.AddInt32(&q.size, 1)
	defer atomic.AddInt32(&q.size, -1)

	q.queue.Store(id, QueueItem{atomic.LoadInt32(&q.iter), atomic.LoadInt32(&q.size)})
	defer q.queue.Delete(id)

	defer atomic.AddInt32(&q.iter, 1)

	select {
	case <-ctx.Done():
		return
	case q.pool <- struct{}{}:
		atomic.AddInt32(&q.released, 1)

		go func() {
			<-ctx.Done()
			atomic.AddInt32(&q.released, -1)
			<-q.pool
		}()

		return
	}
}

func (q *Queue) get(id string) *QueueItem {
	qi, ok := q.queue.Load(id)
	if !ok {
		return nil
	}

	switch qi := qi.(type) {
	case QueueItem:
		return &qi
	default:
		return nil
	}
}

func (q *Queue) Stats() QueueStats {
	return QueueStats{
		Size:     int(q.size),
		Released: int(q.released),
	}
}

func (sf *StarFleet) handleQueue(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")

	var q *Queue
	var qi *QueueItem

	for _, w := range sf.workerPool.workers {
		qi = w.queue.get(id)
		if qi != nil {
			q = &w.queue
			break
		}
	}

	if qi == nil {
		http.Error(w, "ID is not queued", http.StatusBadRequest)
		return
	}

	ahead := qi.ahead - (q.iter - qi.iter)
	data, err := json.Marshal(ahead)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}
