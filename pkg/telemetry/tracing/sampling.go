package tracing

import (
	"fmt"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Sampling strategies determine which traces are recorded and exported.
// Three strategies are supported:
//   - always: Sample 100% of traces (development/debugging)
//   - never: Sample 0% of traces (tracing effectively disabled)
//   - ratio: Sample a percentage of traces (production)

const (
	// SamplerAlways samples all traces
	SamplerAlways = "always"

	// SamplerNever samples no traces
	SamplerNever = "never"

	// SamplerRatio samples a percentage of traces
	SamplerRatio = "ratio"
)

// createSampler creates a sampler based on the strategy and ratio.
//
// # Sampling Strategies
//
// AlwaysOn: Samples all traces. Use in development/debugging.
//
//	telemetry:
//	  tracing:
//	    sampler: always
//
// AlwaysOff: Samples no traces. Use when tracing should be completely disabled.
//
//	telemetry:
//	  tracing:
//	    sampler: never
//
// TraceIDRatioBased: Samples traces based on trace ID hash. This ensures
// consistent sampling decisions across services (same trace ID = same decision).
//
//	telemetry:
//	  tracing:
//	    sampler: ratio
//	    sample_ratio: 0.1  # Sample 10% of traces
//
// # Sampling Decision
//
// The sampling decision is made once at trace creation and propagated to
// all child spans. This ensures either the entire trace is sampled or none of it.
//
// # Parent-Based Sampling
//
// All samplers are wrapped in ParentBased(), which respects the parent span's
// sampling decision when available. This maintains consistency in distributed traces:
//   - If parent span is sampled → child is sampled
//   - If parent span is not sampled → child is not sampled
//   - If no parent span → use configured sampler
func createSampler(strategy string, ratio float64) (sdktrace.Sampler, error) {
	var baseSampler sdktrace.Sampler

	switch strategy {
	case SamplerAlways:
		// AlwaysOn samples all traces
		baseSampler = sdktrace.AlwaysSample()

	case SamplerNever:
		// NeverSample samples no traces
		baseSampler = sdktrace.NeverSample()

	case SamplerRatio:
		// Validate ratio
		if ratio < 0.0 || ratio > 1.0 {
			return nil, fmt.Errorf("sample ratio must be between 0.0 and 1.0, got %f", ratio)
		}

		// TraceIDRatioBased samples based on trace ID hash
		// This ensures consistent sampling across distributed services
		baseSampler = sdktrace.TraceIDRatioBased(ratio)

	default:
		return nil, fmt.Errorf("unknown sampler strategy: %s (valid: always, never, ratio)", strategy)
	}

	// Wrap in ParentBased to respect parent sampling decisions
	// This ensures sampling consistency in distributed traces
	return sdktrace.ParentBased(baseSampler), nil
}

// SamplingConfig contains configuration for trace sampling.
type SamplingConfig struct {
	// Strategy is the sampling strategy ("always", "never", "ratio")
	Strategy string

	// Ratio is the sampling ratio for "ratio" strategy (0.0 to 1.0)
	Ratio float64
}

// ValidateSamplingConfig validates the sampling configuration.
func ValidateSamplingConfig(cfg SamplingConfig) error {
	// Validate strategy
	switch cfg.Strategy {
	case SamplerAlways, SamplerNever, SamplerRatio:
		// Valid strategies
	default:
		return fmt.Errorf("invalid sampling strategy: %s (valid: always, never, ratio)", cfg.Strategy)
	}

	// Validate ratio for ratio-based sampling
	if cfg.Strategy == SamplerRatio {
		if cfg.Ratio < 0.0 || cfg.Ratio > 1.0 {
			return fmt.Errorf("sample ratio must be between 0.0 and 1.0, got %f", cfg.Ratio)
		}
	}

	return nil
}

// SamplingRecommendations provides guidelines for choosing sampling strategies.
//
// # Development
//
// Use "always" sampling to capture all traces for debugging:
//
//	telemetry:
//	  tracing:
//	    enabled: true
//	    sampler: always
//
// # Production (Low Traffic)
//
// For services with <1000 requests/minute, use "always" or high ratio:
//
//	telemetry:
//	  tracing:
//	    enabled: true
//	    sampler: ratio
//	    sample_ratio: 1.0  # or 0.5 for 50%
//
// # Production (Medium Traffic)
//
// For services with 1K-10K requests/minute, use moderate ratio:
//
//	telemetry:
//	  tracing:
//	    enabled: true
//	    sampler: ratio
//	    sample_ratio: 0.1  # 10% sampling
//
// # Production (High Traffic)
//
// For services with >10K requests/minute, use low ratio:
//
//	telemetry:
//	  tracing:
//	    enabled: true
//	    sampler: ratio
//	    sample_ratio: 0.01  # 1% sampling
//
// # Cost Considerations
//
// Trace storage and analysis costs increase with sampling ratio.
// A 1% sampling ratio typically provides sufficient visibility while
// minimizing costs for high-traffic services.
//
// # Error Sampling
//
// Consider implementing error-based sampling in the future:
//   - Always sample traces with errors (even if ratio-based sampling rejects)
//   - Provides full visibility into failures
//   - Requires custom sampler implementation
const SamplingRecommendations = ""
