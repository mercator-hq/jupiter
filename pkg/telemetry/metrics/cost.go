package metrics

import (
	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
)

// CostMetrics tracks cost-related metrics for LLM requests.
//
// Metrics:
//   - mercator_cost_total: Total cost in USD by provider and model
//   - mercator_cost_per_request: Cost distribution per request (histogram)
//   - mercator_cost_per_token: Average cost per token by provider and model
type CostMetrics struct {
	// Total cost counter (in USD)
	costTotal *prometheus.CounterVec

	// Cost per request histogram (in USD)
	costPerRequest *prometheus.HistogramVec

	// Cost per token (derived metric, recorded as gauge)
	costPerToken *prometheus.GaugeVec
}

// NewCostMetrics creates and registers cost metrics with the provided registry.
func NewCostMetrics(cfg *config.MetricsConfig, registry *prometheus.Registry) *CostMetrics {
	cm := &CostMetrics{
		costTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cost_total",
				Help:      "Total cost in USD by provider and model",
			},
			[]string{"provider", "model"},
		),

		costPerRequest: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cost_per_request",
				Help:      "Cost distribution per request in USD",
				// Cost buckets: $0.001 to $10 (optimized for LLM pricing)
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0},
			},
			[]string{"provider", "model"},
		),

		costPerToken: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cost_per_token",
				Help:      "Cost per token in USD by provider and model",
			},
			[]string{"provider", "model"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		cm.costTotal,
		cm.costPerRequest,
		cm.costPerToken,
	)

	return cm
}

// RecordRequestCost records the cost of a single request.
//
// Parameters:
//   - provider: LLM provider name
//   - model: Model name
//   - costUSD: Request cost in USD
//
// This updates both the total cost counter and the cost-per-request histogram.
//
// Example:
//
//	cm.RecordRequestCost("openai", "gpt-4", 0.05)
func (cm *CostMetrics) RecordRequestCost(provider, model string, costUSD float64) {
	if costUSD <= 0 {
		return
	}

	cm.costTotal.WithLabelValues(provider, model).Add(costUSD)
	cm.costPerRequest.WithLabelValues(provider, model).Observe(costUSD)
}

// UpdateCostPerToken updates the average cost per token for a provider/model.
//
// Parameters:
//   - provider: LLM provider name
//   - model: Model name
//   - costPerToken: Cost per token in USD
//
// This is typically called when provider pricing changes or after calculating
// an average over a time window.
//
// Example:
//
//	// GPT-4 Turbo pricing (as of 2024)
//	cm.UpdateCostPerToken("openai", "gpt-4-turbo", 0.00003) // $0.03 per 1K tokens
func (cm *CostMetrics) UpdateCostPerToken(provider, model string, costPerToken float64) {
	cm.costPerToken.WithLabelValues(provider, model).Set(costPerToken)
}
