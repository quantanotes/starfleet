package main

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
)

func (sf *StarFleet) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("www/dashboard.html")
	if err != nil {
		log.Error().Err(err).Msg("")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := tmpl.Execute(w, nil); err != nil {
		log.Error().Err(err).Msg("")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (sf *StarFleet) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("www/stats.html")
	if err := tmpl.Execute(w, sf.workerPool.Stats()); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (sf *StarFleet) handleDashboardRequestCounter(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("www/request-counter.html")
	if err := tmpl.Execute(w, sf.requestCounter.Stats()); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (sf *StarFleet) handleDashboardRevive(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	numStr := path[len("/dashboard-revive/"):]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		http.Error(w, "Invalid number", http.StatusBadRequest)
		return
	}
	sf.workerPool.Revive(num)
}
