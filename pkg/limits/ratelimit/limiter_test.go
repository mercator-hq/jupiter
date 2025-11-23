package ratelimit

import (
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Token Bucket Tests
// ============================================================================

func TestTokenBucket_Basic(t *testing.T) {
	bucket := NewTokenBucket(10, 10) // 10 capacity, 10 tokens/sec

	// Should start with full capacity
	if !bucket.Take(5) {
		t.Error("Expected to take 5 tokens from full bucket")
	}

	// Should have 5 remaining
	remaining := bucket.Remaining()
	if remaining != 5 {
		t.Errorf("Expected 5 remaining, got %d", remaining)
	}

	// Should be able to take remaining 5
	if !bucket.Take(5) {
		t.Error("Expected to take remaining 5 tokens")
	}

	// Should be empty now
	if bucket.Take(1) {
		t.Error("Expected bucket to be empty")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket := NewTokenBucket(10, 10) // 10 capacity, 10 tokens/sec

	// Drain bucket
	bucket.Take(10)
	if bucket.Remaining() != 0 {
		t.Error("Expected bucket to be empty")
	}

	// Wait for refill (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should have refilled at least 1 token
	if !bucket.Take(1) {
		t.Error("Expected bucket to have refilled")
	}
}

func TestTokenBucket_CapacityLimit(t *testing.T) {
	bucket := NewTokenBucket(10, 10)

	// Wait longer than needed to fill beyond capacity
	time.Sleep(200 * time.Millisecond)

	// Should not exceed capacity
	if bucket.Remaining() > 10 {
		t.Errorf("Bucket exceeded capacity: %d", bucket.Remaining())
	}
}

func TestTokenBucket_TimeUntilAvailable(t *testing.T) {
	bucket := NewTokenBucket(10, 10) // 10 tokens/sec

	// Drain bucket
	bucket.Take(10)

	// Check time until 5 tokens available
	timeUntil := bucket.TimeUntilAvailable(5)

	// Should be approximately 0.5 seconds (5 tokens at 10/sec)
	if timeUntil < 400*time.Millisecond || timeUntil > 600*time.Millisecond {
		t.Errorf("Expected ~500ms, got %v", timeUntil)
	}

	// If tokens already available, should return 0
	bucket.Reset()
	timeUntil = bucket.TimeUntilAvailable(5)
	if timeUntil != 0 {
		t.Errorf("Expected 0 for available tokens, got %v", timeUntil)
	}
}

func TestTokenBucket_Concurrent(t *testing.T) {
	bucket := NewTokenBucket(1000, 100) // Large capacity for concurrency test

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Run 100 concurrent Take operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if bucket.Take(1) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All should succeed since capacity is 1000
	if successCount != 100 {
		t.Errorf("Expected 100 successes, got %d", successCount)
	}
}

// ============================================================================
// Sliding Window Tests
// ============================================================================

func TestSlidingWindow_Basic(t *testing.T) {
	sw := NewSlidingWindow(time.Minute, time.Second)

	// Add values
	sw.Add(100)
	sw.Add(200)
	sw.Add(300)

	// Sum should be 600
	sum := sw.Sum()
	if sum != 600 {
		t.Errorf("Expected sum 600, got %d", sum)
	}
}

func TestSlidingWindow_Expiration(t *testing.T) {
	sw := NewSlidingWindow(100*time.Millisecond, 10*time.Millisecond)

	// Add value
	sw.Add(100)

	// Should be present immediately
	if sw.Sum() != 100 {
		t.Error("Expected value to be present")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired and pruned
	sum := sw.Sum()
	if sum != 0 {
		t.Errorf("Expected 0 after expiration, got %d", sum)
	}
}

func TestSlidingWindow_RollingWindow(t *testing.T) {
	// Use longer windows to handle race detector overhead
	sw := NewSlidingWindow(1*time.Second, 100*time.Millisecond)

	// Add value at T=0
	start := time.Now()
	sw.Add(100)
	time.Sleep(550 * time.Millisecond) // Add to bucket at 500ms

	// Add value at T=~550ms (bucket at 500ms)
	sw.Add(200)
	time.Sleep(50 * time.Millisecond)

	// At T=~600ms, sum should be 300
	sum := sw.Sum()
	if sum != 300 {
		t.Errorf("Expected 300 before expiration, got %d", sum)
	}

	// Wait for first value to expire (window is 1s)
	// Sleep until T=1.4s (first bucket at T=0 expires at T=1s, extra 400ms margin)
	// Second bucket at T=500ms will still be within window (1400 - 1000 = 400ms cutoff, 500ms > 400ms)
	elapsed := time.Since(start)
	remainingSleep := (1400 * time.Millisecond) - elapsed
	if remainingSleep > 0 {
		time.Sleep(remainingSleep)
	}

	// First value (T=0) should be expired, only second value (T=~500ms) remains
	sum = sw.Sum()
	if sum != 200 {
		t.Errorf("Expected 200 after first expiration, got %d (elapsed: %v)", sum, time.Since(start))
	}
}

func TestSlidingWindow_Reset(t *testing.T) {
	sw := NewSlidingWindow(time.Minute, time.Second)

	sw.Add(100)
	sw.Add(200)

	sw.Reset()

	sum := sw.Sum()
	if sum != 0 {
		t.Errorf("Expected 0 after reset, got %d", sum)
	}
}

func TestSlidingWindow_Concurrent(t *testing.T) {
	sw := NewSlidingWindow(time.Minute, time.Second)

	var wg sync.WaitGroup

	// Run 100 concurrent Add operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sw.Add(1)
		}()
	}

	wg.Wait()

	// Sum should be 100
	sum := sw.Sum()
	if sum != 100 {
		t.Errorf("Expected sum 100, got %d", sum)
	}
}

// ============================================================================
// Concurrent Limiter Tests
// ============================================================================

func TestConcurrentLimiter_Basic(t *testing.T) {
	limiter := NewConcurrentLimiter(5)

	// Should be able to acquire 5 times
	for i := 0; i < 5; i++ {
		if !limiter.Acquire() {
			t.Errorf("Failed to acquire slot %d", i)
		}
	}

	// 6th acquisition should fail
	if limiter.Acquire() {
		t.Error("Expected 6th acquisition to fail")
	}

	// Release one slot
	limiter.Release()

	// Now should be able to acquire again
	if !limiter.Acquire() {
		t.Error("Expected to acquire after release")
	}
}

func TestConcurrentLimiter_CurrentAndRemaining(t *testing.T) {
	limiter := NewConcurrentLimiter(10)

	// Initially current=0, remaining=10
	if limiter.Current() != 0 {
		t.Errorf("Expected current 0, got %d", limiter.Current())
	}
	if limiter.Remaining() != 10 {
		t.Errorf("Expected remaining 10, got %d", limiter.Remaining())
	}

	// Acquire 3 slots
	limiter.Acquire()
	limiter.Acquire()
	limiter.Acquire()

	// current=3, remaining=7
	if limiter.Current() != 3 {
		t.Errorf("Expected current 3, got %d", limiter.Current())
	}
	if limiter.Remaining() != 7 {
		t.Errorf("Expected remaining 7, got %d", limiter.Remaining())
	}
}

func TestConcurrentLimiter_Concurrent(t *testing.T) {
	limiter := NewConcurrentLimiter(50)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Try to acquire 100 slots concurrently (limit is 50)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Acquire() {
				mu.Lock()
				successCount++
				mu.Unlock()
				defer limiter.Release()
				time.Sleep(10 * time.Millisecond) // Hold for a bit
			}
		}()
	}

	wg.Wait()

	// Exactly 50 should have succeeded
	if successCount != 50 {
		t.Errorf("Expected 50 successes, got %d", successCount)
	}

	// All should be released now
	if limiter.Current() != 0 {
		t.Errorf("Expected current 0 after all released, got %d", limiter.Current())
	}
}

func TestConcurrentLimiter_Reset(t *testing.T) {
	limiter := NewConcurrentLimiter(10)

	// Acquire some slots
	limiter.Acquire()
	limiter.Acquire()
	limiter.Acquire()

	limiter.Reset()

	if limiter.Current() != 0 {
		t.Errorf("Expected current 0 after reset, got %d", limiter.Current())
	}
}

// ============================================================================
// Limiter Integration Tests
// ============================================================================

func TestLimiter_RequestLimits(t *testing.T) {
	limiter := NewLimiter(Config{
		RequestsPerSecond: 10,
		RequestsPerMinute: 500,
	})

	// Should allow first request
	result := limiter.CheckRequest()
	if !result.Allowed {
		t.Error("Expected first request to be allowed")
	}

	// Should allow up to limit
	for i := 0; i < 9; i++ {
		result = limiter.CheckRequest()
		if !result.Allowed {
			t.Errorf("Expected request %d to be allowed", i+2)
		}
	}

	// 11th request should be blocked (burst capacity is 20, but we took 10)
	// Actually with 2x burst, we have 20 tokens, so let's drain them first
	for i := 0; i < 10; i++ {
		limiter.CheckRequest()
	}

	// Now should be blocked
	result = limiter.CheckRequest()
	if result.Allowed {
		t.Error("Expected request to be blocked after exceeding limit")
	}
	if result.Reason != "requests per second limit exceeded" {
		t.Errorf("Expected 'requests per second limit exceeded', got %s", result.Reason)
	}
}

func TestLimiter_TokenLimits(t *testing.T) {
	limiter := NewLimiter(Config{
		TokensPerMinute: 1000,
	})

	// Should allow request within limit
	result := limiter.CheckTokens(500)
	if !result.Allowed {
		t.Error("Expected request to be allowed")
	}

	// Record the tokens
	limiter.RecordTokens(500)

	// Should allow another 500
	result = limiter.CheckTokens(500)
	if !result.Allowed {
		t.Error("Expected second request to be allowed")
	}

	// Record the tokens
	limiter.RecordTokens(500)

	// Should block next request (would exceed 1000)
	result = limiter.CheckTokens(100)
	if result.Allowed {
		t.Error("Expected request to be blocked")
	}
	if result.Reason != "tokens per minute limit exceeded" {
		t.Errorf("Expected 'tokens per minute limit exceeded', got %s", result.Reason)
	}
}

func TestLimiter_ConcurrentLimit(t *testing.T) {
	limiter := NewLimiter(Config{
		MaxConcurrent: 5,
	})

	// Acquire 5 slots
	for i := 0; i < 5; i++ {
		if !limiter.AcquireConcurrent() {
			t.Errorf("Failed to acquire slot %d", i)
		}
	}

	// 6th should fail
	if limiter.AcquireConcurrent() {
		t.Error("Expected 6th acquisition to fail")
	}

	// Release one
	limiter.ReleaseConcurrent()

	// Should work now
	if !limiter.AcquireConcurrent() {
		t.Error("Expected to acquire after release")
	}
}

func TestLimiter_NoLimits(t *testing.T) {
	// Limiter with no limits configured
	limiter := NewLimiter(Config{})

	// Should allow all requests
	for i := 0; i < 100; i++ {
		result := limiter.CheckRequest()
		if !result.Allowed {
			t.Errorf("Expected request %d to be allowed with no limits", i)
		}

		result = limiter.CheckTokens(1000)
		if !result.Allowed {
			t.Errorf("Expected token check %d to be allowed with no limits", i)
		}

		if !limiter.AcquireConcurrent() {
			t.Errorf("Expected concurrent acquire %d to succeed with no limits", i)
		}
	}
}

func TestLimiter_Reset(t *testing.T) {
	limiter := NewLimiter(Config{
		RequestsPerSecond: 5,
		TokensPerMinute:   100,
		MaxConcurrent:     3,
	})

	// Use up some limits
	for i := 0; i < 5; i++ {
		limiter.CheckRequest()
	}
	limiter.RecordTokens(100)
	limiter.AcquireConcurrent()
	limiter.AcquireConcurrent()

	// Reset
	limiter.Reset()

	// Should allow requests again
	result := limiter.CheckRequest()
	if !result.Allowed {
		t.Error("Expected request to be allowed after reset")
	}

	result = limiter.CheckTokens(100)
	if !result.Allowed {
		t.Error("Expected token check to be allowed after reset")
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkTokenBucket_Take(b *testing.B) {
	bucket := NewTokenBucket(1000000, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucket.Take(1)
	}
}

func BenchmarkTokenBucket_Concurrent(b *testing.B) {
	bucket := NewTokenBucket(1000000, 1000)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bucket.Take(1)
		}
	})
}

func BenchmarkSlidingWindow_Add(b *testing.B) {
	sw := NewSlidingWindow(time.Minute, time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Add(1)
	}
}

func BenchmarkSlidingWindow_Sum(b *testing.B) {
	sw := NewSlidingWindow(time.Minute, time.Second)

	// Pre-populate
	for i := 0; i < 100; i++ {
		sw.Add(1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Sum()
	}
}

func BenchmarkConcurrentLimiter_AcquireRelease(b *testing.B) {
	limiter := NewConcurrentLimiter(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if limiter.Acquire() {
			limiter.Release()
		}
	}
}

func BenchmarkConcurrentLimiter_Concurrent(b *testing.B) {
	limiter := NewConcurrentLimiter(1000)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if limiter.Acquire() {
				limiter.Release()
			}
		}
	})
}

func BenchmarkLimiter_CheckRequest(b *testing.B) {
	limiter := NewLimiter(Config{
		RequestsPerSecond: 10000,
		RequestsPerMinute: 500000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.CheckRequest()
	}
}

func BenchmarkLimiter_CheckTokens(b *testing.B) {
	limiter := NewLimiter(Config{
		TokensPerMinute: 1000000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.CheckTokens(100)
	}
}
