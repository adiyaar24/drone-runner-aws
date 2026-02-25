# OTEL Integration — Drone Runner AWS

Native OpenTelemetry integration for shipping logs and metrics from the drone runner to any observability backend via standard OTLP.

## Overview

The runner is **backend-agnostic** — it speaks OTLP to a customer-managed OTEL Collector. The collector handles routing to the desired backend (Splunk, Datadog, CloudWatch, etc.). This keeps the runner surface area minimal while giving customers full flexibility.

**Key design decisions:**
- Runner speaks OTLP only — no backend-specific code
- Prometheus `/metrics` always stays on as a fallback
- Panic recovery in log hook — OTEL failures never crash the app
- SDK errors surfaced via `otel.SetErrorHandler` into app logs

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DRONE_OTEL_ENABLED` | `false` | Enable OTEL integration |
| `DRONE_OTEL_ENDPOINT` | — | Collector endpoint (e.g. `localhost:4317`) |
| `DRONE_OTEL_PROTOCOL` | `grpc` | Transport protocol: `grpc` or `http` |
| `DRONE_OTEL_INSECURE` | `true` | Skip TLS verification |
| `DRONE_OTEL_EXPORT_LOGS` | `true` | Export logs via OTLP |
| `DRONE_OTEL_EXPORT_METRICS` | `true` | Export metrics via OTLP |
| `DRONE_OTEL_SERVICE_NAME` | `drone-runner-aws` | Service name in telemetry resource |
| `DRONE_OTEL_SERVICE_VERSION` | — | Service version |
| `DRONE_OTEL_ENVIRONMENT` | — | Deployment environment label |

## Package Structure

```
otel/
├── config.go            # Config struct
├── exporter.go          # OTLP log + metric exporter creation (gRPC/HTTP)
├── init.go              # Start() / Shutdown() lifecycle, error handler
├── logger_hook.go       # logrus.Hook → OTEL log records (with panic recovery)
├── metrics_exporter.go  # MetricsBridge (MeterProvider + PeriodicReader)
├── resource.go          # OTEL resource attributes (service, host, OS)
└── *_test.go            # Comprehensive unit tests
```

## Lifecycle

```go
// In your main command file — OTEL has its own lifecycle
import "github.com/drone-runners/drone-runner-aws/otel"

// Start OTEL integration
otelConfig := otel.Config{
    Enabled:       true,
    Endpoint:      "localhost:4317",
    Protocol:      "grpc",
    Insecure:      true,
    ExportLogs:    true,
    ExportMetrics: true,
    ServiceName:   "drone-runner-aws",
    Environment:   "production",
}

if err := otel.Start(ctx, &otelConfig, version, logger); err != nil {
    log.WithError(err).Error("Failed to start OTEL")
}

defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    otel.Shutdown(shutdownCtx)
}()
```

`Start()` is safe to call multiple times — the previous manager is shut down before reinitializing. `Shutdown()` flushes all pending data and closes exporter connections with a 15-second timeout.

## Usage Example

### Basic Setup

```go
package main

import (
    "context"
    "time"

    "github.com/drone-runners/drone-runner-aws/otel"
    "github.com/sirupsen/logrus"
)

func main() {
    ctx := context.Background()
    logger := logrus.New()

    config := otel.Config{
        Enabled:       true,
        Endpoint:      "localhost:4317",
        Protocol:      "grpc",
        Insecure:      true,
        ExportLogs:    true,
        ExportMetrics: true,
        ServiceName:   "drone-runner-aws",
        ServiceVersion: "1.0.0",
        Environment:   "development",
    }

    if err := otel.Start(ctx, &config, "1.0.0", logger); err != nil {
        logger.WithError(err).Fatal("Failed to start OTEL")
    }
    defer otel.Shutdown(ctx)

    // Your application code here...
    logger.Info("Application started with OTEL integration")
}
```

### Using the Metrics Bridge

```go
// Get the metrics bridge to create custom metrics
bridge := otel.GetMetricsBridge()
if bridge != nil {
    meter := bridge.GetMeterProvider().Meter("my-metrics")
    counter, _ := meter.Int64Counter("my_counter")
    counter.Add(ctx, 1)
}
```

### Updating Log Context

```go
// Add custom context to all logs
if hook := otel.GetLogHook(); hook != nil {
    hook.UpdateContext(map[string]string{
        "accountId": "abc123",
        "poolId":    "pool-1",
    })
}
```

## Testing

Run the tests with:

```bash
go test ./otel/... -v
```

The test suite covers:
- Configuration defaults and full population
- Resource attribute creation
- Log and metric exporter creation (gRPC and HTTP)
- Logger hook fire/close behavior with panic recovery
- Metrics bridge meter creation and recording
- Start/Shutdown lifecycle management
- Concurrent access safety
- Error handling and edge cases
