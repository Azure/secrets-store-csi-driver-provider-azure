package metrics

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

const prometheusExporter = "prometheus"
const otlpExporter = "otlp"

func InitMetricsExporter(metricsBackend string, prometheusPort int) error {
	mb := strings.ToLower(metricsBackend)
	klog.InfoS("intializing metrics backend", "backend", mb)
	switch mb {
	// Prometheus is the only exporter for now
	case prometheusExporter:
		return initPrometheusExporter(prometheusPort)
	case otlpExporter:
		return initOTLPExporter()
	default:
		return fmt.Errorf("unsupported metrics backend %v", metricsBackend)
	}
}
