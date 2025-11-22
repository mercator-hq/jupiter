package providerfactory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"mercator-hq/jupiter/pkg/providers"
)

// Manager manages a collection of provider instances.
// It handles provider lifecycle (creation, health monitoring, shutdown).
//
// Manager is thread-safe and can be used concurrently.
type Manager struct {
	providers map[string]providers.Provider
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a new provider manager.
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		providers: make(map[string]providers.Provider),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddProvider adds a provider to the manager.
// If a provider with the same name already exists, it is replaced and the old one is closed.
func (m *Manager) AddProvider(config providers.ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if provider already exists
	if existing, ok := m.providers[config.Name]; ok {
		slog.Warn("replacing existing provider", "name", config.Name)
		existing.Close()
		delete(m.providers, config.Name)
	}

	// Create provider with health checking
	provider, err := NewProviderWithHealthCheck(m.ctx, config)
	if err != nil {
		return fmt.Errorf("failed to add provider %q: %w", config.Name, err)
	}

	m.providers[config.Name] = provider

	slog.Info("provider added to manager",
		"name", config.Name,
		"type", provider.GetType(),
		"total_providers", len(m.providers),
	)

	return nil
}

// RemoveProvider removes a provider from the manager and closes it.
func (m *Manager) RemoveProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, ok := m.providers[name]
	if !ok {
		return fmt.Errorf("provider %q not found", name)
	}

	if err := provider.Close(); err != nil {
		slog.Error("error closing provider", "name", name, "error", err)
	}

	delete(m.providers, name)

	slog.Info("provider removed from manager",
		"name", name,
		"remaining_providers", len(m.providers),
	)

	return nil
}

// GetProvider returns a provider by name.
func (m *Manager) GetProvider(name string) (providers.Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}

	return provider, nil
}

// GetProviders returns a map of all providers.
// The returned map is a copy and safe to modify.
func (m *Manager) GetProviders() map[string]providers.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid concurrent modification issues
	providers := make(map[string]providers.Provider, len(m.providers))
	for name, provider := range m.providers {
		providers[name] = provider
	}

	return providers
}

// GetProviderNames returns a list of all provider names.
func (m *Manager) GetProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}

	return names
}

// GetHealthyProviders returns a map of providers that are currently healthy.
func (m *Manager) GetHealthyProviders() map[string]providers.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthy := make(map[string]providers.Provider)
	for name, provider := range m.providers {
		if provider.IsHealthy() {
			healthy[name] = provider
		}
	}

	return healthy
}

// GetUnhealthyProviders returns a map of providers that are currently unhealthy.
func (m *Manager) GetUnhealthyProviders() map[string]providers.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	unhealthy := make(map[string]providers.Provider)
	for name, provider := range m.providers {
		if !provider.IsHealthy() {
			unhealthy[name] = provider
		}
	}

	return unhealthy
}

// ProviderCount returns the total number of providers.
func (m *Manager) ProviderCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.providers)
}

// HealthyProviderCount returns the number of healthy providers.
func (m *Manager) HealthyProviderCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, provider := range m.providers {
		if provider.IsHealthy() {
			count++
		}
	}

	return count
}

// LoadFromConfig loads providers from a list of configurations.
// Any errors are collected and returned as a single error.
func (m *Manager) LoadFromConfig(configs []providers.ProviderConfig) error {
	var errors []error

	for _, config := range configs {
		if err := m.AddProvider(config); err != nil {
			errors = append(errors, err)
			slog.Error("failed to load provider",
				"name", config.Name,
				"error", err,
			)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load %d provider(s)", len(errors))
	}

	slog.Info("all providers loaded successfully", "count", len(configs))
	return nil
}

// Close closes all providers and stops the manager.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel context to stop all health checkers
	m.cancel()

	// Close all providers
	var errors []error
	for name, provider := range m.providers {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close provider %q: %w", name, err))
		}
	}

	// Clear providers map
	m.providers = make(map[string]providers.Provider)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing providers: %v", errors)
	}

	slog.Info("provider manager closed")
	return nil
}

// GetHealthSummary returns a summary of provider health status.
func (m *Manager) GetHealthSummary() HealthSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := HealthSummary{
		Total:   len(m.providers),
		Healthy: 0,
		Details: make(map[string]providers.ProviderHealth),
	}

	for name, provider := range m.providers {
		health := provider.GetHealth()
		summary.Details[name] = health

		if health.IsHealthy {
			summary.Healthy++
		}
	}

	summary.Unhealthy = summary.Total - summary.Healthy

	return summary
}

// HealthSummary provides an overview of provider health across the manager.
type HealthSummary struct {
	// Total is the total number of providers
	Total int

	// Healthy is the number of healthy providers
	Healthy int

	// Unhealthy is the number of unhealthy providers
	Unhealthy int

	// Details contains per-provider health information
	Details map[string]providers.ProviderHealth
}
