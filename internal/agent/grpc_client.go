package agent

import (
	"context"
	"fmt"
	"time"

	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	pb "github.com/ilushka-off/go-musthave-metrics-tpl/internal/proto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// grpcCallTimeout bounds a single UpdateMetrics attempt. It is independent of
// the agent's run context so the final batch can still be delivered while the
// agent is shutting down.
const grpcCallTimeout = 5 * time.Second

func dialGRPC(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}
	return conn, nil
}

func sendMetricsGRPC(client pb.MetricsClient, address string, metrics []models.Metrics) error {
	req := &pb.UpdateMetricsRequest{Metrics: make([]*pb.Metric, 0, len(metrics))}
	for _, m := range metrics {
		pm := &pb.Metric{Id: m.ID}
		switch m.MType {
		case models.Gauge:
			pm.Type = pb.Metric_GAUGE
			if m.Value != nil {
				pm.Value = *m.Value
			}
		case models.Counter:
			pm.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				pm.Delta = *m.Delta
			}
		default:
			return fmt.Errorf("unknown metric type %q for %q", m.MType, m.ID)
		}
		req.Metrics = append(req.Metrics, pm)
	}

	realIP := localIP("grpc://" + address)

	return retry.Do(retry.Delays, isGRPCRetriable, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), grpcCallTimeout)
		defer cancel()

		ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", realIP)
		if _, err := client.UpdateMetrics(ctx, req); err != nil {
			return fmt.Errorf("grpc update metrics: %w", err)
		}
		return nil
	})
}

func isGRPCRetriable(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}
