package agent

import (
	"fmt"
	"time"
)

type Agent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	gauges         map[string]float64
	pollCount      int64
}

func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		gauges:         make(map[string]float64),
		pollCount:      0,
	}
}

func (a *Agent) Run() {
	pollIntervalTicker := time.NewTicker(a.pollInterval)
	defer pollIntervalTicker.Stop()
	reportIntervalTicker := time.NewTicker(a.reportInterval)
	defer reportIntervalTicker.Stop()

	for {
		select {
		case <-pollIntervalTicker.C:
			a.poll()
		case <-reportIntervalTicker.C:
			a.report()
		}
	}
}

func (a *Agent) poll() {
	a.gauges = CollectRunTimeGauges()
	a.pollCount += 1
}

func (a *Agent) report() {
	for name, value := range a.gauges {
		sendMetrics(a.serverAddress, "gauge", name, fmt.Sprintf("%f", value))
	}

	sendMetrics(a.serverAddress, "counter", "PollCount", fmt.Sprintf("%v", a.pollCount))

	a.pollCount = 0
}
