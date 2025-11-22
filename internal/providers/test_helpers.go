package providers

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/providers"
)

// TestConfig returns a test provider configuration.
func TestConfig(name, providerType string) providers.ProviderConfig {
	return providers.ProviderConfig{
		Name:                name,
		Type:                providerType,
		BaseURL:             "http://localhost:8080",
		APIKey:              "test-key",
		Timeout:             5 * time.Second,
		MaxRetries:          2,
		HealthCheckInterval: 1 * time.Second,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}
}

// TestConfigWithURL returns a test config with a specific base URL.
func TestConfigWithURL(name, providerType, baseURL string) providers.ProviderConfig {
	config := TestConfig(name, providerType)
	config.BaseURL = baseURL
	return config
}

// TestMessage creates a test message.
func TestMessage(role, content string) providers.Message {
	return providers.Message{
		Role:    role,
		Content: content,
	}
}

// TestCompletionRequest creates a test completion request.
func TestCompletionRequest(model string, messages ...providers.Message) *providers.CompletionRequest {
	return &providers.CompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   100,
	}
}

// TestStreamingRequest creates a test streaming request.
func TestStreamingRequest(model string, messages ...providers.Message) *providers.CompletionRequest {
	req := TestCompletionRequest(model, messages...)
	req.Stream = true
	return req
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AssertErrorType fails the test if err is not of the expected type.
func AssertErrorType(t *testing.T, err error, expectedType interface{}) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	switch expectedType.(type) {
	case *providers.AuthError:
		if _, ok := err.(*providers.AuthError); !ok {
			t.Fatalf("expected AuthError, got %T: %v", err, err)
		}
	case *providers.RateLimitError:
		if _, ok := err.(*providers.RateLimitError); !ok {
			t.Fatalf("expected RateLimitError, got %T: %v", err, err)
		}
	case *providers.TimeoutError:
		if _, ok := err.(*providers.TimeoutError); !ok {
			t.Fatalf("expected TimeoutError, got %T: %v", err, err)
		}
	case *providers.ProviderError:
		if _, ok := err.(*providers.ProviderError); !ok {
			t.Fatalf("expected ProviderError, got %T: %v", err, err)
		}
	case *providers.ParseError:
		if _, ok := err.(*providers.ParseError); !ok {
			t.Fatalf("expected ParseError, got %T: %v", err, err)
		}
	case *providers.ValidationError:
		if _, ok := err.(*providers.ValidationError); !ok {
			t.Fatalf("expected ValidationError, got %T: %v", err, err)
		}
	default:
		t.Fatalf("unknown error type: %T", expectedType)
	}
}

// AssertEqual fails the test if got != expected.
func AssertEqual(t *testing.T, got, expected interface{}) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

// AssertNotEqual fails the test if got == unexpected.
func AssertNotEqual(t *testing.T, got, unexpected interface{}) {
	t.Helper()
	if got == unexpected {
		t.Fatalf("expected not %v, got %v", unexpected, got)
	}
}

// AssertTrue fails the test if condition is false.
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Fatalf("assertion failed: %s", message)
	}
}

// AssertFalse fails the test if condition is true.
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Fatalf("assertion failed: %s", message)
	}
}

// AssertContains fails the test if haystack doesn't contain needle.
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if haystack == "" {
		t.Fatal("haystack is empty")
	}
	if needle == "" {
		t.Fatal("needle is empty")
	}
	// Simple substring check
	found := false
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}

// WithTimeout runs a function with a timeout context.
func WithTimeout(t *testing.T, timeout time.Duration, fn func(ctx context.Context)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		fn(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-ctx.Done():
		t.Fatalf("test timeout after %s", timeout)
	}
}

// CollectStreamChunks collects all chunks from a stream channel.
func CollectStreamChunks(t *testing.T, chunks <-chan *providers.StreamChunk) ([]*providers.StreamChunk, error) {
	t.Helper()

	var collected []*providers.StreamChunk
	for chunk := range chunks {
		if chunk.Error != nil {
			return collected, chunk.Error
		}
		collected = append(collected, chunk)
	}

	return collected, nil
}

// ConcatenateChunks concatenates the delta content from all chunks.
func ConcatenateChunks(chunks []*providers.StreamChunk) string {
	var result string
	for _, chunk := range chunks {
		result += chunk.Delta
	}
	return result
}

// WaitForCondition waits for a condition to become true within a timeout.
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %s: %s", timeout, message)
		}

		<-ticker.C
	}
}
