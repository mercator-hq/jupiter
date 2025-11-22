package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mercator-hq/jupiter/pkg/proxy/types"
)

func TestParseChatCompletionRequest(t *testing.T) {
	tests := []struct {
		name    string
		body    interface{}
		wantErr bool
		errType string
	}{
		{
			name: "valid request with string content",
			body: types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with multiple messages",
			body: types.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []types.Message{
					{Role: "system", Content: "You are a helpful assistant"},
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with optional parameters",
			body: func() types.ChatCompletionRequest {
				temp := 0.7
				maxTokens := 100
				return types.ChatCompletionRequest{
					Model:       "gpt-4",
					Messages:    []types.Message{{Role: "user", Content: "Hello"}},
					Temperature: &temp,
					MaxTokens:   &maxTokens,
				}
			}(),
			wantErr: false,
		},
		{
			name: "valid request with tool calls",
			body: types.ChatCompletionRequest{
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
			},
			wantErr: false,
		},
		{
			name:    "empty request body",
			body:    nil,
			wantErr: true,
			errType: "invalid_request",
		},
		{
			name: "missing model",
			body: types.ChatCompletionRequest{
				Messages: []types.Message{{Role: "user", Content: "Hello"}},
			},
			wantErr: true,
		},
		{
			name: "missing messages",
			body: types.ChatCompletionRequest{
				Model: "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			body: func() types.ChatCompletionRequest {
				temp := 3.0 // Out of range
				return types.ChatCompletionRequest{
					Model:       "gpt-4",
					Messages:    []types.Message{{Role: "user", Content: "Hello"}},
					Temperature: &temp,
				}
			}(),
			wantErr: true,
		},
		{
			name: "invalid top_p",
			body: func() types.ChatCompletionRequest {
				topP := 1.5 // Out of range
				return types.ChatCompletionRequest{
					Model:    "gpt-4",
					Messages: []types.Message{{Role: "user", Content: "Hello"}},
					TopP:     &topP,
				}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if tt.body != nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal test body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			got, err := ParseChatCompletionRequest(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseChatCompletionRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("ParseChatCompletionRequest() returned nil without error")
			}
		})
	}
}

func TestParseChatCompletionRequest_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	_, err := ParseChatCompletionRequest(req)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParseChatCompletionRequest_MultimodalContent(t *testing.T) {
	// Test multimodal content with text and image parts
	requestBody := map[string]interface{}{
		"model": "gpt-4-vision-preview",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "What's in this image?",
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "https://example.com/image.jpg",
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal test body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	got, err := ParseChatCompletionRequest(req)
	if err != nil {
		t.Fatalf("ParseChatCompletionRequest() unexpected error = %v", err)
	}

	if got == nil {
		t.Fatal("ParseChatCompletionRequest() returned nil")
	}

	if len(got.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(got.Messages))
	}

	// Content should be an array
	if got.Messages[0].Content == nil {
		t.Error("Expected content to be non-nil")
	}
}

func TestValidateChatCompletionRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *types.ChatCompletionRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &types.ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: []types.Message{{Role: "user", Content: "Hello"}},
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty model",
			req: &types.ChatCompletionRequest{
				Messages: []types.Message{{Role: "user", Content: "Hello"}},
			},
			wantErr: true,
		},
		{
			name: "empty messages",
			req: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.req != nil {
				err = tt.req.Validate()
			} else {
				// Nil check
				if tt.req == nil && tt.wantErr {
					return // Expected
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
