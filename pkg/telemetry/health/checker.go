package health

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CheckFunc is a function that performs a health check for a component.
// It returns nil if the component is healthy, or an error describing the problem.
type CheckFunc func(ctx context.Context) error

// CheckResult represents the result of a single health check.
type CheckResult struct {
	// Status is the health status: "ok", "unhealthy", "disabled"
	Status string `json:"status"`

	// Message provides additional context (usually for unhealthy status)
	Message string `json:"message,omitempty"`

	// Duration is how long the check took
	Duration time.Duration `json:"duration_ms,omitempty"`
}

// HealthStatus represents the overall health status of the system.
type HealthStatus struct {
	// Status is the overall status: "ok", "ready", "degraded", "unhealthy"
	Status string `json:"status"`

	// Checks contains the status of individual components (for readiness)
	Checks map[string]CheckResult `json:"checks,omitempty"`

	// Timestamp is when the health check was performed
	Timestamp time.Time `json:"timestamp"`
}

// Checker manages health checks for system components.
type Checker struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc

	// Timeout for individual checks
	checkTimeout time.Duration
}

var (
	// ErrCheckTimeout is returned when a health check times out
	ErrCheckTimeout = errors.New("health check timeout")

	// ErrNoChecks is returned when no checks are registered
	ErrNoChecks = errors.New("no health checks registered")
)

// New creates a new health checker with the specified check timeout.
// If timeout is 0, defaults to 5 seconds per check.
func New(checkTimeout time.Duration) *Checker {
	if checkTimeout == 0 {
		checkTimeout = 5 * time.Second
	}

	return &Checker{
		checks:       make(map[string]CheckFunc),
		checkTimeout: checkTimeout,
	}
}

// RegisterCheck registers a health check function for a named component.
// If a check with the same name already exists, it will be replaced.
func (c *Checker) RegisterCheck(name string, check CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks[name] = check
}

// UnregisterCheck removes a health check for a named component.
func (c *Checker) UnregisterCheck(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.checks, name)
}

// CheckLiveness performs a simple liveness check.
// It returns a healthy status if the process is running.
// This is a fast check meant for Kubernetes liveness probes.
func (c *Checker) CheckLiveness(ctx context.Context) HealthStatus {
	return HealthStatus{
		Status:    "ok",
		Timestamp: time.Now(),
	}
}

// CheckReadiness performs readiness checks on all registered components.
// It returns the aggregated health status of all components.
// This check may take longer as it performs all component checks.
func (c *Checker) CheckReadiness(ctx context.Context) HealthStatus {
	c.mu.RLock()
	checks := make(map[string]CheckFunc, len(c.checks))
	for name, check := range c.checks {
		checks[name] = check
	}
	c.mu.RUnlock()

	// If no checks registered, system is ready by default
	if len(checks) == 0 {
		return HealthStatus{
			Status:    "ready",
			Checks:    make(map[string]CheckResult),
			Timestamp: time.Now(),
		}
	}

	// Run all checks concurrently
	results := make(map[string]CheckResult)
	var resultMu sync.Mutex
	var wg sync.WaitGroup

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check CheckFunc) {
			defer wg.Done()

			result := c.runCheck(ctx, check)

			resultMu.Lock()
			results[name] = result
			resultMu.Unlock()
		}(name, check)
	}

	wg.Wait()

	// Determine overall status
	status := "ready"

	for _, result := range results {
		if result.Status == "unhealthy" {
			status = "degraded"
		}
	}

	return HealthStatus{
		Status:    status,
		Checks:    results,
		Timestamp: time.Now(),
	}
}

// runCheck executes a single health check with timeout.
func (c *Checker) runCheck(ctx context.Context, check CheckFunc) CheckResult {
	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, c.checkTimeout)
	defer cancel()

	start := time.Now()

	// Run check in goroutine to support timeout
	errChan := make(chan error, 1)
	go func() {
		errChan <- check(checkCtx)
	}()

	// Wait for check to complete or timeout
	select {
	case err := <-errChan:
		duration := time.Since(start)
		if err != nil {
			return CheckResult{
				Status:   "unhealthy",
				Message:  err.Error(),
				Duration: duration,
			}
		}
		return CheckResult{
			Status:   "ok",
			Duration: duration,
		}

	case <-checkCtx.Done():
		duration := time.Since(start)
		return CheckResult{
			Status:   "unhealthy",
			Message:  "health check timeout",
			Duration: duration,
		}
	}
}

// GetCheck returns the check function for a named component.
// Returns nil if the check doesn't exist.
func (c *Checker) GetCheck(name string) CheckFunc {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.checks[name]
}

// ListChecks returns the names of all registered health checks.
func (c *Checker) ListChecks() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.checks))
	for name := range c.checks {
		names = append(names, name)
	}

	return names
}

// CheckCount returns the number of registered health checks.
func (c *Checker) CheckCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.checks)
}
