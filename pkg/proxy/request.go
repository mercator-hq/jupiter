package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"mercator-hq/jupiter/pkg/proxy/types"
)

const (
	// MaxRequestBodySize is the maximum allowed request body size (10MB).
	MaxRequestBodySize = 10 * 1024 * 1024

	// AuthorizationHeader is the HTTP header for API key authentication.
	AuthorizationHeader = "Authorization"

	// UserIDHeader is the HTTP header for user ID tracking.
	UserIDHeader = "X-User-ID"

	// RequestIDHeader is the HTTP header for request ID propagation.
	RequestIDHeader = "X-Request-ID"
)

// ParseChatCompletionRequest parses an HTTP request body into a ChatCompletionRequest.
// It validates the JSON format, enforces size limits, and validates required fields.
//
// The request body is limited to MaxRequestBodySize to prevent memory exhaustion.
// If the body exceeds this limit, an error is returned.
//
// Example usage:
//
//	req, err := ParseChatCompletionRequest(r)
//	if err != nil {
//	    // Handle validation error
//	    return err
//	}
func ParseChatCompletionRequest(r *http.Request) (*types.ChatCompletionRequest, error) {
	// Enforce request body size limit
	limitedReader := io.LimitReader(r.Body, MaxRequestBodySize)

	// Read the request body
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Check if body exceeded size limit
	if len(body) >= MaxRequestBodySize {
		return nil, &RequestError{
			Message: fmt.Sprintf("request body exceeds maximum size of %d bytes", MaxRequestBodySize),
			Code:    types.CodeRequestTooLarge,
			Param:   "body",
		}
	}

	// Parse JSON
	var req types.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, &RequestError{
			Message: fmt.Sprintf("invalid JSON: %v", err),
			Code:    types.CodeInvalidJSON,
			Param:   "body",
		}
	}

	// Validate required fields and constraints
	if err := req.Validate(); err != nil {
		if valErr, ok := err.(*types.ValidationError); ok {
			return nil, &RequestError{
				Message: valErr.Message,
				Code:    types.CodeInvalidValue,
				Param:   valErr.Field,
			}
		}
		return nil, err
	}

	return &req, nil
}

// ExtractAPIKey extracts the API key from the Authorization header.
// It expects the format "Bearer <api-key>" following OpenAI conventions.
//
// Example:
//
//	Authorization: Bearer sk-1234567890abcdef
//
// If the header is missing or malformed, an empty string is returned.
func ExtractAPIKey(r *http.Request) string {
	authHeader := r.Header.Get(AuthorizationHeader)
	if authHeader == "" {
		return ""
	}

	// Expected format: "Bearer <api-key>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// ExtractUserID extracts the user ID from the X-User-ID header.
// If the header is not present, it returns an empty string.
//
// The user ID can be used for tracking, rate limiting, and audit logging.
func ExtractUserID(r *http.Request) string {
	return r.Header.Get(UserIDHeader)
}

// ExtractRequestID extracts the request ID from the X-Request-ID header.
// If the header is not present, it returns an empty string.
//
// This allows clients to provide their own request IDs for correlation.
// If not provided, the middleware will generate one.
func ExtractRequestID(r *http.Request) string {
	return r.Header.Get(RequestIDHeader)
}

// RequestError represents a request parsing or validation error.
type RequestError struct {
	Message string
	Code    string
	Param   string
}

// Error implements the error interface.
func (e *RequestError) Error() string {
	return e.Message
}

// ToErrorResponse converts a RequestError to an OpenAI-compatible error response.
func (e *RequestError) ToErrorResponse() *types.ErrorResponse {
	return types.NewInvalidRequestError(e.Message, e.Param, e.Code)
}
