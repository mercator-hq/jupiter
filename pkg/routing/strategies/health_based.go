package strategies

import (
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/routing"
)

// HealthBasedStrategy is a decorator that filters out unhealthy providers
// before delegating to a wrapped strategy.
//
// It ensures that only healthy providers are considered for routing,
// providing automatic failover when providers become unhealthy.
type HealthBasedStrategy struct {
	// wrappedStrategy is the underlying strategy to use after health filtering
	wrappedStrategy RoutingStrategy

	// requireHealthy controls whether to strictly require healthy providers
	// If true and all providers are unhealthy, returns error
	// If false, falls back to unhealthy providers
	requireHealthy bool
}

// NewHealthBasedStrategy creates a new health-based routing strategy.
// It wraps another strategy and filters out unhealthy providers before routing.
func NewHealthBasedStrategy(wrappedStrategy RoutingStrategy, requireHealthy bool) *HealthBasedStrategy {
	return &HealthBasedStrategy{
		wrappedStrategy: wrappedStrategy,
		requireHealthy:  requireHealthy,
	}
}

// SelectProvider selects a provider from healthy providers only.
//
// Algorithm:
//  1. Filter available providers to only healthy ones
//  2. If healthy providers exist, delegate to wrapped strategy
//  3. If no healthy providers and requireHealthy=false, use all providers
//  4. If no healthy providers and requireHealthy=true, return error
func (s *HealthBasedStrategy) SelectProvider(req *routing.RoutingRequest, available []providers.Provider) (providers.Provider, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no providers available for health-based routing")
	}

	// Filter to healthy providers
	healthy := s.filterHealthy(available)

	// If we have healthy providers, use them
	if len(healthy) > 0 {
		return s.wrappedStrategy.SelectProvider(req, healthy)
	}

	// No healthy providers
	if s.requireHealthy {
		return nil, fmt.Errorf("no healthy providers available (total providers: %d)", len(available))
	}

	// Fall back to all providers (even unhealthy)
	return s.wrappedStrategy.SelectProvider(req, available)
}

// filterHealthy returns only healthy providers from the list.
func (s *HealthBasedStrategy) filterHealthy(providerList []providers.Provider) []providers.Provider {
	healthy := make([]providers.Provider, 0, len(providerList))
	for _, p := range providerList {
		if p.IsHealthy() {
			healthy = append(healthy, p)
		}
	}
	return healthy
}

// GetName returns the strategy name.
func (s *HealthBasedStrategy) GetName() string {
	return "health-based"
}

// Reset resets the wrapped strategy.
func (s *HealthBasedStrategy) Reset() {
	s.wrappedStrategy.Reset()
}
