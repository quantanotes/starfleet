package main

import "context"

type Job struct {
	Ctx     context.Context
	Id      string
	Payload []byte
	Output  chan string
	Done    chan struct{}
	Err     chan error
}

func NewJob(ctx context.Context, id string, payload []byte) *Job {
	return &Job{
		Ctx:     ctx,
		Id:      id,
		Payload: payload,
		Output:  make(chan string, 100),
		Done:    make(chan struct{}),
		Err:     make(chan error),
	}
}
