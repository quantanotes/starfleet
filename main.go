package main

import "github.com/rs/zerolog"

func main() {
	config, err := LoadConfig("config.json")
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	sf := New(*config)
	sf.Run()
}
