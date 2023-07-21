package main

import (
	"context"
	"sync"
	"sync/atomic"
)

type Queue struct {
	queue    chan struct{}
	ids      sync.Map
	size     int32
	capacity int
}

func NewQueue(capacity int) Queue {
	return Queue{
		queue:    make(chan struct{}, capacity),
		ids:      sync.Map{},
		size:     0,
		capacity: capacity,
	}
}

func (q *Queue) Size() int {
	return int(q.size)
}

func (q *Queue) Wait(ctx context.Context, id string) {
	atomic.AddInt32(&q.size, 1)
	defer atomic.AddInt32(&q.size, -1)

	select {
	case <-ctx.Done():
		return
	case q.queue <- struct{}{}:
		q.ids.Store(id, struct{}{})
		return
	}
}

func (q *Queue) Release(id string) {
	if _, ok := q.ids.Load(id); ok {
		q.ids.Delete(id)
		<-q.queue
	}
}
