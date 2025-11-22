package strategies

import (
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/routing"
)

// RoutingStrategy is the interface that all routing strategies must implement.
// It defines the contract for selecting a provider from a list of available providers.
//
// Implementations must be thread-safe as they will be called concurrently
// from multiple goroutines handling simultaneous routing requests.
//
// Example usage:
//
//	strategy := NewRoundRobinStrategy(weights)
//	provider, err := strategy.SelectProvider(req, availableProviders)
//	if err != nil {
//	    return nil, err
//	}
//	// Use selected provider...
type RoutingStrategy interface {
	// SelectProvider selects a provider from the list of available providers
	// based on the strategy's algorithm.
	//
	// The request contains metadata (model, user, session) that may influence
	// the selection. The available providers list should already be filtered
	// for health and model capability.
	//
	// Returns the selected provider and nil on success.
	// Returns nil and an error if no provider can be selected.
	//
	// Implementations must be thread-safe.
	SelectProvider(req *routing.RoutingRequest, available []providers.Provider) (providers.Provider, error)

	// GetName returns the strategy name for logging and statistics.
	// Examples: "round-robin", "sticky", "manual", "health-based"
	GetName() string

	// Reset resets the strategy's internal state.
	// This is primarily used for testing to clear counters and caches.
	Reset()
}
