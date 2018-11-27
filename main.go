package main

import (
	"flag"
	"log"

	"github.com/ninnemana/butler/services"
)

var (
	cfg    services.Config
	file   = flag.String("config", "", "file to read configuration")
	envVar = flag.String("env", "", "environment variable to read configuration")
)

func main() {
	flag.Parse()

	cfg, err := services.ReadConfig(file, envVar)
	if err != nil {
		log.Fatalf("failed to read configuration: %v", err)
	}

	if err := services.Start(cfg); err != nil {
		log.Fatalf("fell out of listener: %v", err)
	}
}
