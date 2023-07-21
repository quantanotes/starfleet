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
	auth *AuthMiddleware
	once *OnceMiddleware
}

func NewMiddleware(config MiddlewareConfig) Middleware {
	m := Middleware{}
	// TODO: Refactor such that this can be handled in middleware's constructors
	if config.Auth != nil {
		m.auth = NewAuthMiddleware(*config.Auth)
	}
	if config.Once != nil {
		m.once = NewOnceMiddleware(*config.Once)
	}
	return m
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

		if m.auth != nil {
			next = m.auth.Middleware(next)
		}
		if m.once != nil {
			next = m.once.Middleware(next)
		}
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

// func (m *Middleware) middleware(middleware MiddlewareInterface, next http.HandlerFunc) http.HandlerFunc {
// 	if middleware != nil {
// 		return middleware.Middleware(next)
// 	}
// 	return next
// }

func headers(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
}
