package metrics

import (
	"context"
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	providerAttr = attribute.String("provider", "azure")
	osTypeAttr   = attribute.String("os_type", runtime.GOOS)
	// set service.name attribute explicitly to the provider name as the default service name is "unknown_service:<binary name>"
	serviceNameAttr = attribute.String("service.name", "csi-secrets-store-provider-azure")
	objectTypeKey   = "object_type"
	objectNameKey   = "object_name"
	errorKey        = "error"
	grpcMethodKey   = "grpc_method"
	grpcCodeKey     = "grpc_code"
	grpcMessageKey  = "grpc_message"
	keyvaultRequest metric.Float64Histogram
	grpcRequest     metric.Float64Histogram
)

type reporter struct {
	meter metric.Meter
}

// StatsReporter is the interface for reporting metrics
type StatsReporter interface {
	ReportKeyvaultRequest(ctx context.Context, duration float64, objectType, objectName, err string)
	ReportGRPCRequest(ctx context.Context, duration float64, method, code, message string)
}

// NewStatsReporter creates a new StatsReporter
func NewStatsReporter() StatsReporter {
	meter := otel.GetMeterProvider().Meter("csi-secrets-store-provider-azure")

	var err error
	keyvaultRequest, err = meter.Float64Histogram("keyvault_request", metric.WithDescription("Distribution of how long it took to get from keyvault"))
	if err != nil {
		panic(err)
	}
	grpcRequest, err = meter.Float64Histogram("grpc_request", metric.WithDescription("Distribution of how long it took for the gRPC requests"))
	if err != nil {
		panic(err)
	}
	return &reporter{meter: meter}
}

// ReportKeyvaultRequest reports the duration of the keyvault request
// objectType and objectName are used to identify the object being accessed
// err is used to identify the error if any
func (r *reporter) ReportKeyvaultRequest(ctx context.Context, duration float64, objectType, objectName, err string) {
	attributes := []attribute.KeyValue{
		serviceNameAttr,
		providerAttr,
		osTypeAttr,
		attribute.String(objectTypeKey, objectType),
		attribute.String(objectNameKey, objectName),
		attribute.String(errorKey, err),
	}
	keyvaultRequest.Record(ctx, duration,
		metric.WithAttributes(attributes...),
	)
}

// ReportGRPCRequest reports the duration of the gRPC request
// method and code are used to identify the gRPC request
func (r *reporter) ReportGRPCRequest(ctx context.Context, duration float64, method, code, message string) {
	attributes := []attribute.KeyValue{
		serviceNameAttr,
		providerAttr,
		osTypeAttr,
		attribute.String(grpcMethodKey, method),
		attribute.String(grpcCodeKey, code),
		attribute.String(grpcMessageKey, message),
	}
	grpcRequest.Record(ctx,
		duration,
		metric.WithAttributes(attributes...),
	)
}
