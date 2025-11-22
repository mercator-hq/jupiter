// Package metrics provides Prometheus metrics collection for Mercator Jupiter.
//
// # Overview
//
// The metrics package implements comprehensive Prometheus metrics for monitoring
// LLM request processing, provider health, policy evaluations, costs, and rate
// limits. It provides high-performance metric collection with minimal overhead
// (<50µs per request).
//
// # Metrics Categories
//
//   - Request Metrics: Request count, duration, tokens, and sizes
//   - Provider Metrics: Provider health, latency, and error rates
//   - Policy Metrics: Policy evaluation count, duration, and actions
//   - Cost Metrics: Total cost and cost per request by provider/model
//   - Limit Metrics: Budget usage and rate limit violations
//   - Cache Metrics: Cache hits, misses, and sizes (if caching enabled)
//
// # Usage
//
//	// Create collector
//	collector := metrics.NewCollector(config, registry)
//
//	// Record request metrics
//	collector.RecordRequest(
//		"openai",         // provider
//		"gpt-4",          // model
//		"success",        // status
//		time.Second,      // duration
//		1500,             // tokens
//		0.05,             // cost
//	)
//
//	// Record provider metrics
//	collector.RecordProviderLatency("openai", "gpt-4", 0.95)
//	collector.UpdateProviderHealth("openai", true)
//
//	// Record policy metrics
//	collector.RecordPolicyEvaluation("cost-limit", "allow", 2*time.Millisecond)
//
// # Performance
//
// The metrics package is optimized for minimal overhead:
//
//   - Lock-free counters where possible
//   - Pre-allocated metric instances
//   - Batch updates for high-volume metrics
//   - Configurable cardinality limits
//   - Target: <50µs per metric update
//
// # Custom Histogram Buckets
//
// The collector uses custom histogram buckets optimized for LLM workloads:
//
//	Request Duration: 0.1s, 0.25s, 0.5s, 1s, 2s, 5s, 10s, 30s
//	Token Counts: 100, 500, 1K, 5K, 10K, 50K, 100K
//
// # Prometheus Endpoint
//
// All metrics are exposed on the /metrics endpoint in standard Prometheus format:
//
//	# HELP mercator_requests_total Total number of requests
//	# TYPE mercator_requests_total counter
//	mercator_requests_total{provider="openai",model="gpt-4",status="success"} 1234
//
// # Cardinality Management
//
// The collector implements cardinality limits to prevent memory issues:
//
//   - Maximum 10,000 unique label combinations per metric
//   - Low-frequency labels aggregated into "other"
//   - Warnings logged when approaching limits
//
// # Integration with pkg/limits/metrics.go
//
// The collector extends (but does not replace) the existing metrics in
// pkg/limits/metrics.go. Both coexist:
//
//   - pkg/limits/metrics.go: Rate limit and budget metrics
//   - pkg/telemetry/metrics: Request, provider, policy, cost metrics
package metrics
