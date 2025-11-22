package types

// ErrorResponse represents an OpenAI-compatible error response.
// This is returned for all error conditions to ensure compatibility with
// OpenAI SDKs and tools.
type ErrorResponse struct {
	// Error contains the error details.
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information.
type ErrorDetail struct {
	// Message is a human-readable error message.
	Message string `json:"message"`

	// Type categorizes the error.
	// Possible values: "invalid_request_error", "authentication_error",
	// "permission_denied", "not_found", "rate_limit_exceeded",
	// "server_error", "bad_gateway", "service_unavailable", "gateway_timeout".
	Type string `json:"type"`

	// Param is the name of the parameter that caused the error (if applicable).
	Param string `json:"param,omitempty"`

	// Code is a machine-readable error code.
	Code string `json:"code,omitempty"`
}

// Error type constants matching OpenAI API specification.
const (
	// ErrorTypeInvalidRequest indicates a client-side error (400).
	ErrorTypeInvalidRequest = "invalid_request_error"

	// ErrorTypeAuthentication indicates an authentication failure (401).
	ErrorTypeAuthentication = "authentication_error"

	// ErrorTypePermissionDenied indicates an authorization failure (403).
	ErrorTypePermissionDenied = "permission_denied"

	// ErrorTypeNotFound indicates a resource was not found (404).
	ErrorTypeNotFound = "not_found"

	// ErrorTypeRateLimitExceeded indicates too many requests (429).
	ErrorTypeRateLimitExceeded = "rate_limit_exceeded"

	// ErrorTypeServerError indicates an internal server error (500).
	ErrorTypeServerError = "server_error"

	// ErrorTypeBadGateway indicates a provider error (502).
	ErrorTypeBadGateway = "bad_gateway"

	// ErrorTypeServiceUnavailable indicates temporary unavailability (503).
	ErrorTypeServiceUnavailable = "service_unavailable"

	// ErrorTypeGatewayTimeout indicates a provider timeout (504).
	ErrorTypeGatewayTimeout = "gateway_timeout"
)

// Error code constants for common error scenarios.
const (
	// CodeMissingField indicates a required field is missing.
	CodeMissingField = "missing_field"

	// CodeInvalidValue indicates a field has an invalid value.
	CodeInvalidValue = "invalid_value"

	// CodeInvalidJSON indicates the request body is not valid JSON.
	CodeInvalidJSON = "invalid_json"

	// CodeModelNotFound indicates the requested model is not available.
	CodeModelNotFound = "model_not_found"

	// CodeProviderError indicates an error from the LLM provider.
	CodeProviderError = "provider_error"

	// CodeProviderTimeout indicates the provider request timed out.
	CodeProviderTimeout = "provider_timeout"

	// CodeProviderUnavailable indicates no healthy providers are available.
	CodeProviderUnavailable = "provider_unavailable"

	// CodeRequestTooLarge indicates the request payload is too large.
	CodeRequestTooLarge = "request_too_large"

	// CodeInternalError indicates an internal server error.
	CodeInternalError = "internal_error"
)

// NewErrorResponse creates a new error response with the given details.
func NewErrorResponse(message, errorType, param, code string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
			Param:   param,
			Code:    code,
		},
	}
}

// NewInvalidRequestError creates an error response for invalid requests (400).
func NewInvalidRequestError(message, param, code string) *ErrorResponse {
	return NewErrorResponse(message, ErrorTypeInvalidRequest, param, code)
}

// NewServerError creates an error response for internal server errors (500).
func NewServerError(message string) *ErrorResponse {
	return NewErrorResponse(message, ErrorTypeServerError, "", CodeInternalError)
}

// NewBadGatewayError creates an error response for provider errors (502).
func NewBadGatewayError(message string) *ErrorResponse {
	return NewErrorResponse(message, ErrorTypeBadGateway, "", CodeProviderError)
}

// NewServiceUnavailableError creates an error response for temporary unavailability (503).
func NewServiceUnavailableError(message string) *ErrorResponse {
	return NewErrorResponse(message, ErrorTypeServiceUnavailable, "", CodeProviderUnavailable)
}

// NewGatewayTimeoutError creates an error response for provider timeouts (504).
func NewGatewayTimeoutError(message string) *ErrorResponse {
	return NewErrorResponse(message, ErrorTypeGatewayTimeout, "", CodeProviderTimeout)
}

// HTTPStatusCode returns the appropriate HTTP status code for the error type.
func (e *ErrorDetail) HTTPStatusCode() int {
	switch e.Type {
	case ErrorTypeInvalidRequest:
		return 400
	case ErrorTypeAuthentication:
		return 401
	case ErrorTypePermissionDenied:
		return 403
	case ErrorTypeNotFound:
		return 404
	case ErrorTypeRateLimitExceeded:
		return 429
	case ErrorTypeServerError:
		return 500
	case ErrorTypeBadGateway:
		return 502
	case ErrorTypeServiceUnavailable:
		return 503
	case ErrorTypeGatewayTimeout:
		return 504
	default:
		return 500
	}
}
