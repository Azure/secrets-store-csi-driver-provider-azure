package metrics

import (
	"context"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

func initOTLPExporter(ctx context.Context, otlpMetricsGRPCEndpoint string, arcExtensionResourceID string, tlsCertificatePath string) error {
	if otlpMetricsGRPCEndpoint == "" {
		return errors.Errorf("OTLP exporter specified but endpoint not provided")
	}

	tlsCredentials, err := credentials.NewClientTLSFromFile(tlsCertificatePath, "")
	if err != nil {
		return err
	}

	otlpOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(otlpMetricsGRPCEndpoint),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled: false,
		}),
		otlpmetricgrpc.WithTemporalitySelector(func(kind metric.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}),
		otlpmetricgrpc.WithTLSCredentials(tlsCredentials),
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpOpts...)
	if err != nil {
		return err
	}

	metricOpts := []metric.Option{
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	}

	if arcExtensionResourceID != "" {
		metricOpts = append(metricOpts, metric.WithResource(resource.NewSchemaless(
			attribute.String("microsoft.resourceId", arcExtensionResourceID),
		)))
	}

	meterProvider := metric.NewMeterProvider(metricOpts...)
	otel.SetMeterProvider(meterProvider)

	return nil
}
