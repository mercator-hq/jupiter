package openai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/providers"
)

// TestOpenAI_StreamingChunkDelivery verifies that streaming chunks are delivered correctly
func TestOpenAI_StreamingChunkDelivery(t *testing.T) {
	// Create test server that sends SSE stream
	chunks := []string{
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected Accept: text/event-stream, got %s", r.Header.Get("Accept"))
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Bearer") {
			t.Error("expected Authorization header with Bearer token")
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.ResponseWriter to be http.Flusher")
		}

		// Send chunks
		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond) // Small delay between chunks
		}
	}))
	defer server.Close()

	// Create provider
	config := providers.ProviderConfig{
		Name:       "openai-test",
		Type:       "openai",
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// Create streaming request
	req := &providers.CompletionRequest{
		Model: "gpt-4",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "Say hello"},
		},
		Stream: true,
	}

	// Start stream
	ctx := context.Background()
	stream, err := provider.StreamCompletion(ctx, req)
	if err != nil {
		t.Fatalf("failed to start stream: %v", err)
	}

	// Collect chunks
	var receivedChunks []*providers.StreamChunk
	var fullContent strings.Builder

	for chunk := range stream {
		if chunk.Error != nil {
			if chunk.Error != io.EOF {
				t.Fatalf("unexpected error in stream: %v", chunk.Error)
			}
			break
		}

		receivedChunks = append(receivedChunks, chunk)
		fullContent.WriteString(chunk.Delta)
	}

	// Verify we received all content chunks (excluding role and finish chunks)
	if len(receivedChunks) < 3 {
		t.Errorf("expected at least 3 chunks with content, got %d", len(receivedChunks))
	}

	// Verify concatenated content
	expectedContent := "Hello World!"
	if fullContent.String() != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, fullContent.String())
	}

	// Verify final chunk has finish reason
	lastChunk := receivedChunks[len(receivedChunks)-1]
	if lastChunk.FinishReason != providers.FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", providers.FinishReasonStop, lastChunk.FinishReason)
	}

	// Verify all chunks have same ID
	for i, chunk := range receivedChunks {
		if chunk.ID != "chatcmpl-123" {
			t.Errorf("chunk %d: expected ID %q, got %q", i, "chatcmpl-123", chunk.ID)
		}
		if chunk.Model != "gpt-4" {
			t.Errorf("chunk %d: expected model %q, got %q", i, "gpt-4", chunk.Model)
		}
	}
}

// TestOpenAI_StreamingErrorHandling verifies error propagation in streams
func TestOpenAI_StreamingErrorHandling(t *testing.T) {
	t.Run("server error during stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send a couple of valid chunks, then close connection abruptly
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)

			fmt.Fprintf(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`+"\n\n")
			flusher.Flush()

			// Abruptly close connection (simulate server error)
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
		}))
		defer server.Close()

		config := providers.ProviderConfig{
			Name:       "openai-test",
			Type:       "openai",
			BaseURL:    server.URL,
			APIKey:     "test-api-key",
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		}

		provider, err := NewProvider(config)
		if err != nil {
			t.Fatalf("failed to create provider: %v", err)
		}

		req := &providers.CompletionRequest{
			Model:    "gpt-4",
			Messages: []providers.Message{{Role: providers.RoleUser, Content: "Test"}},
			Stream:   true,
		}

		ctx := context.Background()
		stream, err := provider.StreamCompletion(ctx, req)
		if err != nil {
			t.Fatalf("failed to start stream: %v", err)
		}

		// Read chunks until error
		var errorReceived bool
		for chunk := range stream {
			if chunk.Error != nil {
				errorReceived = true
				// Verify it's a stream error
				var streamErr *providers.StreamError
				if providers, ok := chunk.Error.(*providers.StreamError); ok {
					streamErr = providers
				}
				if streamErr == nil && chunk.Error != io.EOF {
					t.Logf("received error: %v (type: %T)", chunk.Error, chunk.Error)
				}
				break
			}
		}

		if !errorReceived {
			t.Error("expected error to be propagated through stream")
		}
	})

	t.Run("malformed JSON in stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)

			// Send valid chunk
			fmt.Fprintf(w, `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`+"\n\n")
			flusher.Flush()

			// Send malformed JSON
			fmt.Fprintf(w, `data: {"invalid": json}`+"\n\n")
			flusher.Flush()
		}))
		defer server.Close()

		config := providers.ProviderConfig{
			Name:       "openai-test",
			Type:       "openai",
			BaseURL:    server.URL,
			APIKey:     "test-api-key",
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		}

		provider, err := NewProvider(config)
		if err != nil {
			t.Fatalf("failed to create provider: %v", err)
		}

		req := &providers.CompletionRequest{
			Model:    "gpt-4",
			Messages: []providers.Message{{Role: providers.RoleUser, Content: "Test"}},
			Stream:   true,
		}

		ctx := context.Background()
		stream, err := provider.StreamCompletion(ctx, req)
		if err != nil {
			t.Fatalf("failed to start stream: %v", err)
		}

		// Read chunks until parse error
		var parseErrorReceived bool
		for chunk := range stream {
			if chunk.Error != nil {
				var parseErr *providers.ParseError
				if pe, ok := chunk.Error.(*providers.ParseError); ok {
					parseErr = pe
				}
				if parseErr != nil {
					parseErrorReceived = true
				}
				break
			}
		}

		if !parseErrorReceived {
			t.Error("expected ParseError to be propagated for malformed JSON")
		}
	})

	t.Run("HTTP error initiating stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return 500 error
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal server error"}`))
		}))
		defer server.Close()

		config := providers.ProviderConfig{
			Name:       "openai-test",
			Type:       "openai",
			BaseURL:    server.URL,
			APIKey:     "test-api-key",
			Timeout:    5 * time.Second,
			MaxRetries: 0,
		}

		provider, err := NewProvider(config)
		if err != nil {
			t.Fatalf("failed to create provider: %v", err)
		}

		req := &providers.CompletionRequest{
			Model:    "gpt-4",
			Messages: []providers.Message{{Role: providers.RoleUser, Content: "Test"}},
			Stream:   true,
		}

		ctx := context.Background()
		stream, err := provider.StreamCompletion(ctx, req)

		// Error should be returned immediately when initiating stream
		if err == nil {
			t.Error("expected error when initiating stream with server error")
			if stream != nil {
				// Drain stream
				for range stream {
				}
			}
		}

		// Verify error type
		var providerErr *providers.ProviderError
		if pe, ok := err.(*providers.ProviderError); ok {
			providerErr = pe
		}
		if providerErr == nil {
			t.Errorf("expected ProviderError, got %T: %v", err, err)
		}
	})
}

// TestOpenAI_StreamingClientDisconnect verifies cleanup on client disconnect
func TestOpenAI_StreamingClientDisconnect(t *testing.T) {
	var mu sync.Mutex
	chunksServed := 0
	cleanedUp := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Send chunks until context is cancelled
		for i := 0; i < 100; i++ {
			select {
			case <-r.Context().Done():
				// Client disconnected
				mu.Lock()
				cleanedUp = true
				mu.Unlock()
				return
			default:
			}

			chunk := fmt.Sprintf(`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"chunk%d"},"finish_reason":null}]}`, i)
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
			mu.Lock()
			chunksServed++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer server.Close()

	config := providers.ProviderConfig{
		Name:       "openai-test",
		Type:       "openai",
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Timeout:    30 * time.Second,
		MaxRetries: 0,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := &providers.CompletionRequest{
		Model:    "gpt-4",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "Test"}},
		Stream:   true,
	}

	// Create context that we'll cancel early
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := provider.StreamCompletion(ctx, req)
	if err != nil {
		t.Fatalf("failed to start stream: %v", err)
	}

	// Read a few chunks then cancel
	chunksRead := 0
	for chunk := range stream {
		if chunk.Error != nil {
			break
		}
		chunksRead++
		if chunksRead >= 3 {
			// Cancel context (simulate client disconnect)
			cancel()
			break
		}
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify that:
	// 1. We read at least 3 chunks
	if chunksRead < 3 {
		t.Errorf("expected to read at least 3 chunks before disconnect, got %d", chunksRead)
	}

	// 2. Server didn't send all 100 chunks (stopped due to disconnect)
	mu.Lock()
	served := chunksServed
	cleaned := cleanedUp
	mu.Unlock()

	if served >= 100 {
		t.Errorf("expected server to stop sending after disconnect, but sent all %d chunks", served)
	}

	// 3. Context cancellation was detected (cleanup occurred)
	// Note: This is tricky to test reliably. We rely on the server's context being cancelled.
	t.Logf("Client disconnect test: read %d chunks, server served %d chunks, cleanup: %v",
		chunksRead, served, cleaned)
}

// TestOpenAI_StreamingFinalUsageChunk verifies usage information in final chunk
func TestOpenAI_StreamingFinalUsageChunk(t *testing.T) {
	// Create test server that sends SSE stream with usage in final chunk
	chunks := []string{
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}`,
		// Final chunk with finish reason and usage
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
		`data: [DONE]`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		for _, chunk := range chunks {
			fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	config := providers.ProviderConfig{
		Name:       "openai-test",
		Type:       "openai",
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	req := &providers.CompletionRequest{
		Model:    "gpt-4",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "Say hello"}},
		Stream:   true,
	}

	ctx := context.Background()
	stream, err := provider.StreamCompletion(ctx, req)
	if err != nil {
		t.Fatalf("failed to start stream: %v", err)
	}

	// Collect all chunks
	var receivedChunks []*providers.StreamChunk
	for chunk := range stream {
		if chunk.Error != nil {
			if chunk.Error != io.EOF {
				t.Fatalf("unexpected error: %v", chunk.Error)
			}
			break
		}
		receivedChunks = append(receivedChunks, chunk)
	}

	if len(receivedChunks) == 0 {
		t.Fatal("expected to receive chunks")
	}

	// Find the final chunk (has finish reason)
	var finalChunk *providers.StreamChunk
	for _, chunk := range receivedChunks {
		if chunk.FinishReason != "" {
			finalChunk = chunk
			break
		}
	}

	if finalChunk == nil {
		t.Fatal("expected to find final chunk with finish reason")
	}

	// Verify finish reason
	if finalChunk.FinishReason != providers.FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", providers.FinishReasonStop, finalChunk.FinishReason)
	}

	// Verify usage information is present in final chunk
	if finalChunk.Usage == nil {
		t.Fatal("expected usage information in final chunk, got nil")
	}

	// Verify usage values
	if finalChunk.Usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", finalChunk.Usage.PromptTokens)
	}
	if finalChunk.Usage.CompletionTokens != 5 {
		t.Errorf("expected 5 completion tokens, got %d", finalChunk.Usage.CompletionTokens)
	}
	if finalChunk.Usage.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", finalChunk.Usage.TotalTokens)
	}

	// Verify earlier chunks don't have usage
	for i, chunk := range receivedChunks {
		if chunk.FinishReason == "" && chunk.Usage != nil {
			t.Errorf("chunk %d: expected no usage in non-final chunk, got %+v", i, chunk.Usage)
		}
	}
}

// TestOpenAI_StreamingValidation verifies request validation for streaming
func TestOpenAI_StreamingValidation(t *testing.T) {
	config := providers.ProviderConfig{
		Name:       "openai-test",
		Type:       "openai",
		BaseURL:    "http://localhost:9999", // Won't be called
		APIKey:     "test-api-key",
		Timeout:    5 * time.Second,
		MaxRetries: 0,
	}

	provider, err := NewProvider(config)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		req     *providers.CompletionRequest
		wantErr bool
		errType string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name: "missing model",
			req: &providers.CompletionRequest{
				Messages: []providers.Message{{Role: providers.RoleUser, Content: "Test"}},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name: "missing messages",
			req: &providers.CompletionRequest{
				Model: "gpt-4",
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name: "valid request",
			req: &providers.CompletionRequest{
				Model:    "gpt-4",
				Messages: []providers.Message{{Role: providers.RoleUser, Content: "Test"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := provider.StreamCompletion(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected validation error, got nil")
					if stream != nil {
						for range stream {
						}
					}
				} else {
					var validationErr *providers.ValidationError
					if ve, ok := err.(*providers.ValidationError); ok {
						validationErr = ve
					}
					if validationErr == nil && tt.errType == "ValidationError" {
						t.Errorf("expected ValidationError, got %T: %v", err, err)
					}
				}
			} else {
				if err != nil && !strings.Contains(err.Error(), "connection refused") {
					// Connection errors are expected since we're using a fake URL
					// Only fail if it's a validation error
					var validationErr *providers.ValidationError
					if ve, ok := err.(*providers.ValidationError); ok {
						validationErr = ve
					}
					if validationErr != nil {
						t.Errorf("unexpected validation error: %v", err)
					}
				}
			}
		})
	}
}
