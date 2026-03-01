// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// ---- Constructor tests ----

func TestNewMetricsBridgeFromReader(t *testing.T) {
	reader := sdkmetric.NewManualReader()

	bridge, err := newMetricsBridgeFromReader(reader)
	if err != nil {
		t.Fatalf("newMetricsBridgeFromReader failed: %v", err)
	}
	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.meterProvider == nil {
		t.Fatal("expected non-nil meterProvider")
	}

	_ = bridge.Shutdown(context.Background())
}

func TestNewMetricsBridgeFromReaderWithOptions(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	res, _ := NewResource(context.Background(), &Config{ServiceName: "test-svc"}, "1.0")

	bridge, err := newMetricsBridgeFromReader(reader, sdkmetric.WithResource(res))
	if err != nil {
		t.Fatalf("newMetricsBridgeFromReader with options failed: %v", err)
	}
	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}

	_ = bridge.Shutdown(context.Background())
}

func TestNewMetricsBridgeWithProvider(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	bridge := NewMetricsBridgeWithProvider(provider)
	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.meterProvider != provider {
		t.Fatal("expected provider to be assigned")
	}

	_ = bridge.Shutdown(context.Background())
}

// ---- GetMeterProvider ----

func TestGetMeterProvider(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	bridge, _ := newMetricsBridgeFromReader(reader)

	provider := bridge.GetMeterProvider()
	if provider == nil {
		t.Fatal("expected non-nil MeterProvider")
	}

	_ = bridge.Shutdown(context.Background())
}

// ---- Shutdown ----

func TestMetricsBridgeShutdown(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	bridge, err := newMetricsBridgeFromReader(reader)
	if err != nil {
		t.Fatalf("failed to create bridge: %v", err)
	}

	err = bridge.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestMetricsBridgeShutdownIdempotent(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	bridge, _ := newMetricsBridgeFromReader(reader)

	// First shutdown
	err := bridge.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("first Shutdown returned error: %v", err)
	}

	// Second shutdown — should not panic.
	err = bridge.Shutdown(context.Background())
	// May or may not error (depends on SDK), but must not panic.
	_ = err
}

// ---- Meter creation and recording ----

func TestMetricsBridgeCanCreateMeter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	bridge, _ := newMetricsBridgeFromReader(reader)

	meter := bridge.GetMeterProvider().Meter("test-meter")
	if meter == nil {
		// Meter() never returns nil, but check the type
		t.Fatal("expected non-nil meter")
	}

	_ = bridge.Shutdown(context.Background())
}

func TestMetricsBridgeRecordAndCollect(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	bridge, _ := newMetricsBridgeFromReader(reader)

	// Create a counter and record a value
	meter := bridge.GetMeterProvider().Meter("test-meter")
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}

	counter.Add(context.Background(), 42)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Verify we got scope metrics
	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected at least one ScopeMetrics entry")
	}

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_counter" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected to find test_counter metric")
	}

	_ = bridge.Shutdown(context.Background())
}

func TestMetricsBridgeRecordHistogram(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	bridge, _ := newMetricsBridgeFromReader(reader)

	meter := bridge.GetMeterProvider().Meter("test-meter")
	hist, err := meter.Float64Histogram("test_histogram")
	if err != nil {
		t.Fatalf("failed to create histogram: %v", err)
	}

	hist.Record(context.Background(), 3.14)
	hist.Record(context.Background(), 2.72)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_histogram" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected to find test_histogram metric")
	}

	_ = bridge.Shutdown(context.Background())
}

// ---- DefaultMetricsExportInterval ----

func TestDefaultMetricsExportIntervalInMetricsExporter(t *testing.T) {
	// Verify the constant is defined and has the expected value.
	expected := 30 * time.Second
	if DefaultMetricsExportInterval != expected {
		t.Errorf("DefaultMetricsExportInterval = %v, want %v", DefaultMetricsExportInterval, expected)
	}
}
