package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"mercator-hq/jupiter/pkg/providerfactory"
	"mercator-hq/jupiter/pkg/providers"
)

func main() {
	// Example 1: Simple completion request
	simpleCompletion()

	// Example 2: Streaming completion
	streamingCompletion()

	// Example 3: Using provider manager
	providerManager()

	// Example 4: Health checking
	healthChecking()

	// Example 5: Error handling
	errorHandling()
}

// simpleCompletion demonstrates a basic completion request
func simpleCompletion() {
	fmt.Println("=== Simple Completion ===")

	// Create provider configuration
	config := providers.ProviderConfig{
		Name:                "openai",
		Type:                "openai",
		BaseURL:             "https://api.openai.com/v1",
		APIKey:              os.Getenv("OPENAI_API_KEY"),
		Timeout:             60 * time.Second,
		MaxRetries:          3,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	// Create provider
	provider, err := providerfactory.NewProvider(config)
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	// Create completion request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "What is the capital of France?"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Send request
	ctx := context.Background()
	resp, err := provider.SendCompletion(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	// Print response
	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Tokens used: %d (prompt: %d, completion: %d)\n",
		resp.Usage.TotalTokens,
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
	)
	fmt.Println()
}

// streamingCompletion demonstrates streaming responses
func streamingCompletion() {
	fmt.Println("=== Streaming Completion ===")

	// Create provider
	config := providers.ProviderConfig{
		Name:    "openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Timeout: 60 * time.Second,
	}

	provider, err := providerfactory.NewProvider(config)
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	// Create streaming request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Write a haiku about coding"},
		},
		Stream: true,
	}

	// Send request
	ctx := context.Background()
	chunks, err := provider.StreamCompletion(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	// Read stream
	fmt.Print("Streamed response: ")
	for chunk := range chunks {
		if chunk.Error != nil {
			log.Fatal(chunk.Error)
		}
		fmt.Print(chunk.Delta)

		// Check for final chunk with usage info
		if chunk.FinishReason != "" {
			fmt.Printf("\n\nFinish reason: %s\n", chunk.FinishReason)
			if chunk.Usage != nil {
				fmt.Printf("Tokens used: %d\n", chunk.Usage.TotalTokens)
			}
		}
	}
	fmt.Println()
}

// providerManager demonstrates using the provider manager
func providerManager() {
	fmt.Println("=== Provider Manager ===")

	// Create manager
	manager := providerfactory.NewManager()
	defer manager.Close()

	// Add multiple providers
	configs := []providers.ProviderConfig{
		{
			Name:    "openai",
			Type:    "openai",
			BaseURL: "https://api.openai.com/v1",
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Timeout: 60 * time.Second,
		},
		{
			Name:    "anthropic",
			Type:    "anthropic",
			BaseURL: "https://api.anthropic.com",
			APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
			Timeout: 60 * time.Second,
		},
		{
			Name:    "ollama",
			Type:    "generic",
			BaseURL: "http://localhost:11434/v1",
			APIKey:  "",
			Timeout: 120 * time.Second,
		},
	}

	if err := manager.LoadFromConfig(configs); err != nil {
		log.Fatal(err)
	}

	// Get health summary
	summary := manager.GetHealthSummary()
	fmt.Printf("Total providers: %d\n", summary.Total)
	fmt.Printf("Healthy: %d\n", summary.Healthy)
	fmt.Printf("Unhealthy: %d\n", summary.Unhealthy)

	// Use a specific provider
	provider, err := manager.GetProvider("openai")
	if err != nil {
		log.Fatal(err)
	}

	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello!"},
		},
	}

	ctx := context.Background()
	resp, err := provider.SendCompletion(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response from %s: %s\n", provider.GetName(), resp.Content)
	fmt.Println()
}

// healthChecking demonstrates provider health monitoring
func healthChecking() {
	fmt.Println("=== Health Checking ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := providers.ProviderConfig{
		Name:                "openai",
		Type:                "openai",
		BaseURL:             "https://api.openai.com/v1",
		APIKey:              os.Getenv("OPENAI_API_KEY"),
		HealthCheckInterval: 10 * time.Second,
	}

	// Create provider with health checking
	provider, err := providerfactory.NewProviderWithHealthCheck(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	// Check initial health
	fmt.Printf("Provider healthy: %v\n", provider.IsHealthy())

	// Get detailed health info
	health := provider.GetHealth()
	fmt.Printf("Last check: %s\n", health.LastCheck.Format(time.RFC3339))
	fmt.Printf("Consecutive failures: %d\n", health.ConsecutiveFailures)
	fmt.Printf("Total requests: %d\n", health.TotalRequests)
	fmt.Printf("Failed requests: %d\n", health.FailedRequests)

	// Perform on-demand health check
	if err := provider.HealthCheck(ctx); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
	} else {
		fmt.Println("Health check passed")
	}
	fmt.Println()
}

// errorHandling demonstrates handling different error types
func errorHandling() {
	fmt.Println("=== Error Handling ===")

	config := providers.ProviderConfig{
		Name:    "openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "invalid-key", // Intentionally invalid
		Timeout: 5 * time.Second,
	}

	provider, err := providerfactory.NewProvider(config)
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello!"},
		},
	}

	ctx := context.Background()
	_, err = provider.SendCompletion(ctx, req)

	// Handle specific error types
	if err != nil {
		switch e := err.(type) {
		case *providers.AuthError:
			fmt.Printf("Authentication error: %s\n", e.Message)
			fmt.Println("Solution: Check your API key")

		case *providers.RateLimitError:
			fmt.Printf("Rate limited: %s\n", e.Message)
			if e.RetryAfter > 0 {
				fmt.Printf("Retry after: %s\n", e.RetryAfter)
			}

		case *providers.TimeoutError:
			fmt.Printf("Request timeout after %s\n", e.Timeout)
			fmt.Println("Solution: Increase timeout or check network")

		case *providers.ProviderError:
			fmt.Printf("Provider error (status %d): %s\n", e.StatusCode, e.Message)

		case *providers.ValidationError:
			fmt.Printf("Validation error in field %q: %s\n", e.Field, e.Message)

		case *providers.ModelNotFoundError:
			fmt.Printf("Model %q not found in provider %s\n", e.Model, e.Provider)

		default:
			fmt.Printf("Unknown error: %v\n", err)
		}
	}

	fmt.Println()
}
