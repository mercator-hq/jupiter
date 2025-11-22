package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNew tests the creation of a new health checker.
func TestNew(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "default timeout",
			timeout:         0,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "custom timeout",
			timeout:         10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := New(tt.timeout)

			if checker == nil {
				t.Fatal("expected non-nil checker")
			}

			if checker.checkTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, checker.checkTimeout)
			}

			if checker.checks == nil {
				t.Error("expected non-nil checks map")
			}

			if len(checker.checks) != 0 {
				t.Errorf("expected 0 checks, got %d", len(checker.checks))
			}
		})
	}
}

// TestRegisterCheck tests registering health checks.
func TestRegisterCheck(t *testing.T) {
	checker := New(5 * time.Second)

	// Register a check
	called := false
	checker.RegisterCheck("test", func(ctx context.Context) error {
		called = true
		return nil
	})

	if checker.CheckCount() != 1 {
		t.Errorf("expected 1 check, got %d", checker.CheckCount())
	}

	// Call the check
	check := checker.GetCheck("test")
	if check == nil {
		t.Fatal("expected non-nil check")
	}

	_ = check(context.Background())
	if !called {
		t.Error("expected check to be called")
	}

	// Replace check
	called2 := false
	checker.RegisterCheck("test", func(ctx context.Context) error {
		called2 = true
		return nil
	})

	if checker.CheckCount() != 1 {
		t.Errorf("expected 1 check after replacement, got %d", checker.CheckCount())
	}

	check2 := checker.GetCheck("test")
	_ = check2(context.Background())
	if !called2 {
		t.Error("expected replacement check to be called")
	}
}

// TestUnregisterCheck tests unregistering health checks.
func TestUnregisterCheck(t *testing.T) {
	checker := New(5 * time.Second)

	checker.RegisterCheck("test1", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("test2", func(ctx context.Context) error { return nil })

	if checker.CheckCount() != 2 {
		t.Errorf("expected 2 checks, got %d", checker.CheckCount())
	}

	checker.UnregisterCheck("test1")

	if checker.CheckCount() != 1 {
		t.Errorf("expected 1 check after unregister, got %d", checker.CheckCount())
	}

	if checker.GetCheck("test1") != nil {
		t.Error("expected nil for unregistered check")
	}

	if checker.GetCheck("test2") == nil {
		t.Error("expected non-nil for remaining check")
	}
}

// TestListChecks tests listing registered checks.
func TestListChecks(t *testing.T) {
	checker := New(5 * time.Second)

	checker.RegisterCheck("test1", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("test2", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("test3", func(ctx context.Context) error { return nil })

	checks := checker.ListChecks()

	if len(checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(checks))
	}

	// Check names are present
	names := make(map[string]bool)
	for _, name := range checks {
		names[name] = true
	}

	if !names["test1"] || !names["test2"] || !names["test3"] {
		t.Error("expected all check names to be present")
	}
}

// TestCheckLiveness tests the liveness check.
func TestCheckLiveness(t *testing.T) {
	checker := New(5 * time.Second)

	status := checker.CheckLiveness(context.Background())

	if status.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", status.Status)
	}

	if status.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	if status.Checks != nil && len(status.Checks) > 0 {
		t.Error("expected no checks in liveness response")
	}
}

// TestCheckReadiness_NoChecks tests readiness with no checks registered.
func TestCheckReadiness_NoChecks(t *testing.T) {
	checker := New(5 * time.Second)

	status := checker.CheckReadiness(context.Background())

	if status.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", status.Status)
	}

	if status.Checks == nil {
		t.Error("expected non-nil checks map")
	}

	if len(status.Checks) != 0 {
		t.Errorf("expected 0 checks, got %d", len(status.Checks))
	}
}

// TestCheckReadiness_AllHealthy tests readiness with all healthy checks.
func TestCheckReadiness_AllHealthy(t *testing.T) {
	checker := New(5 * time.Second)

	checker.RegisterCheck("test1", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("test2", func(ctx context.Context) error { return nil })

	status := checker.CheckReadiness(context.Background())

	if status.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", status.Status)
	}

	if len(status.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(status.Checks))
	}

	for name, result := range status.Checks {
		if result.Status != "ok" {
			t.Errorf("expected check %q to be ok, got %q", name, result.Status)
		}
	}
}

// TestCheckReadiness_SomeUnhealthy tests readiness with unhealthy checks.
func TestCheckReadiness_SomeUnhealthy(t *testing.T) {
	checker := New(5 * time.Second)

	checker.RegisterCheck("healthy", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("unhealthy", func(ctx context.Context) error {
		return errors.New("component unhealthy")
	})

	status := checker.CheckReadiness(context.Background())

	if status.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", status.Status)
	}

	if len(status.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(status.Checks))
	}

	healthyResult := status.Checks["healthy"]
	if healthyResult.Status != "ok" {
		t.Errorf("expected healthy check to be ok, got %q", healthyResult.Status)
	}

	unhealthyResult := status.Checks["unhealthy"]
	if unhealthyResult.Status != "unhealthy" {
		t.Errorf("expected unhealthy check to be unhealthy, got %q", unhealthyResult.Status)
	}
	if unhealthyResult.Message != "component unhealthy" {
		t.Errorf("expected message 'component unhealthy', got %q", unhealthyResult.Message)
	}
}

// TestCheckReadiness_Timeout tests readiness with a check that times out.
func TestCheckReadiness_Timeout(t *testing.T) {
	checker := New(100 * time.Millisecond)

	checker.RegisterCheck("slow", func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	status := checker.CheckReadiness(context.Background())

	if status.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", status.Status)
	}

	slowResult := status.Checks["slow"]
	if slowResult.Status != "unhealthy" {
		t.Errorf("expected slow check to be unhealthy, got %q", slowResult.Status)
	}
	if slowResult.Message != "health check timeout" {
		t.Errorf("expected timeout message, got %q", slowResult.Message)
	}
}

// TestCheckReadiness_ContextCancellation tests readiness with context cancellation.
func TestCheckReadiness_ContextCancellation(t *testing.T) {
	checker := New(5 * time.Second)

	checker.RegisterCheck("test", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	status := checker.CheckReadiness(ctx)

	// Check should fail due to cancellation
	testResult := status.Checks["test"]
	if testResult.Status != "unhealthy" {
		t.Errorf("expected test check to be unhealthy, got %q", testResult.Status)
	}
}

// TestLivenessHandler tests the liveness HTTP handler.
func TestLivenessHandler(t *testing.T) {
	checker := New(5 * time.Second)
	handler := checker.LivenessHandler()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkBody      bool
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "HEAD request",
			method:         http.MethodHead,
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkBody {
				var status HealthStatus
				if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				if status.Status != "ok" {
					t.Errorf("expected status 'ok', got %q", status.Status)
				}
			}
		})
	}
}

// TestReadinessHandler tests the readiness HTTP handler.
func TestReadinessHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupChecks    func(*Checker)
		expectedStatus int
		expectedHealth string
	}{
		{
			name: "all healthy",
			setupChecks: func(c *Checker) {
				c.RegisterCheck("test", func(ctx context.Context) error { return nil })
			},
			expectedStatus: http.StatusOK,
			expectedHealth: "ready",
		},
		{
			name: "some unhealthy",
			setupChecks: func(c *Checker) {
				c.RegisterCheck("healthy", func(ctx context.Context) error { return nil })
				c.RegisterCheck("unhealthy", func(ctx context.Context) error {
					return errors.New("failed")
				})
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "degraded",
		},
		{
			name:           "no checks",
			setupChecks:    func(c *Checker) {},
			expectedStatus: http.StatusOK,
			expectedHealth: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := New(5 * time.Second)
			tt.setupChecks(checker)

			handler := checker.ReadinessHandler()

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			var status HealthStatus
			if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if status.Status != tt.expectedHealth {
				t.Errorf("expected status %q, got %q", tt.expectedHealth, status.Status)
			}
		})
	}
}

// TestVersionHandler tests the version HTTP handler.
func TestVersionHandler(t *testing.T) {
	version := "1.0.0"
	commit := "abc123"
	buildTime := "2025-11-20T00:00:00Z"

	handler := VersionHandler(version, commit, buildTime)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var info VersionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if info.Version != version {
		t.Errorf("expected version %q, got %q", version, info.Version)
	}
	if info.Commit != commit {
		t.Errorf("expected commit %q, got %q", commit, info.Commit)
	}
	if info.BuildTime != buildTime {
		t.Errorf("expected build time %q, got %q", buildTime, info.BuildTime)
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty go version")
	}
}

// TestCreateHandlers tests creating all handlers at once.
func TestCreateHandlers(t *testing.T) {
	checker := New(5 * time.Second)
	handlers := checker.CreateHandlers("1.0.0", "abc123", "2025-11-20")

	if handlers.LivenessHandler == nil {
		t.Error("expected non-nil liveness handler")
	}
	if handlers.ReadinessHandler == nil {
		t.Error("expected non-nil readiness handler")
	}
	if handlers.VersionHandler == nil {
		t.Error("expected non-nil version handler")
	}
}

// TestHTTPMiddleware tests the HTTP middleware.
func TestHTTPMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	checker := New(5 * time.Second)

	HTTPMiddleware(mux, checker, "1.0.0", "abc123", "2025-11-20")

	tests := []struct {
		path           string
		expectedStatus int
	}{
		{"/health", http.StatusOK},
		{"/ready", http.StatusOK},
		{"/version", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestRateLimitedHandler tests the rate-limited handler.
func TestRateLimitedHandler(t *testing.T) {
	checker := New(5 * time.Second)
	baseHandler := checker.LivenessHandler()

	// Create rate-limited handler (2 req/s)
	handler := RateLimitedHandler(baseHandler, 2)

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, rec.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
}

// TestRateLimitedHandler_Disabled tests rate limiting with 0 or negative limit.
func TestRateLimitedHandler_Disabled(t *testing.T) {
	checker := New(5 * time.Second)
	baseHandler := checker.LivenessHandler()

	// Create rate-limited handler with 0 limit (disabled)
	handler := RateLimitedHandler(baseHandler, 0)

	// All requests should succeed
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, rec.Code)
		}
	}
}

// TestConcurrentChecks tests concurrent health checks.
func TestConcurrentChecks(t *testing.T) {
	checker := New(5 * time.Second)

	// Register multiple checks
	for i := 0; i < 10; i++ {
		checker.RegisterCheck("test", func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	// Run multiple concurrent readiness checks
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			status := checker.CheckReadiness(context.Background())
			if status.Status != "ready" {
				t.Errorf("expected status 'ready', got %q", status.Status)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestCheckResult_Duration tests that check results include duration.
func TestCheckResult_Duration(t *testing.T) {
	checker := New(5 * time.Second)

	checker.RegisterCheck("slow", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	status := checker.CheckReadiness(context.Background())

	slowResult := status.Checks["slow"]
	if slowResult.Duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", slowResult.Duration)
	}
}
