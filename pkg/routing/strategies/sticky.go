package strategies

import (
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/routing"
)

// StickyStrategy implements sticky routing where requests from the same
// user/session/API key are routed to the same provider.
//
// It uses a cache with TTL and LRU eviction to maintain provider affinity.
// On cache miss, it falls back to a wrapped strategy (typically round-robin).
type StickyStrategy struct {
	// cache stores sticky routing entries
	cache *routing.StickyCache

	// fallbackStrategy is used when cache misses occur
	fallbackStrategy RoutingStrategy

	// keyType determines what field to use as the cache key
	// Options: "user", "api_key", "session", "composite"
	keyType string
}

// NewStickyStrategy creates a new sticky routing strategy.
// The fallback strategy is used when there's a cache miss.
// Common fallback is round-robin.
func NewStickyStrategy(cache *routing.StickyCache, fallbackStrategy RoutingStrategy, keyType string) *StickyStrategy {
	if keyType == "" {
		keyType = "user" // Default to user-based sticky routing
	}

	return &StickyStrategy{
		cache:            cache,
		fallbackStrategy: fallbackStrategy,
		keyType:          keyType,
	}
}

// SelectProvider selects a provider using sticky routing.
//
// Algorithm:
//  1. Build cache key from request (based on keyType)
//  2. Check cache for existing provider assignment
//  3. If cache hit and provider is in available list, return it
//  4. If cache miss, use fallback strategy and cache the result
func (s *StickyStrategy) SelectProvider(req *routing.RoutingRequest, available []providers.Provider) (providers.Provider, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no providers available for sticky routing")
	}

	// Build cache key based on configured key type
	key := s.buildCacheKey(req)
	if key == "" {
		// No key available, fall back to wrapped strategy
		return s.fallbackStrategy.SelectProvider(req, available)
	}

	// Check cache for existing assignment
	providerName, found := s.cache.Get(key)
	if found {
		// Cache hit - check if provider is still available
		for _, p := range available {
			if p.GetName() == providerName {
				return p, nil
			}
		}
		// Provider no longer available, fall through to fallback
	}

	// Cache miss or provider unavailable - use fallback strategy
	provider, err := s.fallbackStrategy.SelectProvider(req, available)
	if err != nil {
		return nil, err
	}

	// Cache the new assignment
	s.cache.Set(key, provider.GetName())

	return provider, nil
}

// buildCacheKey builds a cache key from the request based on keyType.
func (s *StickyStrategy) buildCacheKey(req *routing.RoutingRequest) string {
	switch s.keyType {
	case "user":
		return req.User
	case "api_key":
		return req.APIKey
	case "session":
		return req.SessionID
	case "composite":
		// Composite key: use first non-empty value
		if req.User != "" {
			return "user:" + req.User
		}
		if req.APIKey != "" {
			return "apikey:" + req.APIKey
		}
		if req.SessionID != "" {
			return "session:" + req.SessionID
		}
		return ""
	default:
		return req.User // Default to user
	}
}

// GetName returns the strategy name.
func (s *StickyStrategy) GetName() string {
	return "sticky"
}

// Reset resets the sticky routing cache.
func (s *StickyStrategy) Reset() {
	s.cache.Clear()
	s.fallbackStrategy.Reset()
}
