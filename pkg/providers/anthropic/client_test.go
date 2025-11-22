package anthropic

import (
	"context"
	"testing"

	testhelpers "mercator-hq/jupiter/internal/providers"
	"mercator-hq/jupiter/pkg/providers"
)

func TestAnthropicProvider_SendCompletion(t *testing.T) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure mock response
	mock.SetResponse("/v1/messages", testhelpers.MockResponse{
		StatusCode: 200,
		Body:       testhelpers.MockAnthropicResponse("Hello, world!", "claude-3-opus-20240229"),
	})

	// Create provider
	config := testhelpers.TestConfigWithURL("anthropic", "anthropic", mock.URL())
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := &providers.CompletionRequest{
		Model: "claude-3-opus-20240229",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello"},
		},
		MaxTokens: 1024,
	}

	// Send completion
	ctx := context.Background()
	resp, err := provider.SendCompletion(ctx, req)
	if err != nil {
		t.Fatalf("SendCompletion failed: %v", err)
	}

	// Verify response
	if resp.Model != "claude-3-opus-20240229" {
		t.Errorf("expected model claude-3-opus-20240229, got %s", resp.Model)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("expected content %q, got %q", "Hello, world!", resp.Content)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected total tokens 30, got %d", resp.Usage.TotalTokens)
	}
}

func TestAnthropicProvider_ValidationError(t *testing.T) {
	config := testhelpers.TestConfig("anthropic", "anthropic")
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
				Model:    "claude-3-opus-20240229",
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

func TestAnthropicProvider_MessageAlternation(t *testing.T) {
	config := testhelpers.TestConfig("anthropic", "anthropic")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Test: First message must be from user
	req := &providers.CompletionRequest{
		Model: "claude-3-opus-20240229",
		Messages: []providers.Message{
			{Role: providers.RoleAssistant, Content: "Hello"},
		},
		MaxTokens: 1024,
	}

	ctx := context.Background()
	_, err = provider.SendCompletion(ctx, req)
	if err == nil {
		t.Fatal("expected validation error for non-user first message, got nil")
	}

	// Test: Messages must alternate
	req = &providers.CompletionRequest{
		Model: "claude-3-opus-20240229",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Hello"},
			{Role: providers.RoleUser, Content: "Hello again"},
		},
		MaxTokens: 1024,
	}

	_, err = provider.SendCompletion(ctx, req)
	if err == nil {
		t.Fatal("expected validation error for non-alternating messages, got nil")
	}
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
