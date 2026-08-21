package repository

import "sync"

type MemStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MemStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

func (s *MemStorage) Gauge(name string) (float64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.gauges[name]
	return v, ok
}

func (s *MemStorage) Counter(name string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.counters[name]
	return v, ok
}
