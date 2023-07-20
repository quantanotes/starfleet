package main

func main() {
	config, err := LoadConfig("config.json")
	if err != nil {
		panic(err)
	}
	sf := New(*config)
	sf.Run()
}
