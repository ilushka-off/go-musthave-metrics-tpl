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
	hashKey        string
}

func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration, hashKey string) *Agent {
	return &Agent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		gauges:         make(map[string]float64),
		pollCount:      0,
		hashKey:        hashKey,
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

	metrics := make([]models.Metrics, 0, len(a.gauges)+1)

	for name, value := range a.gauges {
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		})
	}

	metrics = append(metrics, models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: &a.pollCount,
	})

	if len(metrics) == 0 {
		return
	}

	err := sendMetricsBatch(a.serverAddress, metrics, a.hashKey)
	if err != nil {
		return
	}

	a.pollCount = 0
}
