/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	otlpMetricsEndpointGrpc = os.Getenv("otlpMetricsEndpointGrpc")
	extensionResourceID     = os.Getenv("EXTENSION_RESOURCE_ID")
	otlpInsecureGrpc        = os.Getenv("otlpInsecureGrpc")
)

func initOTLPExporter() error {
	ctx := context.Background()

	if otlpMetricsEndpointGrpc == "" {
		return errors.Errorf("OTLP exporter specified but endpoint not provided")
	}

	otlpOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(otlpMetricsEndpointGrpc),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled: false,
		}),
		otlpmetricgrpc.WithTemporalitySelector(func(kind metric.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}),
	}

	if otlpInsecureGrpc == "true" {
		otlpOpts = append(otlpOpts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpOpts...)
	if err != nil {
		return err
	}

	metricOpts := []metric.Option{
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	}

	if extensionResourceID != "" {
		metricOpts = append(metricOpts, metric.WithResource(resource.NewSchemaless(
			attribute.String("microsoft.resourceId", extensionResourceID),
		)))
	}

	meterProvider := metric.NewMeterProvider(metricOpts...)
	otel.SetMeterProvider(meterProvider)

	return nil
}
