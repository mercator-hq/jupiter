// Package anthropic implements the Anthropic provider adapter.
//
// This package provides an implementation of the providers.Provider interface
// for Anthropic's Messages API. It supports:
//
//   - Messages API (Claude 3.x models)
//   - Streaming responses (Server-Sent Events)
//   - Tool calling
//   - Token usage tracking
//
// # Basic Usage
//
//	config := providers.ProviderConfig{
//	    Name:    "anthropic",
//	    Type:    "anthropic",
//	    BaseURL: "https://api.anthropic.com",
//	    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
//	}
//
//	provider, err := anthropic.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	req := &providers.CompletionRequest{
//	    Model: "claude-3-opus-20240229",
//	    Messages: []providers.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	    MaxTokens: 1024,  // Required by Anthropic
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
// This adapter is compatible with Anthropic's Messages API version 2023-06-01.
// It supports all Claude 3.x models:
//
//   - claude-3-opus-20240229
//   - claude-3-sonnet-20240229
//   - claude-3-haiku-20240307
//   - And future models that use the Messages API format
//
// # Request Transformation
//
// The adapter transforms provider-agnostic CompletionRequest to Anthropic's format:
//
//   - System messages are extracted and placed in the "system" field
//   - Messages must alternate between user and assistant (enforced by validation)
//   - MaxTokens is required (defaults to 4096 if not provided)
//   - Tools are transformed to Anthropic's tool calling format
//
// # Response Transformation
//
// The adapter normalizes Anthropic responses to provider-agnostic format:
//
//   - Content blocks are concatenated into a single string
//   - Token usage is extracted (input_tokens + output_tokens)
//   - Stop reason is normalized (end_turn -> stop, max_tokens -> length, tool_use -> tool_calls)
//   - Tool use blocks are extracted and converted to tool calls
//
// # Error Handling
//
// The adapter maps Anthropic-specific errors to common error types:
//
//   - 401/403 -> AuthError
//   - 429 -> RateLimitError (includes retry-after)
//   - 400 -> ValidationError
//   - 5xx -> ProviderError (retried automatically)
//
// # Anthropic-Specific Requirements
//
// Important differences from OpenAI:
//
//  1. MaxTokens is required (cannot be 0)
//  2. System messages must be extracted from messages array
//  3. Messages must alternate between user and assistant
//  4. First message must be from user
//  5. Uses x-api-key header instead of Authorization: Bearer
//  6. Requires anthropic-version header
package anthropic
