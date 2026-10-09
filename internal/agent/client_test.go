package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/compress"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

func TestSendMetrics(t *testing.T) {
	var gotMethod, gotPath, gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := sendMetrics(server.URL, "gauge", "Alloc", "123.45"); err != nil {
		t.Fatalf("sendMetrics returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/update/gauge/Alloc/123.45" {
		t.Errorf("path = %q, want %q", gotPath, "/update/gauge/Alloc/123.45")
	}
	if gotContentType != "text/plain" {
		t.Errorf("Content-Type = %q, want %q", gotContentType, "text/plain")
	}
}

func TestSendMetrics_ConnectionError(t *testing.T) {
	// порт 0 никогда не слушается — соединение должно оборваться ошибкой
	if err := sendMetrics("http://127.0.0.1:0", "gauge", "Foo", "1"); err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestSendMetricsBatch(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotContentEncoding string
	var gotMetrics []models.Metrics

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotContentEncoding = r.Header.Get("Content-Encoding")

		reader, err := compress.NewReader(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = reader.Close() }()

		if err := json.NewDecoder(reader).Decode(&gotMetrics); err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	gaugeValue := 42.5
	delta := int64(7)
	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gaugeValue},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
	}

	if err := sendMetricsBatch(context.Background(), server.URL, "127.0.0.1", metrics, "", nil); err != nil {
		t.Fatalf("sendMetricsBatch returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/updates" {
		t.Errorf("path = %q, want %q", gotPath, "/updates")
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", gotContentType, "application/json")
	}
	if gotContentEncoding != "gzip" {
		t.Errorf("Content-Encoding = %q, want %q", gotContentEncoding, "gzip")
	}
	if len(gotMetrics) != 2 {
		t.Fatalf("got %d metrics, want 2", len(gotMetrics))
	}
}

func TestSendMetricsBatch_ConnectionError(t *testing.T) {
	if err := sendMetricsBatch(context.Background(), "http://127.0.0.1:0", "127.0.0.1", []models.Metrics{{ID: "Foo", MType: models.Gauge}}, "", nil); err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}

func TestIsConnRetriable_TrueForDialFailure(t *testing.T) {
	_, err := net.Dial("tcp", "127.0.0.1:1") // порт 1 закрыт — connection refused
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
	if !isConnRetriable(err) {
		t.Fatalf("expected isConnRetriable=true for dial failure, err=%v", err)
	}
}

func TestIsConnRetriable_FalseForUnrelatedError(t *testing.T) {
	if isConnRetriable(errors.New("unexpected status code 500")) {
		t.Fatal("expected isConnRetriable=false for a plain non-network error")
	}
}

func TestIsConnRetriable_FalseForNil(t *testing.T) {
	if isConnRetriable(nil) {
		t.Fatal("expected isConnRetriable=false for nil error")
	}
}

func TestSendMetricsBatch_SetsXRealIP(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Real-IP")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := sendMetricsBatch(context.Background(), server.URL, "10.1.2.3", []models.Metrics{{ID: "Foo", MType: models.Gauge}}, "", nil); err != nil {
		t.Fatal(err)
	}
	if got != "10.1.2.3" {
		t.Fatalf("X-Real-IP = %q, want %q", got, "10.1.2.3")
	}
}

func TestLocalIP_ReturnsValidIP(t *testing.T) {
	for _, hostPort := range []string{"127.0.0.1:8080", httpHostPort("http://localhost"), ""} {
		if got := localIP(hostPort); net.ParseIP(got) == nil {
			t.Fatalf("localIP(%q) = %q, want a valid IP", hostPort, got)
		}
	}
}

func TestHTTPHostPort(t *testing.T) {
	tests := map[string]string{
		"http://localhost:8080": "localhost:8080",
		"http://localhost":      "localhost:80",
		"https://example.com":   "example.com:443",
		"localhost:8080":        "",
		"":                      "",
	}
	for in, want := range tests {
		if got := httpHostPort(in); got != want {
			t.Errorf("httpHostPort(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGRPCHostPort(t *testing.T) {
	tests := map[string]string{
		"localhost:3200": "localhost:3200",
		"localhost":      "localhost:443",
		"[::1]:3200":     "[::1]:3200",
	}
	for in, want := range tests {
		if got := grpcHostPort(in); got != want {
			t.Errorf("grpcHostPort(%q) = %q, want %q", in, got, want)
		}
	}
}
