// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func TestNewResourceMinimal(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "my-service",
	}

	res, err := NewResource(ctx, cfg, "")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil resource")
	}

	// Check that service.name is present
	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceNameKey && attr.Value.AsString() == "my-service" {
			found = true
		}
	}
	if !found {
		t.Error("expected service.name=my-service in resource attributes")
	}
}

func TestNewResourceWithVersion(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "drone-runner-aws",
	}

	res, err := NewResource(ctx, cfg, "2.5.0")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceVersionKey && attr.Value.AsString() == "2.5.0" {
			found = true
		}
	}
	if !found {
		t.Error("expected service.version=2.5.0 in resource attributes")
	}
}

func TestNewResourceWithoutVersion(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "drone-runner-aws",
	}

	res, err := NewResource(ctx, cfg, "")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	// service.version should NOT be present when version is empty
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceVersionKey {
			t.Errorf("service.version should not be present when version is empty, got %q", attr.Value.AsString())
		}
	}
}

func TestNewResourceWithEnvironment(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "drone-runner-aws",
		Environment: "production",
	}

	res, err := NewResource(ctx, cfg, "1.0")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == attribute.Key("deployment.environment") && attr.Value.AsString() == "production" {
			found = true
		}
	}
	if !found {
		t.Error("expected deployment.environment=production in resource attributes")
	}
}

func TestNewResourceWithoutEnvironment(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "drone-runner-aws",
		Environment: "", // empty
	}

	res, err := NewResource(ctx, cfg, "1.0")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	for _, attr := range res.Attributes() {
		if attr.Key == attribute.Key("deployment.environment") {
			t.Errorf("deployment.environment should not be present when empty, got %q", attr.Value.AsString())
		}
	}
}

func TestNewResourceFullConfig(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName:    "drone-runner-aws",
		ServiceVersion: "2.0.0",
		Environment:    "staging",
	}

	res, err := NewResource(ctx, cfg, "2.0.0")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	attrs := res.Attributes()
	attrMap := make(map[attribute.Key]string, len(attrs))
	for _, a := range attrs {
		attrMap[a.Key] = a.Value.AsString()
	}

	if attrMap[semconv.ServiceNameKey] != "drone-runner-aws" {
		t.Errorf("service.name = %q, want %q", attrMap[semconv.ServiceNameKey], "drone-runner-aws")
	}
	if attrMap[semconv.ServiceVersionKey] != "2.0.0" {
		t.Errorf("service.version = %q, want %q", attrMap[semconv.ServiceVersionKey], "2.0.0")
	}
	if attrMap[attribute.Key("deployment.environment")] != "staging" {
		t.Errorf("deployment.environment = %q, want %q", attrMap[attribute.Key("deployment.environment")], "staging")
	}
}

func TestNewResourceIncludesSystemAttributes(t *testing.T) {
	// NewResource should include process runtime, OS, and host info.
	ctx := context.Background()
	cfg := Config{
		ServiceName: "drone-runner-aws",
	}

	res, err := NewResource(ctx, cfg, "1.0")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	attrs := res.Attributes()

	// We should have more than just service.name and service.version,
	// because WithProcessRuntimeDescription, WithOS, WithHost add extra attributes.
	if len(attrs) <= 2 {
		t.Errorf("expected system attributes (OS, host, runtime), got only %d attributes", len(attrs))
	}

	// Check that at least one OS-related or host-related attribute exists
	hasSystem := false
	for _, a := range attrs {
		key := string(a.Key)
		if key == "os.type" || key == "host.name" || key == "process.runtime.description" {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		t.Error("expected at least one system attribute (os.type, host.name, or process.runtime.description)")
	}
}

func TestNewResourceEmptyServiceName(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "", // empty service name
	}

	res, err := NewResource(ctx, cfg, "")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil resource even with empty service name")
	}

	// service.name should be present but empty
	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceNameKey {
			found = true
			if attr.Value.AsString() != "" {
				t.Errorf("expected empty service.name, got %q", attr.Value.AsString())
			}
		}
	}
	if !found {
		t.Error("expected service.name attribute to be present (even if empty)")
	}
}
