package repository

import (
	"errors"
	"testing"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
	s := NewMemStorage()

	s.UpdateGauge("Alloc", 10.5)
	v, err := s.Gauge("Alloc")
	if err != nil || v != 10.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 10.5, nil", v, err)
	}

	// повторное обновление должно перезаписывать значение, а не складывать
	s.UpdateGauge("Alloc", 20)
	v, err = s.Gauge("Alloc")
	if err != nil || v != 20 {
		t.Fatalf("Gauge(Alloc) after overwrite = %v, %v; want 20, nil", v, err)
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	s := NewMemStorage()

	s.UpdateCounter("PollCount", 1)
	s.UpdateCounter("PollCount", 2)

	v, err := s.Counter("PollCount")
	if err != nil || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, nil", v, err)
	}
}

func TestMemStorage_UpdateBatch(t *testing.T) {
	s := NewMemStorage()

	gaugeValue := 10.5
	firstDelta := int64(1)
	secondDelta := int64(2)

	err := s.UpdateBatch([]models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gaugeValue},
		{ID: "PollCount", MType: models.Counter, Delta: &firstDelta},
		{ID: "PollCount", MType: models.Counter, Delta: &secondDelta},
		{ID: "BadGauge", MType: models.Gauge, Value: nil},
		{ID: "BadCounter", MType: models.Counter, Delta: nil},
	})
	if err != nil {
		t.Fatalf("UpdateBatch returned error: %v", err)
	}

	v, err := s.Gauge("Alloc")
	if err != nil || v != 10.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 10.5, nil", v, err)
	}

	c, err := s.Counter("PollCount")
	if err != nil || c != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, nil", c, err)
	}

	if _, err := s.Gauge("BadGauge"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Gauge(BadGauge) err = %v; want ErrNotFound (nil Value must be skipped)", err)
	}
	if _, err := s.Counter("BadCounter"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Counter(BadCounter) err = %v; want ErrNotFound (nil Delta must be skipped)", err)
	}
}

func TestMemStorage_MissingKey(t *testing.T) {
	s := NewMemStorage()

	if _, err := s.Gauge("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Gauge(missing) err = %v; want ErrNotFound", err)
	}
	if _, err := s.Counter("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Counter(missing) err = %v; want ErrNotFound", err)
	}
}
