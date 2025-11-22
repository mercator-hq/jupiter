package tokens

import (
	"testing"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/proxy/types"
)

func TestSimpleEstimator_EstimateText(t *testing.T) {
	cfg := &config.TokensConfig{
		Models: map[string]float64{
			"gpt-4":   4.0,
			"claude":  3.5,
			"default": 4.0,
		},
	}

	estimator := NewSimpleEstimator(cfg)

	tests := []struct {
		name        string
		text        string
		model       string
		expectedMin int
		expectedMax int
		expectError bool
	}{
		{
			name:        "empty text",
			text:        "",
			model:       "gpt-4",
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name:        "short text gpt-4",
			text:        "Hello, world!",
			model:       "gpt-4",
			expectedMin: 2,
			expectedMax: 4,
		},
		{
			name:        "short text claude",
			text:        "Hello, world!",
			model:       "claude",
			expectedMin: 3,
			expectedMax: 5,
		},
		{
			name:        "medium text",
			text:        "This is a longer message that should result in more tokens being estimated for the request.",
			model:       "gpt-4",
			expectedMin: 20,
			expectedMax: 25,
		},
		{
			name:        "unknown model uses default",
			text:        "Hello, world!",
			model:       "unknown-model",
			expectedMin: 2,
			expectedMax: 4,
		},
		{
			name:        "model prefix match",
			text:        "Hello, world!",
			model:       "gpt-4-turbo",
			expectedMin: 2,
			expectedMax: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := estimator.EstimateText(tt.text, tt.model)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tokens < tt.expectedMin || tokens > tt.expectedMax {
				t.Errorf("expected tokens between %d and %d, got %d",
					tt.expectedMin, tt.expectedMax, tokens)
			}
		})
	}
}

func TestSimpleEstimator_EstimateMessages(t *testing.T) {
	cfg := &config.TokensConfig{
		Models: map[string]float64{
			"gpt-4":   4.0,
			"default": 4.0,
		},
	}

	estimator := NewSimpleEstimator(cfg)

	tests := []struct {
		name        string
		messages    []types.Message
		model       string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "empty messages",
			messages:    []types.Message{},
			model:       "gpt-4",
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name: "single user message",
			messages: []types.Message{
				{
					Role:    "user",
					Content: "Hello, how are you?",
				},
			},
			model:       "gpt-4",
			expectedMin: 5,
			expectedMax: 15,
		},
		{
			name: "multi-turn conversation",
			messages: []types.Message{
				{
					Role:    "user",
					Content: "What is the capital of France?",
				},
				{
					Role:    "assistant",
					Content: "The capital of France is Paris.",
				},
				{
					Role:    "user",
					Content: "What about Germany?",
				},
			},
			model:       "gpt-4",
			expectedMin: 20,
			expectedMax: 40,
		},
		{
			name: "message with name",
			messages: []types.Message{
				{
					Role:    "user",
					Name:    "john",
					Content: "Hello!",
				},
			},
			model:       "gpt-4",
			expectedMin: 5,
			expectedMax: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := estimator.EstimateMessages(tt.messages, tt.model)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tokens < tt.expectedMin || tokens > tt.expectedMax {
				t.Errorf("expected tokens between %d and %d, got %d",
					tt.expectedMin, tt.expectedMax, tokens)
			}
		})
	}
}

func TestSimpleEstimator_EstimateTools(t *testing.T) {
	cfg := &config.TokensConfig{
		Models: map[string]float64{
			"gpt-4":   4.0,
			"default": 4.0,
		},
	}

	estimator := NewSimpleEstimator(cfg)

	tests := []struct {
		name        string
		tools       []types.Tool
		model       string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "empty tools",
			tools:       []types.Tool{},
			model:       "gpt-4",
			expectedMin: 0,
			expectedMax: 0,
		},
		{
			name: "single tool",
			tools: []types.Tool{
				{
					Type: "function",
					Function: types.FunctionDefinition{
						Name:        "get_weather",
						Description: "Get the current weather for a location",
						Parameters: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"location": map[string]interface{}{
									"type":        "string",
									"description": "The city name",
								},
							},
						},
					},
				},
			},
			model:       "gpt-4",
			expectedMin: 30,
			expectedMax: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := estimator.EstimateTools(tt.tools, tt.model)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tokens < tt.expectedMin || tokens > tt.expectedMax {
				t.Errorf("expected tokens between %d and %d, got %d",
					tt.expectedMin, tt.expectedMax, tokens)
			}
		})
	}
}

func TestSimpleEstimator_EstimateRequest(t *testing.T) {
	cfg := &config.TokensConfig{
		Models: map[string]float64{
			"gpt-4":   4.0,
			"default": 4.0,
		},
	}

	estimator := NewSimpleEstimator(cfg)

	tests := []struct {
		name        string
		request     *types.ChatCompletionRequest
		expectedMin int
		expectedMax int
		expectError bool
	}{
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "simple request",
			request: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{
						Role:    "user",
						Content: "Hello, how are you?",
					},
				},
			},
			expectedMin: 100, // Includes default completion estimate
			expectedMax: 200,
		},
		{
			name: "request with system prompt",
			request: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{
						Role:    "system",
						Content: "You are a helpful assistant.",
					},
					{
						Role:    "user",
						Content: "Hello!",
					},
				},
			},
			expectedMin: 100,
			expectedMax: 200,
		},
		{
			name: "request with max_tokens",
			request: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{
						Role:    "user",
						Content: "Hello!",
					},
				},
				MaxTokens: intPtr(500),
			},
			expectedMin: 500, // Should include the 500 completion tokens
			expectedMax: 600,
		},
		{
			name: "request with tools",
			request: &types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{
						Role:    "user",
						Content: "What's the weather?",
					},
				},
				Tools: []types.Tool{
					{
						Type: "function",
						Function: types.FunctionDefinition{
							Name:        "get_weather",
							Description: "Get the current weather",
						},
					},
				},
			},
			expectedMin: 100,
			expectedMax: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate, err := estimator.EstimateRequest(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if estimate.TotalTokens < tt.expectedMin || estimate.TotalTokens > tt.expectedMax {
				t.Errorf("expected total tokens between %d and %d, got %d",
					tt.expectedMin, tt.expectedMax, estimate.TotalTokens)
			}

			// Verify that TotalTokens = PromptTokens + EstimatedCompletionTokens
			expectedTotal := estimate.PromptTokens + estimate.EstimatedCompletionTokens
			if estimate.TotalTokens != expectedTotal {
				t.Errorf("total tokens mismatch: TotalTokens=%d, but PromptTokens=%d + EstimatedCompletionTokens=%d = %d",
					estimate.TotalTokens, estimate.PromptTokens, estimate.EstimatedCompletionTokens, expectedTotal)
			}

			// Verify confidence is set
			if estimate.Confidence <= 0 || estimate.Confidence > 1.0 {
				t.Errorf("invalid confidence: %f (should be between 0 and 1)", estimate.Confidence)
			}

			// Verify model is set
			if estimate.Model == "" {
				t.Errorf("model not set in estimate")
			}
		})
	}
}

func TestSimpleEstimator_ExtractContent(t *testing.T) {
	cfg := &config.TokensConfig{
		Models: map[string]float64{
			"gpt-4":   4.0,
			"default": 4.0,
		},
	}

	estimator := NewSimpleEstimator(cfg)

	tests := []struct {
		name     string
		content  interface{}
		contains string
	}{
		{
			name:     "nil content",
			content:  nil,
			contains: "",
		},
		{
			name:     "string content",
			content:  "Hello, world!",
			contains: "Hello, world!",
		},
		{
			name: "array content with text",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello from array",
				},
			},
			contains: "Hello from array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.extractContent(tt.content)
			if tt.contains != "" && result != tt.contains {
				t.Errorf("expected content to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}
