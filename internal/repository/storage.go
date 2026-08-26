package repository

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	Gauge(name string) (float64, bool)
	Counter(name string) (int64, bool)
}
