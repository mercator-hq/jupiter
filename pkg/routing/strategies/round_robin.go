package strategies

import (
	"fmt"
	"sync/atomic"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/routing"
)

// RoundRobinStrategy implements round-robin load balancing across providers.
// It distributes requests evenly across available providers, with optional
// weighting to send more traffic to specific providers.
//
// The strategy is thread-safe and uses atomic counters for concurrent access.
// Counters are reset on overflow to prevent unbounded growth.
type RoundRobinStrategy struct {
	// counter is the global round-robin counter
	counter atomic.Int64

	// weights maps provider names to their weights (default: 1)
	// Higher weight = more traffic
	weights map[string]int
}

// NewRoundRobinStrategy creates a new round-robin strategy.
// Weights is optional; if nil or empty, all providers have equal weight (1).
func NewRoundRobinStrategy(weights map[string]int) *RoundRobinStrategy {
	if weights == nil {
		weights = make(map[string]int)
	}

	return &RoundRobinStrategy{
		weights: weights,
	}
}

// SelectProvider selects the next provider using weighted round-robin.
//
// Algorithm:
//  1. Build a weighted provider list (each provider appears weight times)
//  2. Use atomic counter % list length to select provider
//  3. Increment counter atomically
//
// Returns error if no providers are available.
func (s *RoundRobinStrategy) SelectProvider(req *routing.RoutingRequest, available []providers.Provider) (providers.Provider, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no providers available for round-robin selection")
	}

	// Single provider - no need for round-robin
	if len(available) == 1 {
		return available[0], nil
	}

	// Build weighted provider list
	weightedProviders := s.buildWeightedList(available)
	if len(weightedProviders) == 0 {
		// All providers have zero weight, fall back to unweighted
		weightedProviders = available
	}

	// Get next index using atomic counter and increment
	count := s.counter.Add(1) - 1 // Get value before increment

	// Handle overflow (reset counter when it gets too large)
	// Use modulo to keep counter in reasonable range
	if count >= 1_000_000_000 {
		// Reset to 0 atomically
		s.counter.CompareAndSwap(count+1, 0)
		count = 0
	}

	// Select provider using modulo
	index := int(count % int64(len(weightedProviders)))
	return weightedProviders[index], nil
}

// buildWeightedList creates a weighted provider list where each provider
// appears according to its weight.
//
// Example: Provider A (weight 2), Provider B (weight 1)
// Result: [A, A, B]
func (s *RoundRobinStrategy) buildWeightedList(providerList []providers.Provider) []providers.Provider {
	// If no weights configured, return providers as-is
	if len(s.weights) == 0 {
		return providerList
	}

	var result []providers.Provider

	for _, p := range providerList {
		weight := s.getWeight(p.GetName())

		// Zero or negative weight means exclude provider
		if weight <= 0 {
			continue
		}

		// Add provider weight times
		for i := 0; i < weight; i++ {
			result = append(result, p)
		}
	}

	return result
}

// getWeight returns the configured weight for a provider.
// Returns 1 if no weight is configured (default weight).
func (s *RoundRobinStrategy) getWeight(providerName string) int {
	if weight, ok := s.weights[providerName]; ok {
		return weight
	}
	return 1 // Default weight
}

// GetName returns the strategy name.
func (s *RoundRobinStrategy) GetName() string {
	return "round-robin"
}

// Reset resets the round-robin counter.
// This is primarily used for testing.
func (s *RoundRobinStrategy) Reset() {
	s.counter.Store(0)
}
