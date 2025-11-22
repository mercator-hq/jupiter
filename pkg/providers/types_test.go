package providers

import (
	"testing"
)

func TestMessageConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"system role", RoleSystem, "system"},
		{"user role", RoleUser, "user"},
		{"assistant role", RoleAssistant, "assistant"},
		{"tool role", RoleTool, "tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}

func TestFinishReasonConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"stop reason", FinishReasonStop, "stop"},
		{"length reason", FinishReasonLength, "length"},
		{"tool calls reason", FinishReasonToolCalls, "tool_calls"},
		{"content filter reason", FinishReasonContentFilter, "content_filter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}

func TestToolTypeConstants(t *testing.T) {
	if ToolTypeFunction != "function" {
		t.Errorf("expected %q, got %q", "function", ToolTypeFunction)
	}
}

func TestCompletionRequest(t *testing.T) {
	req := &CompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      false,
	}

	if req.Model != "gpt-4" {
		t.Errorf("expected model %q, got %q", "gpt-4", req.Model)
	}

	if len(req.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, req.Messages[0].Role)
	}

	if req.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", req.Temperature)
	}

	if req.MaxTokens != 100 {
		t.Errorf("expected max_tokens 100, got %d", req.MaxTokens)
	}

	if req.Stream {
		t.Error("expected stream false, got true")
	}
}

func TestCompletionResponse(t *testing.T) {
	resp := &CompletionResponse{
		ID:           "resp-123",
		Model:        "gpt-4",
		Content:      "Hello, world!",
		FinishReason: FinishReasonStop,
		Usage: TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if resp.ID != "resp-123" {
		t.Errorf("expected ID %q, got %q", "resp-123", resp.ID)
	}

	if resp.Model != "gpt-4" {
		t.Errorf("expected model %q, got %q", "gpt-4", resp.Model)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("expected content %q, got %q", "Hello, world!", resp.Content)
	}

	if resp.FinishReason != FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", FinishReasonStop, resp.FinishReason)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected prompt tokens 10, got %d", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("expected completion tokens 5, got %d", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("expected total tokens 15, got %d", resp.Usage.TotalTokens)
	}
}

func TestStreamChunk(t *testing.T) {
	chunk := &StreamChunk{
		ID:           "chunk-123",
		Model:        "gpt-4",
		Delta:        "Hello",
		FinishReason: "",
	}

	if chunk.ID != "chunk-123" {
		t.Errorf("expected ID %q, got %q", "chunk-123", chunk.ID)
	}

	if chunk.Model != "gpt-4" {
		t.Errorf("expected model %q, got %q", "gpt-4", chunk.Model)
	}

	if chunk.Delta != "Hello" {
		t.Errorf("expected delta %q, got %q", "Hello", chunk.Delta)
	}

	if chunk.FinishReason != "" {
		t.Errorf("expected empty finish reason, got %q", chunk.FinishReason)
	}

	if chunk.Error != nil {
		t.Errorf("expected no error, got %v", chunk.Error)
	}
}

func TestToolCall(t *testing.T) {
	toolCall := ToolCall{
		ID:   "call-123",
		Type: ToolTypeFunction,
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `{"location": "San Francisco"}`,
		},
	}

	if toolCall.ID != "call-123" {
		t.Errorf("expected ID %q, got %q", "call-123", toolCall.ID)
	}

	if toolCall.Type != ToolTypeFunction {
		t.Errorf("expected type %q, got %q", ToolTypeFunction, toolCall.Type)
	}

	if toolCall.Function.Name != "get_weather" {
		t.Errorf("expected function name %q, got %q", "get_weather", toolCall.Function.Name)
	}

	expectedArgs := `{"location": "San Francisco"}`
	if toolCall.Function.Arguments != expectedArgs {
		t.Errorf("expected arguments %q, got %q", expectedArgs, toolCall.Function.Arguments)
	}
}

func TestProviderHealth(t *testing.T) {
	health := ProviderHealth{
		IsHealthy:           true,
		ConsecutiveFailures: 0,
		TotalRequests:       100,
		FailedRequests:      5,
	}

	if !health.IsHealthy {
		t.Error("expected healthy provider")
	}

	if health.ConsecutiveFailures != 0 {
		t.Errorf("expected 0 consecutive failures, got %d", health.ConsecutiveFailures)
	}

	if health.TotalRequests != 100 {
		t.Errorf("expected 100 total requests, got %d", health.TotalRequests)
	}

	if health.FailedRequests != 5 {
		t.Errorf("expected 5 failed requests, got %d", health.FailedRequests)
	}

	// Calculate success rate
	successRate := float64(health.TotalRequests-health.FailedRequests) / float64(health.TotalRequests)
	expectedSuccessRate := 0.95
	if successRate != expectedSuccessRate {
		t.Errorf("expected success rate %.2f, got %.2f", expectedSuccessRate, successRate)
	}
}
