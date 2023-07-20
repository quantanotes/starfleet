package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func LoadConfig(path string) (*StarFleetConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(wd, path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &StarFleetConfig{}
	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func IsJsonPath(data any, path []string) bool {
	for _, key := range path {
		if m, ok := data.(map[string]any); ok {
			data = m[key]
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
