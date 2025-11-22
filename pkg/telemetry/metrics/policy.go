package metrics

import (
	"time"

	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
)

// PolicyMetrics tracks metrics related to policy evaluation.
//
// Metrics:
//   - mercator_policy_evaluations_total: Total policy evaluations by rule and action
//   - mercator_policy_evaluation_duration_seconds: Policy evaluation duration
//   - mercator_policy_hits_total: Number of times a policy rule matched
//   - mercator_policy_misses_total: Number of times a policy rule did not match
type PolicyMetrics struct {
	// Total policy evaluations
	evaluationsTotal *prometheus.CounterVec

	// Policy evaluation duration histogram
	evaluationDuration *prometheus.HistogramVec

	// Policy rule hits (rule matched and took action)
	hitsTotal *prometheus.CounterVec

	// Policy rule misses (rule did not match)
	missesTotal *prometheus.CounterVec
}

// NewPolicyMetrics creates and registers policy metrics with the provided registry.
func NewPolicyMetrics(cfg *config.MetricsConfig, registry *prometheus.Registry) *PolicyMetrics {
	pm := &PolicyMetrics{
		evaluationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "policy_evaluations_total",
				Help:      "Total number of policy evaluations",
			},
			[]string{"rule_id", "action"},
		),

		evaluationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "policy_evaluation_duration_seconds",
				Help:      "Duration of policy evaluation in seconds",
				// Policy evaluations should be fast (< 10ms)
				Buckets: prometheus.ExponentialBuckets(0.000001, 2, 15), // 1µs to 16ms
			},
			[]string{"rule_id"},
		),

		hitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "policy_hits_total",
				Help:      "Total number of policy rule matches",
			},
			[]string{"rule_id"},
		),

		missesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "policy_misses_total",
				Help:      "Total number of policy rule misses",
			},
			[]string{"rule_id"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		pm.evaluationsTotal,
		pm.evaluationDuration,
		pm.hitsTotal,
		pm.missesTotal,
	)

	return pm
}

// RecordEvaluation records a policy evaluation.
//
// Parameters:
//   - ruleID: Policy rule identifier
//   - action: Action taken by the policy ("allow", "deny", "modify", "log")
//   - duration: Time taken to evaluate the policy
//
// Example:
//
//	pm.RecordEvaluation("cost-limit-daily", "allow", 1500*time.Microsecond)
func (pm *PolicyMetrics) RecordEvaluation(ruleID, action string, duration time.Duration) {
	pm.evaluationsTotal.WithLabelValues(ruleID, action).Inc()
	pm.evaluationDuration.WithLabelValues(ruleID).Observe(duration.Seconds())
}

// RecordHit records when a policy rule matched and took action.
//
// Parameters:
//   - ruleID: Policy rule identifier
//
// A "hit" means the policy rule's conditions were satisfied and it took action
// (e.g., blocked a request, modified it, or logged it).
func (pm *PolicyMetrics) RecordHit(ruleID string) {
	pm.hitsTotal.WithLabelValues(ruleID).Inc()
}

// RecordMiss records when a policy rule did not match.
//
// Parameters:
//   - ruleID: Policy rule identifier
//
// A "miss" means the policy rule's conditions were not satisfied and it did not take action.
func (pm *PolicyMetrics) RecordMiss(ruleID string) {
	pm.missesTotal.WithLabelValues(ruleID).Inc()
}
