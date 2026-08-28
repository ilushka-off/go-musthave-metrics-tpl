package agent

import (
	"time"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
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
	a.pollCount++
}

func (a *Agent) report() {

	for name, value := range a.gauges {

		modelGauge := models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		}

		err := sendMetricsJSON(a.serverAddress, modelGauge)
		if err != nil {
			return
		}
	}

	modelCounter := models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: new(a.pollCount),
	}

	err := sendMetricsJSON(a.serverAddress, modelCounter)
	if err != nil {
		return
	}

	a.pollCount = 0
}
