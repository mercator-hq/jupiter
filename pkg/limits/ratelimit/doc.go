// Package ratelimit provides rate limiting algorithms for request and token-based limits.
//
// # Overview
//
// The ratelimit package implements multiple rate limiting strategies:
//
//   - Token Bucket: Request-based rate limiting with constant refill rate
//   - Sliding Window: Token-based rate limiting over rolling time windows
//   - Concurrent Limiter: Semaphore-based concurrent request limiting
//
// # Token Bucket Algorithm
//
// The token bucket algorithm allows bursts up to the bucket capacity while
// maintaining an average rate over time:
//
//	bucket := ratelimit.NewTokenBucket(100, 10) // 100 capacity, 10 refill/sec
//	if bucket.Take(1) {
//	    // Request allowed
//	} else {
//	    // Rate limit exceeded
//	}
//
// # Sliding Window
//
// The sliding window tracks token usage over rolling time windows:
//
//	window := ratelimit.NewSlidingWindow(time.Minute, 100000) // 100K tokens/min
//	window.Add(5000) // Add 5K tokens used
//	if window.Sum() > 100000 {
//	    // Rate limit exceeded
//	}
//
// # Concurrent Limiter
//
// The concurrent limiter enforces maximum simultaneous requests:
//
//	limiter := ratelimit.NewConcurrentLimiter(50) // Max 50 concurrent
//	if limiter.Acquire() {
//	    defer limiter.Release()
//	    // Process request
//	}
//
// # Thread Safety
//
// All rate limiters are thread-safe and use fine-grained locking to minimize
// contention under high load.
package ratelimit
