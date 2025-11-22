package routing

import (
	"context"
	"fmt"
	"log/slog"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/providers"
)

// RoutingStrategy defines the interface for routing strategies.
// This is defined here to avoid import cycles with the strategies package.
type RoutingStrategy interface {
	SelectProvider(req *RoutingRequest, available []providers.Provider) (providers.Provider, error)
	GetName() string
	Reset()
}

// DefaultRouter implements the Router interface with full routing orchestration.
type DefaultRouter struct {
	// selector handles provider filtering
	selector *ProviderSelector

	// strategy is the configured routing strategy
	strategy RoutingStrategy

	// stats tracks routing statistics
	stats *AtomicRoutingStats

	// config contains routing configuration
	config *config.RoutingConfig
}

// NewRouter creates a new router with the specified configuration and strategy.
// The strategy should be created using the strategies package (e.g., strategies.NewRoundRobinStrategy).
func NewRouter(cfg *config.RoutingConfig, providerMap map[string]providers.Provider, strategy RoutingStrategy) (Router, error) {
	if cfg == nil {
		return nil, fmt.Errorf("routing config cannot be nil")
	}
	if strategy == nil {
		return nil, fmt.Errorf("routing strategy cannot be nil")
	}

	// Create provider selector
	selector := NewProviderSelector(providerMap, cfg.ModelMapping)

	return &DefaultRouter{
		selector: selector,
		strategy: strategy,
		stats:    NewAtomicRoutingStats(),
		config:   cfg,
	}, nil
}

// RouteRequest selects the optimal provider for the given request.
//
// Routing precedence:
//  1. Policy override (if PolicyDecision contains route action)
//  2. Manual provider selection (if PreferredProvider specified)
//  3. Configured routing strategy (round-robin, sticky, etc.)
//  4. Default provider (if configured and all else fails)
func (r *DefaultRouter) RouteRequest(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	r.stats.IncrementTotal()

	// Check context cancellation
	if ctx.Err() != nil {
		r.stats.IncrementErrors()
		return nil, ctx.Err()
	}

	// Get all available providers
	available := r.selector.GetAvailableProviders()
	if len(available) == 0 {
		r.stats.IncrementErrors()
		return nil, ErrNoProvidersConfigured
	}

	// Filter by model capability
	available = r.selector.FilterByModel(available, req.Model)
	if len(available) == 0 {
		r.stats.IncrementErrors()
		return nil, &ModelNotSupportedError{
			Model:           req.Model,
			AvailableModels: r.selector.GetSupportedModels(),
		}
	}

	// Filter by health status
	healthyBefore := len(available)
	available = r.selector.FilterByHealth(available)
	if len(available) < healthyBefore {
		r.stats.IncrementHealthFiltered()
	}

	// If no healthy providers and health required, return error
	if len(available) == 0 {
		r.stats.IncrementErrors()
		return nil, &NoHealthyProvidersError{
			AttemptedProviders: r.selector.GetProviderNames(),
			Model:              req.Model,
		}
	}

	// Check for policy override
	if req.PolicyDecision != nil && req.PolicyDecision.RoutingTarget != nil {
		return r.routeWithPolicy(req, available)
	}

	// Check for manual provider selection
	if req.PreferredProvider != "" {
		return r.routeManual(req, available)
	}

	// Use configured strategy
	return r.routeWithStrategy(req, available)
}

// routeWithPolicy handles policy-driven routing.
func (r *DefaultRouter) routeWithPolicy(req *RoutingRequest, available []providers.Provider) (*RoutingResult, error) {
	r.stats.IncrementPolicyOverride()

	target := req.PolicyDecision.RoutingTarget
	providerName := target.Provider

	// Find the requested provider
	for _, p := range available {
		if p.GetName() == providerName {
			r.stats.IncrementProvider(providerName)
			r.stats.IncrementStrategy("policy")

			slog.Info("policy-driven routing",
				"request_id", req.RequestID,
				"provider", providerName,
				"model", req.Model,
			)

			return &RoutingResult{
				Provider:           p,
				ProviderName:       providerName,
				Strategy:           "policy",
				Reason:             "policy route action",
				IsHealthy:          p.IsHealthy(),
				IsFallback:         false,
				AttemptedProviders: []string{providerName},
				Metadata:           make(map[string]string),
			}, nil
		}
	}

	// Policy-specified provider not available, try fallback
	if r.config.Fallback.Enabled && len(target.Fallback) > 0 {
		return r.tryFallbacks(req, available, target.Fallback)
	}

	// No fallback configured, return error
	r.stats.IncrementErrors()
	return nil, &ProviderNotFoundError{
		ProviderName:       providerName,
		AvailableProviders: getProviderNames(available),
	}
}

// routeManual handles manual provider selection.
func (r *DefaultRouter) routeManual(req *RoutingRequest, available []providers.Provider) (*RoutingResult, error) {
	r.stats.IncrementManualOverride()

	providerName := req.PreferredProvider

	// Find the requested provider
	for _, p := range available {
		if p.GetName() == providerName {
			r.stats.IncrementProvider(providerName)
			r.stats.IncrementStrategy("manual")

			slog.Info("manual provider selection",
				"request_id", req.RequestID,
				"provider", providerName,
				"model", req.Model,
			)

			return &RoutingResult{
				Provider:           p,
				ProviderName:       providerName,
				Strategy:           "manual",
				Reason:             "explicit provider selection",
				IsHealthy:          p.IsHealthy(),
				IsFallback:         false,
				AttemptedProviders: []string{providerName},
				Metadata:           make(map[string]string),
			}, nil
		}
	}

	// Manual provider not available, try fallback if enabled
	if r.config.Fallback.Enabled {
		slog.Warn("manual provider not available, falling back",
			"request_id", req.RequestID,
			"preferred_provider", providerName,
		)
		return r.routeWithStrategy(req, available)
	}

	// No fallback, return error
	r.stats.IncrementErrors()
	return nil, &ProviderNotFoundError{
		ProviderName:       providerName,
		AvailableProviders: getProviderNames(available),
	}
}

// routeWithStrategy uses the configured routing strategy.
func (r *DefaultRouter) routeWithStrategy(req *RoutingRequest, available []providers.Provider) (*RoutingResult, error) {
	provider, err := r.strategy.SelectProvider(req, available)
	if err != nil {
		r.stats.IncrementErrors()
		return nil, fmt.Errorf("strategy selection failed: %w", err)
	}

	r.stats.IncrementProvider(provider.GetName())
	r.stats.IncrementStrategy(r.strategy.GetName())

	slog.Debug("strategy-based routing",
		"request_id", req.RequestID,
		"provider", provider.GetName(),
		"strategy", r.strategy.GetName(),
		"model", req.Model,
	)

	return &RoutingResult{
		Provider:           provider,
		ProviderName:       provider.GetName(),
		Strategy:           r.strategy.GetName(),
		Reason:             fmt.Sprintf("selected by %s strategy", r.strategy.GetName()),
		IsHealthy:          provider.IsHealthy(),
		IsFallback:         false,
		AttemptedProviders: []string{provider.GetName()},
		Metadata:           make(map[string]string),
	}, nil
}

// tryFallbacks attempts to route to fallback providers.
func (r *DefaultRouter) tryFallbacks(req *RoutingRequest, available []providers.Provider, fallbackNames []string) (*RoutingResult, error) {
	attempted := make([]string, 0, len(fallbackNames))

	for _, fallbackName := range fallbackNames {
		attempted = append(attempted, fallbackName)

		// Find fallback provider
		for _, p := range available {
			if p.GetName() == fallbackName {
				r.stats.IncrementProvider(fallbackName)
				r.stats.IncrementStrategy("fallback")

				slog.Info("using fallback provider",
					"request_id", req.RequestID,
					"provider", fallbackName,
					"attempted", attempted,
				)

				return &RoutingResult{
					Provider:           p,
					ProviderName:       fallbackName,
					Strategy:           "fallback",
					Reason:             "fallback provider",
					IsHealthy:          p.IsHealthy(),
					IsFallback:         true,
					AttemptedProviders: attempted,
					Metadata:           make(map[string]string),
				}, nil
			}
		}
	}

	// All fallbacks failed
	r.stats.IncrementErrors()
	return nil, &AllProvidersFailedError{
		AttemptedProviders: attempted,
		Model:              req.Model,
		LastError:          fmt.Errorf("no fallback providers available"),
	}
}

// GetStrategy returns the name of the configured routing strategy.
func (r *DefaultRouter) GetStrategy() string {
	return r.strategy.GetName()
}

// GetStats returns current routing statistics.
func (r *DefaultRouter) GetStats() *RoutingStats {
	return r.stats.Snapshot()
}

// UpdateProviders updates the available provider pool.
func (r *DefaultRouter) UpdateProviders(providerMap map[string]providers.Provider) error {
	r.selector.UpdateProviders(providerMap)
	return nil
}

// Close closes the router and releases resources.
func (r *DefaultRouter) Close() error {
	// Nothing to close currently
	return nil
}


// getProviderNames extracts provider names from a provider list.
func getProviderNames(providerList []providers.Provider) []string {
	names := make([]string, 0, len(providerList))
	for _, p := range providerList {
		names = append(names, p.GetName())
	}
	return names
}
