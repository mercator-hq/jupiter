package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/types"
)

func TestFormatChatCompletionResponse(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name         string
		providerResp *providers.CompletionResponse
		model        string
		wantModel    string
		wantContent  string
		wantNonEmpty bool
	}{
		{
			name: "basic response",
			providerResp: &providers.CompletionResponse{
				ID:           "resp-123",
				Model:        "gpt-4",
				Content:      "Hello, how can I help you?",
				FinishReason: "stop",
				Usage: providers.TokenUsage{
					PromptTokens:     10,
					CompletionTokens: 8,
					TotalTokens:      18,
				},
				Created: now,
			},
			model:        "gpt-4",
			wantModel:    "gpt-4",
			wantContent:  "Hello, how can I help you?",
			wantNonEmpty: true,
		},
		{
			name: "response with tool calls",
			providerResp: &providers.CompletionResponse{
				ID:           "resp-456",
				Model:        "gpt-4",
				Content:      "",
				FinishReason: "tool_calls",
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
				Usage: providers.TokenUsage{
					PromptTokens:     15,
					CompletionTokens: 5,
					TotalTokens:      20,
				},
				Created: now,
			},
			model:        "gpt-4",
			wantModel:    "gpt-4",
			wantContent:  "",
			wantNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatChatCompletionResponse(tt.providerResp, tt.model)

			if got == nil {
				t.Fatal("FormatChatCompletionResponse() returned nil")
			}

			if got.Model != tt.wantModel {
				t.Errorf("Model = %v, want %v", got.Model, tt.wantModel)
			}

			if tt.wantNonEmpty && got.ID == "" {
				t.Error("ID should not be empty")
			}

			if len(got.Choices) == 0 {
				t.Fatal("Choices should not be empty")
			}

			choice := got.Choices[0]
			if choice.Message.Content != tt.wantContent {
				t.Errorf("Content = %v, want %v", choice.Message.Content, tt.wantContent)
			}

			if got.Usage.PromptTokens != tt.providerResp.Usage.PromptTokens {
				t.Errorf("PromptTokens = %v, want %v", got.Usage.PromptTokens, tt.providerResp.Usage.PromptTokens)
			}
		})
	}
}

func TestFormatStreamChunk(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name       string
		chunk      *providers.StreamChunk
		model      string
		responseID string
		wantDelta  string
	}{
		{
			name: "content chunk",
			chunk: &providers.StreamChunk{
				ID:      "chunk-123",
				Model:   "gpt-4",
				Delta:   "Hello",
				Created: now,
			},
			model:      "gpt-4",
			responseID: "chatcmpl-123",
			wantDelta:  "Hello",
		},
		{
			name: "final chunk with finish reason",
			chunk: &providers.StreamChunk{
				ID:           "chunk-123",
				Model:        "gpt-4",
				Delta:        "",
				FinishReason: "stop",
				Usage: &providers.TokenUsage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
				Created: now,
			},
			model:      "gpt-4",
			responseID: "chatcmpl-123",
			wantDelta:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStreamChunk(tt.chunk, tt.model, tt.responseID)

			if got == nil {
				t.Fatal("FormatStreamChunk() returned nil")
			}

			if got.ID != tt.responseID {
				t.Errorf("ID = %v, want %v", got.ID, tt.responseID)
			}

			if got.Model != tt.model {
				t.Errorf("Model = %v, want %v", got.Model, tt.model)
			}

			if len(got.Choices) == 0 {
				t.Fatal("Choices should not be empty")
			}

			if got.Choices[0].Delta.Content != tt.wantDelta {
				t.Errorf("Delta content = %v, want %v", got.Choices[0].Delta.Content, tt.wantDelta)
			}
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			statusCode: http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created response",
			statusCode: http.StatusCreated,
			data:       map[string]string{"id": "123"},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			err := WriteJSONResponse(w, tt.statusCode, tt.data)
			if err != nil {
				t.Errorf("WriteJSONResponse() error = %v", err)
			}

			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}

			// Verify JSON is valid
			var result map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("Response is not valid JSON: %v", err)
			}
		})
	}
}

func TestWriteErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		err        *types.ErrorResponse
		wantStatus int
	}{
		{
			name: "bad request error",
			err: &types.ErrorResponse{
				Error: types.ErrorDetail{
					Message: "Invalid request",
					Type:    "invalid_request_error",
					Code:    "invalid_request",
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unauthorized error",
			err: &types.ErrorResponse{
				Error: types.ErrorDetail{
					Message: "Invalid API key",
					Type:    "authentication_error",
					Code:    "invalid_api_key",
				},
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "rate limit error",
			err: &types.ErrorResponse{
				Error: types.ErrorDetail{
					Message: "Rate limit exceeded",
					Type:    types.ErrorTypeRateLimitExceeded,
					Code:    "rate_limit_exceeded",
				},
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "server error",
			err: &types.ErrorResponse{
				Error: types.ErrorDetail{
					Message: "Internal server error",
					Type:    "server_error",
					Code:    "internal_error",
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteErrorResponse(w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %v, want %v", w.Code, tt.wantStatus)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", contentType)
			}

			// Verify error response structure
			var errResp types.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Errorf("Response is not valid JSON: %v", err)
			}

			if errResp.Error.Message != tt.err.Error.Message {
				t.Errorf("Error message = %v, want %v", errResp.Error.Message, tt.err.Error.Message)
			}
		})
	}
}

func TestSetSSEHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	SetSSEHeaders(w)

	expectedHeaders := map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"Transfer-Encoding": "chunked",
	}

	for key, want := range expectedHeaders {
		got := w.Header().Get(key)
		if got != want {
			t.Errorf("Header %s = %v, want %v", key, got, want)
		}
	}
}

func TestWriteSSEChunk(t *testing.T) {
	chunk := &types.ChatCompletionStreamChunk{
		ID:      "chatcmpl-123",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []types.StreamChoice{
			{
				Index: 0,
				Delta: types.Delta{
					Content: "Hello",
				},
			},
		},
	}

	w := httptest.NewRecorder()
	err := WriteSSEChunk(w, chunk)
	if err != nil {
		t.Errorf("WriteSSEChunk() error = %v", err)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("SSE chunk body should not be empty")
	}

	// Should start with "data: "
	if len(body) < 6 || body[:6] != "data: " {
		t.Errorf("SSE chunk should start with 'data: ', got: %s", body[:min(20, len(body))])
	}
}

func TestWriteSSEDone(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteSSEDone(w)
	if err != nil {
		t.Errorf("WriteSSEDone() error = %v", err)
	}

	body := w.Body.String()
	expected := "data: [DONE]\n\n"
	if body != expected {
		t.Errorf("WriteSSEDone() body = %v, want %v", body, expected)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
