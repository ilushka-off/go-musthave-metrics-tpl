package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

// failingStorage always fails writes, to verify SyncStorage does not attempt
// to persist to disk when the underlying storage rejects the update.
type failingStorage struct {
	Storage
}

func (f failingStorage) UpdateGauge(name string, value float64) error {
	return errors.New("boom")
}

func (f failingStorage) UpdateCounter(name string, value int64) error {
	return errors.New("boom")
}

func (f failingStorage) UpdateBatch(metrics []models.Metrics) error {
	return errors.New("boom")
}

func TestSyncStorage_UpdateGauge_PersistsToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	s := NewSyncStorage(NewMemStorage(), path)

	if err := s.UpdateGauge("Alloc", 10.5); err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to be written after UpdateGauge, stat error = %v", err)
	}

	restored := NewMemStorage()
	if err := LoadFromFile(restored, path); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if v, err := restored.Gauge("Alloc"); err != nil || v != 10.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 10.5, nil", v, err)
	}
}

func TestSyncStorage_UpdateCounter_PersistsToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	s := NewSyncStorage(NewMemStorage(), path)

	if err := s.UpdateCounter("PollCount", 3); err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	restored := NewMemStorage()
	if err := LoadFromFile(restored, path); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if v, err := restored.Counter("PollCount"); err != nil || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, nil", v, err)
	}
}

func TestSyncStorage_UpdateBatch_PersistsToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	s := NewSyncStorage(NewMemStorage(), path)

	value := 1.5
	if err := s.UpdateBatch([]models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &value}}); err != nil {
		t.Fatalf("UpdateBatch() error = %v", err)
	}

	restored := NewMemStorage()
	if err := LoadFromFile(restored, path); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if v, err := restored.Gauge("Alloc"); err != nil || v != 1.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 1.5, nil", v, err)
	}
}

func TestSyncStorage_UpdateGauge_DoesNotPersistOnUnderlyingError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	s := NewSyncStorage(failingStorage{}, path)

	if err := s.UpdateGauge("Alloc", 10.5); err == nil {
		t.Fatal("UpdateGauge() error = nil; want the underlying storage's error")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("file must not be written when the underlying update fails")
	}
}
