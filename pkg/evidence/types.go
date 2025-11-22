package evidence

import (
	"context"
	"io"
	"time"
)

// EvidenceRecord represents a complete audit trail for a single LLM request/response
// pair. It captures all metadata, policy decisions, costs, and cryptographic hashes
// for compliance and forensics.
type EvidenceRecord struct {
	// Identity
	ID        string `json:"id"`         // UUID v4
	RequestID string `json:"request_id"` // From proxy

	// Timestamps
	RequestTime      time.Time `json:"request_time"`       // When request received
	PolicyEvalTime   time.Time `json:"policy_eval_time"`   // When policy evaluated
	ProviderCallTime time.Time `json:"provider_call_time"` // When provider called
	ResponseTime     time.Time `json:"response_time"`      // When response received
	RecordedTime     time.Time `json:"recorded_time"`      // When evidence recorded

	// Request metadata
	RequestHash    string            `json:"request_hash"`    // SHA-256 of request body
	RequestMethod  string            `json:"request_method"`  // HTTP method
	RequestPath    string            `json:"request_path"`    // HTTP path
	RequestHeaders map[string]string `json:"request_headers"` // Selected headers

	// Request content
	Model        string   `json:"model"`         // Requested model
	Provider     string   `json:"provider"`      // Provider name (from routing)
	Messages     int      `json:"messages"`      // Message count
	SystemPrompt string   `json:"system_prompt"` // System prompt (first 500 chars)
	UserPrompt   string   `json:"user_prompt"`   // User prompt (first 500 chars)
	ToolsUsed    []string `json:"tools_used"`    // Tool names

	// Request metadata (from processing)
	EstimatedTokens int      `json:"estimated_tokens"` // Token estimate
	EstimatedCost   float64  `json:"estimated_cost"`   // Cost estimate
	RiskScore       int      `json:"risk_score"`       // Risk score (1-10)
	ComplexityScore int      `json:"complexity_score"` // Complexity score (1-10)
	PIIDetected     bool     `json:"pii_detected"`     // PII found?
	PIITypes        []string `json:"pii_types"`        // PII types found

	// Policy decisions
	PolicyDecision    string              `json:"policy_decision"`     // "allow", "block", "transform"
	MatchedRules      []MatchedRuleRecord `json:"matched_rules"`       // Rules that matched
	BlockReason       string              `json:"block_reason"`        // If blocked, why
	PolicyVersion     string              `json:"policy_version"`      // Git commit hash (deprecated, use PolicyVersionDetails)
	PolicyVersionInfo *PolicyVersionInfo  `json:"policy_version_info"` // Detailed policy version information (Git mode)

	// Response metadata
	ResponseHash   string `json:"response_hash"`   // SHA-256 of response body
	ResponseStatus int    `json:"response_status"` // HTTP status code

	// Response content
	ResponseContent string `json:"response_content"` // Response text (first 500 chars)
	FinishReason    string `json:"finish_reason"`    // stop, length, tool_calls

	// Actual usage
	PromptTokens     int     `json:"prompt_tokens"`     // Actual prompt tokens
	CompletionTokens int     `json:"completion_tokens"` // Actual completion tokens
	TotalTokens      int     `json:"total_tokens"`      // Total tokens
	ActualCost       float64 `json:"actual_cost"`       // Actual cost

	// Provider info
	ProviderLatency time.Duration `json:"provider_latency"` // Provider round-trip time
	ProviderModel   string        `json:"provider_model"`   // Actual model used

	// User/API key
	UserID    string `json:"user_id"`    // User identifier
	APIKey    string `json:"api_key"`    // API key (hashed or redacted)
	IPAddress string `json:"ip_address"` // Client IP

	// Error info
	Error     string `json:"error"`      // Error message if request failed
	ErrorType string `json:"error_type"` // Error type (timeout, rate_limit, etc.)

	// Conversation context
	TurnNumber   int     `json:"turn_number"`   // Turn in conversation
	ContextUsage float64 `json:"context_usage"` // Context window usage (0-1)
}

// MatchedRuleRecord captures details about a policy rule that matched during
// policy evaluation.
type MatchedRuleRecord struct {
	PolicyID       string        `json:"policy_id"`       // Policy identifier
	RuleID         string        `json:"rule_id"`         // Rule identifier
	Action         string        `json:"action"`          // "block", "allow", "route", etc.
	Reason         string        `json:"reason"`          // Why matched
	EvaluationTime time.Duration `json:"evaluation_time"` // Time to evaluate
}

// PolicyVersionInfo contains detailed version information for Git-based policy management.
// This provides a complete audit trail of which policies were active when a request was processed.
type PolicyVersionInfo struct {
	// CommitSHA is the Git commit hash of the active policies.
	CommitSHA string `json:"commit_sha"`

	// CommitTime is when the commit was created.
	CommitTime time.Time `json:"commit_time"`

	// Branch is the Git branch from which policies were loaded.
	Branch string `json:"branch"`

	// Repository is the Git repository URL.
	Repository string `json:"repository"`

	// Author is the commit author (name and email).
	Author string `json:"author"`

	// Message is the commit message.
	Message string `json:"message,omitempty"`
}

// Query defines filter parameters for querying evidence records.
type Query struct {
	// Time range
	StartTime *time.Time `json:"start_time,omitempty"` // Inclusive start time
	EndTime   *time.Time `json:"end_time,omitempty"`   // Inclusive end time

	// Filters
	UserID         string `json:"user_id,omitempty"`         // Filter by user ID
	APIKey         string `json:"api_key,omitempty"`         // Filter by API key
	Provider       string `json:"provider,omitempty"`        // Filter by provider
	Model          string `json:"model,omitempty"`           // Filter by model
	PolicyID       string `json:"policy_id,omitempty"`       // Filter by policy ID
	RuleID         string `json:"rule_id,omitempty"`         // Filter by rule ID
	PolicyDecision string `json:"policy_decision,omitempty"` // "allow", "block", etc.

	// Thresholds
	MinCost   *float64 `json:"min_cost,omitempty"`   // Minimum cost
	MaxCost   *float64 `json:"max_cost,omitempty"`   // Maximum cost
	MinTokens *int     `json:"min_tokens,omitempty"` // Minimum tokens
	MaxTokens *int     `json:"max_tokens,omitempty"` // Maximum tokens

	// Status
	Status string `json:"status,omitempty"` // "success", "error", "blocked"

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Max records to return
	Offset int `json:"offset,omitempty"` // Skip N records

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // "timestamp", "cost", "tokens"
	SortOrder string `json:"sort_order,omitempty"` // "asc", "desc"
}

// Storage defines the interface for evidence storage backends.
// Implementations must be thread-safe and support concurrent access.
type Storage interface {
	// Store persists an evidence record.
	// Returns an error if the record cannot be written.
	Store(ctx context.Context, record *EvidenceRecord) error

	// Query retrieves evidence records matching the query filters.
	// Returns an empty slice if no records match.
	Query(ctx context.Context, query *Query) ([]*EvidenceRecord, error)

	// QueryStream returns a channel of evidence records for memory-efficient streaming.
	// Use this for large result sets to avoid loading everything in memory.
	//
	// Returns:
	//   - recordsCh: Channel of evidence records (buffered)
	//   - errCh: Channel for errors (buffered, max 1 error)
	//   - error: Immediate error (e.g., invalid query)
	//
	// The channels will be closed when the query completes or errors.
	// Callers should read from both channels until they are closed.
	//
	// Example usage:
	//   recordsCh, errCh, err := storage.QueryStream(ctx, query)
	//   if err != nil {
	//       return err
	//   }
	//   for record := range recordsCh {
	//       // Process record
	//   }
	//   if err := <-errCh; err != nil {
	//       return err
	//   }
	QueryStream(ctx context.Context, query *Query) (<-chan *EvidenceRecord, <-chan error, error)

	// Count returns the number of evidence records matching the query filters.
	Count(ctx context.Context, query *Query) (int64, error)

	// Delete removes evidence records matching the query filters.
	// Returns the number of records deleted.
	// Used for retention policy enforcement.
	Delete(ctx context.Context, query *Query) (int64, error)

	// Close releases any resources held by the storage backend.
	Close() error
}

// Exporter defines the interface for exporting evidence records to various formats.
type Exporter interface {
	// Export writes evidence records to the provided writer in the exporter's format.
	// Returns an error if the export fails.
	Export(ctx context.Context, records []*EvidenceRecord, w io.Writer) error
}
