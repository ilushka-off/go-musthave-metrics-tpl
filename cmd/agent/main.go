package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/agent"
)

func main() {

	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportInterval := flag.Int("r", 10, "Report interval in seconds")
	pollInterval := flag.Int("p", 2, "Poll interval in seconds")
	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok {
		*addr = envAddr
	}

	if envReportInterval, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		var err error
		*reportInterval, err = strconv.Atoi(envReportInterval)

		if err != nil {
			log.Fatal(err)
		}

	}

	if envPollInterval, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		var err error

		*pollInterval, err = strconv.Atoi(envPollInterval)

		if err != nil {
			log.Fatal(err)
		}
	}

	serverAddress := "http://" + *addr

	a := agent.NewAgent(serverAddress, time.Duration(*pollInterval)*time.Second, time.Duration(*reportInterval)*time.Second)
	a.Run()

}
