package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// MockServer is a mock HTTP server for testing provider adapters.
// It simulates various provider API responses including errors, streaming, etc.
type MockServer struct {
	server       *httptest.Server
	responses    map[string]MockResponse
	requestCount int
	mu           sync.Mutex
}

// MockResponse defines a mock response configuration.
type MockResponse struct {
	StatusCode   int
	Body         interface{}
	Delay        time.Duration
	Headers      map[string]string
	StreamChunks []string // For streaming responses
}

// NewMockServer creates a new mock server.
func NewMockServer() *MockServer {
	ms := &MockServer{
		responses: make(map[string]MockResponse),
	}

	// Create HTTP server
	ms.server = httptest.NewServer(http.HandlerFunc(ms.handler))

	return ms
}

// URL returns the mock server's base URL.
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Close closes the mock server.
func (ms *MockServer) Close() {
	ms.server.Close()
}

// SetResponse sets a mock response for a specific endpoint.
func (ms *MockServer) SetResponse(path string, response MockResponse) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.responses[path] = response
}

// GetRequestCount returns the number of requests received.
func (ms *MockServer) GetRequestCount() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.requestCount
}

// ResetRequestCount resets the request counter.
func (ms *MockServer) ResetRequestCount() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.requestCount = 0
}

// handler handles incoming HTTP requests.
func (ms *MockServer) handler(w http.ResponseWriter, r *http.Request) {
	ms.mu.Lock()
	ms.requestCount++
	ms.mu.Unlock()

	// Find matching response
	ms.mu.Lock()
	response, ok := ms.responses[r.URL.Path]
	ms.mu.Unlock()

	if !ok {
		// Default 404 response
		http.NotFound(w, r)
		return
	}

	// Apply delay if specified
	if response.Delay > 0 {
		time.Sleep(response.Delay)
	}

	// Set headers
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	// Handle streaming responses
	if len(response.StreamChunks) > 0 {
		ms.handleStream(w, r, response)
		return
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Write response body
	if response.Body != nil {
		switch v := response.Body.(type) {
		case string:
			_, _ = w.Write([]byte(v)) // Write to response, ignore error
		case []byte:
			_, _ = w.Write(v) // Write to response, ignore error
		default:
			_ = json.NewEncoder(w).Encode(response.Body) // Write to response, ignore error
		}
	}
}

// handleStream handles Server-Sent Events streaming responses.
func (ms *MockServer) handleStream(w http.ResponseWriter, r *http.Request, response MockResponse) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send chunks
	for _, chunk := range response.StreamChunks {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond) // Small delay between chunks
	}

	// Send final [DONE] message
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// MockOpenAIResponse creates a mock OpenAI chat completion response.
func MockOpenAIResponse(content string, model string) map[string]interface{} {
	return map[string]interface{}{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
		},
	}
}

// MockOpenAIStreamChunk creates a mock OpenAI streaming chunk.
func MockOpenAIStreamChunk(delta string, finishReason string) string {
	chunk := map[string]interface{}{
		"id":      "chatcmpl-123",
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "gpt-4",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"content": delta,
				},
				"finish_reason": finishReason,
			},
		},
	}

	bytes, _ := json.Marshal(chunk)
	return string(bytes)
}

// MockAnthropicResponse creates a mock Anthropic messages response.
func MockAnthropicResponse(content string, model string) map[string]interface{} {
	return map[string]interface{}{
		"id":   "msg_123",
		"type": "message",
		"role": "assistant",
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": content,
			},
		},
		"model":       model,
		"stop_reason": "end_turn",
		"usage": map[string]interface{}{
			"input_tokens":  10,
			"output_tokens": 20,
		},
	}
}

// MockAnthropicStreamEvent creates a mock Anthropic stream event.
func MockAnthropicStreamEvent(eventType string, data interface{}) string {
	var eventData string

	if data != nil {
		bytes, _ := json.Marshal(data)
		eventData = string(bytes)
	}

	return fmt.Sprintf("event: %s\ndata: %s\n", eventType, eventData)
}

// MockAnthropicContentBlockDelta creates a content block delta event.
func MockAnthropicContentBlockDelta(text string) string {
	data := map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": text,
		},
	}

	bytes, _ := json.Marshal(data)
	return string(bytes)
}

// MockErrorResponse creates a mock error response.
func MockErrorResponse(statusCode int, message string) MockResponse {
	body := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "invalid_request_error",
			"code":    statusCode,
		},
	}

	return MockResponse{
		StatusCode: statusCode,
		Body:       body,
	}
}

// MockAuthError creates a 401 authentication error response.
func MockAuthError() MockResponse {
	return MockErrorResponse(http.StatusUnauthorized, "Invalid API key")
}

// MockRateLimitError creates a 429 rate limit error response.
func MockRateLimitError(retryAfter int) MockResponse {
	response := MockErrorResponse(http.StatusTooManyRequests, "Rate limit exceeded")
	response.Headers = map[string]string{
		"Retry-After": fmt.Sprintf("%d", retryAfter),
	}
	return response
}

// MockTimeoutError creates a slow response to simulate timeout.
func MockTimeoutError(delay time.Duration) MockResponse {
	return MockResponse{
		StatusCode: http.StatusOK,
		Body:       MockOpenAIResponse("timeout", "gpt-4"),
		Delay:      delay,
	}
}

// MockServerError creates a 500 internal server error response.
func MockServerError() MockResponse {
	return MockErrorResponse(http.StatusInternalServerError, "Internal server error")
}

// Helper functions for testing

// ExpectJSONRequest is a helper to verify JSON request bodies.
func ExpectJSONRequest(r *http.Request, expected interface{}) error {
	var actual interface{}
	if err := json.NewDecoder(r.Body).Decode(&actual); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	// Compare as JSON strings for simplicity
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	if string(expectedJSON) != string(actualJSON) {
		return fmt.Errorf("request mismatch:\nexpected: %s\nactual: %s",
			string(expectedJSON), string(actualJSON))
	}

	return nil
}

// ExpectHeader checks if a request has a specific header value.
func ExpectHeader(r *http.Request, key, value string) error {
	actual := r.Header.Get(key)
	if !strings.Contains(actual, value) {
		return fmt.Errorf("header %q mismatch: expected %q, got %q", key, value, actual)
	}
	return nil
}
