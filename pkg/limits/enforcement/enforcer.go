package enforcement

import (
	"context"
	"fmt"
	"time"
)

// Enforcer executes enforcement actions for limit violations.
//
// The Enforcer is responsible for determining what happens when a limit
// is exceeded (block, queue, downgrade, or alert) and executing that action.
type Enforcer struct {
	config Config
}

// NewEnforcer creates a new enforcement action enforcer.
//
// Example:
//
//	enforcer := NewEnforcer(Config{
//	    DefaultAction: ActionBlock,
//	    QueueDepth:    100,
//	    QueueTimeout:  30 * time.Second,
//	    ModelDowngrades: map[string]string{
//	        "gpt-4":           "gpt-3.5-turbo",
//	        "claude-3-opus":   "claude-3-sonnet",
//	    },
//	})
func NewEnforcer(config Config) *Enforcer {
	// Apply defaults
	if config.DefaultAction == "" {
		config.DefaultAction = ActionBlock
	}
	if config.QueueDepth == 0 {
		config.QueueDepth = 100
	}
	if config.QueueTimeout == 0 {
		config.QueueTimeout = 30 * time.Second
	}
	if config.ModelDowngrades == nil {
		config.ModelDowngrades = make(map[string]string)
	}

	return &Enforcer{
		config: config,
	}
}

// Enforce executes the enforcement action for a limit violation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - action: The enforcement action to take
//   - reason: Why the limit was exceeded
//   - model: The requested model (for downgrades)
//   - retryAfter: How long to wait before retrying
//
// Returns a Result indicating what should happen to the request.
func (e *Enforcer) Enforce(ctx context.Context, action Action, reason string, model string, retryAfter time.Duration) (*Result, error) {
	// If no action specified, use default
	if action == "" {
		action = e.config.DefaultAction
	}

	switch action {
	case ActionAllow:
		return e.enforceAllow(), nil

	case ActionBlock:
		return e.enforceBlock(reason, retryAfter), nil

	case ActionQueue:
		return e.enforceQueue(ctx, reason, retryAfter), nil

	case ActionDowngrade:
		return e.enforceDowngrade(model, reason), nil

	case ActionAlert:
		return e.enforceAlert(reason), nil

	default:
		return e.enforceBlock(reason, retryAfter), nil
	}
}

// enforceAllow allows the request to proceed.
func (e *Enforcer) enforceAllow() *Result {
	return &Result{
		Allowed: true,
		Action:  ActionAllow,
	}
}

// enforceBlock blocks the request with 429 Too Many Requests.
func (e *Enforcer) enforceBlock(reason string, retryAfter time.Duration) *Result {
	return &Result{
		Allowed:    false,
		Action:     ActionBlock,
		Reason:     reason,
		RetryAfter: retryAfter,
	}
}

// enforceQueue queues the request until capacity is available.
// Note: Actual queuing logic would be implemented in the Manager.
// This just indicates that queuing should be attempted.
func (e *Enforcer) enforceQueue(ctx context.Context, reason string, retryAfter time.Duration) *Result {
	// For now, return a result indicating queuing should be attempted.
	// The actual queue implementation would be in the Manager.
	return &Result{
		Allowed:    false, // Not immediately allowed
		Action:     ActionQueue,
		Reason:     reason,
		RetryAfter: retryAfter,
	}
}

// enforceDowngrade downgrades to a cheaper model.
func (e *Enforcer) enforceDowngrade(model string, reason string) *Result {
	// Look up cheaper model
	downgradedModel, exists := e.config.ModelDowngrades[model]
	if !exists {
		// No downgrade available, fall back to blocking
		return &Result{
			Allowed: false,
			Action:  ActionBlock,
			Reason:  fmt.Sprintf("%s (no downgrade available for model %s)", reason, model),
		}
	}

	return &Result{
		Allowed:         true, // Allow but with different model
		Action:          ActionDowngrade,
		DowngradedModel: downgradedModel,
	}
}

// enforceAlert triggers an alert but allows the request.
func (e *Enforcer) enforceAlert(reason string) *Result {
	return &Result{
		Allowed:      true,
		Action:       ActionAlert,
		AlertMessage: reason,
	}
}

// GetConfig returns the current enforcer configuration.
func (e *Enforcer) GetConfig() Config {
	return e.config
}
