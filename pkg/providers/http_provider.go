package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"
)

// HTTPProvider is the base implementation for HTTP-based provider adapters.
// It provides connection pooling, retry logic, timeout handling, and health monitoring.
//
// Concrete provider implementations (OpenAI, Anthropic, etc.) should embed this
// struct and implement the Provider interface methods.
type HTTPProvider struct {
	// config contains the provider configuration
	config ProviderConfig

	// client is the HTTP client with connection pooling
	client *http.Client

	// health tracks the provider's health status
	health ProviderHealth

	// healthMu protects concurrent access to health status
	healthMu sync.RWMutex

	// stopHealthCheck is closed to signal the health checker to stop
	stopHealthCheck chan struct{}

	// healthCheckStopped is closed when the health checker has stopped
	healthCheckStopped chan struct{}
}

// NewHTTPProvider creates a new base HTTP provider with connection pooling.
func NewHTTPProvider(config ProviderConfig) *HTTPProvider {
	// Create HTTP transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DisableCompression:  false,
		// Enable HTTP/2
		ForceAttemptHTTP2: true,
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	p := &HTTPProvider{
		config: config,
		client: client,
		health: ProviderHealth{
			IsHealthy:             true, // Start optimistic
			LastCheck:             time.Now(),
			ConsecutiveFailures:   0,
			LastSuccessfulRequest: time.Now(),
			TotalRequests:         0,
			FailedRequests:        0,
		},
		stopHealthCheck:    make(chan struct{}),
		healthCheckStopped: make(chan struct{}),
	}

	return p
}

// GetName returns the provider's configured name.
func (p *HTTPProvider) GetName() string {
	return p.config.Name
}

// GetType returns the provider's type.
func (p *HTTPProvider) GetType() string {
	return p.config.Type
}

// GetConfig returns the provider's configuration.
func (p *HTTPProvider) GetConfig() ProviderConfig {
	return p.config
}

// IsHealthy returns the current health status.
func (p *HTTPProvider) IsHealthy() bool {
	p.healthMu.RLock()
	defer p.healthMu.RUnlock()
	return p.health.IsHealthy
}

// GetHealth returns detailed health information.
func (p *HTTPProvider) GetHealth() ProviderHealth {
	p.healthMu.RLock()
	defer p.healthMu.RUnlock()
	return p.health
}

// updateHealth updates the provider's health status.
// This is called after each health check or request.
func (p *HTTPProvider) updateHealth(success bool, err error) {
	p.healthMu.Lock()
	defer p.healthMu.Unlock()

	p.health.LastCheck = time.Now()

	if success {
		p.health.IsHealthy = true
		p.health.ConsecutiveFailures = 0
		p.health.LastError = nil
		p.health.LastSuccessfulRequest = time.Now()
	} else {
		p.health.ConsecutiveFailures++
		p.health.LastError = err

		// Mark unhealthy after 3 consecutive failures (circuit breaker)
		if p.health.ConsecutiveFailures >= 3 {
			p.health.IsHealthy = false
			slog.Warn("provider marked unhealthy",
				"provider", p.config.Name,
				"consecutive_failures", p.health.ConsecutiveFailures,
				"error", err,
			)
		}
	}
}

// recordRequest records request metrics.
func (p *HTTPProvider) recordRequest(success bool) {
	p.healthMu.Lock()
	defer p.healthMu.Unlock()

	p.health.TotalRequests++
	if !success {
		p.health.FailedRequests++
	}
}

// DoRequest performs an HTTP request with retry logic and timeout handling.
// It automatically retries transient errors (5xx, timeouts) with exponential backoff.
func (p *HTTPProvider) DoRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) (*http.Response, error) {
	var lastErr error

	// Attempt request with retries
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			slog.Debug("retrying request",
				"provider", p.config.Name,
				"attempt", attempt,
				"max_retries", p.config.MaxRetries,
				"backoff", backoff,
			)

			// Wait with backoff (respect context cancellation)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Create request
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// Set default Content-Type if not provided
		if req.Header.Get("Content-Type") == "" && body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		// Perform request
		slog.Debug("sending request to provider",
			"provider", p.config.Name,
			"method", method,
			"url", url,
		)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			p.recordRequest(false)

			// Check if error is retryable
			if ctx.Err() != nil {
				// Context cancelled or timeout - don't retry
				return nil, &TimeoutError{
					Provider: p.config.Name,
					Timeout:  p.config.Timeout,
				}
			}

			// Network error - retry
			slog.Warn("request failed, will retry",
				"provider", p.config.Name,
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			p.recordRequest(true)
			p.updateHealth(true, nil)
			return resp, nil
		}

		// Read error response body
		errorBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check for specific error types
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			// Authentication error - don't retry
			p.recordRequest(false)
			p.updateHealth(false, fmt.Errorf("authentication failed"))
			return nil, &AuthError{
				Provider: p.config.Name,
				Message:  string(errorBody),
			}

		case http.StatusTooManyRequests:
			// Rate limit error - don't retry (caller should handle)
			p.recordRequest(false)
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			return nil, &RateLimitError{
				Provider:   p.config.Name,
				RetryAfter: retryAfter,
				Message:    string(errorBody),
			}

		case http.StatusBadRequest:
			// Bad request - don't retry
			p.recordRequest(false)
			return nil, &ProviderError{
				Provider:   p.config.Name,
				StatusCode: resp.StatusCode,
				Message:    string(errorBody),
			}

		default:
			// Server error (5xx) or other error - retry
			lastErr = &ProviderError{
				Provider:   p.config.Name,
				StatusCode: resp.StatusCode,
				Message:    string(errorBody),
			}
			p.recordRequest(false)

			slog.Warn("request returned error status, will retry",
				"provider", p.config.Name,
				"status", resp.StatusCode,
				"attempt", attempt+1,
			)
		}
	}

	// All retries exhausted
	p.updateHealth(false, lastErr)
	return nil, lastErr
}

// DoJSONRequest performs a JSON request and decodes the response.
func (p *HTTPProvider) DoJSONRequest(ctx context.Context, method, url string, reqBody interface{}, respBody interface{}, headers map[string]string) error {
	// Marshal request body
	var bodyBytes []byte
	var err error
	if reqBody != nil {
		bodyBytes, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// Perform request
	resp, err := p.DoRequest(ctx, method, url, bodyBytes, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ParseError{
			Provider: p.config.Name,
			Cause:    fmt.Errorf("failed to read response: %w", err),
		}
	}

	// Decode response
	if respBody != nil && len(responseBytes) > 0 {
		if err := json.Unmarshal(responseBytes, respBody); err != nil {
			return &ParseError{
				Provider:    p.config.Name,
				RawResponse: string(responseBytes),
				Cause:       fmt.Errorf("failed to unmarshal response: %w", err),
			}
		}
	}

	return nil
}

// Close closes the HTTP client and stops the health checker.
func (p *HTTPProvider) Close() error {
	// Signal health checker to stop
	close(p.stopHealthCheck)

	// Wait for health checker to stop (with timeout)
	select {
	case <-p.healthCheckStopped:
		slog.Debug("health checker stopped", "provider", p.config.Name)
	case <-time.After(5 * time.Second):
		slog.Warn("health checker did not stop in time", "provider", p.config.Name)
	}

	// Close idle connections
	p.client.CloseIdleConnections()

	slog.Info("provider closed", "provider", p.config.Name)
	return nil
}

// parseRetryAfter parses the Retry-After header value.
// It supports both delay-seconds and HTTP-date formats.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}

	// Try parsing as seconds
	var seconds int
	if _, err := fmt.Sscanf(header, "%d", &seconds); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(header); err == nil {
		return time.Until(t)
	}

	return 0
}
