package main

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

type StarFleetConfig struct {
	Middleware MiddlewareConfig `json:"middleware"`
	Workers    []WorkerConfig   `json:"workers"`
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

func (sf *StarFleet) Run() {
	sf.workerPool.Run()

	http.HandleFunc("/dashboard", sf.handleDashboard)
	http.HandleFunc("/dashboard-stats", sf.handleDashboardStats)

	http.HandleFunc("/generate", sf.middleware.Middleware(sf.handleGenerate))

	log.Info().Msg("Listening on port :8080")
	log.Fatal().Err(http.ListenAndServe(":8080", nil))
}
