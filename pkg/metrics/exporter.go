package metrics

import (
	"flag"
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

var (
	metricsBackend = flag.String("metrics-backend", "Prometheus", "Backend used for metrics")
	prometheusPort = flag.Int("prometheus-port", 8888, "Prometheus port for metrics backend [DEPRECATED]. Use --metrics-addr instead.")
)

const prometheusExporter = "prometheus"

func InitMetricsExporter() error {
	mb := strings.ToLower(*metricsBackend)
	klog.Infof("metrics backend: %s", mb)
	switch mb {
	// Prometheus is the only exporter for now
	case prometheusExporter:
		return initPrometheusExporter()
	default:
		return fmt.Errorf("unsupported metrics backend %v", *metricsBackend)
	}
}
