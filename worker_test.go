package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func simulateWorker() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
			return
		case "/generate":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				fmt.Println("Streaming not supported")
				return
			}

			for i := 'a'; i <= 'z'; i++ {
				fmt.Fprint(w, string(i))
				flusher.Flush()
			}

			fmt.Fprintf(w, "\n\n")
			flusher.Flush()

			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
}

func TestCharIter(t *testing.T) {
	for i := 'a'; i <= 'z'; i++ {
		fmt.Println(string(i))
	}
}
