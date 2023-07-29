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
		LogHttpErr(w, id, "Failed to read request body", err, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	job := NewJob(ctx, id, payload)
	//defer job.Close()

	log.Info().Str("request id", id).Msg("Beginning generation job")
	if err := sf.workerPool.Enlist(job); err != nil {
		LogHttpErr(w, id, "Could not connect to LLM", nil, http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		LogHttpErr(w, id, "Streaming not supported by connection", nil, http.StatusBadRequest)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-job.Ctx.Done():
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
