package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository/mocks"
)

// newMockStorage возвращает сгенерированный mockgen'ом MockStorage,
// подключённый к простым in-memory картам для проверки сохранённых значений в тестах.
func newMockStorage(t *testing.T) *mocks.MockStorage {
	ctrl := gomock.NewController(t)
	storage := mocks.NewMockStorage(ctrl)

	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	storage.EXPECT().UpdateGauge(gomock.Any(), gomock.Any()).DoAndReturn(func(name string, value float64) error {
		gauges[name] = value
		return nil
	}).AnyTimes()

	storage.EXPECT().UpdateCounter(gomock.Any(), gomock.Any()).DoAndReturn(func(name string, value int64) error {
		counters[name] += value
		return nil
	}).AnyTimes()

	storage.EXPECT().Gauge(gomock.Any()).DoAndReturn(func(name string) (float64, bool) {
		v, ok := gauges[name]
		return v, ok
	}).AnyTimes()

	storage.EXPECT().Counter(gomock.Any()).DoAndReturn(func(name string) (int64, bool) {
		v, ok := counters[name]
		return v, ok
	}).AnyTimes()

	return storage
}

func doUpdateRequest(h *MetricsHandler, mType, mName, mValue string) *httptest.ResponseRecorder {
	mux := NewRouter(h, zap.NewNop(), nil)
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
			h := NewMetricsHandler(newMockStorage(t), zap.NewNop())
			rec := doUpdateRequest(h, tt.mType, tt.mName, tt.mValue)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestMetricsHandler_Update_StoresValue(t *testing.T) {
	storage := newMockStorage(t)
	h := NewMetricsHandler(storage, zap.NewNop())

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
	h := NewMetricsHandler(newMockStorage(t), zap.NewNop())
	mux := NewRouter(h, zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/Alloc/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
