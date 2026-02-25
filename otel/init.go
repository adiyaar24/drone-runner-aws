// Copyright 2024 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package otel

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	otelglobal "go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// manager holds references to OTEL components for graceful shutdown.
type manager struct {
	logHook       *OTELLogHook
	metricsBridge *MetricsBridge
	shutdownFuncs []shutdownFunc // exporter shutdown functions to flush/close connections
	config        Config
}

var (
	mu            sync.Mutex
	activeManager *manager
)

// Start initializes the OTEL integration based on configuration.
// It creates OTLP exporters, registers the logrus hook for log shipping,
// and sets up the metrics bridge. Safe to call multiple times — the previous
// manager is shut down before being replaced.
func Start(ctx context.Context, config *Config, version string, logger *logrus.Logger) error {
	if !config.Enabled {
		logrus.Info("OTEL integration is disabled")
		return nil
	}

	if config.Endpoint == "" {
		return fmt.Errorf("OTEL is enabled but DRONE_OTEL_ENDPOINT is not set")
	}

	mu.Lock()
	defer mu.Unlock()

	// Shut down any existing manager to avoid resource leaks
	if activeManager != nil {
		logrus.Info("Shutting down previous OTEL manager before reinitializing")
		if err := activeManager.shutdown(ctx); err != nil {
			logrus.WithError(err).Warn("Failed to shutdown previous OTEL manager")
		}
		activeManager = nil
	}

	logrus.Infof("Initializing OTEL integration: endpoint=%s, protocol=%s",
		config.Endpoint, config.Protocol)

	// Register a global OTEL error handler so that SDK-level errors (export
	// timeouts, connection failures, dropped spans/logs) are surfaced in the
	// application logs instead of only going to stderr.
	otelglobal.SetErrorHandler(otelglobal.ErrorHandlerFunc(func(err error) {
		logrus.Errorf("OTEL SDK error: %v", err)
	}))

	mgr := &manager{
		config: *config,
	}

	// Build resource attributes
	res, err := NewResource(ctx, *config, version)
	if err != nil {
		logrus.WithError(err).Warn("Failed to create OTEL resource, using default")
	}

	// Initialize log exporter if enabled
	if config.ExportLogs {
		logExporter, _, err := createLogExporter(ctx, config)
		if err != nil {
			logrus.WithError(err).Error("Failed to create OTLP log exporter")
		} else {
			// Note: we do NOT add logExporter.Shutdown to shutdownFuncs.
			// The LoggerProvider owns the exporter lifecycle via the BatchProcessor:
			//   logHook.Close() → LoggerProvider.Shutdown() → BatchProcessor.Shutdown()
			//     → flush pending logs → exporter.Shutdown()
			// Calling exporter.Shutdown() separately would kill the transport before
			// the BatchProcessor can flush, causing log loss.
			hook, err := newOTELLogHookFromExporter(logExporter,
				sdklog.WithResource(res),
			)
			if err != nil {
				logrus.WithError(err).Error("Failed to create OTEL log hook")
			} else {
				if logger != nil {
					logger.AddHook(hook)
				} else {
					logrus.AddHook(hook)
				}
				mgr.logHook = hook
				logrus.Infoln("OTEL log export enabled")
			}
		}
	}

	// Initialize metric exporter if enabled
	if config.ExportMetrics {
		metricReader, _, err := createMetricReader(ctx, config)
		if err != nil {
			logrus.WithError(err).Error("Failed to create OTLP metric reader")
		} else {
			// Note: we do NOT add metricExporter.Shutdown to shutdownFuncs.
			// The MeterProvider owns the exporter lifecycle via the PeriodicReader:
			//   MetricsBridge.Shutdown() → MeterProvider.Shutdown() → PeriodicReader.Shutdown()
			//     → collect final metrics → exporter.Shutdown()
			// Adding it to shutdownFuncs would cause a double-shutdown error.
			bridge, err := newMetricsBridgeFromReader(metricReader,
				sdkmetric.WithResource(res),
			)
			if err != nil {
				logrus.WithError(err).Error("Failed to create OTEL metrics bridge")
			} else {
				mgr.metricsBridge = bridge
				logrus.Infoln("OTEL metric export enabled")
			}
		}
	}

	activeManager = mgr
	logrus.Infof("OTEL integration initialized successfully (endpoint=%s)", config.Endpoint)
	return nil
}

// Shutdown gracefully stops all OTEL components, flushing pending data
// and closing exporter connections.
func Shutdown(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	if activeManager == nil {
		return nil
	}

	err := activeManager.shutdown(ctx)
	activeManager = nil
	return err
}

// shutdown performs the actual shutdown of the manager (must be called under mu.Lock).
func (m *manager) shutdown(ctx context.Context) error {
	if m == nil {
		return nil
	}

	logrus.Infoln("Shutting down OTEL integration")

	var errs []error

	// 1. Flush and close the log hook first.
	// LoggerProvider.Shutdown() → BatchProcessor.Shutdown() → flush pending logs → exporter.Shutdown()
	// This must happen before metrics shutdown so any log entries generated during
	// shutdown are still captured.
	if m.logHook != nil {
		if err := m.logHook.Close(); err != nil {
			errs = append(errs, fmt.Errorf("log hook shutdown: %w", err))
		}
	}

	// 2. Flush and close the metrics bridge.
	// MeterProvider.Shutdown() → PeriodicReader.Shutdown() → collect final metrics → exporter.Shutdown()
	if m.metricsBridge != nil {
		if err := m.metricsBridge.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("metrics bridge shutdown: %w", err))
		}
	}

	// 3. Any additional shutdown functions (extensibility for future components).
	for _, fn := range m.shutdownFuncs {
		if err := fn(ctx); err != nil {
			errs = append(errs, fmt.Errorf("exporter shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("OTEL shutdown errors: %v", errs)
	}
	return nil
}

// GetMetricsBridge returns the active metrics bridge (can be nil if metrics are disabled or OTEL is off).
func GetMetricsBridge() *MetricsBridge {
	mu.Lock()
	defer mu.Unlock()

	if activeManager == nil {
		return nil
	}
	return activeManager.metricsBridge
}

// GetLogHook returns the active log hook (can be nil if logs are disabled or OTEL is off).
func GetLogHook() *OTELLogHook {
	mu.Lock()
	defer mu.Unlock()

	if activeManager == nil {
		return nil
	}
	return activeManager.logHook
}
