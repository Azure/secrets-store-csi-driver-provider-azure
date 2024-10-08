package metrics

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

const prometheusExporter = "prometheus"
const otlpExporter = "otlp"

func InitMetricsExporter(ctx context.Context, metricsBackend string, prometheusPort int, otlpMetricsGRPCEndpoint string, arcExtensionResourceID string, tlsCertificatePath string) error {
	mb := strings.ToLower(metricsBackend)
	klog.InfoS("intializing metrics backend", "backend", mb)
	switch mb {
	// Prometheus is the only exporter for now
	case prometheusExporter:
		return initPrometheusExporter(prometheusPort)
	case otlpExporter:
		return initOTLPExporter(ctx, otlpMetricsGRPCEndpoint, arcExtensionResourceID, tlsCertificatePath)
	default:
		return fmt.Errorf("unsupported metrics backend %v", metricsBackend)
	}
}
