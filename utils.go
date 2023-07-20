package main

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

func IsJsonPath(data any, path []string) bool {
	for _, key := range path {
		if m, ok := data.(map[string]any); ok {
			data = m[key].(map[string]any)
		} else {
			return false
		}
	}
	if b, ok := data.(bool); ok {
		return b
	}
	return false
}

func LogHttpErr(w http.ResponseWriter, id string, msg string, err error, status int) {
	log.Error().Err(err).Str("request id", id).Msg(msg)
	http.Error(w, msg, status)
}
