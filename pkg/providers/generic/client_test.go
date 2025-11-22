package generic

import (
	"context"
	"testing"

	testhelpers "mercator-hq/jupiter/internal/providers"
	"mercator-hq/jupiter/pkg/providers"
)

func TestGenericProvider_SendCompletion(t *testing.T) {
	// Create mock server
	mock := testhelpers.NewMockServer()
	defer mock.Close()

	// Configure mock response (OpenAI format)
	mock.SetResponse("/v1/chat/completions", testhelpers.MockResponse{
		StatusCode: 200,
		Body:       testhelpers.MockOpenAIResponse("Hello from Ollama!", "llama2"),
	})

	// Create provider
	config := testhelpers.TestConfigWithURL("ollama", "generic", mock.URL()+"/v1")
	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Create request
	req := &providers.CompletionRequest{
		Model: "llama2",
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
	if resp.Model != "llama2" {
		t.Errorf("expected model llama2, got %s", resp.Model)
	}

	if resp.Content != "Hello from Ollama!" {
		t.Errorf("expected content %q, got %q", "Hello from Ollama!", resp.Content)
	}
}

func TestGenericProvider_NoAPIKey(t *testing.T) {
	// Create provider without API key (local providers don't need it)
	config := providers.ProviderConfig{
		Name:    "ollama",
		Type:    "generic",
		BaseURL: "http://localhost:11434/v1",
		APIKey:  "", // No API key
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider without API key: %v", err)
	}
	defer provider.Close()

	// Verify provider was created successfully
	if provider.GetName() != "ollama" {
		t.Errorf("expected provider name ollama, got %s", provider.GetName())
	}

	if provider.GetType() != "generic" {
		t.Errorf("expected provider type generic, got %s", provider.GetType())
	}
}

func TestGenericProvider_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  providers.ProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: providers.ProviderConfig{
				Name:    "ollama",
				Type:    "generic",
				BaseURL: "http://localhost:11434/v1",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: providers.ProviderConfig{
				Type:    "generic",
				BaseURL: "http://localhost:11434/v1",
			},
			wantErr: true,
		},
		{
			name: "missing base URL",
			config: providers.ProviderConfig{
				Name: "ollama",
				Type: "generic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
