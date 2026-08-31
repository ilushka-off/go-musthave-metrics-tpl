package repository

import (
	"encoding/json"
	"os"
	"path/filepath"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

func NewFileStorage(path string, restore bool) (Storage, error) {
	storage := NewMemStorage()

	if !restore {
		return storage, nil
	}

	return storage, LoadFromFile(storage, path)
}

func SaveToFile(storage Storage, path string) error {
	metrics := []models.Metrics{}

	for name, value := range storage.AllGauges() {
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		})
	}

	for name, value := range storage.AllCounters() {
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &value,
		})
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}

func LoadFromFile(storage Storage, path string) error {
	var metrics []models.Metrics

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				if err := storage.UpdateGauge(m.ID, *m.Value); err != nil {
					return err
				}
			}
		case models.Counter:
			if m.Delta != nil {
				if err := storage.UpdateCounter(m.ID, *m.Delta); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
