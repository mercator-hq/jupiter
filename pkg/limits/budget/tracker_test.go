package budget

import (
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Rolling Window Tests
// ============================================================================

func TestRollingWindow_Basic(t *testing.T) {
	rw := NewRollingWindow(time.Minute, time.Second)

	// Add spending
	rw.Add(10.50)
	rw.Add(5.25)
	rw.Add(3.75)

	// Sum should be total
	sum := rw.Sum()
	expected := 19.50
	if sum != expected {
		t.Errorf("Expected sum %.2f, got %.2f", expected, sum)
	}
}

func TestRollingWindow_Expiration(t *testing.T) {
	rw := NewRollingWindow(100*time.Millisecond, 10*time.Millisecond)

	// Add spending
	rw.Add(25.00)

	// Should be present immediately
	if rw.Sum() != 25.00 {
		t.Error("Expected spending to be present")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	sum := rw.Sum()
	if sum != 0 {
		t.Errorf("Expected 0 after expiration, got %.2f", sum)
	}
}

func TestRollingWindow_RollingBehavior(t *testing.T) {
	// Use longer windows and more generous timing to handle race detector overhead
	// Race detector can add significant latency, making precise timing unreliable
	rw := NewRollingWindow(5*time.Second, 500*time.Millisecond)

	// T=0: Add $10
	start := time.Now()
	rw.Add(10.00)
	time.Sleep(2600 * time.Millisecond) // Wait well into a later bucket (2.6s)

	// T=~2600ms: Add $20 (different bucket, in the 2500ms bucket)
	// This provides 2.5s separation from the first bucket
	rw.Add(20.00)
	time.Sleep(200 * time.Millisecond)

	// T=~2800ms: Should have $30 (both buckets within 5s window)
	sum := rw.Sum()
	if sum != 30.00 {
		t.Errorf("Expected 30.00 before expiration, got %.2f", sum)
	}

	// Wait for first spending to expire (window is 5s, first entry at T=0)
	// Sleep until T=7s (first bucket at T=0 expires at T=5, extra 2s margin)
	// Second bucket at T=~2.5s is still within window (7 - 5 = 2s cutoff, 2.5s > 2s)
	elapsed := time.Since(start)
	targetTime := 7 * time.Second
	remainingSleep := targetTime - elapsed
	if remainingSleep > 0 {
		time.Sleep(remainingSleep)
	}

	// First spending should be expired (>5s old), only second remains
	sum = rw.Sum()
	if sum != 20.00 {
		t.Errorf("Expected 20.00 after first expiration, got %.2f (elapsed: %v)", sum, time.Since(start))
	}
}

func TestRollingWindow_Reset(t *testing.T) {
	rw := NewRollingWindow(time.Hour, time.Minute)

	rw.Add(100.00)
	rw.Add(50.00)

	rw.Reset()

	sum := rw.Sum()
	if sum != 0 {
		t.Errorf("Expected 0 after reset, got %.2f", sum)
	}
}

func TestRollingWindow_OldestTimestamp(t *testing.T) {
	rw := NewRollingWindow(time.Hour, time.Minute)

	// Empty window has zero timestamp
	oldest := rw.OldestTimestamp()
	if !oldest.IsZero() {
		t.Error("Expected zero timestamp for empty window")
	}

	// Add spending
	before := time.Now().Truncate(time.Minute) // Truncate to bucket boundary
	rw.Add(10.00)
	after := time.Now().Add(time.Minute) // Allow for bucket rounding

	oldest = rw.OldestTimestamp()
	if oldest.Before(before) || oldest.After(after) {
		t.Errorf("Oldest timestamp %v should be between %v and %v", oldest, before, after)
	}
}

func TestRollingWindow_Concurrent(t *testing.T) {
	rw := NewRollingWindow(time.Minute, time.Second)

	var wg sync.WaitGroup
	numGoroutines := 10
	amountPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < amountPerGoroutine; j++ {
				rw.Add(1.00)
			}
		}()
	}

	wg.Wait()

	expected := float64(numGoroutines * amountPerGoroutine)
	sum := rw.Sum()
	if sum != expected {
		t.Errorf("Expected sum %.2f, got %.2f", expected, sum)
	}
}

// ============================================================================
// Budget Tracker Tests
// ============================================================================

func TestTracker_HourlyLimit(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly: 10.00,
	})

	// Add spending within limit
	tracker.Add(5.00)

	status := tracker.Check()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}

	// Add more spending to exceed limit
	tracker.Add(6.00) // Total: 11.00

	status = tracker.Check()
	if status.Allowed {
		t.Error("Expected spending to be blocked")
	}
	if status.Reason != "hourly budget limit exceeded" {
		t.Errorf("Expected hourly limit exceeded, got: %s", status.Reason)
	}
	if status.Used != 11.00 {
		t.Errorf("Expected used 11.00, got %.2f", status.Used)
	}
}

func TestTracker_DailyLimit(t *testing.T) {
	tracker := NewTracker(Config{
		Daily: 100.00,
	})

	// Add spending within limit
	tracker.Add(75.00)

	status := tracker.Check()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}

	// Exceed limit
	tracker.Add(30.00) // Total: 105.00

	status = tracker.Check()
	if status.Allowed {
		t.Error("Expected spending to be blocked")
	}
	if status.Reason != "daily budget limit exceeded" {
		t.Errorf("Expected daily limit exceeded, got: %s", status.Reason)
	}
}

func TestTracker_MonthlyLimit(t *testing.T) {
	tracker := NewTracker(Config{
		Monthly: 1000.00,
	})

	// Add spending within limit
	tracker.Add(500.00)

	status := tracker.Check()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}

	// Exceed limit
	tracker.Add(600.00) // Total: 1100.00

	status = tracker.Check()
	if status.Allowed {
		t.Error("Expected spending to be blocked")
	}
	if status.Reason != "monthly budget limit exceeded" {
		t.Errorf("Expected monthly limit exceeded, got: %s", status.Reason)
	}
}

func TestTracker_MultipleLimits(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly:  10.00,
		Daily:   100.00,
		Monthly: 1000.00,
	})

	// Add spending within all limits
	tracker.Add(5.00)

	status := tracker.Check()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}

	// Exceed hourly limit (most restrictive)
	tracker.Add(6.00) // Total: 11.00

	status = tracker.Check()
	if status.Allowed {
		t.Error("Expected spending to be blocked")
	}
	// Should report hourly limit (most restrictive)
	if status.Reason != "hourly budget limit exceeded" {
		t.Errorf("Expected hourly limit to be reported first, got: %s", status.Reason)
	}
}

func TestTracker_AlertThreshold(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly:         10.00,
		AlertThreshold: 0.8, // Alert at 80%
	})

	// Add spending below threshold
	tracker.Add(7.00) // 70%

	status := tracker.Check()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}
	if status.AlertTriggered {
		t.Error("Expected no alert below threshold")
	}

	// Add spending to reach threshold
	tracker.Add(1.50) // 85%

	status = tracker.Check()
	if !status.Allowed {
		t.Error("Expected spending to still be allowed")
	}
	if !status.AlertTriggered {
		t.Error("Expected alert to be triggered")
	}
	if status.Percentage < 0.8 {
		t.Errorf("Expected percentage >= 0.8, got %.2f", status.Percentage)
	}
}

func TestTracker_GetHourlyStatus(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly: 10.00,
	})

	tracker.Add(3.50)

	status := tracker.GetHourlyStatus()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}
	if status.Used != 3.50 {
		t.Errorf("Expected used 3.50, got %.2f", status.Used)
	}
	if status.Remaining != 6.50 {
		t.Errorf("Expected remaining 6.50, got %.2f", status.Remaining)
	}
	if status.Percentage != 0.35 {
		t.Errorf("Expected percentage 0.35, got %.2f", status.Percentage)
	}
}

func TestTracker_GetDailyStatus(t *testing.T) {
	tracker := NewTracker(Config{
		Daily: 100.00,
	})

	tracker.Add(25.00)

	status := tracker.GetDailyStatus()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}
	if status.Used != 25.00 {
		t.Errorf("Expected used 25.00, got %.2f", status.Used)
	}
	if status.Remaining != 75.00 {
		t.Errorf("Expected remaining 75.00, got %.2f", status.Remaining)
	}
}

func TestTracker_GetMonthlyStatus(t *testing.T) {
	tracker := NewTracker(Config{
		Monthly: 1000.00,
	})

	tracker.Add(250.00)

	status := tracker.GetMonthlyStatus()
	if !status.Allowed {
		t.Error("Expected spending to be allowed")
	}
	if status.Used != 250.00 {
		t.Errorf("Expected used 250.00, got %.2f", status.Used)
	}
	if status.Remaining != 750.00 {
		t.Errorf("Expected remaining 750.00, got %.2f", status.Remaining)
	}
}

func TestTracker_GetTotalSpent(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly: 10.00,
	})

	tracker.Add(3.00)
	tracker.Add(5.00)
	tracker.Add(2.00)

	total := tracker.GetTotalSpent()
	expected := 10.00
	if total != expected {
		t.Errorf("Expected total %.2f, got %.2f", expected, total)
	}
}

func TestTracker_NoLimits(t *testing.T) {
	tracker := NewTracker(Config{})

	// Should allow unlimited spending
	for i := 0; i < 100; i++ {
		tracker.Add(100.00)

		status := tracker.Check()
		if !status.Allowed {
			t.Errorf("Expected spending to be allowed with no limits")
		}
	}
}

func TestTracker_Reset(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly:  10.00,
		Daily:   100.00,
		Monthly: 1000.00,
	})

	// Add spending
	tracker.Add(50.00)

	// Verify spending recorded
	if tracker.GetTotalSpent() != 50.00 {
		t.Error("Expected spending to be recorded")
	}

	// Reset
	tracker.Reset()

	// Verify all cleared
	if tracker.GetTotalSpent() != 0 {
		t.Error("Expected total spent to be reset")
	}

	status := tracker.GetHourlyStatus()
	if status.Used != 0 {
		t.Error("Expected hourly usage to be reset")
	}

	status = tracker.GetDailyStatus()
	if status.Used != 0 {
		t.Error("Expected daily usage to be reset")
	}

	status = tracker.GetMonthlyStatus()
	if status.Used != 0 {
		t.Error("Expected monthly usage to be reset")
	}
}

func TestTracker_Concurrent(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly: 1000.00, // Large limit for concurrency test
	})

	var wg sync.WaitGroup
	numGoroutines := 10
	amountPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < amountPerGoroutine; j++ {
				tracker.Add(1.00)
				tracker.Check()
			}
		}()
	}

	wg.Wait()

	expected := float64(numGoroutines * amountPerGoroutine)
	total := tracker.GetTotalSpent()
	if total != expected {
		t.Errorf("Expected total %.2f, got %.2f", expected, total)
	}
}

func TestTracker_StatusFields(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly: 10.00,
	})

	tracker.Add(3.00)

	status := tracker.GetHourlyStatus()

	// Verify all status fields
	if status.Limit != 10.00 {
		t.Errorf("Expected limit 10.00, got %.2f", status.Limit)
	}
	if status.Used != 3.00 {
		t.Errorf("Expected used 3.00, got %.2f", status.Used)
	}
	if status.Remaining != 7.00 {
		t.Errorf("Expected remaining 7.00, got %.2f", status.Remaining)
	}
	if status.Percentage != 0.30 {
		t.Errorf("Expected percentage 0.30, got %.2f", status.Percentage)
	}
	if status.Window != time.Hour {
		t.Errorf("Expected window 1h, got %v", status.Window)
	}
	if status.Reset.IsZero() {
		t.Error("Expected reset time to be set")
	}
}

func TestTracker_ExceededStatusFields(t *testing.T) {
	tracker := NewTracker(Config{
		Hourly: 10.00,
	})

	tracker.Add(15.00) // Exceed limit

	status := tracker.Check()

	// Verify status for exceeded limit
	if status.Allowed {
		t.Error("Expected spending to be blocked")
	}
	if status.Reason == "" {
		t.Error("Expected reason to be set")
	}
	if status.Used != 15.00 {
		t.Errorf("Expected used 15.00, got %.2f", status.Used)
	}
	if status.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %.2f", status.Remaining)
	}
	if status.Percentage <= 1.0 {
		t.Errorf("Expected percentage > 1.0 for exceeded limit, got %.2f", status.Percentage)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkRollingWindow_Add(b *testing.B) {
	rw := NewRollingWindow(time.Hour, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw.Add(1.00)
	}
}

func BenchmarkRollingWindow_Sum(b *testing.B) {
	rw := NewRollingWindow(time.Hour, time.Minute)

	// Pre-populate
	for i := 0; i < 60; i++ {
		rw.Add(1.00)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw.Sum()
	}
}

func BenchmarkTracker_Add(b *testing.B) {
	tracker := NewTracker(Config{
		Hourly:  10.00,
		Daily:   100.00,
		Monthly: 1000.00,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Add(0.01)
	}
}

func BenchmarkTracker_Check(b *testing.B) {
	tracker := NewTracker(Config{
		Hourly:  10.00,
		Daily:   100.00,
		Monthly: 1000.00,
	})

	// Pre-populate with some spending
	tracker.Add(5.00)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Check()
	}
}

func BenchmarkTracker_Concurrent(b *testing.B) {
	tracker := NewTracker(Config{
		Hourly:  10000.00,
		Daily:   100000.00,
		Monthly: 1000000.00,
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tracker.Add(0.01)
			tracker.Check()
		}
	})
}
