package metrics

import (
	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
)

// ProviderMetrics tracks metrics related to LLM provider health and performance.
//
// Metrics:
//   - mercator_provider_health: Provider health status (1=healthy, 0=unhealthy)
//   - mercator_provider_latency_seconds: Provider API latency
//   - mercator_provider_errors_total: Provider error count by type
//   - mercator_provider_requests_total: Total requests to each provider
type ProviderMetrics struct {
	// Provider health status (gauge: 1=healthy, 0=unhealthy)
	health *prometheus.GaugeVec

	// Provider API latency histogram
	latency *prometheus.HistogramVec

	// Provider error counter
	errors *prometheus.CounterVec

	// Total requests to provider
	requests *prometheus.CounterVec
}

// NewProviderMetrics creates and registers provider metrics with the provided registry.
func NewProviderMetrics(cfg *config.MetricsConfig, registry *prometheus.Registry) *ProviderMetrics {
	pm := &ProviderMetrics{
		health: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "provider_health",
				Help:      "Provider health status (1=healthy, 0=unhealthy)",
			},
			[]string{"provider"},
		),

		latency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "provider_latency_seconds",
				Help:      "Provider API call latency in seconds",
				Buckets:   cfg.RequestDurationBuckets, // Reuse request duration buckets
			},
			[]string{"provider", "model"},
		),

		errors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "provider_errors_total",
				Help:      "Total number of provider errors by type",
			},
			[]string{"provider", "error_type"},
		),

		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "provider_requests_total",
				Help:      "Total number of requests to each provider",
			},
			[]string{"provider", "model"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		pm.health,
		pm.latency,
		pm.errors,
		pm.requests,
	)

	return pm
}

// UpdateHealth updates the health status of a provider.
//
// Parameters:
//   - provider: Provider name (e.g., "openai", "anthropic")
//   - healthy: true if provider is healthy, false otherwise
//
// The health metric is a gauge where 1=healthy, 0=unhealthy.
func (pm *ProviderMetrics) UpdateHealth(provider string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	pm.health.WithLabelValues(provider).Set(value)
}

// RecordLatency records the latency of a provider API call.
//
// Parameters:
//   - provider: Provider name
//   - model: Model name
//   - latencySeconds: API call latency in seconds
func (pm *ProviderMetrics) RecordLatency(provider, model string, latencySeconds float64) {
	pm.latency.WithLabelValues(provider, model).Observe(latencySeconds)
}

// RecordError records an error from a provider.
//
// Parameters:
//   - provider: Provider name
//   - errorType: Type of error (e.g., "rate_limit", "timeout", "auth", "server_error", "network")
//
// Common error types:
//   - "rate_limit": Provider rate limit exceeded
//   - "timeout": Request timeout
//   - "auth": Authentication/authorization error
//   - "server_error": Provider server error (5xx)
//   - "client_error": Client error (4xx)
//   - "network": Network connectivity error
//   - "parse": Response parsing error
func (pm *ProviderMetrics) RecordError(provider, errorType string) {
	pm.errors.WithLabelValues(provider, errorType).Inc()
}

// RecordRequest records a request to a provider.
//
// Parameters:
//   - provider: Provider name
//   - model: Model name
//
// This metric tracks the total number of API calls made to each provider/model combination.
func (pm *ProviderMetrics) RecordRequest(provider, model string) {
	pm.requests.WithLabelValues(provider, model).Inc()
}
