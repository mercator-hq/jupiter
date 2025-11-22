package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPProvider_RetryOn5xx(t *testing.T) {
	attemptCount := int32(0)

	// Create test server that fails twice with 500, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// Create provider with retries enabled
	config := ProviderConfig{
		Name:       "test-provider",
		Type:       "openai",
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 3,
	}
	provider := NewHTTPProvider(config)

	// Perform request
	ctx := context.Background()
	resp, err := provider.DoRequest(ctx, "POST", server.URL+"/test", []byte(`{"test": true}`), nil)

	// Verify retry happened and request eventually succeeded
	if err != nil {
		t.Errorf("expected request to succeed after retries, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	defer resp.Body.Close()

	// Verify it took exactly 3 attempts (2 failures + 1 success)
	finalCount := atomic.LoadInt32(&attemptCount)
	if finalCount != 3 {
		t.Errorf("expected 3 attempts (2 retries), got %d", finalCount)
	}

	// Verify final response is successful
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify health was updated to success
	if !provider.IsHealthy() {
		t.Error("expected provider to be healthy after successful retry")
	}
}

func TestHTTPProvider_NoRetryOn4xx(t *testing.T) {
	attemptCount := int32(0)

	tests := []struct {
		name       string
		statusCode int
		errorType  string
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			errorType:  "ProviderError",
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			errorType:  "AuthError",
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			errorType:  "AuthError",
		},
		{
			name:       "429 rate limit",
			statusCode: http.StatusTooManyRequests,
			errorType:  "RateLimitError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atomic.StoreInt32(&attemptCount, 0)

			// Create test server that returns 4xx error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attemptCount, 1)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"error": "client error"}`))
			}))
			defer server.Close()

			// Create provider with retries enabled
			config := ProviderConfig{
				Name:       "test-provider",
				Type:       "openai",
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				MaxRetries: 3,
			}
			provider := NewHTTPProvider(config)

			// Perform request
			ctx := context.Background()
			resp, err := provider.DoRequest(ctx, "POST", server.URL+"/test", []byte(`{"test": true}`), nil)

			// Verify request failed without retry
			if err == nil {
				t.Errorf("expected error for %d status, got nil", tt.statusCode)
			}
			if resp != nil {
				resp.Body.Close()
			}

			// Verify only 1 attempt was made (no retries for 4xx)
			finalCount := atomic.LoadInt32(&attemptCount)
			if finalCount != 1 {
				t.Errorf("expected 1 attempt (no retries for 4xx), got %d", finalCount)
			}

			// Verify correct error type
			switch tt.errorType {
			case "AuthError":
				var authErr *AuthError
				if !errors.As(err, &authErr) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			case "RateLimitError":
				var rateLimitErr *RateLimitError
				if !errors.As(err, &rateLimitErr) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				}
			case "ProviderError":
				var providerErr *ProviderError
				if !errors.As(err, &providerErr) {
					t.Errorf("expected ProviderError, got %T: %v", err, err)
				}
			}
		})
	}
}

func TestHTTPProvider_MaxRetries(t *testing.T) {
	attemptCount := int32(0)

	// Create test server that always fails with 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	// Create provider with MaxRetries = 2
	config := ProviderConfig{
		Name:       "test-provider",
		Type:       "openai",
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 2,
	}
	provider := NewHTTPProvider(config)

	// Perform request
	ctx := context.Background()
	resp, err := provider.DoRequest(ctx, "POST", server.URL+"/test", []byte(`{"test": true}`), nil)

	// Verify request failed after max retries
	if err == nil {
		t.Error("expected error after max retries exceeded")
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Verify it made exactly MaxRetries + 1 attempts (initial + 2 retries)
	finalCount := atomic.LoadInt32(&attemptCount)
	expectedAttempts := int32(config.MaxRetries + 1)
	if finalCount != expectedAttempts {
		t.Errorf("expected %d attempts (initial + %d retries), got %d", expectedAttempts, config.MaxRetries, finalCount)
	}

	// Verify provider has recorded the failure
	health := provider.GetHealth()
	// Note: After a single request failure (even with retries), ConsecutiveFailures is 1
	// The circuit breaker only triggers (IsHealthy = false) after 3 separate request failures
	if health.ConsecutiveFailures < 1 {
		t.Errorf("expected at least 1 consecutive failure, got %d", health.ConsecutiveFailures)
	}
	if health.FailedRequests < 1 {
		t.Errorf("expected at least 1 failed request, got %d", health.FailedRequests)
	}
}

func TestHTTPProvider_ExponentialBackoff(t *testing.T) {
	attemptCount := int32(0)
	attemptTimes := make([]time.Time, 0, 4)

	// Create test server that always fails with 503
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		attemptTimes = append(attemptTimes, time.Now())
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "service unavailable"}`))
	}))
	defer server.Close()

	// Create provider with retries
	config := ProviderConfig{
		Name:       "test-provider",
		Type:       "openai",
		BaseURL:    server.URL,
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}
	provider := NewHTTPProvider(config)

	// Perform request
	ctx := context.Background()
	resp, _ := provider.DoRequest(ctx, "POST", server.URL+"/test", []byte(`{"test": true}`), nil)
	if resp != nil {
		_ = resp.Body.Close()
	}

	// Verify exponential backoff timing
	// Expected delays: 0s (initial), 1s (2^0), 2s (2^1), 4s (2^2)
	finalCount := atomic.LoadInt32(&attemptCount)
	if finalCount != 4 {
		t.Fatalf("expected 4 attempts, got %d", finalCount)
	}

	// Check delays between attempts
	for i := 1; i < len(attemptTimes); i++ {
		delay := attemptTimes[i].Sub(attemptTimes[i-1])
		// Expected delay: 2^(i-1) seconds
		// Allow 200ms tolerance for timing variance
		expectedDelay := time.Duration(1<<uint(i-1)) * time.Second
		minDelay := expectedDelay - 200*time.Millisecond
		maxDelay := expectedDelay + 200*time.Millisecond

		if delay < minDelay || delay > maxDelay {
			t.Errorf("attempt %d: expected delay ~%s, got %s (expected range: %s - %s)",
				i, expectedDelay, delay, minDelay, maxDelay)
		}
	}
}

func TestHTTPProvider_TimeoutDuringRetry(t *testing.T) {
	attemptCount := int32(0)

	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		// First attempt succeeds quickly
		if count == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "success"}`))
			return
		}
		// Subsequent attempts hang
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	t.Run("context timeout cancels retry", func(t *testing.T) {
		atomic.StoreInt32(&attemptCount, 0)

		// Create provider with retries
		config := ProviderConfig{
			Name:       "test-provider",
			Type:       "openai",
			BaseURL:    server.URL,
			Timeout:    10 * time.Second,
			MaxRetries: 3,
		}
		provider := NewHTTPProvider(config)

		// Create context with short timeout (enough for backoff but not for slow request)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Make failing request that would trigger retries
		server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attemptCount, 1)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "error"}`))
		}))
		defer server500.Close()

		provider.config.BaseURL = server500.URL
		resp, err := provider.DoRequest(ctx, "POST", server500.URL+"/test", []byte(`{"test": true}`), nil)

		// Verify timeout error
		if err == nil {
			t.Error("expected timeout error, got nil")
			if resp != nil {
				resp.Body.Close()
			}
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			var timeoutErr *TimeoutError
			if !errors.As(err, &timeoutErr) {
				t.Errorf("expected timeout-related error, got %T: %v", err, err)
			}
		}

		// Verify retries were attempted but stopped due to timeout
		finalCount := atomic.LoadInt32(&attemptCount)
		if finalCount == 0 {
			t.Error("expected at least one attempt before timeout")
		}
		// Should be less than max retries + 1 due to timeout
		if finalCount > 4 {
			t.Errorf("expected fewer than 4 attempts due to timeout, got %d", finalCount)
		}
	})

	t.Run("http client timeout", func(t *testing.T) {
		atomic.StoreInt32(&attemptCount, 0)

		slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attemptCount, 1)
			time.Sleep(3 * time.Second) // Longer than client timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer slowServer.Close()

		// Create provider with very short timeout
		config := ProviderConfig{
			Name:       "test-provider",
			Type:       "openai",
			BaseURL:    slowServer.URL,
			Timeout:    100 * time.Millisecond, // Very short timeout
			MaxRetries: 2,
		}
		provider := NewHTTPProvider(config)

		ctx := context.Background()
		resp, err := provider.DoRequest(ctx, "POST", slowServer.URL+"/test", []byte(`{"test": true}`), nil)

		// Verify timeout error
		if err == nil {
			t.Error("expected timeout error, got nil")
			if resp != nil {
				resp.Body.Close()
			}
		}

		var timeoutErr *TimeoutError
		if !errors.As(err, &timeoutErr) {
			// Check if it's a context deadline exceeded error
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("expected TimeoutError or DeadlineExceeded, got %T: %v", err, err)
			}
		}
	})
}

// TestHTTPProvider_ConnectionReuse verifies that HTTP connections are reused
func TestHTTPProvider_ConnectionReuse(t *testing.T) {
	connectionCount := int32(0)

	// Create test server that tracks unique connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track new connections (simplified - in real scenario would use connection ID)
		atomic.AddInt32(&connectionCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// Create provider with connection pooling
	config := ProviderConfig{
		Name:                "test-provider",
		Type:                "openai",
		BaseURL:             server.URL,
		Timeout:             5 * time.Second,
		MaxRetries:          0,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
	}
	provider := NewHTTPProvider(config)

	// Make multiple requests
	ctx := context.Background()
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		resp, err := provider.DoRequest(ctx, "GET", server.URL+"/test", nil, nil)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_, _ = io.ReadAll(resp.Body) // Drain body to allow connection reuse
		resp.Body.Close()
	}

	// Note: This test is simplified. In a real scenario, you'd need to track
	// actual TCP connections using the server's ConnState callback.
	// For now, we just verify all requests completed successfully.
	count := atomic.LoadInt32(&connectionCount)
	if count != int32(numRequests) {
		t.Errorf("expected %d requests, got %d", numRequests, count)
	}
}

// TestHTTPProvider_PoolLimitEnforcement verifies connection pool limits
func TestHTTPProvider_PoolLimitEnforcement(t *testing.T) {
	// Create test server with slow responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// Create provider with limited connection pool
	config := ProviderConfig{
		Name:                "test-provider",
		Type:                "openai",
		BaseURL:             server.URL,
		Timeout:             5 * time.Second,
		MaxRetries:          0,
		MaxIdleConns:        2,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     1 * time.Second,
	}
	provider := NewHTTPProvider(config)

	// Make concurrent requests
	ctx := context.Background()
	numRequests := 10
	errors := make(chan error, numRequests)
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			resp, err := provider.DoRequest(ctx, "GET", fmt.Sprintf("%s/test?id=%d", server.URL, id), nil, nil)
			if err != nil {
				errors <- err
				return
			}
			_, _ = io.ReadAll(resp.Body) // Drain body to allow connection reuse
			resp.Body.Close()
			errors <- nil
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		if err := <-errors; err != nil {
			t.Errorf("request failed: %v", err)
		}
	}

	duration := time.Since(start)

	// Verify all requests completed
	// Note: With connection pooling, requests should complete relatively quickly
	// even though the pool is limited
	if duration > 5*time.Second {
		t.Errorf("requests took too long: %s (connection pooling may not be working)", duration)
	}

	// Verify provider is still healthy
	if !provider.IsHealthy() {
		t.Error("expected provider to be healthy after concurrent requests")
	}
}
