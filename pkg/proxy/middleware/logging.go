package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// newResponseWriter creates a new response writer wrapper.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200
	}
}

// WriteHeader captures the status code before writing.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write ensures WriteHeader is called if not already done.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware logs HTTP requests and responses with structured logging.
// It records method, path, status code, latency, request ID, and other metadata.
//
// Log format (JSON):
//
//	{
//	  "time": "2025-11-16T10:30:00Z",
//	  "level": "INFO",
//	  "msg": "request completed",
//	  "method": "POST",
//	  "path": "/v1/chat/completions",
//	  "status": 200,
//	  "latency_ms": 1250,
//	  "request_id": "a1b2c3d4...",
//	  "user_agent": "openai-python/1.0.0",
//	  "remote_addr": "192.168.1.100:54321"
//	}
//
// Example usage:
//
//	handler = LoggingMiddleware(handler)
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record start time
		startTime := time.Now()
		ctx := context.WithValue(r.Context(), StartTimeKey, startTime)

		// Wrap response writer to capture status code
		rw := newResponseWriter(w)

		// Log request start (debug level)
		requestID := GetRequestID(ctx)
		slog.DebugContext(ctx, "request started",
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", requestID,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// Call next handler
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Calculate latency
		latency := time.Since(startTime)

		// Log request completion (info level)
		logLevel := slog.LevelInfo
		if rw.statusCode >= 500 {
			logLevel = slog.LevelError
		} else if rw.statusCode >= 400 {
			logLevel = slog.LevelWarn
		}

		slog.Log(ctx, logLevel, "request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"latency_ms", latency.Milliseconds(),
			"request_id", requestID,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

// GetStartTime extracts the request start time from the context.
// Returns zero time if not found.
func GetStartTime(ctx context.Context) time.Time {
	if startTime, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return startTime
	}
	return time.Time{}
}
