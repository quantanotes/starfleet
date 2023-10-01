package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	onceDefaultTimeout = 10
	onceDefaultPrefix  = "sf-once-middleware:"
)

type OnceConfig struct {
	RedisURL    string `json:"redisUrl,omitempty"`
	RedisURLEnv string `json:"redisUrlEnv,omitempty"`
	KeyPrefix   string `json:"keyPrefix,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
}

func (c *OnceConfig) defaults() {
	if c.Timeout == 0 {
		c.Timeout = onceDefaultTimeout
	}
	if c.KeyPrefix == "" {
		c.KeyPrefix = onceDefaultPrefix
	}
	if c.RedisURL == "" {
		c.RedisURL = os.Getenv(c.RedisURLEnv)
	}
}

type Once struct {
	client  *redis.Client
	prefix  string
	timeout time.Duration
}

func NewOnce(config OnceConfig) *Once {
	config.defaults()

	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(opt)

	return &Once{
		client:  client,
		prefix:  config.KeyPrefix,
		timeout: time.Duration(config.Timeout) * time.Second,
	}
}

func (o *Once) Middleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.Header.Get("X-Request-ID")
		key := o.prefix + id

		lock := o.client.SetNX(ctx, key, "-", o.timeout)
		if err := lock.Err(); err != nil {
			LogHttpErr(w, id, "Failed to access cache", err, http.StatusInternalServerError)
			return
		}

		if !lock.Val() {
			LogHttpErr(w, id, "Can only access LLM once at a time", nil, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)

		if err := o.client.Del(context.Background(), key).Err(); err != nil {
			log.Error().Err(err).Str("request_id", id).Msg("Failed to release lock")
		}
	})
}
