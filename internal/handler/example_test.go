package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

// newExampleRouter wires a MetricsHandler and PingHandler-less router backed
// by a fresh in-memory storage, for use by the Example functions below.
func newExampleRouter() http.Handler {
	storage := repository.NewMemStorage()
	h := NewMetricsHandler(storage, zap.NewNop(), audit.NewAuditor(zap.NewNop()))
	return NewRouter(h, zap.NewNop(), nil, "")
}

// ExampleMetricsHandler_Update demonstrates updating a gauge metric by
// passing its type, name and value in the URL path.
func ExampleMetricsHandler_Update() {
	router := newExampleRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output:
	// 200
}

// ExampleMetricsHandler_UpdateJSON demonstrates updating a metric by sending
// a JSON-encoded models.Metrics in the request body. The server echoes back
// the stored metric.
func ExampleMetricsHandler_UpdateJSON() {
	router := newExampleRouter()

	body := `{"id":"Alloc","type":"gauge","value":123.45}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())
	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":123.45}
}

// ExampleMetricsHandler_UpdateBatch demonstrates updating several metrics at
// once by sending a JSON array of models.Metrics to /updates.
func ExampleMetricsHandler_UpdateBatch() {
	router := newExampleRouter()

	body := `[
		{"id":"Alloc","type":"gauge","value":123.45},
		{"id":"PollCount","type":"counter","delta":1},
		{"id":"PollCount","type":"counter","delta":2}
	]`
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output:
	// 200
}

// ExampleMetricsHandler_Value demonstrates reading back a previously stored
// metric's value as plain text.
func ExampleMetricsHandler_Value() {
	router := newExampleRouter()

	seed := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	router.ServeHTTP(httptest.NewRecorder(), seed)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Body.String())
	// Output:
	// 123.45
}

// ExampleMetricsHandler_ValueJSON demonstrates reading back a previously
// stored metric by sending its id and type as JSON to /value.
func ExampleMetricsHandler_ValueJSON() {
	router := newExampleRouter()

	seed := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	router.ServeHTTP(httptest.NewRecorder(), seed)

	body := `{"id":"Alloc","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Body.String())
	// Output:
	// {"id":"Alloc","type":"gauge","value":123.45}
}

// ExampleMetricsHandler_Index demonstrates the HTML page listing every
// stored metric.
func ExampleMetricsHandler_Index() {
	router := newExampleRouter()

	seed := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	router.ServeHTTP(httptest.NewRecorder(), seed)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Body.String())
	// Output:
	// <p>Alloc: 123.45</p>
}

// ExamplePingHandler_Ping demonstrates the database health-check endpoint.
// With no database configured, the handler responds with 503.
func ExamplePingHandler_Ping() {
	h := NewPingHandler(nil, zap.NewNop())
	router := NewRouter(NewMetricsHandler(repository.NewMemStorage(), zap.NewNop(), audit.NewAuditor(zap.NewNop())), zap.NewNop(), h, "")

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	// Output:
	// 503
}
