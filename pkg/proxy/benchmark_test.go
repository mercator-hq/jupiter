package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mercator-hq/jupiter/pkg/proxy/types"
)

func BenchmarkParseChatCompletionRequest(b *testing.B) {
	reqBody := types.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello, world!"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		_, err := ParseChatCompletionRequest(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteJSONResponse(b *testing.B) {
	response := map[string]interface{}{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   "gpt-4",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": "Hello! How can I help you today?",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 15,
			"total_tokens":      25,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		if err := WriteJSONResponse(w, http.StatusOK, response); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExtractRequestMetadata(b *testing.B) {
	reqBody := types.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []types.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		b.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test1234567890abcdef")
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-Request-ID", "req-456")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractRequestMetadata(req, &reqBody)
	}
}

func BenchmarkRedactAPIKey(b *testing.B) {
	apiKey := "sk-1234567890abcdefghijklmnopqrstuvwxyz"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RedactAPIKey(apiKey)
	}
}

func BenchmarkHandleError(b *testing.B) {
	err := &RequestError{
		Message: "Invalid request",
		Code:    "missing_field",
		Param:   "model",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HandleError(err)
	}
}
