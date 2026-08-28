package repository

import (
	"encoding/json"
	"os"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

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

	return os.WriteFile(path, data, 0644)
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
				storage.UpdateGauge(m.ID, *m.Value)
			}
		case models.Counter:
			if m.Delta != nil {
				storage.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}
