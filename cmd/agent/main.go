package main

import (
	"flag"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/agent"
)

func main() {

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportInterval := flag.Int("r", 10, "Report interval in seconds")
	pollInterval := flag.Int("p", 2, "Poll interval in seconds")
	flag.Parse()

	serverAddress := "http://" + *addr

	a := agent.NewAgent(serverAddress, time.Duration(*pollInterval)*time.Second, time.Duration(*reportInterval)*time.Second)
	a.Run()

}
