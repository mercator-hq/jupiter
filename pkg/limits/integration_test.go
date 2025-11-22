package limits

import (
	"context"
	"sync"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/limits/budget"
	"mercator-hq/jupiter/pkg/limits/enforcement"
	"mercator-hq/jupiter/pkg/limits/ratelimit"
)

// TestIntegration_EndToEnd tests the complete flow from limit check to usage recording.
func TestIntegration_EndToEnd(t *testing.T) {
	// Create manager with both rate limits and budgets
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 100,
				TokensPerMinute:   100000,
				MaxConcurrent:     10,
			},
		},
		Budgets: map[string]budget.Config{
			"test-key": {
				Hourly:         10.00,
				Daily:          200.00,
				Monthly:        5000.00,
				AlertThreshold: 0.8,
			},
		},
		Enforcement: enforcement.Config{
			DefaultAction: enforcement.ActionBlock,
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Simulate 10 requests
	for i := 0; i < 10; i++ {
		// Check limits
		result, err := manager.CheckLimits(ctx, "test-key", 1000, 0.05, "gpt-4")
		if err != nil {
			t.Fatalf("Request %d: CheckLimits failed: %v", i, err)
		}

		if !result.Allowed {
			t.Fatalf("Request %d: Expected to be allowed, reason: %s", i, result.Reason)
		}

		// Acquire concurrent slot
		if !manager.AcquireConcurrent("test-key") {
			t.Fatalf("Request %d: Failed to acquire concurrent slot", i)
		}

		// Record usage
		err = manager.RecordUsage(ctx, &UsageRecord{
			Identifier:     "test-key",
			Dimension:      DimensionAPIKey,
			RequestTokens:  1000,
			ResponseTokens: 500,
			TotalTokens:    1500,
			Cost:           0.05,
			Provider:       "openai",
			Model:          "gpt-4",
		})
		if err != nil {
			t.Fatalf("Request %d: RecordUsage failed: %v", i, err)
		}

		// Release concurrent slot
		manager.ReleaseConcurrent("test-key")
	}

	// Verify usage was recorded
	// After 10 requests @ $0.05 each = $0.50 spent
	// Should still be under $10/hour limit
	result, err := manager.CheckLimits(ctx, "test-key", 1000, 0.05, "gpt-4")
	if err != nil {
		t.Fatalf("Final check failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Expected final request to be allowed")
	}
}

// TestIntegration_MultiDimension tests limits across different dimensions.
func TestIntegration_MultiDimension(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"key-1": {RequestsPerSecond: 10},
			"key-2": {RequestsPerSecond: 5},
			"key-3": {RequestsPerSecond: 20},
		},
		Budgets: map[string]budget.Config{
			"key-1": {Daily: 100.00},
			"key-2": {Daily: 50.00},
			"key-3": {Daily: 200.00},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Test all three keys independently
	keys := []string{"key-1", "key-2", "key-3"}
	for _, key := range keys {
		result, err := manager.CheckLimits(ctx, key, 0, 0, "gpt-4")
		if err != nil {
			t.Fatalf("CheckLimits for %s failed: %v", key, err)
		}

		if !result.Allowed {
			t.Errorf("Expected %s to be allowed", key)
		}
	}

	// Verify they have independent budgets
	manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "key-1",
		Dimension:  DimensionAPIKey,
		Cost:       60.00, // Exceeds key-2 budget but not key-1
	})

	// key-1 should still be allowed (budget is $100)
	result, _ := manager.CheckLimits(ctx, "key-1", 0, 0, "gpt-4")
	if !result.Allowed {
		t.Error("Expected key-1 to still be allowed")
	}

	// key-2 should be unaffected (independent budget)
	result, _ = manager.CheckLimits(ctx, "key-2", 0, 0, "gpt-4")
	if !result.Allowed {
		t.Error("Expected key-2 to be allowed (independent budget)")
	}
}

// TestIntegration_AlertThreshold tests alert triggering at threshold.
func TestIntegration_AlertThreshold(t *testing.T) {
	config := Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily:          10.00,
				AlertThreshold: 0.8, // 80%
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Use 70% of budget - should not trigger alert
	manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       7.00,
	})

	result, _ := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if result.Action == ActionAlert {
		t.Error("Expected no alert at 70% usage")
	}

	// Use another 15% (total 85%) - should trigger alert
	manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       1.50,
	})

	result, _ = manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if result.Action != ActionAlert {
		t.Errorf("Expected alert at 85%% usage, got action: %s", result.Action)
	}
	if !result.Allowed {
		t.Error("Expected request to still be allowed with alert")
	}
}

// TestIntegration_ConcurrentLoad tests handling of concurrent requests.
func TestIntegration_ConcurrentLoad(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"load-test": {
				RequestsPerSecond: 1000,
				MaxConcurrent:     50,
			},
		},
		Budgets: map[string]budget.Config{
			"load-test": {
				Daily: 10000.00,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Simulate 100 concurrent requests
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Check limits
			result, err := manager.CheckLimits(ctx, "load-test", 100, 0.01, "gpt-4")
			if err != nil {
				t.Errorf("Request %d: CheckLimits failed: %v", id, err)
				return
			}

			if !result.Allowed {
				// Expected - some will be rejected due to concurrent limit
				return
			}

			// Try to acquire concurrent slot
			if !manager.AcquireConcurrent("load-test") {
				// Expected - concurrent limit reached
				return
			}

			mu.Lock()
			successCount++
			mu.Unlock()

			// Simulate processing
			time.Sleep(10 * time.Millisecond)

			// Release slot
			manager.ReleaseConcurrent("load-test")

			// Record usage
			manager.RecordUsage(ctx, &UsageRecord{
				Identifier: "load-test",
				Dimension:  DimensionAPIKey,
				Cost:       0.01,
			})
		}(i)
	}

	wg.Wait()

	// Should have processed some requests (up to concurrent limit)
	if successCount == 0 {
		t.Error("Expected at least some requests to succeed")
	}

	// Should not exceed concurrent limit
	if successCount > 50 {
		t.Errorf("Expected at most 50 concurrent requests, got %d", successCount)
	}
}

// TestIntegration_ModelDowngrade tests automatic model downgrade.
func TestIntegration_ModelDowngrade(t *testing.T) {
	config := Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily: 5.00, // Low budget
			},
		},
		Enforcement: enforcement.Config{
			DefaultAction: enforcement.ActionDowngrade,
			ModelDowngrades: map[string]string{
				"gpt-4":         "gpt-3.5-turbo",
				"claude-3-opus": "claude-3-sonnet",
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Exceed budget
	manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       6.00,
	})

	// Request GPT-4 - should downgrade to GPT-3.5
	result, err := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed with downgrade")
	}

	if result.Action != ActionDowngrade {
		t.Errorf("Expected downgrade action, got: %s", result.Action)
	}

	if result.DowngradeTo != "gpt-3.5-turbo" {
		t.Errorf("Expected downgrade to gpt-3.5-turbo, got: %s", result.DowngradeTo)
	}
}

// TestIntegration_LoadTest simulates high load with many API keys.
func TestIntegration_LoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Create config for 100 API keys (scaled down from 10K for test speed)
	rateLimits := make(map[string]ratelimit.Config)
	budgets := make(map[string]budget.Config)

	for i := 0; i < 100; i++ {
		key := "load-key-" + string(rune('0'+i%10))
		rateLimits[key] = ratelimit.Config{
			RequestsPerSecond: 100,
		}
		budgets[key] = budget.Config{
			Daily: 100.00,
		}
	}

	config := Config{
		RateLimits: rateLimits,
		Budgets:    budgets,
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Simulate 1000 requests across all keys
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := "load-key-" + string(rune('0'+id%10))

			// Check limits
			result, err := manager.CheckLimits(ctx, key, 100, 0.01, "gpt-4")
			if err != nil {
				t.Errorf("Request %d: CheckLimits failed: %v", id, err)
				return
			}

			if result.Allowed {
				// Record usage
				manager.RecordUsage(ctx, &UsageRecord{
					Identifier: key,
					Dimension:  DimensionAPIKey,
					Cost:       0.01,
				})
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Should complete in reasonable time (< 1 second for 1000 requests)
	if duration > time.Second {
		t.Errorf("Load test took too long: %v", duration)
	}

	t.Logf("Processed 1000 requests in %v (%.2f req/s)",
		duration, float64(1000)/duration.Seconds())
}

// TestIntegration_RollingWindow tests rolling window behavior over time.
func TestIntegration_RollingWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping time-based test in short mode")
	}

	config := Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Hourly: 1.00, // Small hourly budget for testing
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Use most of hourly budget
	manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       0.90,
	})

	// Should be close to limit
	result, _ := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if !result.Allowed {
		t.Error("Expected to be under limit at 90%")
	}

	// Add more to exceed
	manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       0.15,
	})

	// Should now exceed hourly limit
	result, _ = manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if result.Allowed {
		t.Error("Expected to exceed hourly limit at 105%")
	}

	// Note: Full rolling window test would require waiting for time to pass,
	// which is impractical for unit tests. This test verifies accumulation works.
}
