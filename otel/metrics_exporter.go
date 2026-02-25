// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// MetricsBridge reads metrics from the existing Prometheus registry
// and exports them via OTEL using a PeriodicReader.
type MetricsBridge struct {
	meterProvider *sdkmetric.MeterProvider
}

// NewMetricsBridge creates a MetricsBridge that periodically reads and exports metrics.
// The reader is typically created by the provider (e.g., a PeriodicReader wrapping an OTLP exporter).
func NewMetricsBridge(reader sdkmetric.Reader, res interface{ Attributes() []interface{} }, opts ...sdkmetric.Option) (*MetricsBridge, error) {
	return newMetricsBridgeFromReader(reader, opts...)
}

// newMetricsBridgeFromReader creates the bridge from a reader with optional MeterProvider options.
func newMetricsBridgeFromReader(reader sdkmetric.Reader, opts ...sdkmetric.Option) (*MetricsBridge, error) {
	providerOpts := []sdkmetric.Option{
		sdkmetric.WithReader(reader),
	}
	providerOpts = append(providerOpts, opts...)

	provider := sdkmetric.NewMeterProvider(providerOpts...)

	return &MetricsBridge{
		meterProvider: provider,
	}, nil
}

// NewMetricsBridgeWithProvider creates a bridge directly from a MeterProvider.
func NewMetricsBridgeWithProvider(provider *sdkmetric.MeterProvider) *MetricsBridge {
	return &MetricsBridge{
		meterProvider: provider,
	}
}

// GetMeterProvider returns the underlying OTEL MeterProvider.
// This can be used to create custom metrics instruments.
func (b *MetricsBridge) GetMeterProvider() *sdkmetric.MeterProvider {
	return b.meterProvider
}

// Shutdown gracefully stops the metrics bridge.
func (b *MetricsBridge) Shutdown(ctx context.Context) error {
	return b.meterProvider.Shutdown(ctx)
}

// DefaultMetricsExportInterval is the default interval for periodic metric export.
const DefaultMetricsExportInterval = 30 * time.Second
