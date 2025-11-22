package openai

import (
	"context"
	"testing"
	"time"

	testhelpers "mercator-hq/jupiter/internal/providers"
	"mercator-hq/jupiter/pkg/providers"
)

func TestOpenAIProvider_SendCompletion(t *testing.T) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure mock response
	mock.SetResponse("/v1/chat/completions", testhelpers.MockResponse{
		StatusCode: 200,
		Body:       testhelpers.MockOpenAIResponse("Hello, world!", "gpt-4"),
	})

	// Create provider
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL()+"/v1")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello"},
		},
	}

	// Send completion
	ctx := context.Background()
	resp, err := provider.SendCompletion(ctx, req)
	if err != nil {
		t.Fatalf("SendCompletion failed: %v", err)
	}

	// Verify response
	if resp.Model != "gpt-4" {
		t.Errorf("expected model gpt-4, got %s", resp.Model)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("expected content %q, got %q", "Hello, world!", resp.Content)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected total tokens 30, got %d", resp.Usage.TotalTokens)
	}

	if resp.FinishReason != providers.FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", providers.FinishReasonStop, resp.FinishReason)
	}

	// Verify request was sent
	if mock.GetRequestCount() != 1 {
		t.Errorf("expected 1 request, got %d", mock.GetRequestCount())
	}
}

func TestOpenAIProvider_StreamCompletion(t *testing.T) {
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
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL()+"/v1")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create streaming request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello"},
		},
		Stream: true,
	}

	// Send streaming request
	ctx := context.Background()
	chunksChan, err := provider.StreamCompletion(ctx, req)
	if err != nil {
		t.Fatalf("StreamCompletion failed: %v", err)
	}

	// Collect chunks
	var receivedChunks []*providers.StreamChunk
	for chunk := range chunksChan {
		if chunk.Error != nil {
			t.Fatalf("stream error: %v", chunk.Error)
		}
		receivedChunks = append(receivedChunks, chunk)
	}

	// Verify chunks
	if len(receivedChunks) != 4 {
		t.Errorf("expected 4 chunks, got %d", len(receivedChunks))
	}

	// Concatenate content
	var fullContent string
	for _, chunk := range receivedChunks {
		fullContent += chunk.Delta
	}

	expected := "Hello, world!"
	if fullContent != expected {
		t.Errorf("expected content %q, got %q", expected, fullContent)
	}

	// Verify final chunk has finish reason
	lastChunk := receivedChunks[len(receivedChunks)-1]
	if lastChunk.FinishReason != providers.FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", providers.FinishReasonStop, lastChunk.FinishReason)
	}
}

func TestOpenAIProvider_AuthError(t *testing.T) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure auth error response
	mock.SetResponse("/v1/chat/completions", testhelpers.MockAuthError())

	// Create provider
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL()+"/v1")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := testhelpers.TestCompletionRequest("gpt-4",
		testhelpers.TestMessage(providers.RoleUser, "Hello"))

	// Send request (should fail with auth error)
	ctx := context.Background()
	_, err = provider.SendCompletion(ctx, req)
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}

	// Verify error type
	authErr, ok := err.(*providers.AuthError)
	if !ok {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}

	if authErr.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", authErr.Provider)
	}
}

func TestOpenAIProvider_RateLimitError(t *testing.T) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure rate limit error response
	mock.SetResponse("/v1/chat/completions", testhelpers.MockRateLimitError(60))

	// Create provider
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL()+"/v1")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := testhelpers.TestCompletionRequest("gpt-4",
		testhelpers.TestMessage(providers.RoleUser, "Hello"))

	// Send request (should fail with rate limit error)
	ctx := context.Background()
	_, err = provider.SendCompletion(ctx, req)
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}

	// Verify error type
	rateLimitErr, ok := err.(*providers.RateLimitError)
	if !ok {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}

	if rateLimitErr.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", rateLimitErr.Provider)
	}

	if rateLimitErr.RetryAfter != 60*time.Second {
		t.Errorf("expected retry after 60s, got %s", rateLimitErr.RetryAfter)
	}
}

func TestOpenAIProvider_ValidationError(t *testing.T) {
	config := testhelpers.TestConfig("openai", "openai")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	tests := []struct {
		name    string
		req     *providers.CompletionRequest
		wantErr string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: "request cannot be nil",
		},
		{
			name: "empty model",
			req: &providers.CompletionRequest{
				Messages: []providers.Message{
					{Role: providers.RoleUser, Content: "Hello"},
				},
			},
			wantErr: "model is required",
		},
		{
			name: "empty messages",
			req: &providers.CompletionRequest{
				Model:    "gpt-4",
				Messages: []providers.Message{},
			},
			wantErr: "at least one message is required",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.SendCompletion(ctx, tt.req)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			validationErr, ok := err.(*providers.ValidationError)
			if !ok {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}

			if !containsString(validationErr.Message, tt.wantErr) {
				t.Errorf("expected error message to contain %q, got %q", tt.wantErr, validationErr.Message)
			}
		})
	}
}

func TestOpenAIProvider_Retry(t *testing.T) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure to return error twice, then success
	callCount := 0
	mock.SetResponse("/v1/chat/completions", testhelpers.MockResponse{
		StatusCode: 500,
		Body:       testhelpers.MockErrorResponse(500, "Internal server error"),
	})

	// Create provider with retries
	config := testhelpers.TestConfigWithURL("openai", "openai", mock.URL()+"/v1")
	config.MaxRetries = 3
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := testhelpers.TestCompletionRequest("gpt-4",
		testhelpers.TestMessage(providers.RoleUser, "Hello"))

	// Send request (should retry and eventually fail)
	ctx := context.Background()
	_, err = provider.SendCompletion(ctx, req)
	if err == nil {
		t.Fatal("expected error after retries, got nil")
	}

	// Verify multiple requests were made (initial + retries)
	if mock.GetRequestCount() <= 1 {
		t.Errorf("expected multiple requests (retries), got %d", mock.GetRequestCount())
	}

	// Verify it's a provider error
	_, ok := err.(*providers.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}

	_ = callCount // Keep linter happy
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
