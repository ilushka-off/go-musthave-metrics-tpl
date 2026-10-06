// Package grpcserver exposes the metrics storage over gRPC: the Metrics
// service implementation and the interceptor that restricts callers to the
// trusted subnet.
package grpcserver

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/audit"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
	pb "github.com/ilushka-off/go-musthave-metrics-tpl/internal/proto"
	"github.com/ilushka-off/go-musthave-metrics-tpl/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RealIPKey is the metadata key under which the agent passes its IP address.
const RealIPKey = "x-real-ip"

// Server implements pb.MetricsServer on top of a repository.Storage.
type Server struct {
	pb.UnimplementedMetricsServer
	storage repository.Storage
	log     *zap.Logger
	auditor *audit.Auditor
}

// New creates a Server. auditor may have no observers attached.
func New(s repository.Storage, log *zap.Logger, auditor *audit.Auditor) *Server {
	return &Server{storage: s, log: log, auditor: auditor}
}

// UpdateMetrics stores a batch of metrics. It returns InvalidArgument for an
// unknown metric type and Internal if the storage fails.
func (s *Server) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	in := req.GetMetrics()
	if len(in) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	metrics := make([]models.Metrics, 0, len(in))
	names := make([]string, 0, len(in))
	for _, m := range in {
		converted := models.Metrics{ID: m.GetId()}
		switch m.GetType() {
		case pb.Metric_GAUGE:
			converted.MType = models.Gauge
			v := m.GetValue()
			converted.Value = &v
		case pb.Metric_COUNTER:
			converted.MType = models.Counter
			d := m.GetDelta()
			converted.Delta = &d
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unknown metric type %v for %q", m.GetType(), m.GetId())
		}
		metrics = append(metrics, converted)
		names = append(names, m.GetId())
	}

	if err := s.storage.UpdateBatch(metrics); err != nil {
		s.log.Error("failed to update metrics", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update metrics")
	}

	s.auditor.Notify(audit.Event{
		IPAddress: peerHost(ctx),
		Metrics:   names,
		Timestamp: time.Now().Unix(),
	})

	return &pb.UpdateMetricsResponse{}, nil
}

func peerHost(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}

// TrustedSubnetInterceptor returns a unary interceptor that allows only calls
// whose "x-real-ip" metadata holds an IP address inside subnet; any other
// call (missing, malformed or foreign address) fails with PermissionDenied.
// If subnet is nil, the interceptor lets every call through.
func TrustedSubnetInterceptor(subnet *net.IPNet, log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if subnet == nil {
			return handler(ctx, req)
		}

		var value string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(RealIPKey); len(vals) > 0 {
				value = strings.TrimSpace(vals[0])
			}
		}

		ip := net.ParseIP(value)
		if ip == nil || !subnet.Contains(ip) {
			log.Warn("grpc call from untrusted address rejected", zap.String("x_real_ip", value))
			return nil, status.Error(codes.PermissionDenied, "address is not in the trusted subnet")
		}

		return handler(ctx, req)
	}
}
