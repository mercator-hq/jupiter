// Package telemetry provides comprehensive observability for Mercator Jupiter.
//
// # Overview
//
// The telemetry package implements structured logging, Prometheus metrics,
// OpenTelemetry distributed tracing, and health check endpoints. It provides
// visibility into runtime behavior while maintaining low overhead (<50µs per
// request).
//
// # Components
//
//   - logging: Structured logging with PII redaction
//   - metrics: Prometheus metrics collection
//   - tracing: OpenTelemetry distributed tracing
//   - health: Health check endpoints
//
// # Usage
//
//	// Initialize telemetry
//	cfg := config.GetConfig()
//	tel := telemetry.New(&cfg.Telemetry, "v1.0.0", "abc123", "2025-11-20")
//
//	// Get logger
//	logger := tel.Logger()
//	logger.Info("Request processed", "duration_ms", 123)
//
//	// Record metrics
//	tel.Metrics().RecordRequest("openai", "gpt-4", "success", time.Second, 1500, 0.05)
//
//	// Create span
//	ctx, span := tel.Tracer().Start(ctx, "operation")
//	defer span.End()
//
// # Performance
//
// The telemetry package is designed for minimal overhead:
//
//   - Logging: <10µs when enabled, <1µs when disabled
//   - Metrics: <50µs per metric update
//   - Tracing: <100µs per span
//   - Total overhead: <0.5% of request time
//
// # PII Protection
//
// By default, all PII is automatically redacted from logs:
//
//   - API keys: sk-abc123 → sk-***
//   - Emails: user@example.com → u***@example.com
//   - SSN: 123-45-6789 → ***-**-****
//   - IP addresses: 192.168.1.1 → 192.*.*.*
//
// Custom redaction patterns can be configured.
package telemetry
