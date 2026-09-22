// Package repository defines the metric storage abstraction and its
// implementations: an in-memory store, a file-backed store, a
// synchronous-write wrapper, and a PostgreSQL-backed store.
package repository

import (
	"errors"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

//go:generate go tool mockgen -source=storage.go -destination=mocks/storage_mock.go -package=mocks

// ErrNotFound is returned by Storage.Gauge and Storage.Counter when the
// requested metric has never been recorded.
var ErrNotFound = errors.New("metric not found")

// Storage is the persistence interface used by the metrics server. All
// implementations must be safe for concurrent use.
type Storage interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	Gauge(name string) (float64, error)
	Counter(name string) (int64, error)
	AllGauges() map[string]float64
	AllCounters() map[string]int64
	UpdateBatch(metrics []models.Metrics) error
}
