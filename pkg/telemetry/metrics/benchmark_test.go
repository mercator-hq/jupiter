package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Benchmark_Collector_RecordRequest benchmarks request recording
func Benchmark_Collector_RecordRequest(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordRequest("openai", "gpt-4", "success", time.Second, 1500, 0.05)
	}
}

// Benchmark_Collector_RecordRequest_Parallel benchmarks parallel request recording
func Benchmark_Collector_RecordRequest_Parallel(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordRequest("openai", "gpt-4", "success", time.Second, 1500, 0.05)
		}
	})
}

// Benchmark_Collector_UpdateProviderHealth benchmarks health updates
func Benchmark_Collector_UpdateProviderHealth(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.UpdateProviderHealth("openai", true)
	}
}

// Benchmark_Collector_RecordProviderLatency benchmarks latency recording
func Benchmark_Collector_RecordProviderLatency(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordProviderLatency("openai", "gpt-4", 0.95)
	}
}

// Benchmark_Collector_RecordProviderError benchmarks error recording
func Benchmark_Collector_RecordProviderError(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordProviderError("openai", "rate_limit")
	}
}

// Benchmark_Collector_RecordPolicyEvaluation benchmarks policy evaluation recording
func Benchmark_Collector_RecordPolicyEvaluation(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordPolicyEvaluation("cost-limit", "allow", 2*time.Millisecond)
	}
}

// Benchmark_Collector_RecordCacheHit benchmarks cache hit recording
func Benchmark_Collector_RecordCacheHit(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordCacheHit("policy")
	}
}

// Benchmark_RequestMetrics_RecordRequest benchmarks raw request metric recording
func Benchmark_RequestMetrics_RecordRequest(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	rm := NewRequestMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.RecordRequest("openai", "gpt-4", "success", time.Second, 1500)
	}
}

// Benchmark_RequestMetrics_RecordTokens benchmarks token recording
func Benchmark_RequestMetrics_RecordTokens(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	rm := NewRequestMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.RecordTokens("openai", "gpt-4", 1000, 500)
	}
}

// Benchmark_ProviderMetrics_UpdateHealth benchmarks health updates
func Benchmark_ProviderMetrics_UpdateHealth(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	pm := NewProviderMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.UpdateHealth("openai", true)
	}
}

// Benchmark_ProviderMetrics_RecordLatency benchmarks latency recording
func Benchmark_ProviderMetrics_RecordLatency(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	pm := NewProviderMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.RecordLatency("openai", "gpt-4", 0.95)
	}
}

// Benchmark_PolicyMetrics_RecordEvaluation benchmarks policy evaluation recording
func Benchmark_PolicyMetrics_RecordEvaluation(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	pm := NewPolicyMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.RecordEvaluation("cost-limit", "allow", 2*time.Millisecond)
	}
}

// Benchmark_CostMetrics_RecordRequestCost benchmarks cost recording
func Benchmark_CostMetrics_RecordRequestCost(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	cm := NewCostMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.RecordRequestCost("openai", "gpt-4", 0.05)
	}
}

// Benchmark_CacheMetrics_RecordHit benchmarks cache hit recording
func Benchmark_CacheMetrics_RecordHit(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	cm := NewCacheMetrics(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.RecordHit("policy")
	}
}

// Benchmark_CardinalityLimiter_Allow benchmarks cardinality checking
func Benchmark_CardinalityLimiter_Allow(b *testing.B) {
	limiter := NewCardinalityLimiter(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow("label1")
	}
}

// Benchmark_CardinalityLimiter_Allow_New benchmarks cardinality checking with new labels
func Benchmark_CardinalityLimiter_Allow_New(b *testing.B) {
	limiter := NewCardinalityLimiter(100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow("label" + string(rune(i)))
	}
}

// Benchmark_Collector_Disabled benchmarks metrics when disabled
func Benchmark_Collector_Disabled(b *testing.B) {
	cfg := testConfig()
	cfg.Enabled = false
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordRequest("openai", "gpt-4", "success", time.Second, 1500, 0.05)
	}
}

// Benchmark_Collector_ManyLabels benchmarks recording with many different label values
func Benchmark_Collector_ManyLabels(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	providers := []string{"openai", "anthropic", "google", "cohere"}
	models := []string{"gpt-4", "claude-3-opus", "gemini-pro", "command-r"}
	statuses := []string{"success", "error", "blocked"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider := providers[i%len(providers)]
		model := models[i%len(models)]
		status := statuses[i%len(statuses)]
		collector.RecordRequest(provider, model, status, time.Second, 1500, 0.05)
	}
}

// Benchmark_Collector_AllMetrics benchmarks recording all metric types
func Benchmark_Collector_AllMetrics(b *testing.B) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Record request
		collector.RecordRequest("openai", "gpt-4", "success", time.Second, 1500, 0.05)

		// Update provider health
		collector.UpdateProviderHealth("openai", true)

		// Record policy evaluation
		collector.RecordPolicyEvaluation("cost-limit", "allow", 2*time.Millisecond)

		// Record cache hit
		collector.RecordCacheHit("policy")
	}
}
