package routing

import (
	"log/slog"

	"mercator-hq/jupiter/pkg/providers"
)

// ProviderSelector handles filtering and selection of providers based on
// health status, model capability, and other criteria.
type ProviderSelector struct {
	// providers is the map of all configured providers.
	// Key: provider name, Value: Provider instance
	providers map[string]providers.Provider

	// modelMapping maps model names to provider names.
	// Key: model name, Value: list of provider names that support the model
	modelMapping map[string][]string
}

// NewProviderSelector creates a new provider selector.
func NewProviderSelector(providerMap map[string]providers.Provider, modelMapping map[string][]string) *ProviderSelector {
	if providerMap == nil {
		providerMap = make(map[string]providers.Provider)
	}
	if modelMapping == nil {
		modelMapping = make(map[string][]string)
	}

	return &ProviderSelector{
		providers:    providerMap,
		modelMapping: modelMapping,
	}
}

// FilterByHealth filters providers to only include healthy ones.
// It checks each provider's IsHealthy() status and excludes unhealthy providers.
//
// Returns a new slice containing only healthy providers.
// If all providers are unhealthy, returns an empty slice.
func (s *ProviderSelector) FilterByHealth(providerList []providers.Provider) []providers.Provider {
	if len(providerList) == 0 {
		return providerList
	}

	healthy := make([]providers.Provider, 0, len(providerList))
	for _, p := range providerList {
		if p.IsHealthy() {
			healthy = append(healthy, p)
		} else {
			slog.Debug("provider excluded due to health",
				"provider", p.GetName(),
				"healthy", false,
			)
		}
	}

	slog.Debug("filtered providers by health",
		"total", len(providerList),
		"healthy", len(healthy),
		"filtered", len(providerList)-len(healthy),
	)

	return healthy
}

// FilterByModel filters providers to only include those that support the requested model.
// It uses the model mapping configuration to determine which providers support each model.
//
// If the model is not in the mapping, all providers are considered capable (no filtering).
// If the model is mapped to specific providers, only those providers are returned.
//
// Returns a new slice containing only providers that support the model.
func (s *ProviderSelector) FilterByModel(providerList []providers.Provider, model string) []providers.Provider {
	if len(providerList) == 0 {
		return providerList
	}

	// If no model specified, return all providers
	if model == "" {
		return providerList
	}

	// Check if model has explicit mapping
	providerNames, hasMapping := s.modelMapping[model]
	if !hasMapping {
		// No explicit mapping - all providers are considered capable
		slog.Debug("no model mapping found, all providers considered capable",
			"model", model,
		)
		return providerList
	}

	// Create a set of provider names for fast lookup
	providerSet := make(map[string]bool)
	for _, name := range providerNames {
		providerSet[name] = true
	}

	// Filter providers by model capability
	capable := make([]providers.Provider, 0, len(providerList))
	for _, p := range providerList {
		if providerSet[p.GetName()] {
			capable = append(capable, p)
		} else {
			slog.Debug("provider excluded due to model capability",
				"provider", p.GetName(),
				"model", model,
			)
		}
	}

	slog.Debug("filtered providers by model",
		"model", model,
		"total", len(providerList),
		"capable", len(capable),
		"filtered", len(providerList)-len(capable),
	)

	return capable
}

// GetAvailableProviders returns all configured providers as a slice.
// The order is not guaranteed.
func (s *ProviderSelector) GetAvailableProviders() []providers.Provider {
	if len(s.providers) == 0 {
		return nil
	}

	result := make([]providers.Provider, 0, len(s.providers))
	for _, p := range s.providers {
		result = append(result, p)
	}
	return result
}

// GetProvider returns a specific provider by name.
// Returns nil if the provider does not exist.
func (s *ProviderSelector) GetProvider(name string) providers.Provider {
	return s.providers[name]
}

// GetProviderNames returns the names of all configured providers.
func (s *ProviderSelector) GetProviderNames() []string {
	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	return names
}

// GetSupportedModels returns all models that are explicitly mapped.
// This does not include models that might work with unmapped providers.
func (s *ProviderSelector) GetSupportedModels() []string {
	models := make([]string, 0, len(s.modelMapping))
	for model := range s.modelMapping {
		models = append(models, model)
	}
	return models
}

// UpdateProviders updates the provider pool.
// This is called when providers are added/removed via configuration reload.
func (s *ProviderSelector) UpdateProviders(providerMap map[string]providers.Provider) {
	if providerMap == nil {
		providerMap = make(map[string]providers.Provider)
	}
	s.providers = providerMap
}

// UpdateModelMapping updates the model mapping configuration.
func (s *ProviderSelector) UpdateModelMapping(modelMapping map[string][]string) {
	if modelMapping == nil {
		modelMapping = make(map[string][]string)
	}
	s.modelMapping = modelMapping
}
