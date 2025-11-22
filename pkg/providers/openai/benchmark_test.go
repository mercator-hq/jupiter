package openai

import (
	"context"
	"testing"

	testhelpers "mercator-hq/jupiter/internal/providers"
	"mercator-hq/jupiter/pkg/providers"
)

func BenchmarkOpenAIProvider_SendCompletion(b *testing.B) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure mock response
	mock.SetResponse("/v1/chat/completions", testhelpers.MockResponse{
		StatusCode: 200,
		Body:       testhelpers.MockOpenAIResponse("Hello, world!", "gpt-4"),
	})

	// Create provider
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL())
	provider, err := NewProvider(config)
	if err != nil {
		b.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello"},
		},
	}

	ctx := context.Background()

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		_, err := provider.SendCompletion(ctx, req)
		if err != nil {
			b.Fatalf("SendCompletion failed: %v", err)
		}
	}
}

func BenchmarkOpenAIProvider_RequestTransformation(b *testing.B) {
	// Create request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleSystem, Content: "You are a helpful assistant"},
			{Role: providers.RoleUser, Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Tools: []providers.Tool{
			{
				Type: providers.ToolTypeFunction,
				Function: providers.FunctionDefinition{
					Name:        "get_weather",
					Description: "Get the weather",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]string{
								"type": "string",
							},
						},
					},
				},
			},
		},
	}

	b.ResetTimer()

	// Benchmark transformation
	for i := 0; i < b.N; i++ {
		_ = transformRequest(req)
	}
}

func BenchmarkOpenAIProvider_ResponseTransformation(b *testing.B) {
	// Create OpenAI response
	openaiResp := &OpenAIResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	b.ResetTimer()

	// Benchmark transformation
	for i := 0; i < b.N; i++ {
		_, err := transformResponse(openaiResp)
		if err != nil {
			b.Fatalf("transformResponse failed: %v", err)
		}
	}
}

func BenchmarkOpenAIProvider_StreamCompletion(b *testing.B) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure streaming response
	chunks := []string{
		testhelpers.MockOpenAIStreamChunk("Hello", ""),
		testhelpers.MockOpenAIStreamChunk(", ", ""),
		testhelpers.MockOpenAIStreamChunk("world", ""),
		testhelpers.MockOpenAIStreamChunk("!", "stop"),
	}

	mock.SetResponse("/v1/chat/completions", testhelpers.MockResponse{
		StatusCode:   200,
		StreamChunks: chunks,
	})

	// Create provider
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL())
	provider, err := NewProvider(config)
	if err != nil {
		b.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello"},
		},
		Stream: true,
	}

	ctx := context.Background()

	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		chunksChan, err := provider.StreamCompletion(ctx, req)
		if err != nil {
			b.Fatalf("StreamCompletion failed: %v", err)
		}

		// Consume all chunks
		for chunk := range chunksChan {
			if chunk.Error != nil {
				b.Fatalf("stream error: %v", chunk.Error)
			}
		}
	}
}
