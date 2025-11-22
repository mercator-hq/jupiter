package engine

import (
	"fmt"
	"time"
)

// FailSafeMode determines how the engine handles evaluation errors.
type FailSafeMode string

const (
	// FailOpen allows requests to proceed when evaluation errors occur.
	// Use this for high-availability scenarios where blocking legitimate traffic
	// is worse than allowing potentially risky requests.
	FailOpen FailSafeMode = "fail-open"

	// FailClosed blocks requests when evaluation errors occur.
	// Use this for security-critical scenarios where blocking is preferable
	// to allowing potentially risky requests. This is the default.
	FailClosed FailSafeMode = "fail-closed"

	// FailSafeDefault applies a default action when evaluation errors occur.
	// The default action is specified in the configuration.
	FailSafeDefault FailSafeMode = "fail-safe-default"
)

// EngineConfig contains configuration for the policy evaluation engine.
type EngineConfig struct {
	// FailSafeMode determines how to handle evaluation errors.
	// Default: FailClosed (block on error).
	FailSafeMode FailSafeMode

	// DefaultAction is the action to take when FailSafeMode is FailSafeDefault.
	// Default: "block".
	DefaultAction PolicyAction

	// RuleTimeout is the maximum time allowed to evaluate a single rule.
	// Default: 50ms.
	RuleTimeout time.Duration

	// PolicyTimeout is the maximum time allowed to evaluate a single policy.
	// Default: 100ms.
	PolicyTimeout time.Duration

	// EnableTrace enables detailed evaluation tracing for debugging.
	// Warning: Enabling trace adds performance overhead.
	// Default: false.
	EnableTrace bool

	// MaxPolicies is the maximum number of policies to load.
	// This prevents DoS via excessive policy count.
	// Default: 100.
	MaxPolicies int

	// MaxRulesPerPolicy is the maximum number of rules per policy.
	// This prevents DoS via excessive rule count.
	// Default: 50.
	MaxRulesPerPolicy int

	// BusinessHours defines business hours for time-based conditions.
	// Default: Mon-Fri, 9am-5pm UTC.
	BusinessHours *BusinessHoursConfig
}

// DefaultEngineConfig returns the default engine configuration.
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		FailSafeMode:      FailClosed,
		DefaultAction:     ActionBlock,
		RuleTimeout:       50 * time.Millisecond,
		PolicyTimeout:     100 * time.Millisecond,
		EnableTrace:       false,
		MaxPolicies:       100,
		MaxRulesPerPolicy: 50,
		BusinessHours:     DefaultBusinessHoursConfig(),
	}
}

// Validate validates the engine configuration.
func (c *EngineConfig) Validate() error {
	// Validate fail-safe mode
	switch c.FailSafeMode {
	case FailOpen, FailClosed, FailSafeDefault:
		// Valid
	default:
		return fmt.Errorf("%w: invalid fail-safe mode %q", ErrInvalidConfig, c.FailSafeMode)
	}

	// Validate default action
	if c.FailSafeMode == FailSafeDefault {
		switch c.DefaultAction {
		case ActionAllow, ActionBlock:
			// Valid
		default:
			return fmt.Errorf("%w: invalid default action %q for fail-safe-default mode", ErrInvalidConfig, c.DefaultAction)
		}
	}

	// Validate timeouts
	if c.RuleTimeout <= 0 {
		return fmt.Errorf("%w: rule timeout must be positive", ErrInvalidConfig)
	}
	if c.PolicyTimeout <= 0 {
		return fmt.Errorf("%w: policy timeout must be positive", ErrInvalidConfig)
	}
	if c.RuleTimeout > c.PolicyTimeout {
		return fmt.Errorf("%w: rule timeout cannot exceed policy timeout", ErrInvalidConfig)
	}

	// Validate limits
	if c.MaxPolicies <= 0 {
		return fmt.Errorf("%w: max policies must be positive", ErrInvalidConfig)
	}
	if c.MaxRulesPerPolicy <= 0 {
		return fmt.Errorf("%w: max rules per policy must be positive", ErrInvalidConfig)
	}

	return nil
}

// WithFailSafeMode sets the fail-safe mode.
func (c *EngineConfig) WithFailSafeMode(mode FailSafeMode) *EngineConfig {
	c.FailSafeMode = mode
	return c
}

// WithDefaultAction sets the default action for fail-safe-default mode.
func (c *EngineConfig) WithDefaultAction(action PolicyAction) *EngineConfig {
	c.DefaultAction = action
	return c
}

// WithRuleTimeout sets the rule evaluation timeout.
func (c *EngineConfig) WithRuleTimeout(timeout time.Duration) *EngineConfig {
	c.RuleTimeout = timeout
	return c
}

// WithPolicyTimeout sets the policy evaluation timeout.
func (c *EngineConfig) WithPolicyTimeout(timeout time.Duration) *EngineConfig {
	c.PolicyTimeout = timeout
	return c
}

// WithTrace enables or disables evaluation tracing.
func (c *EngineConfig) WithTrace(enabled bool) *EngineConfig {
	c.EnableTrace = enabled
	return c
}

// WithMaxPolicies sets the maximum number of policies.
func (c *EngineConfig) WithMaxPolicies(max int) *EngineConfig {
	c.MaxPolicies = max
	return c
}

// WithMaxRulesPerPolicy sets the maximum number of rules per policy.
func (c *EngineConfig) WithMaxRulesPerPolicy(max int) *EngineConfig {
	c.MaxRulesPerPolicy = max
	return c
}
