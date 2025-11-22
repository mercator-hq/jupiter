package providerfactory

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/providers"
)

func TestNewProvider_OpenAI(t *testing.T) {
	config := providers.ProviderConfig{
		Name:    "openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}
	defer provider.Close()

	if provider.GetName() != "openai" {
		t.Errorf("expected provider name openai, got %s", provider.GetName())
	}

	if provider.GetType() != "openai" {
		t.Errorf("expected provider type openai, got %s", provider.GetType())
	}
}

func TestNewProvider_Anthropic(t *testing.T) {
	config := providers.ProviderConfig{
		Name:    "anthropic",
		Type:    "anthropic",
		BaseURL: "https://api.anthropic.com",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}
	defer provider.Close()

	if provider.GetName() != "anthropic" {
		t.Errorf("expected provider name anthropic, got %s", provider.GetName())
	}

	if provider.GetType() != "anthropic" {
		t.Errorf("expected provider type anthropic, got %s", provider.GetType())
	}
}

func TestNewProvider_Generic(t *testing.T) {
	config := providers.ProviderConfig{
		Name:    "ollama",
		Type:    "generic",
		BaseURL: "http://localhost:11434/v1",
		APIKey:  "",
		Timeout: 30 * time.Second,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("NewProvider() failed: %v", err)
	}
	defer provider.Close()

	if provider.GetName() != "ollama" {
		t.Errorf("expected provider name ollama, got %s", provider.GetName())
	}

	if provider.GetType() != "generic" {
		t.Errorf("expected provider type generic, got %s", provider.GetType())
	}
}

func TestNewProvider_TypeInference(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		wantType     string
	}{
		{
			name:         "openai inferred",
			providerName: "openai",
			wantType:     "openai",
		},
		{
			name:         "anthropic inferred",
			providerName: "anthropic",
			wantType:     "anthropic",
		},
		{
			name:         "ollama inferred as generic",
			providerName: "ollama",
			wantType:     "generic",
		},
		{
			name:         "lmstudio inferred as generic",
			providerName: "lmstudio",
			wantType:     "generic",
		},
		{
			name:         "vllm inferred as generic",
			providerName: "vllm",
			wantType:     "generic",
		},
		{
			name:         "unknown inferred as generic",
			providerName: "custom-llm",
			wantType:     "generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := providers.ProviderConfig{
				Name: tt.providerName,
				// Type not specified - should be inferred
				BaseURL: "http://localhost:8080",
				APIKey:  "test-key",
			}

			provider, err := NewProvider(config)
			if err != nil {
				t.Fatalf("NewProvider() failed: %v", err)
			}
			defer provider.Close()

			if provider.GetType() != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, provider.GetType())
			}
		})
	}
}

func TestNewProvider_UnsupportedType(t *testing.T) {
	config := providers.ProviderConfig{
		Name:    "test",
		Type:    "unsupported-type",
		BaseURL: "http://localhost:8080",
		APIKey:  "test-key",
	}

	_, err := NewProvider(config)
	if err == nil {
		t.Fatal("expected error for unsupported provider type, got nil")
	}

	configErr, ok := err.(*providers.ConfigError)
	if !ok {
		t.Fatalf("expected ConfigError, got %T: %v", err, err)
	}

	if configErr.Field != "type" {
		t.Errorf("expected error for field 'type', got %q", configErr.Field)
	}
}

func TestNewProviderWithHealthCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := providers.ProviderConfig{
		Name:                "openai",
		Type:                "openai",
		BaseURL:             "https://api.openai.com/v1",
		APIKey:              "test-key",
		HealthCheckInterval: 1 * time.Second,
	}

	provider, err := NewProviderWithHealthCheck(ctx, config)
	if err != nil {
		t.Fatalf("NewProviderWithHealthCheck() failed: %v", err)
	}
	defer provider.Close()

	// Verify provider was created
	if provider.GetName() != "openai" {
		t.Errorf("expected provider name openai, got %s", provider.GetName())
	}

	// Verify health status can be queried
	_ = provider.IsHealthy()
}

func TestInferProviderType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"openai", "openai"},
		{"anthropic", "anthropic"},
		{"ollama", "generic"},
		{"lmstudio", "generic"},
		{"vllm", "generic"},
		{"localai", "generic"},
		{"unknown-provider", "generic"},
		{"custom", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferProviderType(tt.name)
			if result != tt.expected {
				t.Errorf("inferProviderType(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
