package metrics

import (
	"fmt"
	"net/http"
	"time"

	crprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	readHeaderTimeout = 5 * time.Second
)

func initPrometheusExporter(port int) error {
	reg := metrics.Registry.(*crprometheus.Registry)

	exporter, err := prometheus.New(prometheus.WithRegisterer(reg))
	if err != nil {
		return err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithView(metric.NewView(
			metric.Instrument{Kind: metric.InstrumentKindHistogram},
			metric.Stream{
				Aggregation: metric.AggregationExplicitBucketHistogram{
					Boundaries: []float64{
						0.1, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 2.5, 3.0, 5.0, 10.0, 15.0, 30.0,
					}}},
		)),
	)

	otel.SetMeterProvider(meterProvider)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	go func() {
		server := &http.Server{
			Addr:              fmt.Sprintf(":%v", port),
			ReadHeaderTimeout: readHeaderTimeout,
		}
		klog.ErrorS(server.ListenAndServe(), "listen and server error")
	}()

	return err
}
