package agent

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/compress"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	pb "github.com/ilushka-off/go-musthave-metrics-tpl/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

type fakeMetricsServer struct {
	pb.UnimplementedMetricsServer
	mu      sync.Mutex
	batches [][]*pb.Metric
	ips     []string
	// block, if set, makes UpdateMetrics hang until the call is cancelled.
	block bool
}

func (f *fakeMetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	md, _ := metadata.FromIncomingContext(ctx)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.batches = append(f.batches, req.GetMetrics())
	f.ips = append(f.ips, md.Get("x-real-ip")...)
	return &pb.UpdateMetricsResponse{}, nil
}

func startFakeGRPC(t *testing.T, fake *fakeMetricsServer) string {
	t.Helper()
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	pb.RegisterMetricsServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestAgent_RunContext_GRPC(t *testing.T) {
	fake := &fakeMetricsServer{}
	addr := startFakeGRPC(t, fake)

	a := NewAgent("http://unused.invalid", 10*time.Millisecond, 50*time.Millisecond, "", 1, nil)
	if err := a.UseGRPC(addr); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		a.RunContext(ctx)
		close(done)
	}()
	time.Sleep(300 * time.Millisecond)
	cancel()
	<-done

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.batches) == 0 {
		t.Fatal("no batches received over gRPC")
	}
	if net.ParseIP(fake.ips[0]) == nil {
		t.Fatalf("x-real-ip = %q, want a valid IP", fake.ips[0])
	}
	var hasPollCount bool
	for _, m := range fake.batches[0] {
		if m.GetId() == "PollCount" && m.GetType() == pb.Metric_COUNTER {
			hasPollCount = true
		}
	}
	if !hasPollCount {
		t.Fatal("PollCount counter missing from gRPC batch")
	}
}

func TestSendMetricsGRPC_AbortsOnCancel(t *testing.T) {
	addr := startFakeGRPC(t, &fakeMetricsServer{block: true})

	a := NewAgent("", time.Second, time.Second, "", 1, nil)
	if err := a.UseGRPC(addr); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.grpcConn.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	if err := sendMetricsGRPC(ctx, a.grpcClient, a.realIP, []models.Metrics{{ID: "PollCount", MType: models.Counter, Delta: new(int64(1))}}); err == nil {
		t.Fatal("expected error for a call cancelled by ctx, got nil")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("call took %v after ctx expired, want it aborted promptly", elapsed)
	}
}

func TestAgent_Accumulate_StopsOnCancel(t *testing.T) {
	a := NewAgent("", time.Second, time.Second, "", 1, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		a.accumulate(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("accumulate did not return after cancel")
	}
}
