package limits

import (
	"errors"
	"fmt"
	"time"
)

// Dimension represents a limiting dimension (API key, user, team).
type Dimension string

const (
	// DimensionAPIKey limits by API key.
	DimensionAPIKey Dimension = "api_key"

	// DimensionUser limits by user ID.
	DimensionUser Dimension = "user"

	// DimensionTeam limits by team ID.
	DimensionTeam Dimension = "team"
)

// EnforcementAction defines what to do when a limit is exceeded.
type EnforcementAction string

const (
	// ActionAllow permits the request to proceed.
	ActionAllow EnforcementAction = "allow"

	// ActionBlock rejects the request with 429 Too Many Requests.
	ActionBlock EnforcementAction = "block"

	// ActionQueue holds the request until capacity is available.
	ActionQueue EnforcementAction = "queue"

	// ActionDowngrade routes to a cheaper model.
	ActionDowngrade EnforcementAction = "downgrade"

	// ActionAlert triggers an alert but allows the request.
	ActionAlert EnforcementAction = "alert"
)

// LimitCheckResult contains the decision and metadata from a limit check.
// This is returned by Manager.CheckLimits() to indicate whether a request
// should be allowed or rejected based on rate limits and budgets.
type LimitCheckResult struct {
	// Allowed indicates if the request is permitted.
	Allowed bool

	// Reason explains why the request was rejected (if Allowed=false).
	Reason string

	// RateLimit contains current rate limit status.
	RateLimit *RateLimitInfo

	// Budget contains current budget status.
	Budget *BudgetInfo

	// Action specifies the enforcement action to take.
	Action EnforcementAction

	// RetryAfter specifies how long to wait before retrying (if blocked).
	RetryAfter time.Duration

	// DowngradeTo suggests a cheaper model (if action=downgrade).
	DowngradeTo string
}

// RateLimitInfo contains current rate limit status for a dimension.
// This is used to populate HTTP response headers (X-RateLimit-*).
type RateLimitInfo struct {
	// Dimension is the limiting dimension (api_key, user, team).
	Dimension string

	// Identifier is the specific identifier within the dimension.
	Identifier string

	// Limit is the maximum allowed requests in the window.
	Limit int64

	// Remaining is the number of requests remaining in the window.
	Remaining int64

	// Reset is when the limit window resets.
	Reset time.Time

	// Window is the time window duration.
	Window time.Duration
}

// BudgetInfo contains current budget status for a dimension.
// This is used to populate HTTP response headers (X-Budget-*).
type BudgetInfo struct {
	// Dimension is the limiting dimension (api_key, user, team).
	Dimension string

	// Identifier is the specific identifier within the dimension.
	Identifier string

	// Limit is the maximum budget in USD for the window.
	Limit float64

	// Used is the amount of budget consumed in USD.
	Used float64

	// Remaining is the budget remaining in USD.
	Remaining float64

	// Percentage is the percentage of budget used (0.0-1.0).
	Percentage float64

	// Reset is when the budget window resets.
	Reset time.Time

	// Window is the time window duration.
	Window time.Duration
}

// UsageRecord tracks a single request's usage across all dimensions.
// This is recorded after a request completes and is used to update
// both rate limits and budget counters.
type UsageRecord struct {
	// Timestamp is when the request was made.
	Timestamp time.Time

	// Identifier is the dimension identifier (API key, user ID, team ID).
	Identifier string

	// Dimension is the limiting dimension.
	Dimension Dimension

	// RequestTokens is the number of tokens in the prompt.
	RequestTokens int

	// ResponseTokens is the number of tokens in the completion.
	ResponseTokens int

	// TotalTokens is the total token count.
	TotalTokens int

	// Cost is the actual cost in USD for this request.
	Cost float64

	// Provider is the LLM provider used (openai, anthropic, etc.).
	Provider string

	// Model is the specific model used (gpt-4, claude-3-opus, etc.).
	Model string
}

// NewUsageRecord creates a UsageRecord from response data.
// This is the primary way to create usage records for tracking.
func NewUsageRecord(
	dimension Dimension,
	identifier string,
	requestTokens int,
	responseTokens int,
	cost float64,
	provider string,
	model string,
) *UsageRecord {
	return &UsageRecord{
		Timestamp:      time.Now(),
		Identifier:     identifier,
		Dimension:      dimension,
		RequestTokens:  requestTokens,
		ResponseTokens: responseTokens,
		TotalTokens:    requestTokens + responseTokens,
		Cost:           cost,
		Provider:       provider,
		Model:          model,
	}
}

// Error types for limit violations and system errors.
var (
	// ErrRateLimitExceeded is returned when a rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrBudgetExceeded is returned when a budget limit is exceeded.
	ErrBudgetExceeded = errors.New("budget exceeded")

	// ErrInvalidIdentifier is returned when an identifier is invalid.
	ErrInvalidIdentifier = errors.New("invalid identifier")

	// ErrStorageFailure is returned when the storage backend fails.
	ErrStorageFailure = errors.New("storage backend failure")

	// ErrQueueFull is returned when the request queue is full.
	ErrQueueFull = errors.New("request queue full")

	// ErrConfigInvalid is returned when the limits configuration is invalid.
	ErrConfigInvalid = errors.New("invalid limits configuration")
)

// LimitError provides detailed context about a limit violation.
// This wraps the base error types with additional information for debugging.
type LimitError struct {
	// Type is the error type (rate_limit, budget, etc.).
	Type string

	// Identifier is the dimension identifier.
	Identifier string

	// Limit is the configured limit value.
	Limit interface{}

	// Current is the current value that exceeded the limit.
	Current interface{}

	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *LimitError) Error() string {
	return fmt.Sprintf("%s limit exceeded for %s: current=%v, limit=%v",
		e.Type, e.Identifier, e.Current, e.Limit)
}

// Unwrap returns the underlying error for error wrapping.
func (e *LimitError) Unwrap() error {
	return e.Err
}
