package repository

import (
	"errors"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

//go:generate go tool mockgen -source=storage.go -destination=mocks/storage_mock.go -package=mocks

var ErrNotFound = errors.New("metric not found")

type Storage interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	Gauge(name string) (float64, error)
	Counter(name string) (int64, error)
	AllGauges() map[string]float64
	AllCounters() map[string]int64
	UpdateBatch(metrics []models.Metrics) error
}
