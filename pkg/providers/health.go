package providers

import (
	"context"
	"log/slog"
	"time"
)

// StartHealthChecker starts a background goroutine that periodically checks
// the provider's health. It updates the provider's health status atomically.
//
// The health checker runs until the provider is closed or the context is cancelled.
// It implements exponential backoff when the provider is unhealthy to reduce load.
func (p *HTTPProvider) StartHealthChecker(ctx context.Context) {
	go p.runHealthChecker(ctx)
}

// runHealthChecker is the main health checking loop.
func (p *HTTPProvider) runHealthChecker(ctx context.Context) {
	defer close(p.healthCheckStopped)

	interval := p.config.HealthCheckInterval
	if interval == 0 {
		interval = 30 * time.Second // Default to 30 seconds
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("health checker started",
		"provider", p.config.Name,
		"interval", interval,
	)

	for {
		select {
		case <-ctx.Done():
			slog.Debug("health checker stopped (context cancelled)", "provider", p.config.Name)
			return

		case <-p.stopHealthCheck:
			slog.Debug("health checker stopped (provider closed)", "provider", p.config.Name)
			return

		case <-ticker.C:
			p.performHealthCheck(ctx)

			// If provider is unhealthy, use exponential backoff
			if !p.IsHealthy() {
				health := p.GetHealth()
				backoffInterval := calculateBackoff(health.ConsecutiveFailures, interval)
				ticker.Reset(backoffInterval)

				slog.Debug("health check backoff",
					"provider", p.config.Name,
					"consecutive_failures", health.ConsecutiveFailures,
					"next_check_in", backoffInterval,
				)
			} else {
				// Reset to normal interval when healthy
				ticker.Reset(interval)
			}
		}
	}
}

// performHealthCheck executes a single health check.
func (p *HTTPProvider) performHealthCheck(ctx context.Context) {
	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	err := p.healthCheckImpl(checkCtx)
	latency := time.Since(start)

	if err != nil {
		p.updateHealth(false, err)
		slog.Error("health check failed",
			"provider", p.config.Name,
			"error", err,
			"latency", latency,
		)
	} else {
		p.updateHealth(true, nil)
		slog.Debug("health check passed",
			"provider", p.config.Name,
			"latency", latency,
		)

		// Log when provider recovers from unhealthy state
		health := p.GetHealth()
		if health.ConsecutiveFailures > 0 {
			slog.Info("provider marked healthy",
				"provider", p.config.Name,
				"previous_failures", health.ConsecutiveFailures,
			)
		}
	}
}

// healthCheckImpl performs the actual health check.
// This is a lightweight HEAD request to verify the provider is reachable.
func (p *HTTPProvider) healthCheckImpl(ctx context.Context) error {
	// Construct health check URL
	// For most providers, we can use a HEAD request to the base URL
	url := p.config.BaseURL

	// Prepare headers
	headers := make(map[string]string)
	if p.config.APIKey != "" {
		// Different providers use different auth header formats
		// This is a generic implementation - specific providers may override
		headers["Authorization"] = "Bearer " + p.config.APIKey
	}

	// Perform HEAD request
	resp, err := p.DoRequest(ctx, "GET", url, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// calculateBackoff calculates the backoff interval based on consecutive failures.
// It uses exponential backoff with a maximum interval of 5 minutes.
func calculateBackoff(consecutiveFailures int, baseInterval time.Duration) time.Duration {
	if consecutiveFailures <= 0 {
		return baseInterval
	}

	// Exponential backoff: base * 2^failures
	multiplier := 1 << uint(consecutiveFailures) // 2^failures
	if multiplier > 10 {
		multiplier = 10 // Cap at 10x the base interval
	}

	backoff := baseInterval * time.Duration(multiplier)

	// Cap at 5 minutes
	maxBackoff := 5 * time.Minute
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// HealthCheck performs a synchronous health check (part of Provider interface).
// This is called on-demand, while StartHealthChecker runs periodic checks.
func (p *HTTPProvider) HealthCheck(ctx context.Context) error {
	return p.healthCheckImpl(ctx)
}
