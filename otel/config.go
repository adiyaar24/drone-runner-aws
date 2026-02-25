// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

// Config holds all OTEL-related configuration.
// The runner speaks standard OTLP to a collector. The collector (customer-managed)
// handles routing to the desired backend (Splunk, Datadog, etc.).
type Config struct {
	// Core OTEL settings
	Enabled  bool   // DRONE_OTEL_ENABLED
	Endpoint string // DRONE_OTEL_ENDPOINT (collector endpoint, e.g. "localhost:4317")
	Protocol string // DRONE_OTEL_PROTOCOL: "grpc" or "http"
	Insecure bool   // DRONE_OTEL_INSECURE

	// What to export
	ExportLogs    bool // DRONE_OTEL_EXPORT_LOGS
	ExportMetrics bool // DRONE_OTEL_EXPORT_METRICS

	// Resource attributes
	ServiceName    string // DRONE_OTEL_SERVICE_NAME
	ServiceVersion string // DRONE_OTEL_SERVICE_VERSION
	Environment    string // DRONE_OTEL_ENVIRONMENT

	// Custom headers for OTLP exporters (e.g., auth tokens)
	Headers map[string]string
}
