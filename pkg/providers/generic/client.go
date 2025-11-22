package generic

import (
	"context"
	"log/slog"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/providers/openai"
)

// Provider is a generic OpenAI-compatible provider adapter.
// It supports any provider that implements the OpenAI API format,
// such as Ollama, LM Studio, vLLM, FastChat, etc.
//
// This adapter reuses the OpenAI request/response format but allows
// for custom base URLs and optional API keys.
type Provider struct {
	*openai.Provider
}

// NewProvider creates a new generic OpenAI-compatible provider instance.
func NewProvider(config providers.ProviderConfig) (*Provider, error) {
	// Validate configuration
	if config.Name == "" {
		return nil, &providers.ConfigError{
			Provider: "generic",
			Field:    "name",
			Message:  "provider name is required",
		}
	}

	if config.BaseURL == "" {
		return nil, &providers.ConfigError{
			Provider: config.Name,
			Field:    "base_url",
			Message:  "base URL is required for generic provider",
		}
	}

	// API key is optional for generic providers (local models don't need it)
	// Set a dummy key if not provided to avoid validation errors in OpenAI adapter
	if config.APIKey == "" {
		config.APIKey = "not-required"
	}

	// Set defaults if not provided
	if config.MaxRetries == 0 {
		config.MaxRetries = 1 // Local providers typically don't need retries
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}
	if config.MaxIdleConnsPerHost == 0 {
		config.MaxIdleConnsPerHost = 5
	}

	// Create OpenAI provider with custom config
	openaiProvider, err := openai.NewProvider(config)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		Provider: openaiProvider,
	}

	slog.Info("Generic OpenAI-compatible provider initialized",
		"provider", config.Name,
		"base_url", config.BaseURL,
		"type", "generic",
	)

	return p, nil
}

// SendCompletion sends a completion request to the generic provider.
// This delegates to the OpenAI adapter since the request/response format is the same.
func (p *Provider) SendCompletion(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	return p.Provider.SendCompletion(ctx, req)
}

// StreamCompletion sends a streaming completion request to the generic provider.
// This delegates to the OpenAI adapter since the streaming format is the same.
func (p *Provider) StreamCompletion(ctx context.Context, req *providers.CompletionRequest) (<-chan *providers.StreamChunk, error) {
	return p.Provider.StreamCompletion(ctx, req)
}

// GetType returns "generic" as the provider type.
func (p *Provider) GetType() string {
	return "generic"
}
