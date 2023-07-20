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
	Auth AuthConfig `json:"auth"`
	Once OnceConfig `json:"once"`
}

type Middleware struct {
	auth *AuthMiddleware
	once *OnceMiddleware
}

func NewMiddleware(config MiddlewareConfig) Middleware {
	return Middleware{
		auth: NewAuthMiddleware(config.Auth),
		once: NewOnceMiddleware(config.Once),
	}
}

func (m *Middleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
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

		m.middleware(m.auth, m.middleware(m.once, next)).ServeHTTP(w, r)

		log.
			Info().
			Str("method", method).
			Str("path", url.RawPath).
			Str("request id", id).
			Str("time elapsed", time.Since(timer).String()).
			Msg("Finished request")
	})
}

func (m *Middleware) middleware(middleware MiddlewareInterface, next http.HandlerFunc) http.HandlerFunc {
	if middleware != nil {
		return middleware.Middleware(next)
	}
	return next
}

func headers(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
}
