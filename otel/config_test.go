// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	// Zero-value Config should have everything disabled.
	var cfg Config

	if cfg.Enabled {
		t.Error("default Enabled should be false")
	}
	if cfg.Endpoint != "" {
		t.Errorf("default Endpoint should be empty, got %q", cfg.Endpoint)
	}
	if cfg.Protocol != "" {
		t.Errorf("default Protocol should be empty, got %q", cfg.Protocol)
	}
	if cfg.Insecure {
		t.Error("default Insecure should be false")
	}
	if cfg.ExportLogs {
		t.Error("default ExportLogs should be false")
	}
	if cfg.ExportMetrics {
		t.Error("default ExportMetrics should be false")
	}
	if cfg.ServiceName != "" {
		t.Errorf("default ServiceName should be empty, got %q", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "" {
		t.Errorf("default ServiceVersion should be empty, got %q", cfg.ServiceVersion)
	}
	if cfg.Environment != "" {
		t.Errorf("default Environment should be empty, got %q", cfg.Environment)
	}
	if cfg.Headers != nil {
		t.Errorf("default Headers should be nil, got %v", cfg.Headers)
	}
}

func TestConfigFullyPopulated(t *testing.T) {
	cfg := Config{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		Protocol:       "grpc",
		Insecure:       true,
		ExportLogs:     true,
		ExportMetrics:  true,
		ServiceName:    "drone-runner-aws",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "value",
		},
	}

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.Endpoint != "localhost:4317" {
		t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "localhost:4317")
	}
	if cfg.Protocol != "grpc" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "grpc")
	}
	if !cfg.Insecure {
		t.Error("Insecure should be true")
	}
	if !cfg.ExportLogs {
		t.Error("ExportLogs should be true")
	}
	if !cfg.ExportMetrics {
		t.Error("ExportMetrics should be true")
	}
	if cfg.ServiceName != "drone-runner-aws" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "drone-runner-aws")
	}
	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "1.0.0")
	}
	if cfg.Environment != "production" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "production")
	}
	if len(cfg.Headers) != 2 {
		t.Errorf("Headers length = %d, want 2", len(cfg.Headers))
	}
	if cfg.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization header = %q, want %q", cfg.Headers["Authorization"], "Bearer token123")
	}
}

func TestConfigHTTPProtocol(t *testing.T) {
	cfg := Config{
		Enabled:  true,
		Endpoint: "localhost:4318",
		Protocol: "http",
	}

	if cfg.Protocol != "http" {
		t.Errorf("Protocol = %q, want %q", cfg.Protocol, "http")
	}
}

func TestConfigEmptyHeaders(t *testing.T) {
	// Config with explicitly empty headers map.
	cfg := Config{
		Headers: map[string]string{},
	}

	if len(cfg.Headers) != 0 {
		t.Errorf("Headers length = %d, want 0", len(cfg.Headers))
	}
}
