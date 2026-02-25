// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	// protocolHTTP is the HTTP protocol for OTLP export.
	protocolHTTP = "http"
	// metricExportTimeout is the timeout for metric export operations.
	metricExportTimeout = 30 * time.Second
)

// shutdownFunc is a function that gracefully shuts down an exporter.
type shutdownFunc func(ctx context.Context) error

// createLogExporter creates an OTLP log exporter based on config.
// Returns the exporter and a shutdown function to flush/close it.
func createLogExporter(ctx context.Context, config *Config) (sdklog.Exporter, shutdownFunc, error) {
	switch config.Protocol {
	case protocolHTTP:
		opts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(config.Endpoint),
		}
		if config.Insecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlploghttp.WithHeaders(config.Headers))
		}
		exp, err := otlploghttp.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OTLP HTTP log exporter: %w", err)
		}
		return exp, exp.Shutdown, nil

	default: // grpc
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(config.Endpoint),
		}
		if config.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlploggrpc.WithHeaders(config.Headers))
		}
		exp, err := otlploggrpc.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OTLP gRPC log exporter: %w", err)
		}
		return exp, exp.Shutdown, nil
	}
}

// createMetricReader creates an OTLP metric periodic reader based on config.
// Returns the reader and a shutdown function to flush/close the underlying exporter.
func createMetricReader(ctx context.Context, config *Config) (sdkmetric.Reader, shutdownFunc, error) {
	switch config.Protocol {
	case protocolHTTP:
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(config.Endpoint),
		}
		if config.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(config.Headers))
		}
		exporter, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OTLP HTTP metric exporter: %w", err)
		}
		reader := sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(DefaultMetricsExportInterval),
			sdkmetric.WithTimeout(metricExportTimeout),
		)
		return reader, exporter.Shutdown, nil

	default: // grpc
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(config.Endpoint),
		}
		if config.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(config.Headers))
		}
		exporter, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OTLP gRPC metric exporter: %w", err)
		}
		reader := sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(DefaultMetricsExportInterval),
			sdkmetric.WithTimeout(metricExportTimeout),
		)
		return reader, exporter.Shutdown, nil
	}
}
