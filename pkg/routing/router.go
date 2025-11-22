package routing

import (
	"context"

	"mercator-hq/jupiter/pkg/providers"
)

// Router is the main interface for routing LLM requests to providers.
// It orchestrates the routing decision process by:
//   - Applying policy-driven routing (if specified)
//   - Using the configured routing strategy
//   - Filtering providers by health and model capability
//   - Handling fallback to alternative providers
//   - Tracking routing statistics
//
// Router implementations must be thread-safe for concurrent use.
//
// Example usage:
//
//	router, err := NewRouter(config, providers)
//	if err != nil {
//	    return err
//	}
//
//	req := &RoutingRequest{
//	    RequestID: "req-123",
//	    Model:     "gpt-4",
//	    User:      "user-456",
//	}
//
//	result, err := router.RouteRequest(ctx, req)
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Routed to: %s\n", result.ProviderName)
//	provider := result.Provider
//	// Use provider to send request...
type Router interface {
	// RouteRequest selects the optimal provider for the given request.
	//
	// The routing decision follows this precedence:
	//  1. Policy override (if PolicyDecision contains route action)
	//  2. Manual provider selection (if PreferredProvider specified)
	//  3. Configured routing strategy (round-robin, sticky, etc.)
	//  4. Default provider (if configured and all else fails)
	//
	// The context is used for cancellation and timeout control.
	// If the context is cancelled, RouteRequest returns immediately.
	//
	// Returns RoutingResult with selected provider on success.
	// Returns error if no suitable provider can be found.
	RouteRequest(ctx context.Context, req *RoutingRequest) (*RoutingResult, error)

	// GetStrategy returns the name of the configured routing strategy.
	GetStrategy() string

	// GetStats returns current routing statistics.
	// The returned stats are a snapshot and won't be updated.
	GetStats() *RoutingStats

	// UpdateProviders updates the available provider pool.
	// This is called when providers are added/removed via configuration reload.
	//
	// The providers map keys are provider names, values are Provider instances.
	// Returns error if the update fails (e.g., invalid provider configuration).
	UpdateProviders(providers map[string]providers.Provider) error

	// Close closes the router and releases resources.
	// After calling Close, the router should not be used.
	Close() error
}
