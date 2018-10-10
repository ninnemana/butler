package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/ninnemana/butler/services"
)

var (
	cfg services.Config
)

func main() {

	f, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("failed to decode JSON: %v", err)
	}

	if err := services.Start(cfg); err != nil {
		log.Fatalf("fell out of listener: %v", err)
	}
}
