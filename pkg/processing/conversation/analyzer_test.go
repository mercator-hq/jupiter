package conversation

import (
	"testing"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/proxy/types"
)

func TestAnalyzer_AnalyzeConversation(t *testing.T) {
	cfg := &config.ConversationConfig{
		MaxContextWindow: map[string]int{
			"gpt-4":       8192,
			"gpt-4-turbo": 128000,
			"claude-3":    200000,
			"default":     4096,
		},
		WarnThreshold: 0.8,
	}

	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name                 string
		messages             []types.Message
		model                string
		totalTokens          int
		expectedTurnCount    int
		expectedMessageCount int
		expectedHasHistory   bool
		expectedPercentMin   float64
		expectedPercentMax   float64
	}{
		{
			name:                 "empty conversation",
			messages:             []types.Message{},
			model:                "gpt-4",
			totalTokens:          0,
			expectedTurnCount:    0,
			expectedMessageCount: 0,
			expectedHasHistory:   false,
		},
		{
			name: "single user message",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
			},
			model:                "gpt-4",
			totalTokens:          10,
			expectedTurnCount:    1,
			expectedMessageCount: 1,
			expectedHasHistory:   false,
			expectedPercentMin:   0.001,
			expectedPercentMax:   0.002,
		},
		{
			name: "single turn conversation",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
			model:                "gpt-4",
			totalTokens:          20,
			expectedTurnCount:    1,
			expectedMessageCount: 2,
			expectedHasHistory:   true,
			expectedPercentMin:   0.002,
			expectedPercentMax:   0.003,
		},
		{
			name: "multi-turn conversation",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm doing well, thanks!"},
			},
			model:                "gpt-4",
			totalTokens:          50,
			expectedTurnCount:    2,
			expectedMessageCount: 4,
			expectedHasHistory:   true,
			expectedPercentMin:   0.006,
			expectedPercentMax:   0.007,
		},
		{
			name: "conversation with system prompt",
			messages: []types.Message{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
			},
			model:                "gpt-4",
			totalTokens:          30,
			expectedTurnCount:    1,
			expectedMessageCount: 3,
			expectedHasHistory:   true,
		},
		{
			name: "incomplete turn (user message without response)",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "How are you?"},
			},
			model:                "gpt-4",
			totalTokens:          30,
			expectedTurnCount:    2, // 2 user messages
			expectedMessageCount: 3,
			expectedHasHistory:   true,
		},
		{
			name: "high context window usage",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
			},
			model:                "gpt-4",
			totalTokens:          7000, // ~85% of 8192
			expectedTurnCount:    1,
			expectedMessageCount: 1,
			expectedPercentMin:   0.85,
			expectedPercentMax:   0.86,
		},
		{
			name: "model prefix match",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
			},
			model:                "gpt-4-turbo-preview",
			totalTokens:          1000,
			expectedTurnCount:    1,
			expectedMessageCount: 1,
			expectedPercentMin:   0.007,
			expectedPercentMax:   0.009,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := analyzer.AnalyzeConversation(tt.messages, tt.model, tt.totalTokens)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if ctx.TurnCount != tt.expectedTurnCount {
				t.Errorf("expected turn count %d, got %d", tt.expectedTurnCount, ctx.TurnCount)
			}

			if ctx.MessageCount != tt.expectedMessageCount {
				t.Errorf("expected message count %d, got %d", tt.expectedMessageCount, ctx.MessageCount)
			}

			if ctx.HasConversationHistory != tt.expectedHasHistory {
				t.Errorf("expected HasConversationHistory=%v, got %v", tt.expectedHasHistory, ctx.HasConversationHistory)
			}

			if ctx.ContextWindowUsage != tt.totalTokens {
				t.Errorf("expected context window usage %d, got %d", tt.totalTokens, ctx.ContextWindowUsage)
			}

			if tt.expectedPercentMin > 0 {
				if ctx.ContextWindowPercent < tt.expectedPercentMin || ctx.ContextWindowPercent > tt.expectedPercentMax {
					t.Errorf("expected context window percent between %.3f and %.3f, got %.3f",
						tt.expectedPercentMin, tt.expectedPercentMax, ctx.ContextWindowPercent)
				}
			}
		})
	}
}

func TestAnalyzer_ExtractSystemPrompts(t *testing.T) {
	cfg := &config.ConversationConfig{
		MaxContextWindow: map[string]int{
			"default": 4096,
		},
	}

	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name                  string
		messages              []types.Message
		expectedSystemPrompts int
	}{
		{
			name: "no system prompts",
			messages: []types.Message{
				{Role: "user", Content: "Hello"},
			},
			expectedSystemPrompts: 0,
		},
		{
			name: "one system prompt",
			messages: []types.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hello"},
			},
			expectedSystemPrompts: 1,
		},
		{
			name: "multiple system prompts",
			messages: []types.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "system", Content: "Be concise."},
				{Role: "user", Content: "Hello"},
			},
			expectedSystemPrompts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := analyzer.AnalyzeConversation(tt.messages, "gpt-4", 100)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(ctx.SystemPrompts) != tt.expectedSystemPrompts {
				t.Errorf("expected %d system prompts, got %d", tt.expectedSystemPrompts, len(ctx.SystemPrompts))
			}
		})
	}
}

func TestAnalyzer_GetContextWindowLimit(t *testing.T) {
	cfg := &config.ConversationConfig{
		MaxContextWindow: map[string]int{
			"gpt-4":       8192,
			"gpt-4-turbo": 128000,
			"default":     4096,
		},
	}

	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{
			name:     "exact match",
			model:    "gpt-4",
			expected: 8192,
		},
		{
			name:     "prefix match",
			model:    "gpt-4-turbo-preview",
			expected: 128000,
		},
		{
			name:     "unknown model uses default",
			model:    "unknown-model",
			expected: 4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := analyzer.getContextWindowLimit(tt.model)
			if limit != tt.expected {
				t.Errorf("expected limit %d, got %d", tt.expected, limit)
			}
		})
	}
}

func TestExtractMessageContent(t *testing.T) {
	tests := []struct {
		name     string
		content  interface{}
		expected string
	}{
		{
			name:     "nil content",
			content:  nil,
			expected: "",
		},
		{
			name:     "string content",
			content:  "Hello world",
			expected: "Hello world",
		},
		{
			name: "array content with text",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
				map[string]interface{}{
					"type": "text",
					"text": "World",
				},
			},
			expected: "Hello World",
		},
		{
			name: "array content with mixed types",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
				map[string]interface{}{
					"type": "image_url",
					"url":  "https://example.com/image.jpg",
				},
			},
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMessageContent(tt.content)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
