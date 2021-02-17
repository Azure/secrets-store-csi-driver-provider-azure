package provider

import (
	"context"
	"runtime"

	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	providerLabel         = label.String("provider", "azure")
	osTypeLabel           = label.String("os_type", runtime.GOOS)
	objectTypeKey         = "object_type"
	keyvaultGetTotal      metric.Int64Counter
	keyvaultGetErrorTotal metric.Int64Counter
	keyvaultGetDuration   metric.Float64ValueRecorder
)

type reporter struct {
	meter metric.Meter
}

type StatsReporter interface {
	ReportKeyvaultGetCtMetric(objectType string)
	ReportKeyvaultGetErrorCtMetric(objectType string)
	ReportKeyvaultGetDuration(duration float64)
}

func NewStatsReporter() StatsReporter {
	meter := global.Meter("csi-secrets-store-provider-azure")
	keyvaultGetTotal = metric.Must(meter).NewInt64Counter("total_keyvault_get", metric.WithDescription("Total number of GET from keyvault"))
	keyvaultGetErrorTotal = metric.Must(meter).NewInt64Counter("total_keyvault_get_error", metric.WithDescription("Total number of GET from keyvault with error"))
	keyvaultGetDuration = metric.Must(meter).NewFloat64ValueRecorder("keyvault_get_duration_sec", metric.WithDescription("Distribution of how long it took to get from keyvault"))
	return &reporter{meter: meter}
}

func (r *reporter) ReportKeyvaultGetCtMetric(objectType string) {
	labels := []label.KeyValue{providerLabel, osTypeLabel, label.String(objectTypeKey, objectType)}
	keyvaultGetTotal.Add(context.Background(), 1, labels...)
}

func (r *reporter) ReportKeyvaultGetErrorCtMetric(objectType string) {
	labels := []label.KeyValue{providerLabel, osTypeLabel, label.String(objectTypeKey, objectType)}
	keyvaultGetErrorTotal.Add(context.Background(), 1, labels...)
}

func (r *reporter) ReportKeyvaultGetDuration(duration float64) {
	r.meter.RecordBatch(context.Background(), []label.KeyValue{providerLabel, osTypeLabel}, keyvaultGetDuration.Measurement(duration))
}
