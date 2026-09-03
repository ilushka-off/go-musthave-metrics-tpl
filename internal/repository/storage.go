package repository

import models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"

//go:generate go tool mockgen -source=storage.go -destination=mocks/storage_mock.go -package=mocks

type Storage interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	Gauge(name string) (float64, bool)
	Counter(name string) (int64, bool)
	AllGauges() map[string]float64
	AllCounters() map[string]int64
	UpdateBatch(metrics []models.Metrics) error
}
