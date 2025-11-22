package anthropic

import (
	"context"
	"fmt"
	"log/slog"

	"mercator-hq/jupiter/pkg/providers"
)

// Provider is the Anthropic provider adapter.
// It implements the providers.Provider interface for Anthropic's Messages API.
type Provider struct {
	*providers.HTTPProvider
}

const (
	// DefaultAnthropicVersion is the API version to use
	DefaultAnthropicVersion = "2023-06-01"
)

// NewProvider creates a new Anthropic provider instance.
func NewProvider(config providers.ProviderConfig) (*Provider, error) {
	// Validate configuration
	if config.Name == "" {
		return nil, &providers.ConfigError{
			Provider: "anthropic",
			Field:    "name",
			Message:  "provider name is required",
		}
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}

	if config.APIKey == "" {
		return nil, &providers.ConfigError{
			Provider: config.Name,
			Field:    "api_key",
			Message:  "API key is required for Anthropic",
		}
	}

	// Set defaults if not provided
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.MaxIdleConnsPerHost == 0 {
		config.MaxIdleConnsPerHost = 10
	}

	// Create base HTTP provider
	httpProvider := providers.NewHTTPProvider(config)

	p := &Provider{
		HTTPProvider: httpProvider,
	}

	slog.Info("Anthropic provider initialized",
		"provider", config.Name,
		"base_url", config.BaseURL,
	)

	return p, nil
}

// SendCompletion sends a completion request to Anthropic.
func (p *Provider) SendCompletion(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	// Validate request
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// Transform to Anthropic format
	anthropicReq, err := transformRequest(req)
	if err != nil {
		return nil, err
	}

	// Prepare request
	url := fmt.Sprintf("%s/v1/messages", p.GetConfig().BaseURL)
	headers := map[string]string{
		"x-api-key":         p.GetConfig().APIKey,
		"anthropic-version": DefaultAnthropicVersion,
		"Content-Type":      "application/json",
	}

	// Send request
	var anthropicResp AnthropicResponse
	if err := p.DoJSONRequest(ctx, "POST", url, anthropicReq, &anthropicResp, headers); err != nil {
		return nil, err
	}

	// Transform response to provider-agnostic format
	resp, err := transformResponse(&anthropicResp)
	if err != nil {
		return nil, &providers.ParseError{
			Provider: p.GetName(),
			Cause:    err,
		}
	}

	slog.Debug("completion request succeeded",
		"provider", p.GetName(),
		"model", resp.Model,
		"tokens", resp.Usage.TotalTokens,
	)

	return resp, nil
}

// StreamCompletion sends a streaming completion request to Anthropic.
func (p *Provider) StreamCompletion(ctx context.Context, req *providers.CompletionRequest) (<-chan *providers.StreamChunk, error) {
	// Validate request
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// Transform to Anthropic format
	anthropicReq, err := transformRequest(req)
	if err != nil {
		return nil, err
	}
	anthropicReq.Stream = true

	// Prepare request
	url := fmt.Sprintf("%s/v1/messages", p.GetConfig().BaseURL)
	headers := map[string]string{
		"x-api-key":         p.GetConfig().APIKey,
		"anthropic-version": DefaultAnthropicVersion,
		"Content-Type":      "application/json",
		"Accept":            "text/event-stream",
	}

	// Create stream reader
	stream, err := newStreamReader(ctx, p.HTTPProvider, url, anthropicReq, headers)
	if err != nil {
		return nil, err
	}

	// Create output channel
	chunks := make(chan *providers.StreamChunk, 100) // Buffered channel

	// Start goroutine to read stream and send chunks
	go func() {
		defer close(chunks)
		defer stream.Close()

		for {
			chunk, err := stream.Read(ctx)
			if err != nil {
				// Send error chunk and exit
				chunks <- &providers.StreamChunk{
					Error: err,
				}
				return
			}

			if chunk == nil {
				// Stream ended normally
				return
			}

			// Send chunk
			select {
			case chunks <- chunk:
			case <-ctx.Done():
				return
			}

			// Check if this is the final chunk
			if chunk.FinishReason != "" {
				return
			}
		}
	}()

	return chunks, nil
}

// validateRequest validates the completion request.
func validateRequest(req *providers.CompletionRequest) error {
	if req == nil {
		return &providers.ValidationError{
			Field:   "request",
			Message: "request cannot be nil",
		}
	}

	if req.Model == "" {
		return &providers.ValidationError{
			Field:   "model",
			Message: "model is required",
		}
	}

	if len(req.Messages) == 0 {
		return &providers.ValidationError{
			Field:   "messages",
			Message: "at least one message is required",
		}
	}

	return nil
}
