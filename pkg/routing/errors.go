package routing

import (
	"errors"
	"fmt"
	"strings"
)

// Common routing errors that can be checked with errors.Is().
var (
	// ErrNoHealthyProviders is returned when all providers are unhealthy.
	ErrNoHealthyProviders = errors.New("no healthy providers available")

	// ErrModelNotSupported is returned when no provider supports the requested model.
	ErrModelNotSupported = errors.New("model not supported by any provider")

	// ErrProviderNotFound is returned when manual provider selection fails.
	ErrProviderNotFound = errors.New("provider not found")

	// ErrAllProvidersFailed is returned when all fallback attempts are exhausted.
	ErrAllProvidersFailed = errors.New("all providers failed")

	// ErrInvalidStrategy is returned when an unknown routing strategy is configured.
	ErrInvalidStrategy = errors.New("invalid routing strategy")

	// ErrNoProvidersConfigured is returned when no providers are available.
	ErrNoProvidersConfigured = errors.New("no providers configured")
)

// NoHealthyProvidersError is returned when no healthy providers are available
// for routing a request.
type NoHealthyProvidersError struct {
	// AttemptedProviders contains the names of providers that were checked.
	AttemptedProviders []string

	// Model is the requested model.
	Model string
}

// Error implements the error interface.
func (e *NoHealthyProvidersError) Error() string {
	return fmt.Sprintf("no healthy providers available for model %q (attempted: %s)",
		e.Model, strings.Join(e.AttemptedProviders, ", "))
}

// Is implements error matching for errors.Is().
func (e *NoHealthyProvidersError) Is(target error) bool {
	return target == ErrNoHealthyProviders
}

// ModelNotSupportedError is returned when the requested model is not supported
// by any configured provider.
type ModelNotSupportedError struct {
	// Model is the requested model that is not supported.
	Model string

	// AvailableModels contains models that are supported.
	AvailableModels []string
}

// Error implements the error interface.
func (e *ModelNotSupportedError) Error() string {
	if len(e.AvailableModels) == 0 {
		return fmt.Sprintf("model %q not supported by any provider", e.Model)
	}
	return fmt.Sprintf("model %q not supported by any provider (available models: %s)",
		e.Model, strings.Join(e.AvailableModels, ", "))
}

// Is implements error matching for errors.Is().
func (e *ModelNotSupportedError) Is(target error) bool {
	return target == ErrModelNotSupported
}

// ProviderNotFoundError is returned when an explicitly requested provider
// (via manual selection) does not exist.
type ProviderNotFoundError struct {
	// ProviderName is the requested provider that was not found.
	ProviderName string

	// AvailableProviders contains the names of configured providers.
	AvailableProviders []string
}

// Error implements the error interface.
func (e *ProviderNotFoundError) Error() string {
	return fmt.Sprintf("provider %q not found (available providers: %s)",
		e.ProviderName, strings.Join(e.AvailableProviders, ", "))
}

// Is implements error matching for errors.Is().
func (e *ProviderNotFoundError) Is(target error) bool {
	return target == ErrProviderNotFound
}

// AllProvidersFailedError is returned when all fallback attempts have been
// exhausted and no provider could successfully handle the request.
type AllProvidersFailedError struct {
	// AttemptedProviders contains the names of providers that were tried.
	AttemptedProviders []string

	// LastError is the error from the last attempted provider.
	LastError error

	// Model is the requested model.
	Model string
}

// Error implements the error interface.
func (e *AllProvidersFailedError) Error() string {
	return fmt.Sprintf("all providers failed for model %q (attempted: %s, last error: %v)",
		e.Model, strings.Join(e.AttemptedProviders, ", "), e.LastError)
}

// Is implements error matching for errors.Is().
func (e *AllProvidersFailedError) Is(target error) bool {
	return target == ErrAllProvidersFailed
}

// Unwrap returns the wrapped error for error chain traversal.
func (e *AllProvidersFailedError) Unwrap() error {
	return e.LastError
}

// InvalidStrategyError is returned when the configured routing strategy
// is not recognized.
type InvalidStrategyError struct {
	// Strategy is the invalid strategy name.
	Strategy string

	// AvailableStrategies contains the valid strategy names.
	AvailableStrategies []string
}

// Error implements the error interface.
func (e *InvalidStrategyError) Error() string {
	return fmt.Sprintf("invalid routing strategy %q (available strategies: %s)",
		e.Strategy, strings.Join(e.AvailableStrategies, ", "))
}

// Is implements error matching for errors.Is().
func (e *InvalidStrategyError) Is(target error) bool {
	return target == ErrInvalidStrategy
}
