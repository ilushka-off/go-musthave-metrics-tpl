package repository

import (
	"sync"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

// MemStorage is an in-memory, mutex-protected implementation of Storage. It
// does not persist data across process restarts.
type MemStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage creates an empty MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge sets the value of the named gauge, overwriting any previous
// value.
func (s *MemStorage) UpdateGauge(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return nil
}

// UpdateCounter adds value to the named counter's running total.
func (s *MemStorage) UpdateCounter(name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return nil
}

// Gauge returns the current value of the named gauge, or ErrNotFound if it
// has never been set.
func (s *MemStorage) Gauge(name string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.gauges[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

// Counter returns the current value of the named counter, or ErrNotFound if
// it has never been set.
func (s *MemStorage) Counter(name string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.counters[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

// AllGauges returns a snapshot copy of every gauge and its current value.
func (s *MemStorage) AllGauges() map[string]float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]float64, len(s.gauges))
	for name, value := range s.gauges {
		result[name] = value
	}

	return result
}

// AllCounters returns a snapshot copy of every counter and its current
// value.
func (s *MemStorage) AllCounters() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]int64, len(s.counters))
	for name, value := range s.counters {
		result[name] = value
	}
	return result
}

// UpdateBatch applies every metric in metrics in a single locked pass.
// Entries with a nil Value/Delta for their type are silently skipped.
func (s *MemStorage) UpdateBatch(metrics []models.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				continue
			}
			s.gauges[metric.ID] = *metric.Value
		case models.Counter:
			if metric.Delta == nil {
				continue
			}
			s.counters[metric.ID] += *metric.Delta
		}
	}
	return nil
}
