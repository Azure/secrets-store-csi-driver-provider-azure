package metrics

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"k8s.io/klog/v2"
)

const (
	readHeaderTimeout = 5 * time.Second
)

func initPrometheusExporter(port int) error {
	pusher, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{
			0.1, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 2.5, 3.0, 5.0, 10.0, 15.0, 30.0,
		}})
	if err != nil {
		return err
	}
	http.HandleFunc("/metrics", pusher.ServeHTTP)
	go func() {
		server := &http.Server{
			Addr:              fmt.Sprintf(":%v", port),
			ReadHeaderTimeout: readHeaderTimeout,
		}
		klog.ErrorS(server.ListenAndServe(), "listen and server error")
	}()

	return err
}
