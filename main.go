package main

import (
	"github.com/ninnemana/butler/services"
)

var (
	configs = []services.Service{
		services.Service{
			LocalAddress:   "localhost:8080",
			ServiceAddress: "butler-test:8080",
		},
	}
)

func main() {
	services.Start(configs...)
}
