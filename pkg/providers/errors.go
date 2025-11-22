package providers

import (
	"fmt"
	"time"
)

// ProviderError represents a general provider error.
// It includes the provider name, HTTP status code, and underlying error.
type ProviderError struct {
	// Provider is the name of the provider that returned the error
	Provider string

	// StatusCode is the HTTP status code (0 if not applicable)
	StatusCode int

	// Message is the error message
	Message string

	// Cause is the underlying error (if any)
	Cause error
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("provider %q error (status %d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("provider %q error: %s", e.Provider, e.Message)
}

// Unwrap returns the underlying error for error chain support.
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// AuthError represents an authentication failure.
// This occurs when the provider rejects the API key (HTTP 401 or 403).
type AuthError struct {
	// Provider is the name of the provider that rejected authentication
	Provider string

	// Message is the error message from the provider
	Message string
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	return fmt.Sprintf("provider %q authentication failed: %s", e.Provider, e.Message)
}

// RateLimitError represents a rate limit exceeded error (HTTP 429).
// It includes the retry-after duration if provided by the provider.
type RateLimitError struct {
	// Provider is the name of the provider that rate limited the request
	Provider string

	// RetryAfter is the duration to wait before retrying (if provided)
	RetryAfter time.Duration

	// Message is the error message from the provider
	Message string
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("provider %q rate limit exceeded (retry after %s): %s",
			e.Provider, e.RetryAfter, e.Message)
	}
	return fmt.Sprintf("provider %q rate limit exceeded: %s", e.Provider, e.Message)
}

// TimeoutError represents a request timeout.
// This occurs when a request exceeds the configured timeout duration.
type TimeoutError struct {
	// Provider is the name of the provider where the timeout occurred
	Provider string

	// Timeout is the configured timeout duration
	Timeout time.Duration
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("provider %q request timeout after %s", e.Provider, e.Timeout)
}

// ParseError represents a response parsing failure.
// This occurs when the provider returns a malformed response.
type ParseError struct {
	// Provider is the name of the provider that returned the malformed response
	Provider string

	// RawResponse is the raw response body that failed to parse
	RawResponse string

	// Cause is the underlying parse error
	Cause error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return fmt.Sprintf("provider %q response parse error: %v", e.Provider, e.Cause)
}

// Unwrap returns the underlying error for error chain support.
func (e *ParseError) Unwrap() error {
	return e.Cause
}

// ModelNotFoundError represents an unknown model error.
// This occurs when a requested model is not available from the provider.
type ModelNotFoundError struct {
	// Provider is the name of the provider
	Provider string

	// Model is the requested model identifier
	Model string
}

// Error implements the error interface.
func (e *ModelNotFoundError) Error() string {
	return fmt.Sprintf("provider %q does not support model %q", e.Provider, e.Model)
}

// ValidationError represents a request validation failure.
// This occurs when the request has invalid fields before sending to the provider.
type ValidationError struct {
	// Field is the name of the invalid field
	Field string

	// Message describes what is invalid about the field
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %q: %s", e.Field, e.Message)
}

// StreamError represents an error that occurred during streaming.
// This is sent through the stream channel to indicate an error.
type StreamError struct {
	// Provider is the name of the provider where the error occurred
	Provider string

	// Message is the error message
	Message string

	// Cause is the underlying error
	Cause error
}

// Error implements the error interface.
func (e *StreamError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("provider %q stream error: %s: %v", e.Provider, e.Message, e.Cause)
	}
	return fmt.Sprintf("provider %q stream error: %s", e.Provider, e.Message)
}

// Unwrap returns the underlying error for error chain support.
func (e *StreamError) Unwrap() error {
	return e.Cause
}

// ConfigError represents a provider configuration error.
// This occurs when the provider configuration is invalid.
type ConfigError struct {
	// Provider is the name of the provider with invalid configuration
	Provider string

	// Field is the configuration field that is invalid
	Field string

	// Message describes the configuration error
	Message string
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return fmt.Sprintf("provider %q configuration error for field %q: %s",
		e.Provider, e.Field, e.Message)
}
