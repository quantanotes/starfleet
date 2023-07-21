package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (sf *StarFleet) handleGenerate(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		LogHttpErr(w, id, "Failed to read request body", err, http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	job := NewJob(ctx, id, payload)

	log.Info().Str("request id", id).Msg("Beginning generation job")
	sf.workerPool.Enlist(job)

	flusher, ok := w.(http.Flusher)
	if !ok {
		LogHttpErr(w, id, "Streaming not supported by connection", nil, http.StatusBadRequest)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-job.Done:
			return
		case token := <-job.Output:
			fmt.Fprint(w, token)
			flusher.Flush()
		case err := <-job.Err:
			LogHttpErr(w, id, err.Error(), err, http.StatusInternalServerError)
			return
		}
	}
}
