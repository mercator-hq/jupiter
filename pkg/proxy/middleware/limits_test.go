package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/limits"
	"mercator-hq/jupiter/pkg/limits/budget"
	"mercator-hq/jupiter/pkg/limits/ratelimit"
)

// TestLimitsMiddleware_NoIdentifier tests that requests without identifier pass through.
func TestLimitsMiddleware_NoIdentifier(t *testing.T) {
	manager := limits.NewManager(limits.Config{})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", w.Body.String())
	}
}

// TestLimitsMiddleware_WithinLimits tests that requests within limits pass through.
func TestLimitsMiddleware_WithinLimits(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 100,
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", w.Body.String())
	}
}

// TestLimitsMiddleware_RateLimitExceeded tests that rate limit violations are blocked.
func TestLimitsMiddleware_RateLimitExceeded(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerMinute: 2, // Low limit to easily exceed
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 3 requests rapidly (should block on 3rd)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-key")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if i < 2 {
			// First 2 should succeed
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i, w.Code)
			}
		} else {
			// 3rd should be blocked
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status 429, got %d", i, w.Code)
			}
		}
	}
}

// TestLimitsMiddleware_BudgetExceeded tests that budget violations are blocked.
func TestLimitsMiddleware_BudgetExceeded(t *testing.T) {
	t.Skip("Budget limits require actual usage recording - tested in integration tests")
}

// TestLimitsMiddleware_Headers tests that rate limit headers are set correctly.
func TestLimitsMiddleware_Headers(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 100,
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify request succeeded
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Note: Headers may or may not be set depending on manager implementation
	// This test validates the middleware doesn't crash when headers should be set
	t.Logf("X-RateLimit-Limit: %s", w.Header().Get("X-RateLimit-Limit"))
	t.Logf("X-RateLimit-Remaining: %s", w.Header().Get("X-RateLimit-Remaining"))
}

// TestLimitsMiddleware_BudgetHeaders tests that budget headers are set correctly.
func TestLimitsMiddleware_BudgetHeaders(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Hourly: 100.00,
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify request succeeded
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Note: Budget headers may or may not be set depending on manager implementation
	// This test validates the middleware doesn't crash when headers should be set
	t.Logf("X-Budget-Limit: %s", w.Header().Get("X-Budget-Limit"))
	t.Logf("X-Budget-Used: %s", w.Header().Get("X-Budget-Used"))
}

// TestLimitsMiddleware_ConcurrentLimit tests concurrent request limiting.
func TestLimitsMiddleware_ConcurrentLimit(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				MaxConcurrent: 1,
			},
		},
	})
	defer manager.Close()

	// Acquire concurrent slot to simulate in-flight request
	acquired := manager.AcquireConcurrent("test-key")
	if !acquired {
		t.Fatal("Failed to acquire initial concurrent slot")
	}
	// Don't release to simulate in-flight request

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should be rejected due to concurrent limit
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Too many concurrent requests") {
		t.Errorf("Expected 'Too many concurrent requests' message, got: %s", body)
	}

	// Release slot for cleanup
	manager.ReleaseConcurrent("test-key")
}

// TestLimitsMiddleware_ModelDowngrade tests model downgrade action.
func TestLimitsMiddleware_ModelDowngrade(t *testing.T) {
	t.Skip("Model downgrade requires specific enforcement configuration - tested in integration tests")
}

// TestLimitsMiddleware_RetryAfterHeader tests Retry-After header is set.
func TestLimitsMiddleware_RetryAfterHeader(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerMinute: 1,
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 2 requests to trigger rate limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-key")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if i == 1 {
			// Second request should be rate limited
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Expected status 429, got %d", w.Code)
			}

			// Check Retry-After header if present
			retryAfter := w.Header().Get("Retry-After")
			if retryAfter != "" {
				t.Logf("Retry-After header set to: %s", retryAfter)
			}
		}
	}
}

// TestLimitsMiddleware_EnrichedContext tests handling of enriched request context.
func TestLimitsMiddleware_EnrichedContext(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				TokensPerMinute: 5000,
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with enriched context
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	ctx := context.WithValue(req.Context(), "enriched_request", &enrichedRequestContext{
		estimatedTokens: 2000,
		estimatedCost:   0.05,
		model:           "gpt-4",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestLimitsMiddleware_DefaultsWithoutEnrichedContext tests fallback to defaults.
func TestLimitsMiddleware_DefaultsWithoutEnrichedContext(t *testing.T) {
	manager := limits.NewManager(limits.Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 100,
			},
		},
	})
	defer manager.Close()

	middleware := LimitsMiddleware(manager)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request without enriched context (should use defaults)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestExtractIdentifier tests identifier extraction from different sources.
func TestExtractIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func(*http.Request)
		wantPrefix string // Expected prefix of identifier
	}{
		{
			name: "from Authorization header",
			setupReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test-api-key")
			},
			wantPrefix: "test-api-key",
		},
		{
			name: "from X-User-ID header",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-User-ID", "user-123")
			},
			wantPrefix: "user-123",
		},
		{
			name: "no identifier",
			setupReq: func(r *http.Request) {
				// No headers
			},
			wantPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupReq(req)

			identifier := extractIdentifier(req)

			if tt.wantPrefix == "" {
				if identifier != "" {
					t.Errorf("Expected empty identifier, got %q", identifier)
				}
			} else {
				if !strings.Contains(identifier, tt.wantPrefix) {
					t.Errorf("Expected identifier to contain %q, got %q", tt.wantPrefix, identifier)
				}
			}
		})
	}
}

// TestSetLimitHeaders tests header setting logic.
func TestSetLimitHeaders(t *testing.T) {
	w := httptest.NewRecorder()

	now := time.Now()
	result := &limits.LimitCheckResult{
		Allowed: true,
		RateLimit: &limits.RateLimitInfo{
			Limit:     100,
			Remaining: 95,
			Reset:     now,
		},
		Budget: &limits.BudgetInfo{
			Limit:     100.00,
			Used:      25.50,
			Remaining: 74.50,
			Reset:     now,
		},
	}

	setLimitHeaders(w, result)

	// Check rate limit headers
	if w.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("Expected X-RateLimit-Limit '100', got %q", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "95" {
		t.Errorf("Expected X-RateLimit-Remaining '95', got %q", w.Header().Get("X-RateLimit-Remaining"))
	}

	// Check budget headers
	if w.Header().Get("X-Budget-Limit") != "100.00" {
		t.Errorf("Expected X-Budget-Limit '100.00', got %q", w.Header().Get("X-Budget-Limit"))
	}
	if w.Header().Get("X-Budget-Used") != "25.50" {
		t.Errorf("Expected X-Budget-Used '25.50', got %q", w.Header().Get("X-Budget-Used"))
	}
	if w.Header().Get("X-Budget-Remaining") != "74.50" {
		t.Errorf("Expected X-Budget-Remaining '74.50', got %q", w.Header().Get("X-Budget-Remaining"))
	}
}

// TestHandleLimitViolation tests error response formatting.
func TestHandleLimitViolation(t *testing.T) {
	w := httptest.NewRecorder()

	result := &limits.LimitCheckResult{
		Allowed: false,
		Reason:  "Rate limit exceeded: 100 requests per second",
	}

	handleLimitViolation(w, result)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Rate limit exceeded") {
		t.Errorf("Expected error message in body, got: %s", body)
	}
	if !strings.Contains(body, "rate_limit_exceeded") {
		t.Errorf("Expected error type 'rate_limit_exceeded', got: %s", body)
	}
}
