// Package openai implements the OpenAI provider adapter.
//
// This package provides an implementation of the providers.Provider interface
// for OpenAI's chat completions API. It supports:
//
//   - Chat completions
//   - Streaming responses (Server-Sent Events)
//   - Function/tool calling
//   - Token usage tracking
//
// # Basic Usage
//
//	config := providers.ProviderConfig{
//	    Name:    "openai",
//	    Type:    "openai",
//	    BaseURL: "https://api.openai.com/v1",
//	    APIKey:  os.Getenv("OPENAI_API_KEY"),
//	}
//
//	provider, err := openai.NewProvider(config)
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
// # Streaming
//
//	req.Stream = true
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
// # API Compatibility
//
// This adapter is compatible with OpenAI's v1 API as of November 2023.
// It supports all models that implement the chat completions endpoint:
//
//   - gpt-4, gpt-4-turbo
//   - gpt-3.5-turbo
//   - And future models that use the same API format
//
// # Request Transformation
//
// The adapter transforms provider-agnostic CompletionRequest to OpenAI's format:
//
//   - Messages are passed through as-is (OpenAI format is the baseline)
//   - Tools are transformed to OpenAI's function calling format
//   - System messages are kept in the messages array
//
// # Response Transformation
//
// The adapter normalizes OpenAI responses to provider-agnostic format:
//
//   - Token usage is extracted from the usage field
//   - Finish reason is normalized (stop, length, tool_calls, content_filter)
//   - Tool calls are extracted and normalized
//
// # Error Handling
//
// The adapter maps OpenAI-specific errors to common error types:
//
//   - 401/403 -> AuthError
//   - 429 -> RateLimitError (includes retry-after)
//   - 400 -> ValidationError
//   - 5xx -> ProviderError (retried automatically)
package openai
