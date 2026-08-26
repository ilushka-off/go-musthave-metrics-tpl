package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
