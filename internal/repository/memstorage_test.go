package repository

import (
	"testing"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
	s := NewMemStorage()

	s.UpdateGauge("Alloc", 10.5)
	v, ok := s.Gauge("Alloc")
	if !ok || v != 10.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 10.5, true", v, ok)
	}

	// повторное обновление должно перезаписывать значение, а не складывать
	s.UpdateGauge("Alloc", 20)
	v, ok = s.Gauge("Alloc")
	if !ok || v != 20 {
		t.Fatalf("Gauge(Alloc) after overwrite = %v, %v; want 20, true", v, ok)
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	s := NewMemStorage()

	s.UpdateCounter("PollCount", 1)
	s.UpdateCounter("PollCount", 2)

	v, ok := s.Counter("PollCount")
	if !ok || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, true", v, ok)
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

	v, ok := s.Gauge("Alloc")
	if !ok || v != 10.5 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 10.5, true", v, ok)
	}

	c, ok := s.Counter("PollCount")
	if !ok || c != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, true", c, ok)
	}

	if _, ok := s.Gauge("BadGauge"); ok {
		t.Fatal("Gauge(BadGauge) ok = true; want false (nil Value must be skipped)")
	}
	if _, ok := s.Counter("BadCounter"); ok {
		t.Fatal("Counter(BadCounter) ok = true; want false (nil Delta must be skipped)")
	}
}

func TestMemStorage_MissingKey(t *testing.T) {
	s := NewMemStorage()

	if _, ok := s.Gauge("missing"); ok {
		t.Fatal("Gauge(missing) ok = true; want false")
	}
	if _, ok := s.Counter("missing"); ok {
		t.Fatal("Counter(missing) ok = true; want false")
	}
}
