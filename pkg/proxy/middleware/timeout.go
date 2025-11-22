package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"mercator-hq/jupiter/pkg/proxy/types"
)

// TimeoutMiddleware enforces a per-request timeout using context.WithTimeout.
// If the timeout is exceeded, the request context is cancelled and a 504
// Gateway Timeout error is returned.
//
// The timeout applies to the entire request processing pipeline including
// provider requests. Handlers should check context.Done() to detect cancellation.
//
// Example usage:
//
//	handler = TimeoutMiddleware(60 * time.Second)(handler)
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create timeout context
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			// Run handler in goroutine
			go func() {
				defer close(done)
				next.ServeHTTP(w, r.WithContext(ctx))
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed successfully
				return

			case <-ctx.Done():
				// Timeout occurred
				if ctx.Err() == context.DeadlineExceeded {
					// Get request ID for logging
					requestID := GetRequestID(r.Context())

					// Create timeout error response
					errResp := types.NewGatewayTimeoutError(
						"Request timeout: the request took too long to complete",
					)

					// Log the timeout
					// Note: Using original context r.Context() since ctx is cancelled
					// slog.ErrorContext(r.Context(), "request timeout",
					// 	"request_id", requestID,
					// 	"method", r.Method,
					// 	"path", r.URL.Path,
					// 	"timeout", timeout.String(),
					// )

					// Write timeout error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusGatewayTimeout)

					// Encode error response (ignore encoding errors)
					_ = json.NewEncoder(w).Encode(errResp)

					// Note: The handler goroutine will receive ctx.Done() signal
					// and should clean up resources accordingly
					_ = requestID // Avoid unused variable warning
				}
			}
		})
	}
}
