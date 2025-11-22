package ratelimit

import "time"

// Config contains configuration for all rate limiters for a single identifier.
// This is passed to the Limiter to configure all rate limiting dimensions.
type Config struct {
	// RequestsPerSecond limits requests per second using token bucket.
	RequestsPerSecond int

	// RequestsPerMinute limits requests per minute using token bucket.
	RequestsPerMinute int

	// RequestsPerHour limits requests per hour using token bucket.
	RequestsPerHour int

	// TokensPerMinute limits tokens (prompt+completion) per minute.
	TokensPerMinute int

	// TokensPerHour limits tokens per hour.
	TokensPerHour int

	// MaxConcurrent limits simultaneous requests.
	MaxConcurrent int
}

// CheckResult contains the result of a rate limit check.
// This is returned by Limiter.Check() to indicate if a request is allowed.
type CheckResult struct {
	// Allowed indicates if the request is permitted.
	Allowed bool

	// Reason explains why the request was rejected (if Allowed=false).
	Reason string

	// Limit is the configured limit value.
	Limit int64

	// Remaining is how many requests/tokens remain in the window.
	Remaining int64

	// Reset is when the limit window resets.
	Reset time.Time

	// RetryAfter suggests how long to wait before retrying.
	RetryAfter time.Duration
}
