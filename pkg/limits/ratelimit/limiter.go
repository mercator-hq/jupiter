package ratelimit

import (
	"time"
)

// Limiter coordinates multiple rate limiting strategies.
//
// The Limiter combines token bucket, sliding window, and concurrent limiters
// to provide comprehensive rate limiting across multiple dimensions:
//
//   - Request-based limits (requests per second/minute/hour)
//   - Token-based limits (tokens per minute/hour)
//   - Concurrent request limits
//
// All limits are evaluated together - if any limit is exceeded, the request
// is rejected with details about which limit was hit.
type Limiter struct {
	// Request-based limits (token buckets)
	reqPerSecond *TokenBucket
	reqPerMinute *TokenBucket
	reqPerHour   *TokenBucket

	// Token-based limits (sliding windows)
	tokensPerMinute *SlidingWindow
	tokensPerHour   *SlidingWindow

	// Concurrent limit
	concurrent *ConcurrentLimiter

	// Configuration
	config Config
}

// NewLimiter creates a new rate limiter with the given configuration.
//
// Only non-zero limits in the config are enforced. Zero values mean no limit.
//
// Example:
//
//	limiter := NewLimiter(Config{
//	    RequestsPerSecond: 10,
//	    RequestsPerMinute: 500,
//	    TokensPerMinute:   100000,
//	    MaxConcurrent:     50,
//	})
func NewLimiter(config Config) *Limiter {
	limiter := &Limiter{
		config: config,
	}

	// Initialize request-based limits (token buckets)
	if config.RequestsPerSecond > 0 {
		// Allow burst up to 2x the per-second rate
		capacity := int64(config.RequestsPerSecond * 2)
		limiter.reqPerSecond = NewTokenBucket(capacity, float64(config.RequestsPerSecond))
	}

	if config.RequestsPerMinute > 0 {
		// Allow burst up to the full minute rate
		capacity := int64(config.RequestsPerMinute)
		limiter.reqPerMinute = NewTokenBucket(capacity, float64(config.RequestsPerMinute)/60.0)
	}

	if config.RequestsPerHour > 0 {
		// Allow burst up to 5 minutes worth
		capacity := int64(config.RequestsPerHour / 12)
		limiter.reqPerHour = NewTokenBucket(capacity, float64(config.RequestsPerHour)/3600.0)
	}

	// Initialize token-based limits (sliding windows)
	if config.TokensPerMinute > 0 {
		// 1-second granularity for per-minute window
		limiter.tokensPerMinute = NewSlidingWindow(time.Minute, time.Second)
	}

	if config.TokensPerHour > 0 {
		// 1-minute granularity for per-hour window
		limiter.tokensPerHour = NewSlidingWindow(time.Hour, time.Minute)
	}

	// Initialize concurrent limiter
	if config.MaxConcurrent > 0 {
		limiter.concurrent = NewConcurrentLimiter(config.MaxConcurrent)
	}

	return limiter
}

// CheckRequest checks if a request is allowed based on request-based limits.
// This should be called before processing the request.
//
// Returns CheckResult indicating if the request is allowed and why.
func (l *Limiter) CheckRequest() *CheckResult {
	// Check requests per second
	if l.reqPerSecond != nil {
		if !l.reqPerSecond.Take(1) {
			retryAfter := l.reqPerSecond.TimeUntilAvailable(1)
			return &CheckResult{
				Allowed:    false,
				Reason:     "requests per second limit exceeded",
				Limit:      l.reqPerSecond.Capacity(),
				Remaining:  l.reqPerSecond.Remaining(),
				Reset:      time.Now().Add(time.Second),
				RetryAfter: retryAfter,
			}
		}
	}

	// Check requests per minute
	if l.reqPerMinute != nil {
		if !l.reqPerMinute.Take(1) {
			retryAfter := l.reqPerMinute.TimeUntilAvailable(1)
			return &CheckResult{
				Allowed:    false,
				Reason:     "requests per minute limit exceeded",
				Limit:      l.reqPerMinute.Capacity(),
				Remaining:  l.reqPerMinute.Remaining(),
				Reset:      time.Now().Add(time.Minute),
				RetryAfter: retryAfter,
			}
		}
	}

	// Check requests per hour
	if l.reqPerHour != nil {
		if !l.reqPerHour.Take(1) {
			retryAfter := l.reqPerHour.TimeUntilAvailable(1)
			return &CheckResult{
				Allowed:    false,
				Reason:     "requests per hour limit exceeded",
				Limit:      l.reqPerHour.Capacity(),
				Remaining:  l.reqPerHour.Remaining(),
				Reset:      time.Now().Add(time.Hour),
				RetryAfter: retryAfter,
			}
		}
	}

	// All request limits passed
	return &CheckResult{
		Allowed: true,
	}
}

// CheckTokens checks if a request is allowed based on token-based limits.
// This should be called before processing the request.
//
// Parameters:
//   - estimatedTokens: Estimated number of tokens this request will use
//
// Returns CheckResult indicating if the request is allowed and why.
func (l *Limiter) CheckTokens(estimatedTokens int) *CheckResult {
	// Check tokens per minute
	if l.tokensPerMinute != nil {
		currentUsage := l.tokensPerMinute.Sum()
		if currentUsage+int64(estimatedTokens) > int64(l.config.TokensPerMinute) {
			return &CheckResult{
				Allowed:    false,
				Reason:     "tokens per minute limit exceeded",
				Limit:      int64(l.config.TokensPerMinute),
				Remaining:  int64(l.config.TokensPerMinute) - currentUsage,
				Reset:      time.Now().Add(time.Minute),
				RetryAfter: time.Minute, // Conservative estimate
			}
		}
	}

	// Check tokens per hour
	if l.tokensPerHour != nil {
		currentUsage := l.tokensPerHour.Sum()
		if currentUsage+int64(estimatedTokens) > int64(l.config.TokensPerHour) {
			return &CheckResult{
				Allowed:    false,
				Reason:     "tokens per hour limit exceeded",
				Limit:      int64(l.config.TokensPerHour),
				Remaining:  int64(l.config.TokensPerHour) - currentUsage,
				Reset:      time.Now().Add(time.Hour),
				RetryAfter: time.Hour, // Conservative estimate
			}
		}
	}

	// All token limits passed
	return &CheckResult{
		Allowed: true,
	}
}

// RecordTokens records actual token usage after a request completes.
// This updates the sliding window counters.
//
// Parameters:
//   - actualTokens: Actual number of tokens used by the request
func (l *Limiter) RecordTokens(actualTokens int) {
	if l.tokensPerMinute != nil {
		l.tokensPerMinute.Add(int64(actualTokens))
	}

	if l.tokensPerHour != nil {
		l.tokensPerHour.Add(int64(actualTokens))
	}
}

// AcquireConcurrent attempts to acquire a concurrency slot.
// Returns true if acquired, false if limit reached.
//
// If this returns true, the caller MUST call ReleaseConcurrent() when done.
func (l *Limiter) AcquireConcurrent() bool {
	if l.concurrent == nil {
		return true // No concurrent limit configured
	}

	return l.concurrent.Acquire()
}

// ReleaseConcurrent releases a concurrency slot.
// This MUST be called after a successful AcquireConcurrent().
func (l *Limiter) ReleaseConcurrent() {
	if l.concurrent != nil {
		l.concurrent.Release()
	}
}

// GetConcurrentStatus returns the current concurrent request status.
func (l *Limiter) GetConcurrentStatus() *CheckResult {
	if l.concurrent == nil {
		return &CheckResult{Allowed: true}
	}

	return &CheckResult{
		Allowed:   true,
		Limit:     l.concurrent.Limit(),
		Remaining: l.concurrent.Remaining(),
	}
}

// Reset resets all limits. This is primarily for testing.
func (l *Limiter) Reset() {
	if l.reqPerSecond != nil {
		l.reqPerSecond.Reset()
	}
	if l.reqPerMinute != nil {
		l.reqPerMinute.Reset()
	}
	if l.reqPerHour != nil {
		l.reqPerHour.Reset()
	}
	if l.tokensPerMinute != nil {
		l.tokensPerMinute.Reset()
	}
	if l.tokensPerHour != nil {
		l.tokensPerHour.Reset()
	}
	if l.concurrent != nil {
		l.concurrent.Reset()
	}
}
