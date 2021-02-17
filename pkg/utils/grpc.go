package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

// LogInterceptor is a gRPC interceptor that logs the gRPC requests and responses.
// It also publishes metrics for the gRPC requests.
func LogInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		reporter := metrics.NewStatsReporter()

		ctxDeadline, _ := ctx.Deadline()
		klog.V(5).InfoS("request", "method", info.FullMethod, "deadline", time.Until(ctxDeadline).String())

		resp, err := handler(ctx, req)
		s, _ := status.FromError(err)
		klog.V(5).InfoS("response", "method", info.FullMethod, "duration", time.Since(start).String(), "code", s.Code().String(), "message", s.Message())
		reporter.ReportGRPCRequest(ctx, time.Since(start).Seconds(), info.FullMethod, s.Code().String(), s.Message())

		return resp, err
	}
}
