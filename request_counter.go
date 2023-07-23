package main

import (
	"net/http"
	"sync/atomic"
)

type RequestCounterMiddlewareStats struct {
	Requests int `json:"requests"`
	Finished int `json:"finished"`
}

type RequestCounterMiddleware struct {
	requests int64
	finished int64
}

func NewRequestCounterMiddleware() RequestCounterMiddleware {
	return RequestCounterMiddleware{
		requests: 0,
		finished: 0,
	}
}

func (rcm *RequestCounterMiddleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&rcm.requests, 1)
		next.ServeHTTP(w, r)
		atomic.AddInt64(&rcm.finished, 1)
	})
}

func (rcm *RequestCounterMiddleware) Stats() RequestCounterMiddlewareStats {
	return RequestCounterMiddlewareStats{
		Requests: int(rcm.requests),
		Finished: int(rcm.finished),
	}
}
