package limits

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/limits/budget"
	"mercator-hq/jupiter/pkg/limits/enforcement"
	"mercator-hq/jupiter/pkg/limits/ratelimit"
)

func TestNewManager_Basic(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 10,
			},
		},
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily: 100.00,
			},
		},
		Enforcement: enforcement.Config{
			DefaultAction: enforcement.ActionBlock,
		},
	}

	manager := NewManager(config)
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	defer manager.Close()

	// Verify pre-initialized limiters
	if len(manager.rateLimiters) != 1 {
		t.Errorf("Expected 1 rate limiter, got %d", len(manager.rateLimiters))
	}
	if len(manager.budgets) != 1 {
		t.Errorf("Expected 1 budget tracker, got %d", len(manager.budgets))
	}
}

func TestManager_CheckLimits_Allow(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 100,
				TokensPerMinute:   100000,
			},
		},
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily: 100.00,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Should allow request within limits
	result, err := manager.CheckLimits(ctx, "test-key", 1000, 0.05, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Expected request to be allowed, reason: %s", result.Reason)
	}
}

func TestManager_CheckLimits_RateLimitExceeded(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				RequestsPerSecond: 2,
			},
		},
		Enforcement: enforcement.Config{
			DefaultAction: enforcement.ActionBlock,
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Exhaust rate limit (burst capacity is 2x, so 4 requests)
	for i := 0; i < 4; i++ {
		_, _ = manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	}

	// Next request should be blocked
	result, err := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits failed: %v", err)
	}

	if result.Allowed {
		t.Error("Expected request to be blocked due to rate limit")
	}
	if result.RateLimit == nil {
		t.Error("Expected rate limit info to be populated")
	}
}

func TestManager_CheckLimits_BudgetExceeded(t *testing.T) {
	config := Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily: 10.00,
			},
		},
		Enforcement: enforcement.Config{
			DefaultAction: enforcement.ActionBlock,
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// First record usage that exceeds budget
	_ = manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       15.00,
	})

	// Now check should fail due to budget
	result, err := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits failed: %v", err)
	}

	if result.Allowed {
		t.Error("Expected request to be blocked due to budget limit")
	}
	if result.Budget == nil {
		t.Error("Expected budget info to be populated")
	}
}

func TestManager_CheckLimits_AlertThreshold(t *testing.T) {
	config := Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily:          10.00,
				AlertThreshold: 0.8,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Record usage to reach threshold
	_ = manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       8.50, // 85% of budget
	})

	// Check should trigger alert but still allow
	result, err := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed with alert")
	}
	if result.Action != ActionAlert {
		t.Errorf("Expected action Alert, got %s", result.Action)
	}
}

func TestManager_RecordUsage(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				TokensPerMinute: 10000,
			},
		},
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily: 100.00,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Record usage
	err := manager.RecordUsage(ctx, &UsageRecord{
		Identifier:     "test-key",
		Dimension:      DimensionAPIKey,
		RequestTokens:  1000,
		ResponseTokens: 500,
		TotalTokens:    1500,
		Cost:           5.00,
		Provider:       "openai",
		Model:          "gpt-4",
	})
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	// Give async persistence a moment
	time.Sleep(10 * time.Millisecond)

	// Verify usage was recorded (check via budget tracker)
	tracker := manager.budgets["test-key"]
	if tracker == nil {
		t.Fatal("Expected budget tracker to exist")
	}

	total := tracker.GetTotalSpent()
	if total != 5.00 {
		t.Errorf("Expected total spent 5.00, got %.2f", total)
	}
}

func TestManager_ConcurrentLimits(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"test-key": {
				MaxConcurrent: 3,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	// Acquire 3 slots
	for i := 0; i < 3; i++ {
		if !manager.AcquireConcurrent("test-key") {
			t.Errorf("Failed to acquire slot %d", i)
		}
	}

	// 4th should fail
	if manager.AcquireConcurrent("test-key") {
		t.Error("Expected 4th acquisition to fail")
	}

	// Release one
	manager.ReleaseConcurrent("test-key")

	// Should work now
	if !manager.AcquireConcurrent("test-key") {
		t.Error("Expected acquisition to succeed after release")
	}
}

func TestManager_NoLimits(t *testing.T) {
	// Manager with no limits configured
	config := Config{}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Should allow unlimited requests
	for i := 0; i < 100; i++ {
		result, err := manager.CheckLimits(ctx, "test-key", 10000, 10.00, "gpt-4")
		if err != nil {
			t.Fatalf("CheckLimits failed: %v", err)
		}
		if !result.Allowed {
			t.Error("Expected request to be allowed with no limits")
		}
	}
}

func TestManager_MultipleIdentifiers(t *testing.T) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"key-1": {
				RequestsPerSecond: 10,
			},
			"key-2": {
				RequestsPerSecond: 5,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Check limits for both keys
	result1, err := manager.CheckLimits(ctx, "key-1", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits for key-1 failed: %v", err)
	}
	if !result1.Allowed {
		t.Error("Expected key-1 to be allowed")
	}

	result2, err := manager.CheckLimits(ctx, "key-2", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits for key-2 failed: %v", err)
	}
	if !result2.Allowed {
		t.Error("Expected key-2 to be allowed")
	}

	// Verify they have independent limits
	if manager.rateLimiters["key-1"] == manager.rateLimiters["key-2"] {
		t.Error("Expected independent rate limiters for different keys")
	}
}

func TestManager_Downgrade(t *testing.T) {
	config := Config{
		Budgets: map[string]budget.Config{
			"test-key": {
				Daily: 1.00, // Very low budget
			},
		},
		Enforcement: enforcement.Config{
			DefaultAction: enforcement.ActionDowngrade,
			ModelDowngrades: map[string]string{
				"gpt-4": "gpt-3.5-turbo",
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	// Record usage that exceeds budget
	_ = manager.RecordUsage(ctx, &UsageRecord{
		Identifier: "test-key",
		Dimension:  DimensionAPIKey,
		Cost:       2.00, // Exceeds daily limit of 1.00
	})

	// Request expensive model - should downgrade
	result, err := manager.CheckLimits(ctx, "test-key", 0, 0, "gpt-4")
	if err != nil {
		t.Fatalf("CheckLimits failed: %v", err)
	}

	// Should downgrade instead of blocking
	if !result.Allowed {
		t.Error("Expected request to be allowed with downgrade")
	}
	if result.Action != ActionDowngrade {
		t.Errorf("Expected action Downgrade, got %s", result.Action)
	}
	if result.DowngradeTo != "gpt-3.5-turbo" {
		t.Errorf("Expected downgrade to gpt-3.5-turbo, got %s", result.DowngradeTo)
	}
}

func BenchmarkManager_CheckLimits(b *testing.B) {
	config := Config{
		RateLimits: map[string]ratelimit.Config{
			"bench-key": {
				RequestsPerSecond: 10000,
				TokensPerMinute:   1000000,
			},
		},
		Budgets: map[string]budget.Config{
			"bench-key": {
				Daily: 10000.00,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.CheckLimits(ctx, "bench-key", 1000, 0.05, "gpt-4")
	}
}

func BenchmarkManager_RecordUsage(b *testing.B) {
	config := Config{
		Budgets: map[string]budget.Config{
			"bench-key": {
				Daily: 10000.00,
			},
		},
	}

	manager := NewManager(config)
	defer manager.Close()

	ctx := context.Background()
	record := &UsageRecord{
		Identifier:     "bench-key",
		Dimension:      DimensionAPIKey,
		RequestTokens:  1000,
		ResponseTokens: 500,
		TotalTokens:    1500,
		Cost:           0.05,
		Provider:       "openai",
		Model:          "gpt-4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.RecordUsage(ctx, record)
	}
}
