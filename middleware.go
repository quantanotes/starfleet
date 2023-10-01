package main

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type MiddlewareInterface interface {
	Middleware(next http.Handler) http.HandlerFunc
}

type MiddlewareConfig struct {
	Auth *AuthConfig `json:"auth,omitempty"`
	Once *OnceConfig `json:"once,omitempty"`
}

type Middleware struct {
	middlewares []MiddlewareInterface
}

func NewMiddleware(config MiddlewareConfig) Middleware {
	m := Middleware{}
	if config.Once != nil {
		m.middlewares = append(m.middlewares, NewOnce(*config.Once))
	}
	if config.Auth != nil {
		m.middlewares = append(m.middlewares, NewAuth(*config.Auth))
	}
	return m
}

func (m *Middleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	for _, middleware := range m.middlewares {
		next = middleware.Middleware(next)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers(w)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		id := r.Header.Get("X-Request-ID")
		url := r.URL
		method := r.Method
		timer := time.Now()

		log.
			Info().
			Str("method", method).
			Str("path", url.RawPath).
			Str("request id", id).
			Msg("Recieving request")

		next.ServeHTTP(w, r)

		log.
			Info().
			Str("method", method).
			Str("path", url.RawPath).
			Str("request id", id).
			Str("time elapsed", time.Since(timer).String()).
			Msg("Finished request")
	})
}

func headers(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
}
