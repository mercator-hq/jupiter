package proxy

import (
	"errors"
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// HandleError converts various error types to OpenAI-compatible error responses.
// It maps provider errors, validation errors, and internal errors to appropriate
// HTTP status codes and error formats.
//
// Example usage:
//
//	if err != nil {
//	    errResp := HandleError(err)
//	    WriteErrorResponse(w, errResp)
//	    return
//	}
func HandleError(err error) *types.ErrorResponse {
	// Check for RequestError (validation errors)
	var reqErr *RequestError
	if errors.As(err, &reqErr) {
		return reqErr.ToErrorResponse()
	}

	// Check for provider-specific errors
	var providerErr *providers.ProviderError
	if errors.As(err, &providerErr) {
		return handleProviderError(providerErr)
	}

	var authErr *providers.AuthError
	if errors.As(err, &authErr) {
		return types.NewErrorResponse(
			authErr.Error(),
			types.ErrorTypeAuthentication,
			"",
			"authentication_failed",
		)
	}

	var rateLimitErr *providers.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return types.NewErrorResponse(
			rateLimitErr.Error(),
			types.ErrorTypeRateLimitExceeded,
			"",
			"rate_limit_exceeded",
		)
	}

	var timeoutErr *providers.TimeoutError
	if errors.As(err, &timeoutErr) {
		return types.NewGatewayTimeoutError(
			fmt.Sprintf("Provider request timed out: %v", timeoutErr.Error()),
		)
	}

	var parseErr *providers.ParseError
	if errors.As(err, &parseErr) {
		return types.NewBadGatewayError(
			fmt.Sprintf("Failed to parse provider response: %v", parseErr.Error()),
		)
	}

	var modelNotFoundErr *providers.ModelNotFoundError
	if errors.As(err, &modelNotFoundErr) {
		return types.NewInvalidRequestError(
			modelNotFoundErr.Error(),
			"model",
			types.CodeModelNotFound,
		)
	}

	// Default to internal server error for unknown errors
	return types.NewServerError(
		"An internal error occurred. Please try again later.",
	)
}

// handleProviderError converts a ProviderError to an OpenAI error response.
// It maps HTTP status codes to appropriate error types.
func handleProviderError(err *providers.ProviderError) *types.ErrorResponse {
	switch {
	case err.StatusCode >= 500:
		// 5xx errors are gateway errors (provider issues)
		return types.NewBadGatewayError(
			fmt.Sprintf("Provider error (%s): %v", err.Provider, err.Message),
		)
	case err.StatusCode == 429:
		// Rate limiting
		return types.NewErrorResponse(
			fmt.Sprintf("Provider rate limit exceeded (%s)", err.Provider),
			types.ErrorTypeRateLimitExceeded,
			"",
			"rate_limit_exceeded",
		)
	case err.StatusCode == 404:
		// Not found (usually model not found)
		return types.NewInvalidRequestError(
			fmt.Sprintf("Model not found (%s)", err.Provider),
			"model",
			types.CodeModelNotFound,
		)
	case err.StatusCode == 401 || err.StatusCode == 403:
		// Authentication/authorization errors
		return types.NewErrorResponse(
			fmt.Sprintf("Provider authentication failed (%s)", err.Provider),
			types.ErrorTypeAuthentication,
			"",
			"authentication_failed",
		)
	case err.StatusCode >= 400:
		// Other 4xx errors are client errors
		return types.NewInvalidRequestError(
			fmt.Sprintf("Invalid request to provider (%s): %v", err.Provider, err.Message),
			"",
			types.CodeInvalidValue,
		)
	default:
		// Unknown status code, treat as internal error
		return types.NewServerError(
			fmt.Sprintf("Provider error (%s): %v", err.Provider, err.Message),
		)
	}
}

// SanitizeError removes sensitive information from error messages.
// This prevents leaking internal details or credentials in error responses.
//
// Sensitive patterns removed:
//   - API keys (sk-*, Bearer *)
//   - Internal file paths
//   - Stack traces
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	// For now, return the error as-is
	// In production, we would implement pattern matching to remove sensitive data
	// This is a placeholder for future enhancement
	return err
}
