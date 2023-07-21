package main

import (
	"html/template"
	"net/http"
)

func (sf *StarFleet) handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("www/dashboard.html")
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (sf *StarFleet) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("www/stats.html")
	if err := tmpl.Execute(w, sf.workerPool.Stats()); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}
