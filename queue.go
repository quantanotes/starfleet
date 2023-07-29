package main

import (
	"context"
	"sync"
	"sync/atomic"
)

type QueueStats struct {
	Size     int
	Released int
}

type Queue struct {
	queue    chan struct{}
	ids      sync.Map
	capacity int
	size     int32
	released int32
}

func NewQueue(capacity int) Queue {
	return Queue{
		queue:    make(chan struct{}, capacity),
		ids:      sync.Map{},
		capacity: capacity,
		size:     0,
		released: 0,
	}
}

func (q *Queue) Wait(ctx context.Context, id string) {
	atomic.AddInt32(&q.size, 1)
	defer atomic.AddInt32(&q.size, -1)

	select {
	case <-ctx.Done():
		return
	case q.queue <- struct{}{}:
		atomic.AddInt32(&q.released, 1)
		q.ids.Store(id, struct{}{})
		go func() {
			<-ctx.Done()
			atomic.AddInt32(&q.released, -1)
			q.ids.Delete(id)
			<-q.queue
		}()
		return
	}
}

func (q *Queue) Stats() QueueStats {
	return QueueStats{
		Size:     int(q.size),
		Released: int(q.released),
	}
}
