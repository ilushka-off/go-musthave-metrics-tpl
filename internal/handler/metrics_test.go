package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
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

	storage.EXPECT().Gauge(gomock.Any()).DoAndReturn(func(name string) (float64, error) {
		v, ok := gauges[name]
		if !ok {
			return 0, repository.ErrNotFound
		}
		return v, nil
	}).AnyTimes()

	storage.EXPECT().Counter(gomock.Any()).DoAndReturn(func(name string) (int64, error) {
		v, ok := counters[name]
		if !ok {
			return 0, repository.ErrNotFound
		}
		return v, nil
	}).AnyTimes()

	storage.EXPECT().UpdateBatch(gomock.Any()).DoAndReturn(func(metrics []models.Metrics) error {
		for _, m := range metrics {
			switch m.MType {
			case models.Gauge:
				if m.Value != nil {
					gauges[m.ID] = *m.Value
				}
			case models.Counter:
				if m.Delta != nil {
					counters[m.ID] += *m.Delta
				}
			}
		}
		return nil
	}).AnyTimes()

	return storage
}

func doUpdateBatchRequest(h *MetricsHandler, body string) *httptest.ResponseRecorder {
	mux := NewRouter(h, zap.NewNop(), nil)
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
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
	if v, err := storage.Gauge("Alloc"); err != nil || v != 123.45 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 123.45, nil", v, err)
	}

	doUpdateRequest(h, "counter", "PollCount", "1")
	doUpdateRequest(h, "counter", "PollCount", "2")
	if v, err := storage.Counter("PollCount"); err != nil || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, nil", v, err)
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

func TestMetricsHandler_UpdateBatch_StoresValues(t *testing.T) {
	storage := newMockStorage(t)
	h := NewMetricsHandler(storage, zap.NewNop())

	body := `[
		{"id":"Alloc","type":"gauge","value":123.45},
		{"id":"PollCount","type":"counter","delta":1},
		{"id":"PollCount","type":"counter","delta":2}
	]`

	rec := doUpdateBatchRequest(h, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}

	if v, err := storage.Gauge("Alloc"); err != nil || v != 123.45 {
		t.Fatalf("Gauge(Alloc) = %v, %v; want 123.45, nil", v, err)
	}
	if v, err := storage.Counter("PollCount"); err != nil || v != 3 {
		t.Fatalf("Counter(PollCount) = %v, %v; want 3, nil", v, err)
	}
}

func TestMetricsHandler_UpdateBatch_Empty(t *testing.T) {
	h := NewMetricsHandler(newMockStorage(t), zap.NewNop())

	rec := doUpdateBatchRequest(h, `[]`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusOK)
	}
}

func TestMetricsHandler_UpdateBatch_InvalidJSON(t *testing.T) {
	h := NewMetricsHandler(newMockStorage(t), zap.NewNop())

	rec := doUpdateBatchRequest(h, `not-json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", rec.Code, http.StatusBadRequest)
	}
}
