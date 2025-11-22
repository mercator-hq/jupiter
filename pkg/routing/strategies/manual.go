package strategies

import (
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/routing"
)

// ManualStrategy implements manual provider selection where the request
// explicitly specifies which provider to use.
//
// If the specified provider is not available, it can optionally fall back
// to a wrapped strategy.
type ManualStrategy struct {
	// fallbackStrategy is used when the requested provider is unavailable
	fallbackStrategy RoutingStrategy

	// allowFallback controls whether to fall back if provider unavailable
	allowFallback bool
}

// NewManualStrategy creates a new manual selection strategy.
// If allowFallback is true and the requested provider is unavailable,
// it will fall back to the wrapped strategy.
func NewManualStrategy(fallbackStrategy RoutingStrategy, allowFallback bool) *ManualStrategy {
	return &ManualStrategy{
		fallbackStrategy: fallbackStrategy,
		allowFallback:    allowFallback,
	}
}

// SelectProvider selects the explicitly requested provider.
//
// Algorithm:
//  1. Check if PreferredProvider is specified in request
//  2. If yes, search for it in available providers
//  3. If found, return it
//  4. If not found and fallback allowed, use fallback strategy
//  5. If not found and fallback not allowed, return error
func (s *ManualStrategy) SelectProvider(req *routing.RoutingRequest, available []providers.Provider) (providers.Provider, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no providers available for manual selection")
	}

	// Check if manual provider selection is requested
	if req.PreferredProvider == "" {
		// No preference specified, use fallback strategy
		if s.fallbackStrategy != nil {
			return s.fallbackStrategy.SelectProvider(req, available)
		}
		// No preference and no fallback - return first available
		return available[0], nil
	}

	// Search for the requested provider
	for _, p := range available {
		if p.GetName() == req.PreferredProvider {
			return p, nil
		}
	}

	// Requested provider not found
	if s.allowFallback && s.fallbackStrategy != nil {
		// Fall back to wrapped strategy
		return s.fallbackStrategy.SelectProvider(req, available)
	}

	// No fallback - return error
	return nil, fmt.Errorf("requested provider %q not found in available providers", req.PreferredProvider)
}

// GetName returns the strategy name.
func (s *ManualStrategy) GetName() string {
	return "manual"
}

// Reset resets the strategy state.
func (s *ManualStrategy) Reset() {
	if s.fallbackStrategy != nil {
		s.fallbackStrategy.Reset()
	}
}
