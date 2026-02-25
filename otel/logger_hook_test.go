// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// ---- Mock log exporter ----

type mockLogExporter struct {
	exported int
	err      error
}

func (m *mockLogExporter) Export(_ context.Context, records []sdklog.Record) error {
	m.exported += len(records)
	return m.err
}

func (m *mockLogExporter) Shutdown(_ context.Context) error   { return nil }
func (m *mockLogExporter) ForceFlush(_ context.Context) error { return nil }

// ---- mapLogrusLevel ----

func TestMapLogrusLevel(t *testing.T) {
	tests := []struct {
		level    logrus.Level
		expected otellog.Severity
	}{
		{logrus.TraceLevel, otellog.SeverityTrace},
		{logrus.DebugLevel, otellog.SeverityDebug},
		{logrus.InfoLevel, otellog.SeverityInfo},
		{logrus.WarnLevel, otellog.SeverityWarn},
		{logrus.ErrorLevel, otellog.SeverityError},
		{logrus.FatalLevel, otellog.SeverityFatal},
		{logrus.PanicLevel, otellog.SeverityFatal4},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := mapLogrusLevel(tt.level)
			if got != tt.expected {
				t.Errorf("mapLogrusLevel(%v) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestMapLogrusLevelDefault(t *testing.T) {
	// An unknown/invalid level should fall through to SeverityInfo.
	got := mapLogrusLevel(logrus.Level(99))
	if got != otellog.SeverityInfo {
		t.Errorf("mapLogrusLevel(99) = %v, want SeverityInfo", got)
	}
}

// ---- Levels ----

func TestOTELLogHookLevels(t *testing.T) {
	hook := &OTELLogHook{
		context: make(map[string]string),
	}

	levels := hook.Levels()
	if len(levels) != len(logrus.AllLevels) {
		t.Fatalf("expected %d levels (all), got %d", len(logrus.AllLevels), len(levels))
	}
}

// ---- UpdateContext ----

func TestOTELLogHookUpdateContext(t *testing.T) {
	hook := &OTELLogHook{
		context: make(map[string]string),
	}

	hook.UpdateContext(map[string]string{
		"accountId": "abc123",
		"service":   "runner",
	})

	if hook.context["accountId"] != "abc123" {
		t.Fatalf("expected accountId 'abc123', got %q", hook.context["accountId"])
	}
	if hook.context["service"] != "runner" {
		t.Fatalf("expected service 'runner', got %q", hook.context["service"])
	}

	// Update should merge, not replace
	hook.UpdateContext(map[string]string{
		"version": "1.0",
	})

	if hook.context["accountId"] != "abc123" {
		t.Fatal("existing context should not be lost on update")
	}
	if hook.context["version"] != "1.0" {
		t.Fatalf("expected version '1.0', got %q", hook.context["version"])
	}
}

func TestOTELLogHookUpdateContextOverwrite(t *testing.T) {
	hook := &OTELLogHook{
		context: make(map[string]string),
	}

	hook.UpdateContext(map[string]string{"key": "v1"})
	if hook.context["key"] != "v1" {
		t.Fatalf("expected 'v1', got %q", hook.context["key"])
	}

	// Overwrite same key
	hook.UpdateContext(map[string]string{"key": "v2"})
	if hook.context["key"] != "v2" {
		t.Fatalf("expected 'v2' after overwrite, got %q", hook.context["key"])
	}
}

func TestOTELLogHookUpdateContextConcurrency(t *testing.T) {
	hook := &OTELLogHook{
		context: make(map[string]string),
	}

	done := make(chan struct{})

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			hook.UpdateContext(map[string]string{"key": "value"})
		}
		close(done)
	}()

	// Concurrent reads via Fire (simulated by reading context)
	for i := 0; i < 100; i++ {
		hook.contextMu.RLock()
		_ = hook.context["key"]
		hook.contextMu.RUnlock()
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for concurrent operations")
	}
}

// ---- Constructor tests ----

func TestNewOTELLogHookFromExporter(t *testing.T) {
	mock := &mockLogExporter{}

	hook, err := newOTELLogHookFromExporter(mock)
	if err != nil {
		t.Fatalf("newOTELLogHookFromExporter failed: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if hook.loggerProvider == nil {
		t.Fatal("expected non-nil loggerProvider")
	}
	if hook.otelLogger == nil {
		t.Fatal("expected non-nil otelLogger")
	}
	if hook.context == nil {
		t.Fatal("expected non-nil context map")
	}

	// Cleanup
	_ = hook.Close()
}

func TestNewOTELLogHookFromExporterWithOptions(t *testing.T) {
	mock := &mockLogExporter{}

	// Pass a resource option
	res, _ := NewResource(context.Background(), Config{ServiceName: "test-service"}, "1.0.0")
	hook, err := newOTELLogHookFromExporter(mock, sdklog.WithResource(res))
	if err != nil {
		t.Fatalf("newOTELLogHookFromExporter with options failed: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	_ = hook.Close()
}

func TestNewOTELLogHookPublicConstructor(t *testing.T) {
	mock := &mockLogExporter{}

	hook, err := NewOTELLogHook(mock, nil)
	if err != nil {
		t.Fatalf("NewOTELLogHook failed: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}

	_ = hook.Close()
}

func TestNewOTELLogHookWithProvider(t *testing.T) {
	mock := &mockLogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(mock)),
	)

	hook := NewOTELLogHookWithProvider(provider)
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if hook.loggerProvider != provider {
		t.Fatal("expected provider to be assigned")
	}
	if hook.otelLogger == nil {
		t.Fatal("expected non-nil otelLogger")
	}

	_ = hook.Close()
}

// ---- Fire tests ----

func TestOTELLogHookFire(t *testing.T) {
	mock := &mockLogExporter{}

	// Use a SimpleProcessor so records export synchronously (easier to test).
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(mock)),
	)
	hook := NewOTELLogHookWithProvider(provider)
	hook.UpdateContext(map[string]string{
		"accountId": "acc123",
		"service":   "runner",
	})

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.InfoLevel,
		Message: "test log message",
		Time:    time.Now(),
		Data: logrus.Fields{
			"taskId": "task-1",
		},
	}

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	// The SimpleProcessor exports synchronously, so records should be captured.
	if mock.exported == 0 {
		t.Error("expected at least 1 exported record")
	}

	_ = hook.Close()
}

func TestOTELLogHookFireWithContext(t *testing.T) {
	mock := &mockLogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(mock)),
	)
	hook := NewOTELLogHookWithProvider(provider)

	ctx := context.Background()
	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.WarnLevel,
		Message: "warning message",
		Time:    time.Now(),
		Data:    logrus.Fields{},
		Context: ctx,
	}

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	_ = hook.Close()
}

func TestOTELLogHookFireWithErrorField(t *testing.T) {
	mock := &mockLogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(mock)),
	)
	hook := NewOTELLogHookWithProvider(provider)

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.ErrorLevel,
		Message: "something failed",
		Time:    time.Now(),
		Data: logrus.Fields{
			logrus.ErrorKey: context.DeadlineExceeded,
		},
	}

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	_ = hook.Close()
}

func TestOTELLogHookFireNilContext(t *testing.T) {
	// Fire with a nil context in the entry should not panic.
	mock := &mockLogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(mock)),
	)
	hook := NewOTELLogHookWithProvider(provider)

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.DebugLevel,
		Message: "debug with nil context",
		Time:    time.Now(),
		Data:    logrus.Fields{},
		Context: nil, // explicitly nil
	}

	err := hook.Fire(entry)
	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	_ = hook.Close()
}

func TestOTELLogHookFireAllLevels(t *testing.T) {
	mock := &mockLogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(mock)),
	)
	hook := NewOTELLogHookWithProvider(provider)

	for _, level := range logrus.AllLevels {
		entry := &logrus.Entry{
			Logger:  logrus.StandardLogger(),
			Level:   level,
			Message: "test " + level.String(),
			Time:    time.Now(),
			Data:    logrus.Fields{},
		}
		err := hook.Fire(entry)
		if err != nil {
			t.Fatalf("Fire(%s) returned error: %v", level, err)
		}
	}

	_ = hook.Close()
}

// ---- Panic recovery ----

func TestOTELLogHookFirePanicRecovery(t *testing.T) {
	// A hook with a nil otelLogger should panic internally but recover.
	hook := &OTELLogHook{
		loggerProvider: nil, // intentionally nil
		otelLogger:     nil, // will cause panic on Emit
		context:        make(map[string]string),
	}

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.InfoLevel,
		Message: "this should not crash",
		Time:    time.Now(),
		Data:    logrus.Fields{},
	}

	// Fire should recover from the panic and return an error, not crash.
	err := hook.Fire(entry)
	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
}

func TestOTELLogHookFirePanicRecoveryErrorFormat(t *testing.T) {
	// Verify the recovered error contains the expected prefix.
	hook := &OTELLogHook{
		loggerProvider: nil,
		otelLogger:     nil,
		context:        make(map[string]string),
	}

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.InfoLevel,
		Message: "test",
		Time:    time.Now(),
		Data:    logrus.Fields{},
	}

	err := hook.Fire(entry)
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
	errStr := err.Error()
	if !containsSubstring(errStr, "otel log hook panic") {
		t.Errorf("error should contain 'otel log hook panic', got: %s", errStr)
	}
}

func TestOTELLogHookFirePanicRecoveryWithContextFields(t *testing.T) {
	// Even with populated context fields, a nil otelLogger should panic and recover.
	hook := &OTELLogHook{
		loggerProvider: nil,
		otelLogger:     nil,
		context: map[string]string{
			"accountId":  "test-account",
			"service":    "runner",
			"runnerName": "aws-runner",
		},
	}

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.ErrorLevel,
		Message: "error with context",
		Time:    time.Now(),
		Data: logrus.Fields{
			"taskId": "task-123",
			"step":   "clone",
		},
	}

	err := hook.Fire(entry)
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
}

func TestOTELLogHookFirePanicDoesNotCrashApp(t *testing.T) {
	// Simulate rapid Fire calls with a nil logger — none should crash.
	hook := &OTELLogHook{
		loggerProvider: nil,
		otelLogger:     nil,
		context:        make(map[string]string),
	}

	for i := 0; i < 100; i++ {
		entry := &logrus.Entry{
			Logger:  logrus.StandardLogger(),
			Level:   logrus.InfoLevel,
			Message: "rapid fire",
			Time:    time.Now(),
			Data:    logrus.Fields{},
		}
		err := hook.Fire(entry)
		if err == nil {
			t.Fatalf("iteration %d: expected error from panic recovery", i)
		}
	}
}

func TestOTELLogHookFirePanicRecoveryReturnedError(t *testing.T) {
	// Verify that the error returned by panic recovery is non-nil and
	// doesn't itself cause any issues when inspected.
	hook := &OTELLogHook{
		loggerProvider: nil,
		otelLogger:     nil,
		context:        make(map[string]string),
	}

	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.WarnLevel,
		Message: "trigger panic",
		Time:    time.Now(),
		Data:    logrus.Fields{"key": "value"},
	}

	err := hook.Fire(entry)
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}

	// The error should be safe to format via Error() and %v
	_ = err.Error()
	_ = fmt.Sprintf("%v", err)
}

// ---- Close ----

func TestOTELLogHookClose(t *testing.T) {
	mock := &mockLogExporter{}
	hook, err := newOTELLogHookFromExporter(mock)
	if err != nil {
		t.Fatalf("failed to create hook: %v", err)
	}

	err = hook.Close()
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
}

func TestOTELLogHookCloseFlushes(t *testing.T) {
	// Ensure Close triggers shutdown of the provider (which flushes).
	mock := &mockLogExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(mock)),
	)
	hook := NewOTELLogHookWithProvider(provider)

	// Fire a log
	entry := &logrus.Entry{
		Logger:  logrus.StandardLogger(),
		Level:   logrus.InfoLevel,
		Message: "before close",
		Time:    time.Now(),
		Data:    logrus.Fields{},
	}
	_ = hook.Fire(entry)

	err := hook.Close()
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
}

// ---- Resource test (previously here, kept for coverage) ----

func TestNewResourceWithConfig(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		ServiceName: "test-service",
		Environment: "testing",
	}

	res, err := NewResource(ctx, cfg, "1.2.3")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil resource")
	}
}

// ---- helpers ----

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
