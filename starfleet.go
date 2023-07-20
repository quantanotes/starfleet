package main

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

type StarFleetConfig struct {
	Middleware MiddlewareConfig `json:"middleware"`
	Workers    WorkerPoolConfig `json:"workers"`
}

type StarFleet struct {
	middleware Middleware
	workerPool WorkerPool
}

func New(config StarFleetConfig) *StarFleet {
	return &StarFleet{
		middleware: NewMiddleware(config.Middleware),
		workerPool: NewWorkerPool(config.Workers),
	}
}

func (s *StarFleet) Run() {
	http.HandleFunc("/dashboard", s.handleDashboard)
	http.HandleFunc("/generate", s.middleware.Middleware(s.handleGenerate))

	log.Info().Msg("Listening on port :8080")
	log.Fatal().Err(http.ListenAndServe(":8080", nil))
}
