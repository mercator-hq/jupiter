package ratelimit

import (
	"sync/atomic"
)

// ConcurrentLimiter limits the number of simultaneous in-flight requests.
//
// This implements a counting semaphore using atomic operations for
// lock-free performance. It's useful for limiting the number of concurrent
// requests to prevent resource exhaustion.
//
// # Algorithm
//
//  1. Atomically increment counter
//  2. Check if counter exceeds limit
//  3. If yes: decrement and reject
//  4. If no: allow request
//  5. On completion: decrement counter
//
// # Thread Safety
//
// ConcurrentLimiter is lock-free and thread-safe using atomic operations.
type ConcurrentLimiter struct {
	limit   int64 // Maximum concurrent requests
	current int64 // Current number of in-flight requests
}

// NewConcurrentLimiter creates a new concurrent request limiter.
//
// Parameters:
//   - limit: Maximum number of simultaneous requests allowed
//
// Example:
//
//	limiter := NewConcurrentLimiter(50) // Max 50 concurrent requests
//	if limiter.Acquire() {
//	    defer limiter.Release()
//	    // Process request
//	} else {
//	    // Too many concurrent requests
//	}
func NewConcurrentLimiter(limit int) *ConcurrentLimiter {
	return &ConcurrentLimiter{
		limit:   int64(limit),
		current: 0,
	}
}

// Acquire attempts to acquire a concurrency slot.
// Returns true if acquired, false if limit reached.
//
// If this returns true, the caller MUST call Release() when done.
// Use defer immediately after checking the return value:
//
//	if limiter.Acquire() {
//	    defer limiter.Release()
//	    // ... process request ...
//	}
func (cl *ConcurrentLimiter) Acquire() bool {
	// Atomically increment counter
	current := atomic.AddInt64(&cl.current, 1)

	// Check if limit exceeded
	if current > cl.limit {
		// Exceeded limit, decrement and reject
		atomic.AddInt64(&cl.current, -1)
		return false
	}

	// Within limit, allow
	return true
}

// Release releases a concurrency slot.
// This MUST be called after a successful Acquire().
//
// It is safe to call Release() multiple times or without Acquire(),
// though this may lead to incorrect accounting. Always pair with Acquire().
func (cl *ConcurrentLimiter) Release() {
	atomic.AddInt64(&cl.current, -1)
}

// Current returns the current number of in-flight requests.
func (cl *ConcurrentLimiter) Current() int64 {
	return atomic.LoadInt64(&cl.current)
}

// Limit returns the configured concurrency limit.
func (cl *ConcurrentLimiter) Limit() int64 {
	return atomic.LoadInt64(&cl.limit)
}

// Remaining returns the number of available concurrency slots.
func (cl *ConcurrentLimiter) Remaining() int64 {
	current := atomic.LoadInt64(&cl.current)
	limit := atomic.LoadInt64(&cl.limit)

	remaining := limit - current
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset resets the concurrent request count to zero.
// This should only be used in testing or error recovery scenarios.
func (cl *ConcurrentLimiter) Reset() {
	atomic.StoreInt64(&cl.current, 0)
}
