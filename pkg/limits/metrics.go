package limits

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains Prometheus metrics for the limits package.
type Metrics struct {
	// Rate limit checks
	rateLimitChecks *prometheus.CounterVec
	rateLimitHits   *prometheus.CounterVec

	// Budget checks
	budgetChecks *prometheus.CounterVec
	budgetHits   *prometheus.CounterVec
	budgetUsage  *prometheus.GaugeVec

	// Enforcement actions
	enforcementActions *prometheus.CounterVec

	// Concurrent requests
	concurrentRequests *prometheus.GaugeVec

	// Check latency
	checkDuration *prometheus.HistogramVec
}

// NewMetrics creates a new Metrics instance with Prometheus collectors.
func NewMetrics() *Metrics {
	return &Metrics{
		rateLimitChecks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mercator_limits_rate_limit_checks_total",
				Help: "Total number of rate limit checks performed",
			},
			[]string{"identifier", "result"},
		),

		rateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mercator_limits_rate_limit_hits_total",
				Help: "Total number of rate limit violations",
			},
			[]string{"identifier", "limit_type"},
		),

		budgetChecks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mercator_limits_budget_checks_total",
				Help: "Total number of budget checks performed",
			},
			[]string{"identifier", "result"},
		),

		budgetHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mercator_limits_budget_hits_total",
				Help: "Total number of budget violations",
			},
			[]string{"identifier", "window"},
		),

		budgetUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "mercator_limits_budget_usage_percentage",
				Help: "Current budget usage as percentage (0.0-1.0)",
			},
			[]string{"identifier", "window"},
		),

		enforcementActions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mercator_limits_enforcement_actions_total",
				Help: "Total number of enforcement actions taken",
			},
			[]string{"identifier", "action"},
		),

		concurrentRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "mercator_limits_concurrent_requests",
				Help: "Current number of concurrent requests",
			},
			[]string{"identifier"},
		),

		checkDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mercator_limits_check_duration_seconds",
				Help:    "Duration of limit checks in seconds",
				Buckets: prometheus.ExponentialBuckets(0.000001, 2, 15), // 1µs to 16ms
			},
			[]string{"operation"},
		),
	}
}

// RecordRateLimitCheck records a rate limit check.
func (m *Metrics) RecordRateLimitCheck(identifier string, allowed bool) {
	result := "allowed"
	if !allowed {
		result = "blocked"
	}
	m.rateLimitChecks.WithLabelValues(identifier, result).Inc()
}

// RecordRateLimitHit records a rate limit violation.
func (m *Metrics) RecordRateLimitHit(identifier string, limitType string) {
	m.rateLimitHits.WithLabelValues(identifier, limitType).Inc()
}

// RecordBudgetCheck records a budget check.
func (m *Metrics) RecordBudgetCheck(identifier string, allowed bool) {
	result := "allowed"
	if !allowed {
		result = "blocked"
	}
	m.budgetChecks.WithLabelValues(identifier, result).Inc()
}

// RecordBudgetHit records a budget violation.
func (m *Metrics) RecordBudgetHit(identifier string, window string) {
	m.budgetHits.WithLabelValues(identifier, window).Inc()
}

// UpdateBudgetUsage updates the current budget usage percentage.
func (m *Metrics) UpdateBudgetUsage(identifier string, window string, percentage float64) {
	m.budgetUsage.WithLabelValues(identifier, window).Set(percentage)
}

// RecordEnforcementAction records an enforcement action.
func (m *Metrics) RecordEnforcementAction(identifier string, action EnforcementAction) {
	m.enforcementActions.WithLabelValues(identifier, string(action)).Inc()
}

// UpdateConcurrentRequests updates the current concurrent request count.
func (m *Metrics) UpdateConcurrentRequests(identifier string, count int64) {
	m.concurrentRequests.WithLabelValues(identifier).Set(float64(count))
}

// RecordCheckDuration records the duration of a limit check operation.
func (m *Metrics) RecordCheckDuration(operation string, duration float64) {
	m.checkDuration.WithLabelValues(operation).Observe(duration)
}
