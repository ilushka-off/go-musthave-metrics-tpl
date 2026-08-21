package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgent_Poll(t *testing.T) {
	a := NewAgent("http://localhost:8080", time.Second, time.Second)

	a.poll()
	if len(a.gauges) == 0 {
		t.Fatal("poll() did not populate gauges")
	}
	if a.pollCount != 1 {
		t.Fatalf("pollCount = %d, want 1", a.pollCount)
	}

	a.poll()
	if a.pollCount != 2 {
		t.Fatalf("pollCount = %d, want 2", a.pollCount)
	}
}

func TestAgent_Report(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, time.Second, time.Second)
	a.poll()
	wantRequests := len(a.gauges) + 1 // все gauges + один counter PollCount

	a.report()

	if requestCount != wantRequests {
		t.Fatalf("requestCount = %d, want %d", requestCount, wantRequests)
	}
	if a.pollCount != 0 {
		t.Fatalf("pollCount after report = %d, want 0 (must reset after sending)", a.pollCount)
	}
}
