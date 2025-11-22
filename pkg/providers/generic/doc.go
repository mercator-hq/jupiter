// Package generic implements a generic OpenAI-compatible provider adapter.
//
// This package provides an implementation of the providers.Provider interface
// for any provider that implements the OpenAI API format. It supports:
//
//   - Local LLM servers (Ollama, LM Studio, vLLM, FastChat)
//   - Custom OpenAI-compatible endpoints
//   - Self-hosted LLM APIs
//
// # Supported Platforms
//
// The generic adapter works with any OpenAI-compatible API, including:
//
//   - Ollama (http://localhost:11434/v1)
//   - LM Studio (http://localhost:1234/v1)
//   - vLLM (http://localhost:8000/v1)
//   - FastChat (http://localhost:8000/v1)
//   - Text Generation Inference (http://localhost:8080/v1)
//   - LocalAI (http://localhost:8080/v1)
//   - Custom OpenAI-compatible endpoints
//
// # Basic Usage
//
//	config := providers.ProviderConfig{
//	    Name:    "ollama",
//	    Type:    "generic",
//	    BaseURL: "http://localhost:11434/v1",
//	    // API key is optional for local providers
//	}
//
//	provider, err := generic.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	req := &providers.CompletionRequest{
//	    Model: "llama2",
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
// # Ollama Example
//
//	config := providers.ProviderConfig{
//	    Name:    "ollama",
//	    BaseURL: "http://localhost:11434/v1",
//	    Timeout: 120 * time.Second,  // Local inference can be slow
//	}
//
//	provider, err := generic.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	req := &providers.CompletionRequest{
//	    Model: "llama2:13b",
//	    Messages: []providers.Message{
//	        {Role: "user", Content: "Tell me about Go"},
//	    },
//	}
//
//	resp, err := provider.SendCompletion(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # LM Studio Example
//
//	config := providers.ProviderConfig{
//	    Name:    "lmstudio",
//	    BaseURL: "http://localhost:1234/v1",
//	    Timeout: 60 * time.Second,
//	}
//
//	provider, err := generic.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # vLLM Example
//
//	config := providers.ProviderConfig{
//	    Name:    "vllm",
//	    BaseURL: "http://localhost:8000/v1",
//	    APIKey:  "your-api-key",  // If authentication is enabled
//	}
//
//	provider, err := generic.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Implementation Details
//
// The generic adapter reuses the OpenAI adapter implementation since most
// local LLM servers implement the OpenAI API format. The only difference
// is the base URL and optional API key.
//
// Request/response format:
//   - Uses OpenAI's chat completions format
//   - Supports streaming via Server-Sent Events (SSE)
//   - Tool calling support depends on the backend
//
// # Configuration Differences
//
// Compared to cloud providers, local models typically:
//
//   - Don't require API keys (set to "not-required" by default)
//   - Need longer timeouts (inference can be slow)
//   - Have fewer retry attempts (no point retrying local failures)
//   - Use smaller connection pools (single instance)
//
// # Compatibility Notes
//
// Not all OpenAI-compatible servers implement the full API:
//
//   - Tool/function calling may not be supported
//   - Streaming may not be supported
//   - Token usage may not be reported
//   - Some parameters may be ignored
//
// The adapter will work as long as the server implements the basic
// chat completions endpoint with the OpenAI request/response format.
package generic
