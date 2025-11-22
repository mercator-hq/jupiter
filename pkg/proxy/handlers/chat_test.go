package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/types"
)

func TestConvertMessageContent(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    string
	}{
		{
			name:    "string content",
			content: "Hello, world!",
			want:    "Hello, world!",
		},
		{
			name:    "nil content",
			content: nil,
			want:    "",
		},
		{
			name: "multimodal content with text only",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "What's in this image?",
				},
			},
			want: "What's in this image?",
		},
		{
			name: "multimodal content with text and image",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Part 1",
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]string{
						"url": "https://example.com/image.jpg",
					},
				},
				map[string]interface{}{
					"type": "text",
					"text": "Part 2",
				},
			},
			want: "Part 1 Part 2",
		},
		{
			name: "multimodal content with only images",
			content: []interface{}{
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]string{
						"url": "https://example.com/image.jpg",
					},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMessageContent(tt.content)
			if got != tt.want {
				t.Errorf("convertMessageContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		toolCalls []types.ToolCall
		want      int
	}{
		{
			name:      "nil tool calls",
			toolCalls: nil,
			want:      0,
		},
		{
			name:      "empty tool calls",
			toolCalls: []types.ToolCall{},
			want:      0,
		},
		{
			name: "single tool call",
			toolCalls: []types.ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "Boston"}`,
					},
				},
			},
			want: 1,
		},
		{
			name: "multiple tool calls",
			toolCalls: []types.ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "Boston"}`,
					},
				},
				{
					ID:   "call_456",
					Type: "function",
					Function: types.FunctionCall{
						Name:      "get_time",
						Arguments: `{}`,
					},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToolCalls(tt.toolCalls)

			if tt.want == 0 && got != nil {
				t.Errorf("convertToolCalls() should return nil for empty input, got %v", got)
				return
			}

			if len(got) != tt.want {
				t.Errorf("convertToolCalls() length = %v, want %v", len(got), tt.want)
			}

			// Verify conversion is correct
			for i, tc := range tt.toolCalls {
				if i >= len(got) {
					break
				}
				if got[i].ID != tc.ID {
					t.Errorf("ID[%d] = %v, want %v", i, got[i].ID, tc.ID)
				}
				if got[i].Type != tc.Type {
					t.Errorf("Type[%d] = %v, want %v", i, got[i].Type, tc.Type)
				}
				if got[i].Function.Name != tc.Function.Name {
					t.Errorf("Function.Name[%d] = %v, want %v", i, got[i].Function.Name, tc.Function.Name)
				}
			}
		})
	}
}

func TestConvertTools(t *testing.T) {
	tests := []struct {
		name  string
		tools []types.Tool
		want  int
	}{
		{
			name:  "nil tools",
			tools: nil,
			want:  0,
		},
		{
			name:  "empty tools",
			tools: []types.Tool{},
			want:  0,
		},
		{
			name: "single tool",
			tools: []types.Tool{
				{
					Type: "function",
					Function: types.FunctionDefinition{
						Name:        "get_weather",
						Description: "Get the current weather",
						Parameters: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"location": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertTools(tt.tools)

			if tt.want == 0 && got != nil {
				t.Errorf("convertTools() should return nil for empty input, got %v", got)
				return
			}

			if len(got) != tt.want {
				t.Errorf("convertTools() length = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestConvertToProviderRequest(t *testing.T) {
	temp := 0.7
	maxTokens := 100

	tests := []struct {
		name string
		req  *types.ChatCompletionRequest
		want *providers.CompletionRequest
	}{
		{
			name: "basic request",
			req: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			want: &providers.CompletionRequest{
				Model: "gpt-4",
				Messages: []providers.Message{
					{Role: "user", Content: "Hello"},
				},
			},
		},
		{
			name: "request with optional parameters",
			req: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: &temp,
				MaxTokens:   &maxTokens,
			},
			want: &providers.CompletionRequest{
				Model: "gpt-4",
				Messages: []providers.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			},
		},
		{
			name: "request with tool calls",
			req: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{
						Role: "assistant",
						ToolCalls: []types.ToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: types.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"location": "Boston"}`,
								},
							},
						},
					},
				},
				Tools: []types.Tool{
					{
						Type: "function",
						Function: types.FunctionDefinition{
							Name:        "get_weather",
							Description: "Get weather",
						},
					},
				},
			},
			want: &providers.CompletionRequest{
				Model: "gpt-4",
				Messages: []providers.Message{
					{
						Role: "assistant",
						ToolCalls: []providers.ToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: providers.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"location": "Boston"}`,
								},
							},
						},
					},
				},
				Tools: []providers.Tool{
					{
						Type: "function",
						Function: providers.FunctionDefinition{
							Name:        "get_weather",
							Description: "Get weather",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToProviderRequest(tt.req)

			if got.Model != tt.want.Model {
				t.Errorf("Model = %v, want %v", got.Model, tt.want.Model)
			}

			if len(got.Messages) != len(tt.want.Messages) {
				t.Errorf("Messages length = %v, want %v", len(got.Messages), len(tt.want.Messages))
			}

			if tt.want.Temperature != 0 && got.Temperature != tt.want.Temperature {
				t.Errorf("Temperature = %v, want %v", got.Temperature, tt.want.Temperature)
			}

			if tt.want.MaxTokens != 0 && got.MaxTokens != tt.want.MaxTokens {
				t.Errorf("MaxTokens = %v, want %v", got.MaxTokens, tt.want.MaxTokens)
			}
		})
	}
}

func TestSelectProviderByModel(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		healthyProviders map[string]providers.Provider
		wantProviderName string
	}{
		{
			name:  "gpt-4 model selects openai",
			model: "gpt-4",
			healthyProviders: map[string]providers.Provider{
				"openai":    &mockProvider{name: "openai"},
				"anthropic": &mockProvider{name: "anthropic"},
			},
			wantProviderName: "openai",
		},
		{
			name:  "claude model selects anthropic",
			model: "claude-3-opus-20240229",
			healthyProviders: map[string]providers.Provider{
				"openai":    &mockProvider{name: "openai"},
				"anthropic": &mockProvider{name: "anthropic"},
			},
			wantProviderName: "anthropic",
		},
		{
			name:  "unknown model returns nil",
			model: "unknown-model",
			healthyProviders: map[string]providers.Provider{
				"openai": &mockProvider{name: "openai"},
			},
			wantProviderName: "",
		},
		{
			name:  "gpt model without matching provider returns nil",
			model: "gpt-4",
			healthyProviders: map[string]providers.Provider{
				"anthropic": &mockProvider{name: "anthropic"},
			},
			wantProviderName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectProviderByModel(tt.healthyProviders, tt.model)

			if tt.wantProviderName == "" {
				if got != nil {
					t.Errorf("selectProviderByModel() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("selectProviderByModel() returned nil, expected provider")
			}

			if got.GetName() != tt.wantProviderName {
				t.Errorf("Provider name = %v, want %v", got.GetName(), tt.wantProviderName)
			}
		})
	}
}

// Mock provider for testing
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
		ID:           "test-123",
		Model:        req.Model,
		Content:      "Test response",
		FinishReason: "stop",
	}, nil
}

func (m *mockProvider) StreamCompletion(ctx context.Context, req *providers.CompletionRequest) (<-chan *providers.StreamChunk, error) {
	ch := make(chan *providers.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockProvider) IsHealthy() bool {
	return true
}

func (m *mockProvider) GetHealth() providers.ProviderHealth {
	return providers.ProviderHealth{IsHealthy: true}
}

func (m *mockProvider) Close() error {
	return nil
}

// Mock provider manager for testing
type mockProviderManager struct {
	providers map[string]providers.Provider
}

func (m *mockProviderManager) GetProvider(name string) (providers.Provider, error) {
	if p, ok := m.providers[name]; ok {
		return p, nil
	}
	return nil, &providers.ProviderError{
		Message:  "Provider not found",
		Provider: name,
	}
}

func (m *mockProviderManager) GetHealthyProviders() map[string]providers.Provider {
	return m.providers
}

func (m *mockProviderManager) Close() error {
	return nil
}

func TestHandleChatRequest_Integration(t *testing.T) {
	// Create mock provider manager
	pm := &mockProviderManager{
		providers: map[string]providers.Provider{
			"openai": &mockProvider{name: "openai"},
		},
	}

	// Create test request
	reqBody := types.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Call handler
	handleChatRequest(w, req, pm)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify response is valid JSON
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}
}
