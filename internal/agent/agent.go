// Package agent implements the metrics-collection agent: it periodically
// polls runtime and system metrics and reports them to the server over
// HTTP.
package agent

import (
	"time"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

// Agent periodically collects runtime and system metrics and reports them
// to a metrics server. Create one with NewAgent and start it with Run.
type Agent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	hashKey        string
	rateLimit      int
	metricsCh      chan models.Metrics
	snapshotCh     chan chan []models.Metrics
}

// NewAgent creates an Agent that polls metrics every pollInterval and
// reports them to serverAddress every reportInterval. hashKey, if non-empty,
// is used to sign each report. rateLimit is the number of concurrent report
// workers and is clamped to at least 1.
func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration, hashKey string, rateLimit int) *Agent {
	if rateLimit < 1 {
		rateLimit = 1
	}

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

// Run starts polling and reporting. It blocks forever, driving the
// collection and reporting loops on background goroutines.
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
		_ = sendMetricsBatch(a.serverAddress, metrics, a.hashKey)
	}
}
