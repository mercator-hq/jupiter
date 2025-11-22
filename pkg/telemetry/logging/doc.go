// Package logging provides structured logging with PII redaction.
//
// # Overview
//
// The logging package wraps Go's standard log/slog package to provide:
//   - Structured logging with JSON, text, and console formats
//   - Automatic PII redaction (API keys, emails, SSN, etc.)
//   - Context-aware logging with request IDs and metadata
//   - Async buffering for non-blocking writes
//   - Configurable log levels (debug, info, warn, error)
//
// # Usage
//
//	// Create a logger
//	logger := logging.New(logging.Config{
//	    Level:     "info",
//	    Format:    "json",
//	    RedactPII: true,
//	})
//
//	// Log structured data
//	logger.Info("Request processed",
//	    "request_id", "req-123",
//	    "api_key", "sk-abc123",  // Automatically redacted
//	    "duration_ms", 1234,
//	)
//
//	// Create context-aware logger
//	ctx := context.WithValue(ctx, logging.RequestIDKey, "req-123")
//	ctxLogger := logger.WithContext(ctx)
//	ctxLogger.Info("Processing")  // Includes request_id automatically
//
// # PII Redaction
//
// PII is automatically redacted from log fields when RedactPII is enabled:
//
//   - API keys: sk-abc123xyz → sk-***
//   - Emails: user@example.com → u***@example.com
//   - SSN: 123-45-6789 → ***-**-****
//   - IP addresses: 192.168.1.100 → 192.*.*.*
//   - Credit cards: 4111-1111-1111-1111 → ****-****-****-1111
//
// # Performance
//
// Async buffering ensures logging doesn't block request processing:
//   - <1µs when log level filters out the message
//   - <10µs when writing to buffer
//   - Dropped logs are counted if buffer is full
package logging
