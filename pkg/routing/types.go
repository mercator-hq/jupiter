package routing

import (
	"time"

	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/providers"
)

// RoutingRequest contains all information needed to make a routing decision.
// It includes the request metadata, model, user/session information for sticky
// routing, optional manual provider selection, and policy decision.
type RoutingRequest struct {
	// RequestID is the unique identifier for this request.
	RequestID string

	// Model is the LLM model requested (e.g., "gpt-4", "claude-3-opus").
	Model string

	// User is the user identifier for sticky routing (optional).
	User string

	// APIKey is the API key identifier for sticky routing (optional).
	APIKey string

	// SessionID is the session identifier for sticky routing (optional).
	SessionID string

	// PreferredProvider is an explicit provider override (manual selection).
	// If specified, routing will attempt to use this provider first.
	PreferredProvider string

	// PolicyDecision contains the policy evaluation result.
	// If the policy includes a route action, it takes precedence.
	PolicyDecision *engine.PolicyDecision

	// Metadata contains additional routing metadata.
	Metadata map[string]string
}

// RoutingResult contains the result of a routing decision.
// It includes the selected provider, the strategy used, and metadata
// about the routing decision for audit trail.
type RoutingResult struct {
	// Provider is the selected provider instance.
	Provider providers.Provider

	// ProviderName is the name of the selected provider.
	ProviderName string

	// Strategy is the routing strategy that was used.
	// Values: "round-robin", "sticky", "manual", "policy", "health-based"
	Strategy string

	// Reason explains why this provider was selected.
	Reason string

	// IsHealthy indicates whether the selected provider is healthy.
	IsHealthy bool

	// IsFallback indicates whether this is a fallback selection.
	// True if the initially selected provider was unavailable.
	IsFallback bool

	// AttemptedProviders contains the names of providers tried before selection.
	AttemptedProviders []string

	// Metadata contains additional routing metadata.
	Metadata map[string]string
}

// StickyEntry represents a single sticky routing cache entry.
// It records which provider is assigned to a specific user/session/API key.
type StickyEntry struct {
	// ProviderName is the name of the provider assigned to this entry.
	ProviderName string

	// ExpiresAt is when this entry expires (0 for no expiry).
	ExpiresAt time.Time

	// CreatedAt is when this entry was created.
	CreatedAt time.Time

	// AccessCount tracks how many times this entry was accessed.
	// Used for LRU eviction.
	AccessCount int

	// LastAccessedAt tracks the last access time for LRU eviction.
	LastAccessedAt time.Time
}

// RoutingStats contains statistics about routing decisions.
// All counters are updated atomically for thread safety.
type RoutingStats struct {
	// TotalRequests is the total number of routing requests processed.
	TotalRequests int64

	// RequestsPerProvider tracks requests routed to each provider.
	// Key: provider name, Value: request count
	RequestsPerProvider map[string]int64

	// StrategyUseCount tracks how many times each strategy was used.
	// Key: strategy name, Value: use count
	StrategyUseCount map[string]int64

	// HealthFilteredCount is the number of requests where unhealthy providers were filtered.
	HealthFilteredCount int64

	// ManualOverrideCount is the number of manual provider selections.
	ManualOverrideCount int64

	// PolicyOverrideCount is the number of policy-driven routing decisions.
	PolicyOverrideCount int64

	// Errors is the total number of routing errors.
	Errors int64

	// LastResetTime is when statistics were last reset.
	LastResetTime time.Time
}
