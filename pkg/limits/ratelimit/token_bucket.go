package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm.
//
// The token bucket allows bursts up to the capacity while maintaining
// an average rate over time. Tokens are added at a constant refill rate.
// Each request consumes one or more tokens. If insufficient tokens are
// available, the request is rejected.
//
// This implementation uses monotonic time to avoid clock skew issues.
//
// # Algorithm
//
//  1. Calculate tokens to add based on elapsed time since last refill
//  2. Add tokens (up to capacity)
//  3. Check if enough tokens available for request
//  4. If yes: consume tokens and allow request
//  5. If no: reject request
//
// # Thread Safety
//
// TokenBucket is thread-safe using sync.Mutex for all operations.
type TokenBucket struct {
	capacity   int64     // Maximum tokens in bucket
	tokens     int64     // Current available tokens
	refillRate float64   // Tokens added per second
	lastRefill time.Time // Last time tokens were refilled
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter.
//
// Parameters:
//   - capacity: Maximum number of tokens in the bucket (burst size)
//   - refillRate: Number of tokens added per second (average rate)
//
// Example:
//
//	// 100 requests/sec average, burst up to 100
//	bucket := NewTokenBucket(100, 100)
//
//	// 10 requests/sec average, burst up to 50
//	bucket := NewTokenBucket(50, 10)
func NewTokenBucket(capacity int64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity, // Start with full bucket
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Take attempts to consume n tokens from the bucket.
// Returns true if tokens were available and consumed, false otherwise.
//
// This method refills tokens based on elapsed time before checking availability.
func (tb *TokenBucket) Take(n int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on elapsed time
	tb.refillLocked()

	// Check if enough tokens available
	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}

	return false
}

// Remaining returns the number of tokens currently available.
// This does NOT trigger a refill - call Take() for that.
func (tb *TokenBucket) Remaining() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill before returning remaining count
	tb.refillLocked()
	return tb.tokens
}

// Capacity returns the maximum bucket capacity.
func (tb *TokenBucket) Capacity() int64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.capacity
}

// Reset resets the bucket to full capacity.
// This is useful for testing or manual limit resets.
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
}

// TimeUntilAvailable returns how long until n tokens will be available.
// Returns 0 if tokens are immediately available.
func (tb *TokenBucket) TimeUntilAvailable(n int64) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill first
	tb.refillLocked()

	// If already available, return immediately
	if tb.tokens >= n {
		return 0
	}

	// Calculate tokens needed
	tokensNeeded := n - tb.tokens

	// Calculate time needed to refill
	secondsNeeded := float64(tokensNeeded) / tb.refillRate

	return time.Duration(secondsNeeded * float64(time.Second))
}

// refillLocked adds tokens based on elapsed time since last refill.
// Caller must hold lock.
func (tb *TokenBucket) refillLocked() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Calculate tokens to add
	tokensToAdd := int64(elapsed.Seconds() * tb.refillRate)

	if tokensToAdd > 0 {
		// Add tokens (up to capacity)
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}

		// Update last refill time
		tb.lastRefill = now
	}
}
