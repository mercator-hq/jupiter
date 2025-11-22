package storage

import (
	"context"
	"time"
)

// Backend defines the interface for limit state persistence.
// Implementations must be thread-safe and support concurrent access.
type Backend interface {
	// Save persists the limit state for an identifier.
	// If state already exists, it is updated. Returns error on failure.
	Save(ctx context.Context, state *LimitState) error

	// Load retrieves the limit state for an identifier and dimension.
	// Returns nil if no state exists. Returns error on system failure.
	Load(ctx context.Context, identifier string, dimension string) (*LimitState, error)

	// Delete removes the limit state for an identifier and dimension.
	// Returns error on failure. No-op if state doesn't exist.
	Delete(ctx context.Context, identifier string, dimension string) error

	// List returns all limit states for a dimension.
	// Returns empty slice if no states exist. Returns error on failure.
	List(ctx context.Context, dimension string) ([]*LimitState, error)

	// Cleanup removes expired state entries based on retention policy.
	// Returns the number of entries deleted and any error.
	Cleanup(ctx context.Context, olderThan time.Time) (int, error)

	// Close releases any resources held by the backend.
	// The backend should not be used after calling Close.
	Close() error
}

// LimitState represents the persisted state for a single identifier.
// This includes both rate limit counters and budget usage.
type LimitState struct {
	// Identifier is the dimension identifier (API key, user ID, team ID).
	Identifier string

	// Dimension is the limiting dimension (api_key, user, team).
	Dimension string

	// RateLimit contains rate limiter state (token bucket, sliding window).
	RateLimit *RateLimitState

	// Budget contains budget tracker state (rolling window buckets).
	Budget *BudgetState

	// LastUpdated is when this state was last modified.
	LastUpdated time.Time

	// CreatedAt is when this state was first created.
	CreatedAt time.Time
}

// RateLimitState contains the state for rate limiters.
// This is backend-agnostic and can be serialized to any storage.
type RateLimitState struct {
	// TokenBucket contains token bucket state for request-based limits.
	TokenBucket *TokenBucketState

	// SlidingWindow contains sliding window state for token-based limits.
	SlidingWindow *SlidingWindowState

	// Concurrent tracks current concurrent requests.
	Concurrent int

	// MaxConcurrent is the concurrent request limit.
	MaxConcurrent int
}

// TokenBucketState contains the state for a token bucket rate limiter.
// The bucket refills at a constant rate up to a maximum capacity.
type TokenBucketState struct {
	// Capacity is the maximum number of tokens.
	Capacity int64

	// Tokens is the current number of tokens available.
	Tokens int64

	// RefillRate is the number of tokens added per second.
	RefillRate int64

	// LastRefill is when tokens were last refilled.
	LastRefill time.Time
}

// SlidingWindowState contains the state for a sliding window counter.
// Used for token-based rate limiting over time windows.
type SlidingWindowState struct {
	// Window is the time window duration.
	Window time.Duration

	// Buckets contains time-stamped buckets with token counts.
	Buckets []WindowBucket

	// BucketSize is the granularity of each bucket.
	BucketSize time.Duration
}

// WindowBucket represents a single bucket in a sliding window.
type WindowBucket struct {
	// Timestamp is when this bucket started.
	Timestamp time.Time

	// Value is the counter value for this bucket.
	Value int64
}

// BudgetState contains the state for budget tracking.
// Uses rolling windows for hourly, daily, and monthly budgets.
type BudgetState struct {
	// HourlyBuckets tracks spending in the last 60 minutes.
	HourlyBuckets []BudgetBucket

	// DailyBuckets tracks spending in the last 24 hours.
	DailyBuckets []BudgetBucket

	// MonthlyBuckets tracks spending in the last 30 days.
	MonthlyBuckets []BudgetBucket

	// TotalSpent is the all-time total spending for this identifier.
	TotalSpent float64
}

// BudgetBucket represents a single bucket in a budget rolling window.
// Each bucket tracks spending for a specific time interval.
type BudgetBucket struct {
	// Timestamp is when this bucket started.
	Timestamp time.Time

	// Amount is the spending in USD for this bucket.
	Amount float64
}
