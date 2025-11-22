package middleware

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// Context keys for storing values in request context.
const (
	// RequestIDKey stores the unique request ID.
	RequestIDKey contextKey = "request_id"

	// StartTimeKey stores the request start time for latency calculation.
	StartTimeKey contextKey = "start_time"

	// UserIDKey stores the user ID extracted from headers.
	UserIDKey contextKey = "user_id"

	// ModelKey stores the requested model name.
	ModelKey contextKey = "model"

	// DowngradedModelKey stores the downgraded model name when limit enforcement downgrades a request.
	DowngradedModelKey contextKey = "downgraded_model"
)
