package main

import "context"

type Job struct {
	ReqCtx  context.Context
	Ctx     context.Context
	Id      string
	Payload []byte
	Output  chan string
	Err     chan error
	Finish  context.CancelFunc
}

func NewJob(reqCtx context.Context, id string, payload []byte) *Job {
	ctx, cancel := context.WithCancel(reqCtx)
	return &Job{
		ReqCtx:  reqCtx,
		Ctx:     ctx,
		Id:      id,
		Payload: payload,
		Output:  make(chan string, 100),
		Err:     make(chan error, 10),
		Finish:  cancel,
	}
}

func (j *Job) Close() {
	close(j.Output)
	close(j.Err)
}
