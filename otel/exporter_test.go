// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"testing"
	"time"
)

// ---- createLogExporter tests ----

func TestCreateLogExporterGRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "grpc",
		Insecure: true,
	}

	exp, shutdown, err := createLogExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("createLogExporter(grpc) unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil log exporter")
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	// Clean up — shutdown should not panic even without a live server.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateLogExporterHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4318",
		Protocol: "http",
		Insecure: true,
	}

	exp, shutdown, err := createLogExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("createLogExporter(http) unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil log exporter")
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateLogExporterDefaultProtocol(t *testing.T) {
	// Empty protocol should default to gRPC.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "", // should fall through to default (grpc)
		Insecure: true,
	}

	exp, shutdown, err := createLogExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("createLogExporter(default) unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil log exporter")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateLogExporterWithHeaders(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "grpc",
		Insecure: true,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"X-Custom":      "value",
		},
	}

	exp, shutdown, err := createLogExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("createLogExporter(with headers) unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil log exporter")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateLogExporterHTTPWithHeaders(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4318",
		Protocol: "http",
		Insecure: true,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
	}

	exp, shutdown, err := createLogExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("createLogExporter(http with headers) unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil log exporter")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateLogExporterHTTPSecure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4318",
		Protocol: "http",
		Insecure: false, // TLS enabled
	}

	exp, shutdown, err := createLogExporter(ctx, cfg)
	if err != nil {
		t.Fatalf("createLogExporter(http secure) unexpected error: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil log exporter")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

// ---- createMetricReader tests ----

func TestCreateMetricReaderGRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "grpc",
		Insecure: true,
	}

	reader, shutdown, err := createMetricReader(ctx, cfg)
	if err != nil {
		t.Fatalf("createMetricReader(grpc) unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil metric reader")
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateMetricReaderHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4318",
		Protocol: "http",
		Insecure: true,
	}

	reader, shutdown, err := createMetricReader(ctx, cfg)
	if err != nil {
		t.Fatalf("createMetricReader(http) unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil metric reader")
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateMetricReaderDefaultProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "", // defaults to grpc
		Insecure: true,
	}

	reader, shutdown, err := createMetricReader(ctx, cfg)
	if err != nil {
		t.Fatalf("createMetricReader(default) unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil metric reader")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateMetricReaderWithHeaders(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "grpc",
		Insecure: true,
		Headers: map[string]string{
			"Authorization": "Bearer metrics-token",
		},
	}

	reader, shutdown, err := createMetricReader(ctx, cfg)
	if err != nil {
		t.Fatalf("createMetricReader(with headers) unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil metric reader")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateMetricReaderHTTPWithHeaders(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4318",
		Protocol: "http",
		Insecure: true,
		Headers: map[string]string{
			"X-Scope-OrgID": "test-org",
		},
	}

	reader, shutdown, err := createMetricReader(ctx, cfg)
	if err != nil {
		t.Fatalf("createMetricReader(http with headers) unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil metric reader")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

func TestCreateMetricReaderGRPCSecure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := Config{
		Endpoint: "localhost:4317",
		Protocol: "grpc",
		Insecure: false, // TLS enabled
	}

	reader, shutdown, err := createMetricReader(ctx, cfg)
	if err != nil {
		t.Fatalf("createMetricReader(grpc secure) unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil metric reader")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = shutdown(shutdownCtx)
}

// ---- DefaultMetricsExportInterval ----

func TestDefaultMetricsExportIntervalValue(t *testing.T) {
	expected := 30 * time.Second
	if DefaultMetricsExportInterval != expected {
		t.Errorf("DefaultMetricsExportInterval = %v, want %v", DefaultMetricsExportInterval, expected)
	}
}
