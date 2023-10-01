package main

import (
	"net/http"

	"github.com/redis/go-redis/v9"
)

type RateLimiterConfig struct {
	Groups map[string]int
}

type RateLimiter struct {
	client *redis.Client
}

func (rl *RateLimiter) Middleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Get("X-Request-ID")

	})
}
