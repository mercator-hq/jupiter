package providers

import "context"

// Provider is the core interface that all LLM provider adapters must implement.
// It provides a unified abstraction for interacting with different LLM providers
// (OpenAI, Anthropic, local models, etc.).
//
// All methods accept a context.Context for cancellation and timeout control.
// Implementations must respect context cancellation and return immediately when
// the context is cancelled.
//
// Example usage:
//
//	provider, err := NewProvider("openai", config)
//	if err != nil {
//	    return err
//	}
//
//	req := &CompletionRequest{
//	    Model: "gpt-4",
//	    Messages: []Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	}
//
//	resp, err := provider.SendCompletion(ctx, req)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(resp.Content)
type Provider interface {
	// SendCompletion sends a completion request to the provider and returns the response.
	// The request is transformed to the provider-specific format, sent to the provider,
	// and the response is normalized to the provider-agnostic format.
	//
	// Returns an error if the request fails, times out, or the provider returns an error.
	// Implements automatic retry with exponential backoff for transient errors.
	SendCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// StreamCompletion sends a streaming completion request to the provider.
	// It returns a channel that yields incremental response chunks as they arrive.
	//
	// The caller must read from the channel until it closes. If an error occurs during
	// streaming, it will be set in the Error field of the final StreamChunk.
	//
	// The context is used for cancellation. If the context is cancelled, the stream
	// will be closed and no more chunks will be sent.
	//
	// Example:
	//
	//  chunks, err := provider.StreamCompletion(ctx, req)
	//  if err != nil {
	//      return err
	//  }
	//  for chunk := range chunks {
	//      if chunk.Error != nil {
	//          return chunk.Error
	//      }
	//      fmt.Print(chunk.Delta)
	//  }
	StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan *StreamChunk, error)

	// HealthCheck performs a health check against the provider.
	// It sends a lightweight request to verify the provider is reachable and responding.
	//
	// Returns nil if the provider is healthy, or an error describing the health issue.
	// This is called periodically by the health checker to monitor provider availability.
	HealthCheck(ctx context.Context) error

	// GetName returns the provider's configured name (e.g., "openai", "anthropic").
	GetName() string

	// GetType returns the provider's type (e.g., "openai", "anthropic", "generic").
	GetType() string

	// GetConfig returns the provider's configuration.
	GetConfig() ProviderConfig

	// IsHealthy returns the current health status of the provider.
	// This is updated by the health checker and can be used for routing decisions.
	IsHealthy() bool

	// GetHealth returns detailed health information including last check time,
	// consecutive failures, and error details.
	GetHealth() ProviderHealth

	// Close closes the provider and releases any resources (HTTP connections, etc.).
	// After calling Close, the provider should not be used.
	Close() error
}

// StreamReader is a helper interface for providers that support streaming.
// It abstracts the underlying SSE or streaming protocol used by the provider.
type StreamReader interface {
	// Read reads the next chunk from the stream.
	// Returns the chunk and nil on success.
	// Returns nil and io.EOF when the stream ends normally.
	// Returns nil and an error if an error occurs.
	Read(ctx context.Context) (*StreamChunk, error)

	// Close closes the stream and releases resources.
	Close() error
}
