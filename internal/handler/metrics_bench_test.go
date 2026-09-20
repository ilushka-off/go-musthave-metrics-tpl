package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

// newBenchHandler wires a MetricsHandler to a real in-memory storage (not a
// gomock) so the benchmark measures the handler's own work, not mock overhead.
func newBenchHandler() *MetricsHandler {
	storage := repository.NewMemStorage()
	return NewMetricsHandler(storage, zap.NewNop(), audit.NewAuditor(zap.NewNop()))
}

func BenchmarkMetricsHandler_Update(b *testing.B) {
	h := newBenchHandler()
	mux := NewRouter(h, zap.NewNop(), nil, "")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsHandler_UpdateJSON(b *testing.B) {
	h := newBenchHandler()
	mux := NewRouter(h, zap.NewNop(), nil, "")
	body := `{"id":"Alloc","type":"gauge","value":123.45}`
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsHandler_UpdateBatch(b *testing.B) {
	h := newBenchHandler()
	mux := NewRouter(h, zap.NewNop(), nil, "")

	const batchSize = 100
	metrics := make([]models.Metrics, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		if i%2 == 0 {
			value := float64(i)
			metrics = append(metrics, models.Metrics{ID: fmt.Sprintf("Gauge%d", i), MType: models.Gauge, Value: &value})
		} else {
			delta := int64(i)
			metrics = append(metrics, models.Metrics{ID: fmt.Sprintf("Counter%d", i), MType: models.Counter, Delta: &delta})
		}
	}
	body, err := json.Marshal(metrics)
	if err != nil {
		b.Fatalf("failed to marshal batch: %v", err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsHandler_Value(b *testing.B) {
	h := newBenchHandler()
	mux := NewRouter(h, zap.NewNop(), nil, "")
	seed := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	mux.ServeHTTP(httptest.NewRecorder(), seed)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsHandler_ValueJSON(b *testing.B) {
	h := newBenchHandler()
	mux := NewRouter(h, zap.NewNop(), nil, "")
	seed := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	mux.ServeHTTP(httptest.NewRecorder(), seed)
	body := `{"id":"Alloc","type":"gauge"}`
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsHandler_Index(b *testing.B) {
	h := newBenchHandler()
	mux := NewRouter(h, zap.NewNop(), nil, "")
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/update/gauge/Metric%d/%d.5", i, i), nil)
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}
