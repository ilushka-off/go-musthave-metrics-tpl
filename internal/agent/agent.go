// Package agent implements the metrics-collection agent: it periodically
// polls runtime and system metrics and reports them to the server over
// HTTP.
package agent

import (
	"context"
	"crypto/rsa"
	"sync"
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
	publicKey      *rsa.PublicKey
	rateLimit      int
	metricsCh      chan models.Metrics
	snapshotCh     chan chan []models.Metrics
}

// NewAgent creates an Agent that polls metrics every pollInterval and
// reports them to serverAddress every reportInterval. hashKey, if non-empty,
// is used to sign each report. publicKey, if non-nil, is used to encrypt each
// report body. rateLimit is the number of concurrent report
// workers and is clamped to at least 1.
func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration, hashKey string, rateLimit int, publicKey *rsa.PublicKey) *Agent {
	if rateLimit < 1 {
		rateLimit = 1
	}

	return &Agent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		hashKey:        hashKey,
		publicKey:      publicKey,
		rateLimit:      rateLimit,
		metricsCh:      make(chan models.Metrics),
		snapshotCh:     make(chan chan []models.Metrics),
	}
}

// Run starts polling and reporting. It blocks forever, driving the
// collection and reporting loops on background goroutines.
func (a *Agent) Run() {
	a.RunContext(context.Background())
}

// RunContext is like Run but stops when ctx is cancelled. On cancellation
// polling stops, the metrics accumulated since the last report are sent as a
// final batch, and RunContext returns only after every queued and in-flight
// report has been delivered (or has exhausted its retries).
func (a *Agent) RunContext(ctx context.Context) {
	// accumulate gets its own context: it must outlive ctx to serve the final
	// snapshot below, and is stopped only after that.
	accCtx, stopAcc := context.WithCancel(context.Background())
	defer stopAcc()
	go a.accumulate(accCtx)

	var pollers sync.WaitGroup
	pollers.Add(2)
	go func() { defer pollers.Done(); a.pollRuntime(ctx) }()
	go func() { defer pollers.Done(); a.pollGopsutilMetrics(ctx) }()

	jobs := make(chan []models.Metrics, a.rateLimit)
	var workers sync.WaitGroup
	for i := 0; i < a.rateLimit; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); a.worker(jobs) }()
	}

	a.scheduleReports(ctx, jobs)

	// Pollers have stopped, so this snapshot holds everything collected.
	pollers.Wait()
	a.reportOnce(jobs)
	stopAcc()
	close(jobs)
	workers.Wait()
}

func (a *Agent) accumulate(ctx context.Context) {
	gauges := make(map[string]float64)
	var pollCount int64

	for {
		select {
		case <-ctx.Done():
			return
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

func (a *Agent) pollRuntime(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		gauges := CollectRunTimeGauges()
		for name, value := range gauges {
			v := value
			a.metricsCh <- models.Metrics{ID: name, MType: models.Gauge, Value: &v}
		}

		delta := int64(1)
		a.metricsCh <- models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta}
	}
}

func (a *Agent) pollGopsutilMetrics(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

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

func (a *Agent) scheduleReports(ctx context.Context, jobs chan<- []models.Metrics) {
	ticker := time.NewTicker(a.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.reportOnce(jobs)
		}
	}
}

// reportOnce takes a snapshot of the accumulated metrics and queues it for
// sending. It blocks while the job queue is full.
func (a *Agent) reportOnce(jobs chan<- []models.Metrics) {
	reply := make(chan []models.Metrics)
	a.snapshotCh <- reply
	metrics := <-reply

	if len(metrics) == 0 {
		return
	}

	jobs <- metrics
}

func (a *Agent) worker(jobs <-chan []models.Metrics) {
	for metrics := range jobs {
		_ = sendMetricsBatch(a.serverAddress, metrics, a.hashKey, a.publicKey)
	}
}
