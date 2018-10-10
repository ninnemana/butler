package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/ninnemana/butler/services"
)

var (
	cfg  services.Config
	file = flag.String("config", "", "file to read configuration")
)

func main() {
	flag.Parse()

	if file == nil {
		log.Fatalf("invalid configuration file")
	}

	if _, err := os.Stat(*file); err != nil {
		log.Fatalf("failed to read file: %s", *file)
	}

	f, err := os.Open(*file)
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
