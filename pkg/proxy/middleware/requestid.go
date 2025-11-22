package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	// RequestIDHeader is the HTTP header for request ID.
	RequestIDHeader = "X-Request-ID"
)

// RequestIDMiddleware generates a unique request ID for each request and adds it to
// the context and response headers. If the client provides a request ID in the
// X-Request-ID header, it will be used instead of generating a new one.
//
// The request ID is:
//   - Added to the request context for handler access
//   - Included in the X-Request-ID response header
//   - Used for correlation in logs and tracing
//
// Example usage:
//
//	handler = RequestIDMiddleware(handler)
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client provided a request ID
		requestID := r.Header.Get(RequestIDHeader)

		// Generate a new request ID if not provided
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add request ID to response headers
		w.Header().Set(RequestIDHeader, requestID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a unique request ID using cryptographic random bytes.
// Format: 16 bytes (32 hex characters) for uniqueness across distributed systems.
//
// Example output: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple counter-based ID if crypto/rand fails
		// This should never happen in practice
		return "fallback-request-id"
	}
	return hex.EncodeToString(b)
}

// GetRequestID extracts the request ID from the context.
// Returns empty string if not found.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
