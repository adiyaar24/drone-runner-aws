// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

const (
	// logExportTimeout is the timeout for log batch export operations.
	logExportTimeout = 10 * time.Second
	// logHookCloseTimeout is the timeout for closing the log hook.
	logHookCloseTimeout = 15 * time.Second
	// baseAttributeCapacity is the base capacity for log record attributes.
	baseAttributeCapacity = 4
)

// OTELLogHook is a logrus.Hook that ships log entries to an OTEL log exporter.
type OTELLogHook struct {
	loggerProvider *sdklog.LoggerProvider
	otelLogger     otellog.Logger
	contextMu      sync.RWMutex
	context        map[string]string
}

// NewOTELLogHook creates a new logrus hook that exports logs via the given OTEL log exporter.
func NewOTELLogHook(exporter sdklog.Exporter, res interface{ Attributes() []interface{} }, opts ...sdklog.LoggerProviderOption) (*OTELLogHook, error) {
	return newOTELLogHookFromExporter(exporter, opts...)
}

// newOTELLogHookFromExporter creates the hook from an exporter with optional LoggerProvider options.
func newOTELLogHookFromExporter(exporter sdklog.Exporter, opts ...sdklog.LoggerProviderOption) (*OTELLogHook, error) {
	// Build logger provider options: always include a batch processor for the exporter
	providerOpts := []sdklog.LoggerProviderOption{
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter,
			sdklog.WithExportTimeout(logExportTimeout),
		)),
	}
	providerOpts = append(providerOpts, opts...)

	provider := sdklog.NewLoggerProvider(providerOpts...)
	otelLogger := provider.Logger("drone-runner-aws")

	return &OTELLogHook{
		loggerProvider: provider,
		otelLogger:     otelLogger,
		context:        make(map[string]string),
	}, nil
}

// NewOTELLogHookWithProvider creates a hook directly from a LoggerProvider.
func NewOTELLogHookWithProvider(provider *sdklog.LoggerProvider) *OTELLogHook {
	return &OTELLogHook{
		loggerProvider: provider,
		otelLogger:     provider.Logger("drone-runner-aws"),
		context:        make(map[string]string),
	}
}

// Levels returns all log levels — ship everything.
func (h *OTELLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire converts a logrus entry into an OTEL log record and emits it.
func (h *OTELLogHook) Fire(entry *logrus.Entry) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			// Swallow the panic — OTEL issues must never crash the app.
			// We can't use the logger here (would re-enter the hook), so this is silent.
			retErr = fmt.Errorf("otel log hook panic: %v", r)
		}
	}()

	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Build OTEL log record
	var record otellog.Record
	record.SetTimestamp(entry.Time)
	record.SetBody(otellog.StringValue(entry.Message))
	record.SetSeverity(mapLogrusLevel(entry.Level))
	record.SetSeverityText(entry.Level.String())

	// Collect attributes from various sources
	attrs := make([]otellog.KeyValue, 0, len(entry.Data)+len(h.context)+baseAttributeCapacity)

	// Add logrus entry fields (entry.Data)
	for k, v := range entry.Data {
		attrs = append(attrs, otellog.String(k, fmt.Sprintf("%v", v)))
	}

	// Add hook-level context (accountId, runnerName, service, etc.)
	h.contextMu.RLock()
	for k, v := range h.context {
		attrs = append(attrs, otellog.String(k, v))
	}
	h.contextMu.RUnlock()

	// Add caller info if available
	if entry.HasCaller() {
		attrs = append(attrs,
			otellog.String("code.filepath", entry.Caller.File),
			otellog.String("code.function", entry.Caller.Function),
			otellog.Int("code.lineno", entry.Caller.Line),
		)
	}

	// Add error info if present
	if errValue, ok := entry.Data[logrus.ErrorKey]; ok {
		if err, isErr := errValue.(error); isErr {
			attrs = append(attrs, otellog.String("exception.message", err.Error()))
		}
	}

	record.AddAttributes(attrs...)

	h.otelLogger.Emit(ctx, record)
	return nil
}

// Close flushes pending logs and shuts down the logger provider.
func (h *OTELLogHook) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), logHookCloseTimeout)
	defer cancel()
	return h.loggerProvider.Shutdown(ctx)
}

// UpdateContext adds extra key-value pairs to every log record.
func (h *OTELLogHook) UpdateContext(ctxMap map[string]string) {
	h.contextMu.Lock()
	defer h.contextMu.Unlock()
	for key, value := range ctxMap {
		h.context[key] = value
	}
}

// mapLogrusLevel converts logrus log level to OTEL severity.
func mapLogrusLevel(level logrus.Level) otellog.Severity {
	switch level {
	case logrus.TraceLevel:
		return otellog.SeverityTrace
	case logrus.DebugLevel:
		return otellog.SeverityDebug
	case logrus.InfoLevel:
		return otellog.SeverityInfo
	case logrus.WarnLevel:
		return otellog.SeverityWarn
	case logrus.ErrorLevel:
		return otellog.SeverityError
	case logrus.FatalLevel:
		return otellog.SeverityFatal
	case logrus.PanicLevel:
		return otellog.SeverityFatal4
	default:
		return otellog.SeverityInfo
	}
}
