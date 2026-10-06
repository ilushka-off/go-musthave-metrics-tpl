package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/compress"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

func TestNewAgent_ClampsInvalidRateLimit(t *testing.T) {
	for _, rl := range []int{0, -1, -100} {
		a := NewAgent("http://localhost:8080", time.Second, time.Second, "", rl, nil)
		if a.rateLimit != 1 {
			t.Fatalf("rateLimit=%d -> a.rateLimit=%d, want 1 (0 или отрицательный лимит не должен оставлять агента без воркеров)", rl, a.rateLimit)
		}
	}
}

func TestAgent_Run_CollectsAndSends(t *testing.T) {
	var mu sync.Mutex
	var requestCount int
	var lastMetrics []models.Metrics

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader, err := compress.NewReader(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = reader.Close() }()

		var metrics []models.Metrics
		if err := json.NewDecoder(reader).Decode(&metrics); err != nil {
			t.Error(err)
			return
		}

		mu.Lock()
		requestCount++
		lastMetrics = metrics
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 20*time.Millisecond, 50*time.Millisecond, "", 2, nil)
	go a.Run()

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if requestCount == 0 {
		t.Fatal("agent did not send any batch")
	}

	var hasPollCount, hasRuntimeGauge, hasGopsutilGauge bool
	for _, m := range lastMetrics {
		switch m.ID {
		case "PollCount":
			hasPollCount = true
		case "Alloc":
			hasRuntimeGauge = true
		case "TotalMemory":
			hasGopsutilGauge = true
		}
	}
	if !hasPollCount {
		t.Error("в батче нет PollCount")
	}
	if !hasRuntimeGauge {
		t.Error("в батче нет runtime-метрики (Alloc)")
	}
	if !hasGopsutilGauge {
		t.Error("в батче нет gopsutil-метрики (TotalMemory)")
	}
}

func TestAgent_Run_RespectsRateLimit(t *testing.T) {
	var mu sync.Mutex
	var inFlight, maxInFlight int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()

		time.Sleep(30 * time.Millisecond) // имитируем медленный сервер

		mu.Lock()
		inFlight--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	const rateLimit = 2
	a := NewAgent(server.URL, 5*time.Millisecond, 5*time.Millisecond, "", rateLimit, nil)
	go a.Run()

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if maxInFlight > rateLimit {
		t.Fatalf("одновременных запросов было %d, лимит %d", maxInFlight, rateLimit)
	}
	if maxInFlight == 0 {
		t.Fatal("не зафиксировано ни одного запроса")
	}
	t.Logf("максимум одновременных запросов: %d (лимит %d)", maxInFlight, rateLimit)
}

func TestAgent_RunContext_FlushesOnCancel(t *testing.T) {
	var mu sync.Mutex
	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		requests++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Report interval is far longer than the test, so the only batch that can
	// reach the server is the final one sent on cancellation.
	a := NewAgent(server.URL, 10*time.Millisecond, time.Hour, "", 1, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		a.RunContext(ctx)
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunContext did not return after cancel")
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("expected exactly one final batch delivered, got %d", requests)
	}
}
