package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileStorage_NoRestore_StartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	s, err := NewFileStorage(path, false)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	if len(s.AllGauges()) != 0 || len(s.AllCounters()) != 0 {
		t.Fatal("NewFileStorage(restore=false) should start empty and must not read the file")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("NewFileStorage(restore=false) must not create the file")
	}
}

func TestNewFileStorage_Restore_MissingFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	if _, err := NewFileStorage(path, true); err == nil {
		t.Fatal("NewFileStorage(restore=true) error = nil; want non-nil for a missing file")
	}
}

func TestSaveToFile_ThenLoadFromFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	original := NewMemStorage()
	original.UpdateGauge("Alloc", 10.5)
	original.UpdateCounter("PollCount", 3)

	if err := SaveToFile(original, path); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	restored := NewMemStorage()
	if err := LoadFromFile(restored, path); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if v, err := restored.Gauge("Alloc"); err != nil || v != 10.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 10.5, nil", v, err)
	}
	if v, err := restored.Counter("PollCount"); err != nil || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, nil", v, err)
	}
}

func TestNewFileStorage_Restore_LoadsExistingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	seed := NewMemStorage()
	seed.UpdateGauge("Alloc", 42)
	if err := SaveToFile(seed, path); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	s, err := NewFileStorage(path, true)
	if err != nil {
		t.Fatalf("NewFileStorage(restore=true) error = %v", err)
	}

	if v, err := s.Gauge("Alloc"); err != nil || v != 42 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 42, nil", v, err)
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	if err := os.WriteFile(path, []byte("not-json"), 0644); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := LoadFromFile(NewMemStorage(), path); err == nil {
		t.Fatal("LoadFromFile() error = nil; want non-nil for invalid JSON")
	}
}
