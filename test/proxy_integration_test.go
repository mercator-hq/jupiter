//go:build integration

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/types"
	"mercator-hq/jupiter/pkg/server"
)

// TestProxyIntegration tests the end-to-end flow from HTTP request to provider response.
func TestProxyIntegration(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Proxy: config.ProxyConfig{
			ListenAddress:   "127.0.0.1:0", // Use dynamic port
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			MaxHeaderBytes:  1048576,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
				MaxAge:         3600,
			},
		},
		Security: config.SecurityConfig{
			TLS: config.TLSConfig{
				Enabled: false,
			},
		},
	}

	// Create mock provider manager
	pm := &mockProviderManager{
		providers: map[string]providers.Provider{
			"openai": &mockProvider{name: "openai"},
		},
	}

	// Create server
	srv := server.NewServer(&cfg.Proxy, &cfg.Security, pm)

	// Create test server
	testServer := httptest.NewServer(srv.Handler())
	defer testServer.Close()

	t.Run("chat completion request", func(t *testing.T) {
		// Create request
		reqBody := types.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []types.Message{
				{Role: "user", Content: "Hello, world!"},
			},
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		// Send request
		resp, err := http.Post(testServer.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		// Parse response
		var chatResp types.ChatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify response structure
		if chatResp.Object != "chat.completion" {
			t.Errorf("Object = %v, want chat.completion", chatResp.Object)
		}

		if len(chatResp.Choices) == 0 {
			t.Fatal("No choices in response")
		}

		if chatResp.Choices[0].Message.Role != "assistant" {
			t.Errorf("Message role = %v, want assistant", chatResp.Choices[0].Message.Role)
		}

		if chatResp.Choices[0].Message.Content == "" {
			t.Error("Message content should not be empty")
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		// Create invalid request (missing model)
		reqBody := types.ChatCompletionRequest{
			Messages: []types.Message{
				{Role: "user", Content: "Hello"},
			},
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		// Send request
		resp, err := http.Post(testServer.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Should return 400 Bad Request
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status code = %v, want %v", resp.StatusCode, http.StatusBadRequest)
		}

		// Parse error response
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if errResp.Error.Type != types.ErrorTypeInvalidRequest {
			t.Errorf("Error type = %v, want %v", errResp.Error.Type, types.ErrorTypeInvalidRequest)
		}
	})

	t.Run("health check", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("readiness check", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/ready")
		if err != nil {
			t.Fatalf("Failed to send readiness check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("provider health check", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/health/providers")
		if err != nil {
			t.Fatalf("Failed to send provider health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["providers"] == nil {
			t.Error("Response should include providers")
		}

		// Check that providers is a map
		providers, ok := result["providers"].(map[string]interface{})
		if !ok {
			t.Error("providers should be a map")
		}

		// Should have our mock openai provider
		if len(providers) == 0 {
			t.Error("providers map should not be empty")
		}
	})
}

// Mock provider for integration testing
type mockProvider struct {
	name   string
	pType  string
	config providers.ProviderConfig
}

func (m *mockProvider) GetName() string {
	return m.name
}

func (m *mockProvider) GetType() string {
	if m.pType == "" {
		return m.name
	}
	return m.pType
}

func (m *mockProvider) GetConfig() providers.ProviderConfig {
	return m.config
}

func (m *mockProvider) SendCompletion(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	return &providers.CompletionResponse{
		ID:           "test-completion-123",
		Model:        req.Model,
		Content:      "This is a test response from the mock provider.",
		FinishReason: providers.FinishReasonStop,
		Usage: providers.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
		Created: time.Now().Unix(),
	}, nil
}

func (m *mockProvider) StreamCompletion(ctx context.Context, req *providers.CompletionRequest) (<-chan *providers.StreamChunk, error) {
	ch := make(chan *providers.StreamChunk, 3)

	// Send a few test chunks
	go func() {
		defer close(ch)

		chunks := []string{"This ", "is ", "a test."}
		for i, text := range chunks {
			select {
			case <-ctx.Done():
				return
			case ch <- &providers.StreamChunk{
				ID:      "test-stream-123",
				Model:   req.Model,
				Delta:   text,
				Created: time.Now().Unix(),
			}:
			}

			// Send finish reason in last chunk
			if i == len(chunks)-1 {
				ch <- &providers.StreamChunk{
					ID:           "test-stream-123",
					Model:        req.Model,
					Delta:        "",
					FinishReason: providers.FinishReasonStop,
					Usage: &providers.TokenUsage{
						PromptTokens:     10,
						CompletionTokens: 15,
						TotalTokens:      25,
					},
					Created: time.Now().Unix(),
				}
			}
		}
	}()

	return ch, nil
}

func (m *mockProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockProvider) IsHealthy() bool {
	return true
}

func (m *mockProvider) GetHealth() providers.ProviderHealth {
	return providers.ProviderHealth{
		IsHealthy:             true,
		LastCheck:             time.Now(),
		LastSuccessfulRequest: time.Now(),
		TotalRequests:         100,
		FailedRequests:        0,
	}
}

func (m *mockProvider) Close() error {
	return nil
}

// Mock provider manager for integration testing
type mockProviderManager struct {
	providers map[string]providers.Provider
}

func (m *mockProviderManager) GetProvider(name string) (providers.Provider, error) {
	if p, ok := m.providers[name]; ok {
		return p, nil
	}
	return nil, &providers.ProviderError{
		Message:    "Provider not found",
		Provider:   name,
		StatusCode: 404,
	}
}

func (m *mockProviderManager) GetHealthyProviders() map[string]providers.Provider {
	return m.providers
}

func (m *mockProviderManager) Close() error {
	return nil
}
