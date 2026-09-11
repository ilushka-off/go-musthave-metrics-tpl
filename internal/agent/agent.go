package agent

import (
	"time"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

type Agent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	hashKey        string
	rateLimit      int
	metricsCh      chan models.Metrics
	snapshotCh     chan chan []models.Metrics
}

func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration, hashKey string, rateLimit int) *Agent {
	return &Agent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		hashKey:        hashKey,
		rateLimit:      rateLimit,
		metricsCh:      make(chan models.Metrics),
		snapshotCh:     make(chan chan []models.Metrics),
	}
}

func (a *Agent) Run() {
	go a.accumulate()
	go a.pollRuntime()
	go a.pollGopsutilMetrics()

	jobs := make(chan []models.Metrics, a.rateLimit)
	for i := 0; i < a.rateLimit; i++ {
		go a.worker(jobs)
	}

	a.scheduleReports(jobs)
}

func (a *Agent) accumulate() {
	gauges := make(map[string]float64)
	var pollCount int64

	for {
		select {
		case m := <-a.metricsCh:
			switch m.MType {
			case models.Gauge:
				gauges[m.ID] = *m.Value
			case models.Counter:
				pollCount += *m.Delta
			}
		case reply := <-a.snapshotCh:
			metrics := make([]models.Metrics, 0, len(gauges)+1)
			for name, value := range gauges {
				metrics = append(metrics, models.Metrics{
					ID:    name,
					MType: models.Gauge,
					Value: &value,
				})
			}

			metrics = append(metrics, models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: new(pollCount),
			})

			reply <- metrics
			pollCount = 0
		}
	}
}

func (a *Agent) pollRuntime() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		gauges := CollectRunTimeGauges()
		for name, value := range gauges {
			v := value
			a.metricsCh <- models.Metrics{ID: name, MType: models.Gauge, Value: &v}
		}

		delta := int64(1)
		a.metricsCh <- models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta}
	}
}

func (a *Agent) pollGopsutilMetrics() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		gauges, err := CollectGopsutilGauges()
		if err != nil {
			continue
		}

		for name, value := range gauges {
			v := value
			a.metricsCh <- models.Metrics{ID: name, MType: models.Gauge, Value: &v}
		}
	}
}

func (a *Agent) scheduleReports(jobs chan<- []models.Metrics) {
	ticker := time.NewTicker(a.reportInterval)
	defer ticker.Stop()

	for range ticker.C {
		reply := make(chan []models.Metrics)
		a.snapshotCh <- reply
		metrics := <-reply

		if len(metrics) == 0 {
			continue
		}

		jobs <- metrics
	}
}

func (a *Agent) worker(jobs <-chan []models.Metrics) {
	for metrics := range jobs {
		err := sendMetricsBatch(a.serverAddress, metrics, a.hashKey)
		if err != nil {
			return
		}
	}
}
