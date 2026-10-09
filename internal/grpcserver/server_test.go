package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	pb "github.com/ilushka-off/go-musthave-metrics-tpl/internal/proto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func start(t *testing.T, subnet string) (pb.MetricsClient, repository.Storage) {
	t.Helper()

	var trusted *net.IPNet
	if subnet != "" {
		var err error
		_, trusted, err = net.ParseCIDR(subnet)
		if err != nil {
			t.Fatal(err)
		}
	}

	log := zap.NewNop()
	storage := repository.NewMemStorage()
	// A real TCP listener (not bufconn) so the interceptor sees an IP peer
	// address: calls always come from 127.0.0.1.
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(TrustedSubnetInterceptor(trusted, log)))
	pb.RegisterMetricsServer(srv, New(storage, log, audit.NewAuditor(log)))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return pb.NewMetricsClient(conn), storage
}

func batch() *pb.UpdateMetricsRequest {
	return &pb.UpdateMetricsRequest{Metrics: []*pb.Metric{
		{Id: "g", Type: pb.Metric_GAUGE, Value: 1.5},
		{Id: "c", Type: pb.Metric_COUNTER, Delta: 3},
	}}
}

func withIP(ip string) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), RealIPKey, ip)
}

func TestUpdateMetrics_StoresBatch(t *testing.T) {
	client, storage := start(t, "")

	if _, err := client.UpdateMetrics(context.Background(), batch()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateMetrics(context.Background(), batch()); err != nil {
		t.Fatal(err)
	}

	if v, err := storage.Gauge("g"); err != nil || v != 1.5 {
		t.Fatalf("gauge = %v, %v", v, err)
	}
	if v, err := storage.Counter("c"); err != nil || v != 6 {
		t.Fatalf("counter = %v, %v (counters must accumulate)", v, err)
	}
}

func TestUpdateMetrics_EmptyAndInvalid(t *testing.T) {
	client, _ := start(t, "")

	if _, err := client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{}); err != nil {
		t.Fatalf("empty batch: %v", err)
	}

	_, err := client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{{Id: "x", Type: pb.Metric_MType(42)}},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", err)
	}
}

func TestTrustedSubnetInterceptor(t *testing.T) {
	// The peer address is always 127.0.0.1; only it decides, x-real-ip does not.
	tests := []struct {
		name   string
		subnet string
		ctx    context.Context
		want   codes.Code
	}{
		{"peer inside subnet", "127.0.0.0/8", withIP("127.0.0.1"), codes.OK},
		{"peer inside subnet, missing metadata", "127.0.0.0/8", context.Background(), codes.OK},
		{"peer inside subnet, mismatched x-real-ip", "127.0.0.0/8", withIP("10.0.0.1"), codes.OK},
		{"peer outside subnet", "192.168.1.0/24", context.Background(), codes.PermissionDenied},
		{"peer outside subnet, forged x-real-ip", "192.168.1.0/24", withIP("192.168.1.7"), codes.PermissionDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := start(t, tt.subnet)
			_, err := client.UpdateMetrics(tt.ctx, batch())
			if got := status.Code(err); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
