package repository

import models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"

// SyncStorage wraps a Storage and persists its full contents to a file after
// every write, used when the server is configured to save on every update
// rather than on a timer.
type SyncStorage struct {
	Storage
	path string
}

// NewSyncStorage wraps storage so that every update is immediately saved to
// the file at path via SaveToFile.
func NewSyncStorage(storage Storage, path string) *SyncStorage {
	return &SyncStorage{
		Storage: storage,
		path:    path,
	}
}

// UpdateGauge updates the underlying storage and then persists it to disk.
// The file is not written if the underlying update fails.
func (s *SyncStorage) UpdateGauge(name string, value float64) error {
	if err := s.Storage.UpdateGauge(name, value); err != nil {
		return err
	}
	return SaveToFile(s.Storage, s.path)
}

// UpdateCounter updates the underlying storage and then persists it to disk.
// The file is not written if the underlying update fails.
func (s *SyncStorage) UpdateCounter(name string, value int64) error {
	if err := s.Storage.UpdateCounter(name, value); err != nil {
		return err
	}
	return SaveToFile(s.Storage, s.path)
}

// UpdateBatch updates the underlying storage and then persists it to disk.
// The file is not written if the underlying update fails.
func (s *SyncStorage) UpdateBatch(metrics []models.Metrics) error {
	if err := s.Storage.UpdateBatch(metrics); err != nil {
		return err
	}
	return SaveToFile(s.Storage, s.path)
}
