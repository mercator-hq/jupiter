package handlers

import "mercator-hq/jupiter/pkg/providers"

// ProviderManager is the interface for managing LLM providers.
type ProviderManager interface {
	GetProvider(name string) (providers.Provider, error)
	GetHealthyProviders() map[string]providers.Provider
	Close() error
}
