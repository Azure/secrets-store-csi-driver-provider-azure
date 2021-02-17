package metrics

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"k8s.io/klog/v2"
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
		klog.ErrorS(http.ListenAndServe(fmt.Sprintf(":%v", port), nil), "listen and server error")
	}()

	return err
}
