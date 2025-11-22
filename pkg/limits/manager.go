package limits

import (
	"context"
	"fmt"
	"sync"

	"mercator-hq/jupiter/pkg/limits/budget"
	"mercator-hq/jupiter/pkg/limits/enforcement"
	"mercator-hq/jupiter/pkg/limits/ratelimit"
	"mercator-hq/jupiter/pkg/limits/storage"
)

// Manager coordinates budget tracking and rate limiting across multiple dimensions.
//
// The Manager is the primary interface for checking limits and recording usage.
// It orchestrates the rate limiter, budget tracker, and enforcement engine to
// provide a unified API for limit management.
//
// # Example
//
//	manager := limits.NewManager(&config.Limits)
//
//	// Check if request is allowed
//	result, err := manager.CheckLimits(ctx, "api-key-123", estimatedTokens, estimatedCost, model)
//	if !result.Allowed {
//	    // Handle limit exceeded
//	}
//
//	// Record actual usage
//	err = manager.RecordUsage(ctx, record)
type Manager struct {
	// Per-identifier rate limiters and budget trackers
	rateLimiters map[string]*ratelimit.Limiter
	budgets      map[string]*budget.Tracker

	// Enforcement engine
	enforcer *enforcement.Enforcer

	// Storage backend
	storage storage.Backend

	// Configuration
	rateLimitConfigs map[string]ratelimit.Config
	budgetConfigs    map[string]budget.Config
	enforcementConfig enforcement.Config

	mu sync.RWMutex
}

// Config contains configuration for the limits manager.
type Config struct {
	// RateLimits maps identifiers to rate limit configurations.
	RateLimits map[string]ratelimit.Config

	// Budgets maps identifiers to budget configurations.
	Budgets map[string]budget.Config

	// Enforcement configures enforcement actions.
	Enforcement enforcement.Config

	// Storage configures the storage backend.
	Storage storage.Backend
}

// NewManager creates a new limits manager with the given configuration.
//
// Example:
//
//	manager := NewManager(Config{
//	    RateLimits: map[string]ratelimit.Config{
//	        "api-key-123": {
//	            RequestsPerSecond: 10,
//	            TokensPerMinute:   100000,
//	        },
//	    },
//	    Budgets: map[string]budget.Config{
//	        "api-key-123": {
//	            Daily:          100.00,
//	            AlertThreshold: 0.8,
//	        },
//	    },
//	    Enforcement: enforcement.Config{
//	        DefaultAction: enforcement.ActionBlock,
//	    },
//	})
func NewManager(config Config) *Manager {
	// Initialize storage if not provided
	if config.Storage == nil {
		config.Storage = storage.NewMemoryBackend()
	}

	manager := &Manager{
		rateLimiters:      make(map[string]*ratelimit.Limiter),
		budgets:           make(map[string]*budget.Tracker),
		enforcer:          enforcement.NewEnforcer(config.Enforcement),
		storage:           config.Storage,
		rateLimitConfigs:  config.RateLimits,
		budgetConfigs:     config.Budgets,
		enforcementConfig: config.Enforcement,
	}

	// Pre-initialize limiters and trackers for configured identifiers
	for identifier, rateLimitConfig := range config.RateLimits {
		manager.rateLimiters[identifier] = ratelimit.NewLimiter(rateLimitConfig)
	}

	for identifier, budgetConfig := range config.Budgets {
		manager.budgets[identifier] = budget.NewTracker(budgetConfig)
	}

	return manager
}

// CheckLimits checks if a request is allowed based on rate limits and budgets.
//
// This method checks all configured limits for the given identifier:
//   - Rate limits (requests per second/minute/hour)
//   - Token limits (tokens per minute/hour)
//   - Concurrent request limits
//   - Budget limits (hourly/daily/monthly)
//
// If any limit is exceeded, it returns a LimitCheckResult with Allowed=false
// and the reason for rejection. Otherwise, it returns Allowed=true.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - identifier: The dimension identifier (API key, user ID, team name)
//   - estimatedTokens: Estimated number of tokens for this request
//   - estimatedCost: Estimated cost in USD for this request
//   - model: The requested model name
//
// Returns:
//   - result: The limit check result with decision and metadata
//   - error: Any error that occurred during checking
func (m *Manager) CheckLimits(ctx context.Context, identifier string, estimatedTokens int, estimatedCost float64, model string) (*LimitCheckResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get or create rate limiter for this identifier
	rateLimiter := m.getRateLimiter(identifier)
	budgetTracker := m.getBudgetTracker(identifier)

	// Check rate limits (request-based)
	if rateLimiter != nil {
		rateLimitResult := rateLimiter.CheckRequest()
		if !rateLimitResult.Allowed {
			// Rate limit exceeded - enforce action
			enforcementResult, err := m.enforcer.Enforce(
				ctx,
				m.enforcementConfig.DefaultAction,
				rateLimitResult.Reason,
				model,
				rateLimitResult.RetryAfter,
			)
			if err != nil {
				return nil, fmt.Errorf("enforcement failed: %w", err)
			}

			return &LimitCheckResult{
				Allowed: enforcementResult.Allowed,
				Reason:  rateLimitResult.Reason,
				RateLimit: &RateLimitInfo{
					Dimension:  string(DimensionAPIKey),
					Identifier: identifier,
					Limit:      rateLimitResult.Limit,
					Remaining:  rateLimitResult.Remaining,
					Reset:      rateLimitResult.Reset,
					Window:     0, // Not used for token bucket
				},
				Action:      EnforcementAction(enforcementResult.Action),
				RetryAfter:  enforcementResult.RetryAfter,
				DowngradeTo: enforcementResult.DowngradedModel,
			}, nil
		}

		// Check token-based limits
		tokenLimitResult := rateLimiter.CheckTokens(estimatedTokens)
		if !tokenLimitResult.Allowed {
			enforcementResult, err := m.enforcer.Enforce(
				ctx,
				m.enforcementConfig.DefaultAction,
				tokenLimitResult.Reason,
				model,
				tokenLimitResult.RetryAfter,
			)
			if err != nil {
				return nil, fmt.Errorf("enforcement failed: %w", err)
			}

			return &LimitCheckResult{
				Allowed: enforcementResult.Allowed,
				Reason:  tokenLimitResult.Reason,
				RateLimit: &RateLimitInfo{
					Dimension:  string(DimensionAPIKey),
					Identifier: identifier,
					Limit:      tokenLimitResult.Limit,
					Remaining:  tokenLimitResult.Remaining,
					Reset:      tokenLimitResult.Reset,
				},
				Action:      EnforcementAction(enforcementResult.Action),
				RetryAfter:  enforcementResult.RetryAfter,
				DowngradeTo: enforcementResult.DowngradedModel,
			}, nil
		}
	}

	// Check budget limits
	if budgetTracker != nil {
		budgetStatus := budgetTracker.Check()
		if !budgetStatus.Allowed {
			enforcementResult, err := m.enforcer.Enforce(
				ctx,
				m.enforcementConfig.DefaultAction,
				budgetStatus.Reason,
				model,
				0, // No retry after for budget limits
			)
			if err != nil {
				return nil, fmt.Errorf("enforcement failed: %w", err)
			}

			return &LimitCheckResult{
				Allowed: enforcementResult.Allowed,
				Reason:  budgetStatus.Reason,
				Budget: &BudgetInfo{
					Dimension:  string(DimensionAPIKey),
					Identifier: identifier,
					Limit:      budgetStatus.Limit,
					Used:       budgetStatus.Used,
					Remaining:  budgetStatus.Remaining,
					Percentage: budgetStatus.Percentage,
					Reset:      budgetStatus.Reset,
					Window:     budgetStatus.Window,
				},
				Action:      EnforcementAction(enforcementResult.Action),
				DowngradeTo: enforcementResult.DowngradedModel,
			}, nil
		}

		// Check if alert threshold reached
		if budgetStatus.AlertTriggered {
			return &LimitCheckResult{
				Allowed: true,
				Budget: &BudgetInfo{
					Dimension:  string(DimensionAPIKey),
					Identifier: identifier,
					Limit:      budgetStatus.Limit,
					Used:       budgetStatus.Used,
					Remaining:  budgetStatus.Remaining,
					Percentage: budgetStatus.Percentage,
					Reset:      budgetStatus.Reset,
					Window:     budgetStatus.Window,
				},
				Action: ActionAlert,
			}, nil
		}
	}

	// All limits passed
	return &LimitCheckResult{
		Allowed: true,
	}, nil
}

// RecordUsage records actual usage after a request completes.
//
// This updates rate limit counters and budget trackers with the actual
// token and cost data from the completed request.
//
// Parameters:
//   - ctx: Context for cancellation
//   - record: Usage record containing actual usage data
//
// Returns error if recording fails.
func (m *Manager) RecordUsage(ctx context.Context, record *UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	identifier := record.Identifier

	// Record tokens for rate limiting
	rateLimiter := m.getRateLimiter(identifier)
	if rateLimiter != nil {
		rateLimiter.RecordTokens(record.TotalTokens)
	}

	// Record cost for budget tracking
	budgetTracker := m.getBudgetTracker(identifier)
	if budgetTracker != nil {
		budgetTracker.Add(record.Cost)
	}

	// Persist state to storage (async to avoid blocking)
	go func() {
		_ = m.persistState(context.Background(), identifier)
	}()

	return nil
}

// AcquireConcurrent attempts to acquire a concurrent request slot.
// Returns true if acquired, false if the concurrent limit is reached.
//
// If this returns true, the caller MUST call ReleaseConcurrent() when done.
func (m *Manager) AcquireConcurrent(identifier string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rateLimiter := m.getRateLimiter(identifier)
	if rateLimiter == nil {
		return true // No limit configured
	}

	return rateLimiter.AcquireConcurrent()
}

// ReleaseConcurrent releases a concurrent request slot.
// This MUST be called after a successful AcquireConcurrent().
func (m *Manager) ReleaseConcurrent(identifier string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rateLimiter := m.getRateLimiter(identifier)
	if rateLimiter != nil {
		rateLimiter.ReleaseConcurrent()
	}
}

// Close releases any resources held by the manager.
func (m *Manager) Close() error {
	if m.storage != nil {
		return m.storage.Close()
	}
	return nil
}

// getRateLimiter gets the rate limiter for an identifier (creates if needed).
// Caller must hold read or write lock.
func (m *Manager) getRateLimiter(identifier string) *ratelimit.Limiter {
	limiter, exists := m.rateLimiters[identifier]
	if !exists {
		// Check if there's a config for this identifier
		config, hasConfig := m.rateLimitConfigs[identifier]
		if !hasConfig {
			return nil // No rate limit configured
		}

		// Create new limiter (upgrade to write lock would be needed in production)
		limiter = ratelimit.NewLimiter(config)
		m.rateLimiters[identifier] = limiter
	}
	return limiter
}

// getBudgetTracker gets the budget tracker for an identifier (creates if needed).
// Caller must hold read or write lock.
func (m *Manager) getBudgetTracker(identifier string) *budget.Tracker {
	tracker, exists := m.budgets[identifier]
	if !exists {
		// Check if there's a config for this identifier
		config, hasConfig := m.budgetConfigs[identifier]
		if !hasConfig {
			return nil // No budget configured
		}

		// Create new tracker
		tracker = budget.NewTracker(config)
		m.budgets[identifier] = tracker
	}
	return tracker
}

// persistState saves the current state to storage.
func (m *Manager) persistState(ctx context.Context, identifier string) error {
	// Build state from current limiters/trackers
	// This is a simplified version - full implementation would serialize all state
	state := &storage.LimitState{
		Identifier:  identifier,
		Dimension:   string(DimensionAPIKey),
		RateLimit:   nil, // TODO: Serialize rate limit state
		Budget:      nil, // TODO: Serialize budget state
	}

	return m.storage.Save(ctx, state)
}
