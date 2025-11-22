package metrics

import (
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Helper function to create test config
func testConfig() *config.MetricsConfig {
	return &config.MetricsConfig{
		Enabled:                true,
		Namespace:              "test",
		Subsystem:              "metrics",
		RequestDurationBuckets: []float64{0.1, 0.5, 1.0, 5.0},
		TokenCountBuckets:      []float64{100, 500, 1000, 5000},
	}
}

// TestCollector_NewCollector tests collector creation
func TestCollector_NewCollector(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()

	collector := NewCollector(cfg, registry)

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}
	if collector.config != cfg {
		t.Error("Collector config not set correctly")
	}
	if collector.registry != registry {
		t.Error("Collector registry not set correctly")
	}
}

// TestCollector_RecordRequest tests request recording
func TestCollector_RecordRequest(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	tests := []struct {
		name     string
		provider string
		model    string
		status   string
		duration time.Duration
		tokens   int
		cost     float64
	}{
		{
			name:     "success request",
			provider: "openai",
			model:    "gpt-4",
			status:   "success",
			duration: 1200 * time.Millisecond,
			tokens:   1500,
			cost:     0.05,
		},
		{
			name:     "error request",
			provider: "anthropic",
			model:    "claude-3-opus",
			status:   "error",
			duration: 500 * time.Millisecond,
			tokens:   0,
			cost:     0.0,
		},
		{
			name:     "blocked request",
			provider: "openai",
			model:    "gpt-4",
			status:   "blocked",
			duration: 10 * time.Millisecond,
			tokens:   0,
			cost:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector.RecordRequest(tt.provider, tt.model, tt.status, tt.duration, tt.tokens, tt.cost)

			// Verify request counter was incremented
			count := testutil.ToFloat64(collector.requestMetrics.requestsTotal.WithLabelValues(tt.provider, tt.model, tt.status))
			if count < 1 {
				t.Errorf("Expected request counter >= 1, got %f", count)
			}
		})
	}
}

// TestCollector_ProviderMetrics tests provider metric recording
func TestCollector_ProviderMetrics(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	// Test health update
	t.Run("update health", func(t *testing.T) {
		collector.UpdateProviderHealth("openai", true)
		health := testutil.ToFloat64(collector.providerMetrics.health.WithLabelValues("openai"))
		if health != 1.0 {
			t.Errorf("Expected health=1.0, got %f", health)
		}

		collector.UpdateProviderHealth("openai", false)
		health = testutil.ToFloat64(collector.providerMetrics.health.WithLabelValues("openai"))
		if health != 0.0 {
			t.Errorf("Expected health=0.0, got %f", health)
		}
	})

	// Test latency recording
	t.Run("record latency", func(t *testing.T) {
		collector.RecordProviderLatency("openai", "gpt-4", 0.95)
		// Just verify it doesn't panic
	})

	// Test error recording
	t.Run("record error", func(t *testing.T) {
		collector.RecordProviderError("openai", "rate_limit")
		count := testutil.ToFloat64(collector.providerMetrics.errors.WithLabelValues("openai", "rate_limit"))
		if count < 1 {
			t.Errorf("Expected error count >= 1, got %f", count)
		}
	})
}

// TestCollector_PolicyMetrics tests policy metric recording
func TestCollector_PolicyMetrics(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	// Test evaluation recording
	t.Run("record evaluation", func(t *testing.T) {
		collector.RecordPolicyEvaluation("cost-limit", "allow", 2*time.Millisecond)
		count := testutil.ToFloat64(collector.policyMetrics.evaluationsTotal.WithLabelValues("cost-limit", "allow"))
		if count < 1 {
			t.Errorf("Expected evaluation count >= 1, got %f", count)
		}
	})

	// Test hit recording
	t.Run("record hit", func(t *testing.T) {
		collector.RecordPolicyHit("cost-limit")
		count := testutil.ToFloat64(collector.policyMetrics.hitsTotal.WithLabelValues("cost-limit"))
		if count < 1 {
			t.Errorf("Expected hit count >= 1, got %f", count)
		}
	})

	// Test miss recording
	t.Run("record miss", func(t *testing.T) {
		collector.RecordPolicyMiss("cost-limit")
		count := testutil.ToFloat64(collector.policyMetrics.missesTotal.WithLabelValues("cost-limit"))
		if count < 1 {
			t.Errorf("Expected miss count >= 1, got %f", count)
		}
	})
}

// TestCollector_CacheMetrics tests cache metric recording
func TestCollector_CacheMetrics(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	// Test hit recording
	t.Run("record cache hit", func(t *testing.T) {
		collector.RecordCacheHit("policy")
		count := testutil.ToFloat64(collector.cacheMetrics.hitsTotal.WithLabelValues("policy"))
		if count < 1 {
			t.Errorf("Expected hit count >= 1, got %f", count)
		}
	})

	// Test miss recording
	t.Run("record cache miss", func(t *testing.T) {
		collector.RecordCacheMiss("policy")
		count := testutil.ToFloat64(collector.cacheMetrics.missesTotal.WithLabelValues("policy"))
		if count < 1 {
			t.Errorf("Expected miss count >= 1, got %f", count)
		}
	})

	// Test size update
	t.Run("update cache size", func(t *testing.T) {
		collector.UpdateCacheSize("policy", 42)
		size := testutil.ToFloat64(collector.cacheMetrics.entries.WithLabelValues("policy"))
		if size != 42 {
			t.Errorf("Expected size=42, got %f", size)
		}
	})
}

// TestCollector_Disabled tests that metrics are not recorded when disabled
func TestCollector_Disabled(t *testing.T) {
	cfg := testConfig()
	cfg.Enabled = false
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	// These should not panic
	collector.RecordRequest("openai", "gpt-4", "success", time.Second, 1000, 0.05)
	collector.UpdateProviderHealth("openai", true)
	collector.RecordPolicyEvaluation("test", "allow", time.Millisecond)
	collector.RecordCacheHit("policy")
}

// TestCardinalityLimiter tests cardinality limiting
func TestCardinalityLimiter(t *testing.T) {
	limiter := NewCardinalityLimiter(3)

	// First 3 should be allowed
	if !limiter.Allow("label1") {
		t.Error("Expected first label to be allowed")
	}
	if !limiter.Allow("label2") {
		t.Error("Expected second label to be allowed")
	}
	if !limiter.Allow("label3") {
		t.Error("Expected third label to be allowed")
	}

	// Fourth should be rejected
	if limiter.Allow("label4") {
		t.Error("Expected fourth label to be rejected")
	}

	// Existing labels should still be allowed
	if !limiter.Allow("label1") {
		t.Error("Expected existing label to be allowed")
	}

	// Check count
	if limiter.Count() != 3 {
		t.Errorf("Expected count=3, got %d", limiter.Count())
	}
}

// TestRequestMetrics_RecordTokens tests token recording
func TestRequestMetrics_RecordTokens(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	rm := NewRequestMetrics(cfg, registry)

	rm.RecordTokens("openai", "gpt-4", 1000, 500)

	// Verify prompt tokens
	promptCount := testutil.ToFloat64(rm.tokensTotal.WithLabelValues("openai", "gpt-4", "prompt"))
	if promptCount < 1000 {
		t.Errorf("Expected prompt tokens >= 1000, got %f", promptCount)
	}

	// Verify completion tokens
	completionCount := testutil.ToFloat64(rm.tokensTotal.WithLabelValues("openai", "gpt-4", "completion"))
	if completionCount < 500 {
		t.Errorf("Expected completion tokens >= 500, got %f", completionCount)
	}
}

// TestRequestMetrics_RecordSize tests size recording
func TestRequestMetrics_RecordSize(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	rm := NewRequestMetrics(cfg, registry)

	rm.RecordSize("openai", "gpt-4", "request", 5120)
	rm.RecordSize("openai", "gpt-4", "response", 10240)

	// Just verify it doesn't panic
}

// TestProviderMetrics_RecordRequest tests provider request recording
func TestProviderMetrics_RecordRequest(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	pm := NewProviderMetrics(cfg, registry)

	pm.RecordRequest("openai", "gpt-4")
	count := testutil.ToFloat64(pm.requests.WithLabelValues("openai", "gpt-4"))
	if count < 1 {
		t.Errorf("Expected request count >= 1, got %f", count)
	}
}

// TestCostMetrics_RecordRequestCost tests cost recording
func TestCostMetrics_RecordRequestCost(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	cm := NewCostMetrics(cfg, registry)

	cm.RecordRequestCost("openai", "gpt-4", 0.05)

	// Verify cost was recorded
	cost := testutil.ToFloat64(cm.costTotal.WithLabelValues("openai", "gpt-4"))
	if cost < 0.05 {
		t.Errorf("Expected cost >= 0.05, got %f", cost)
	}
}

// TestCostMetrics_UpdateCostPerToken tests cost per token update
func TestCostMetrics_UpdateCostPerToken(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	cm := NewCostMetrics(cfg, registry)

	cm.UpdateCostPerToken("openai", "gpt-4", 0.00003)

	// Verify cost per token was set
	costPerToken := testutil.ToFloat64(cm.costPerToken.WithLabelValues("openai", "gpt-4"))
	if costPerToken != 0.00003 {
		t.Errorf("Expected cost per token = 0.00003, got %f", costPerToken)
	}
}

// TestCacheMetrics_RecordEviction tests eviction recording
func TestCacheMetrics_RecordEviction(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	cm := NewCacheMetrics(cfg, registry)

	cm.RecordEviction("policy")

	// Verify eviction was recorded
	count := testutil.ToFloat64(cm.evictionsTotal.WithLabelValues("policy"))
	if count < 1 {
		t.Errorf("Expected eviction count >= 1, got %f", count)
	}
}

// TestCollector_ConcurrentRecording tests thread-safety
func TestCollector_ConcurrentRecording(t *testing.T) {
	cfg := testConfig()
	registry := prometheus.NewRegistry()
	collector := NewCollector(cfg, registry)

	done := make(chan bool)

	// Spawn multiple goroutines recording metrics
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.RecordRequest("openai", "gpt-4", "success", time.Second, 1000, 0.05)
				collector.UpdateProviderHealth("openai", true)
				collector.RecordPolicyEvaluation("test", "allow", time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify we got all requests recorded
	count := testutil.ToFloat64(collector.requestMetrics.requestsTotal.WithLabelValues("openai", "gpt-4", "success"))
	if count != 1000 {
		t.Errorf("Expected 1000 requests, got %f", count)
	}
}
