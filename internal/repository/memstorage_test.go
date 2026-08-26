package repository

import "testing"

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

func TestMemStorage_MissingKey(t *testing.T) {
	s := NewMemStorage()

	if _, ok := s.Gauge("missing"); ok {
		t.Fatal("Gauge(missing) ok = true; want false")
	}
	if _, ok := s.Counter("missing"); ok {
		t.Fatal("Counter(missing) ok = true; want false")
	}
}
