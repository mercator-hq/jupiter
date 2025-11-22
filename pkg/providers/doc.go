// Package providers implements a unified abstraction layer for LLM providers.
//
// # Overview
//
// The providers package provides a consistent interface for interacting with
// different LLM providers (OpenAI, Anthropic, local models, etc.). It normalizes
// requests and responses, manages connections, performs health checks, and enables
// provider-agnostic routing.
//
// # Architecture
//
// The package is organized into several layers:
//
//  1. Provider Interface - Defines the contract all providers must implement
//  2. Base HTTP Provider - Implements common HTTP client logic (connection pooling, retries, timeouts)
//  3. Provider Adapters - Provider-specific implementations (OpenAI, Anthropic, Generic)
//  4. Provider Factory - Creates providers from configuration
//  5. Provider Manager - Manages a collection of providers with health monitoring
//
// # Basic Usage
//
// Create a single provider:
//
//	config := providers.ProviderConfig{
//	    Name:     "openai",
//	    Type:     "openai",
//	    BaseURL:  "https://api.openai.com/v1",
//	    APIKey:   os.Getenv("OPENAI_API_KEY"),
//	    Timeout:  60 * time.Second,
//	}
//
//	provider, err := providers.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	req := &providers.CompletionRequest{
//	    Model: "gpt-4",
//	    Messages: []providers.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	}
//
//	resp, err := provider.SendCompletion(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Provider Manager
//
// For managing multiple providers:
//
//	manager := providers.NewManager()
//	defer manager.Close()
//
//	configs := []providers.ProviderConfig{
//	    {Name: "openai", Type: "openai", BaseURL: "...", APIKey: "..."},
//	    {Name: "anthropic", Type: "anthropic", BaseURL: "...", APIKey: "..."},
//	    {Name: "ollama", Type: "generic", BaseURL: "http://localhost:11434/v1"},
//	}
//
//	if err := manager.LoadFromConfig(configs); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get a specific provider
//	provider, err := manager.GetProvider("openai")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get all healthy providers
//	healthy := manager.GetHealthyProviders()
//	fmt.Printf("Healthy providers: %d\n", len(healthy))
//
// # Streaming
//
// Stream responses from providers:
//
//	req := &providers.CompletionRequest{
//	    Model: "gpt-4",
//	    Messages: []providers.Message{
//	        {Role: "user", Content: "Write a poem"},
//	    },
//	    Stream: true,
//	}
//
//	chunks, err := provider.StreamCompletion(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for chunk := range chunks {
//	    if chunk.Error != nil {
//	        log.Fatal(chunk.Error)
//	    }
//	    fmt.Print(chunk.Delta)
//	}
//
// # Health Checking
//
// Providers support automatic health monitoring:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	provider, err := providers.NewProviderWithHealthCheck(ctx, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check health status
//	if !provider.IsHealthy() {
//	    health := provider.GetHealth()
//	    fmt.Printf("Provider unhealthy: %v\n", health.LastError)
//	}
//
// # Error Handling
//
// The package defines specific error types for common failure scenarios:
//
//   - ProviderError: General provider errors
//   - AuthError: Authentication failures (HTTP 401/403)
//   - RateLimitError: Rate limit exceeded (HTTP 429)
//   - TimeoutError: Request timeout
//   - ParseError: Response parsing failure
//   - ModelNotFoundError: Unknown model
//   - ValidationError: Invalid request
//
// Example error handling:
//
//	resp, err := provider.SendCompletion(ctx, req)
//	if err != nil {
//	    switch e := err.(type) {
//	    case *providers.AuthError:
//	        fmt.Printf("Authentication failed: %v\n", e)
//	    case *providers.RateLimitError:
//	        fmt.Printf("Rate limited, retry after: %v\n", e.RetryAfter)
//	    case *providers.TimeoutError:
//	        fmt.Printf("Request timeout: %v\n", e)
//	    default:
//	        fmt.Printf("Error: %v\n", e)
//	    }
//	}
//
// # Supported Providers
//
// The package supports three provider types:
//
//  1. OpenAI - OpenAI's chat completions API
//  2. Anthropic - Anthropic's messages API
//  3. Generic - Any OpenAI-compatible API (Ollama, LM Studio, vLLM, etc.)
//
// # Connection Pooling
//
// All providers use HTTP connection pooling to reduce latency:
//
//	config := providers.ProviderConfig{
//	    Name:                "openai",
//	    MaxIdleConns:        100,
//	    MaxIdleConnsPerHost: 10,
//	    IdleConnTimeout:     90 * time.Second,
//	}
//
// # Retry Logic
//
// Providers automatically retry transient errors with exponential backoff:
//
//	config := providers.ProviderConfig{
//	    Name:       "openai",
//	    MaxRetries: 3,  // Retry up to 3 times
//	}
//
// # Thread Safety
//
// All provider implementations and the Manager are thread-safe and can be
// used concurrently from multiple goroutines.
//
// # Performance
//
// The package is designed for high performance:
//
//   - Connection pooling reduces connection establishment overhead
//   - Retry logic uses exponential backoff to avoid overwhelming providers
//   - Health checking uses circuit breaker pattern to fail fast
//   - Streaming uses buffered channels to handle backpressure
//
// Expected performance:
//   - Adapter overhead: <1ms per request
//   - Connection reuse: >90% under sustained load
//   - Health check latency: <100ms
//   - Memory per provider: <10MB
package providers
