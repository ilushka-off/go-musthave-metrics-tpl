package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockStorage — простая реализация repository.Storage для теста хендлера в изоляции от MemStorage.
type mockStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *mockStorage) UpdateGauge(name string, value float64) { m.gauges[name] = value }
func (m *mockStorage) UpdateCounter(name string, value int64) { m.counters[name] += value }
func (m *mockStorage) Gauge(name string) (float64, bool)      { v, ok := m.gauges[name]; return v, ok }
func (m *mockStorage) Counter(name string) (int64, bool)      { v, ok := m.counters[name]; return v, ok }
func (m *mockStorage) AllGauges() map[string]float64          { return m.gauges }
func (m *mockStorage) AllCounters() map[string]int64          { return m.counters }

func doUpdateRequest(h *MetricsHandler, mType, mName, mValue string) *httptest.ResponseRecorder {
	mux := NewRouter(h)
	req := httptest.NewRequest(http.MethodPost, "/update/"+mType+"/"+mName+"/"+mValue, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestMetricsHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		mType      string
		mName      string
		mValue     string
		wantStatus int
	}{
		{"valid gauge", "gauge", "Alloc", "123.45", http.StatusOK},
		{"valid counter", "counter", "PollCount", "1", http.StatusOK},
		{"unknown type", "unknown", "Foo", "1", http.StatusBadRequest},
		{"invalid gauge value", "gauge", "Foo", "notanumber", http.StatusBadRequest},
		{"invalid counter value", "counter", "Foo", "notanumber", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricsHandler(newMockStorage())
			rec := doUpdateRequest(h, tt.mType, tt.mName, tt.mValue)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestMetricsHandler_Update_StoresValue(t *testing.T) {
	storage := newMockStorage()
	h := NewMetricsHandler(storage)

	doUpdateRequest(h, "gauge", "Alloc", "123.45")
	if v, ok := storage.Gauge("Alloc"); !ok || v != 123.45 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 123.45, true", v, ok)
	}

	doUpdateRequest(h, "counter", "PollCount", "1")
	doUpdateRequest(h, "counter", "PollCount", "2")
	if v, ok := storage.Counter("PollCount"); !ok || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, true", v, ok)
	}
}

func TestMetricsHandler_Update_WrongMethod(t *testing.T) {
	h := NewMetricsHandler(newMockStorage())
	mux := NewRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/Alloc/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
