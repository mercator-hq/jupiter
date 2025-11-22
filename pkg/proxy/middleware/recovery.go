package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"mercator-hq/jupiter/pkg/proxy/types"
)

// RecoveryMiddleware recovers from panics in HTTP handlers and returns a 500
// Internal Server Error response in OpenAI error format. It logs the panic
// with stack trace for debugging but does not expose internal details to clients.
//
// Example usage:
//
//	handler = RecoveryMiddleware(handler)
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID for correlation
				requestID := GetRequestID(r.Context())

				// Capture stack trace
				stack := debug.Stack()

				// Log the panic with stack trace
				slog.ErrorContext(r.Context(), "panic in handler",
					"error", err,
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(stack),
				)

				// Create OpenAI-compatible error response
				errResp := types.NewServerError(
					"An internal error occurred. Please try again later.",
				)

				// Write error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				// Encode error response (ignore encoding errors at this point)
				_ = json.NewEncoder(w).Encode(errResp)
			}
		}()

		// Call next handler
		next.ServeHTTP(w, r)
	})
}
