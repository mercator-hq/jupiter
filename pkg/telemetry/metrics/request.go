package metrics

import (
	"time"

	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
)

// RequestMetrics tracks metrics related to LLM request processing.
//
// Metrics:
//   - mercator_requests_total: Total request count by provider, model, status
//   - mercator_request_duration_seconds: Request duration histogram
//   - mercator_request_tokens_total: Total tokens processed
//   - mercator_request_size_bytes: Request/response size (if applicable)
type RequestMetrics struct {
	// Total request count
	requestsTotal *prometheus.CounterVec

	// Request duration histogram
	requestDuration *prometheus.HistogramVec

	// Token counts (prompt and completion)
	tokensTotal *prometheus.CounterVec

	// Request/response size in bytes
	sizeBytes *prometheus.HistogramVec
}

// NewRequestMetrics creates and registers request metrics with the provided registry.
func NewRequestMetrics(cfg *config.MetricsConfig, registry *prometheus.Registry) *RequestMetrics {
	rm := &RequestMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_total",
				Help:      "Total number of LLM requests processed",
			},
			[]string{"provider", "model", "status"},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_duration_seconds",
				Help:      "Duration of LLM requests in seconds",
				Buckets:   cfg.RequestDurationBuckets,
			},
			[]string{"provider", "model"},
		),

		tokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_tokens_total",
				Help:      "Total number of tokens processed",
			},
			[]string{"provider", "model", "type"},
		),

		sizeBytes: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_size_bytes",
				Help:      "Size of request/response in bytes",
				Buckets:   prometheus.ExponentialBuckets(1024, 2, 12), // 1KB to 4MB
			},
			[]string{"provider", "model", "direction"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		rm.requestsTotal,
		rm.requestDuration,
		rm.tokensTotal,
		rm.sizeBytes,
	)

	return rm
}

// RecordRequest records metrics for a completed request.
//
// Parameters:
//   - provider: LLM provider name
//   - model: Model name
//   - status: Request status ("success", "error", "blocked")
//   - duration: Request duration
//   - tokens: Total token count
func (rm *RequestMetrics) RecordRequest(provider, model, status string, duration time.Duration, tokens int) {
	// Increment request counter
	rm.requestsTotal.WithLabelValues(provider, model, status).Inc()

	// Record duration
	rm.requestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())

	// Record tokens (if known)
	if tokens > 0 {
		rm.tokensTotal.WithLabelValues(provider, model, "total").Add(float64(tokens))
	}
}

// RecordTokens records token counts separately for prompt and completion.
//
// Parameters:
//   - provider: LLM provider name
//   - model: Model name
//   - promptTokens: Number of tokens in the prompt
//   - completionTokens: Number of tokens in the completion
func (rm *RequestMetrics) RecordTokens(provider, model string, promptTokens, completionTokens int) {
	if promptTokens > 0 {
		rm.tokensTotal.WithLabelValues(provider, model, "prompt").Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		rm.tokensTotal.WithLabelValues(provider, model, "completion").Add(float64(completionTokens))
	}
}

// RecordSize records the size of a request or response.
//
// Parameters:
//   - provider: LLM provider name
//   - model: Model name
//   - direction: "request" or "response"
//   - sizeBytes: Size in bytes
func (rm *RequestMetrics) RecordSize(provider, model, direction string, sizeBytes int) {
	if sizeBytes > 0 {
		rm.sizeBytes.WithLabelValues(provider, model, direction).Observe(float64(sizeBytes))
	}
}
