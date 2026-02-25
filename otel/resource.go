// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// NewResource creates an OTEL resource with standard and custom attributes.
// The resource identifies the entity producing telemetry (service name, version, etc.).
func NewResource(ctx context.Context, config Config, version string) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(config.ServiceName),
	}

	if version != "" {
		attrs = append(attrs, semconv.ServiceVersion(version))
	}

	if config.Environment != "" {
		attrs = append(attrs, attribute.String("deployment.environment", config.Environment))
	}

	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithProcessRuntimeDescription(),
		resource.WithOS(),
		resource.WithHost(),
	)
}
