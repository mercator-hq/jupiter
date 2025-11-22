package providerfactory

import (
	"context"
	"fmt"
	"log/slog"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/providers/anthropic"
	"mercator-hq/jupiter/pkg/providers/generic"
	"mercator-hq/jupiter/pkg/providers/openai"
)

// NewProvider creates a new provider instance based on the configuration.
// It automatically detects the provider type and creates the appropriate adapter.
//
// Supported provider types:
//   - "openai": OpenAI API
//   - "anthropic": Anthropic Messages API
//   - "generic": OpenAI-compatible APIs (Ollama, LM Studio, vLLM, etc.)
//
// The provider type is determined from the config.Type field. If not specified,
// it is inferred from the provider name:
//   - "openai" -> OpenAI
//   - "anthropic" -> Anthropic
//   - Everything else -> Generic
//
// Example:
//
//	config := ProviderConfig{
//	    Name: "openai",
//	    Type: "openai",
//	    BaseURL: "https://api.openai.com/v1",
//	    APIKey: "sk-...",
//	}
//	provider, err := NewProvider(config)
//	if err != nil {
//	    return err
//	}
//	defer provider.Close()
func NewProvider(config providers.ProviderConfig) (providers.Provider, error) {
	// Determine provider type
	providerType := config.Type
	if providerType == "" {
		// Infer from name
		providerType = inferProviderType(config.Name)
		config.Type = providerType
	}

	slog.Debug("creating provider",
		"name", config.Name,
		"type", providerType,
		"base_url", config.BaseURL,
	)

	// Create provider based on type
	var provider providers.Provider
	var err error

	switch providerType {
	case "openai":
		provider, err = openai.NewProvider(config)

	case "anthropic":
		provider, err = anthropic.NewProvider(config)

	case "generic":
		provider, err = generic.NewProvider(config)

	default:
		return nil, &providers.ConfigError{
			Provider: config.Name,
			Field:    "type",
			Message:  fmt.Sprintf("unsupported provider type: %q (supported: openai, anthropic, generic)", providerType),
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create provider %q: %w", config.Name, err)
	}

	slog.Info("provider created successfully",
		"name", config.Name,
		"type", providerType,
	)

	return provider, nil
}

// NewProviderWithHealthCheck creates a provider and starts the health checker.
// This is a convenience function that combines provider creation and health monitoring.
//
// The health checker runs in a background goroutine and updates the provider's
// health status periodically. The context is used to stop the health checker.
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	provider, err := NewProviderWithHealthCheck(ctx, config)
//	if err != nil {
//	    return err
//	}
//	defer provider.Close()
func NewProviderWithHealthCheck(ctx context.Context, config providers.ProviderConfig) (providers.Provider, error) {
	provider, err := NewProvider(config)
	if err != nil {
		return nil, err
	}

	// Start health checker if provider supports it
	// Check if provider has a StartHealthChecker method
	type healthCheckStarter interface {
		StartHealthChecker(context.Context)
	}

	if hcs, ok := provider.(healthCheckStarter); ok {
		hcs.StartHealthChecker(ctx)
		slog.Debug("health checker started", "provider", config.Name)
	} else {
		slog.Debug("provider does not support health checking", "name", config.Name)
	}

	return provider, nil
}

// inferProviderType infers the provider type from the provider name.
func inferProviderType(name string) string {
	switch name {
	case "openai":
		return "openai"
	case "anthropic":
		return "anthropic"
	case "ollama", "lmstudio", "vllm", "localai":
		return "generic"
	default:
		// Default to generic for unknown providers
		return "generic"
	}
}
