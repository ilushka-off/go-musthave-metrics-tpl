package main

import (
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/agent"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	serverAddress  = "http://localhost:8080"
)

func main() {

	a := agent.NewAgent(serverAddress, pollInterval, reportInterval)
	a.Run()

}
