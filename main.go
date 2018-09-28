package main

import (
	"log"

	"github.com/ninnemana/butler/services"
)

var (
	configs = []*services.Service{
		&services.Service{
			LocalAddress:   "localhost:8080",
			ServiceAddress: "butler-test:8080",
		},
	}
)

func main() {
	if err := services.Start(services.Config{
		LocalAddress: "localhost:8081",
	}); err != nil {
		log.Fatalf("fell out of listener: %v", err)
	}
}
