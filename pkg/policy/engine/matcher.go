package engine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// DefaultMatcher is the default implementation of ConditionMatcher.
type DefaultMatcher struct {
	logger        *slog.Logger
	businessHours *BusinessHoursConfig
	failSafeMode  FailSafeMode
}

// NewDefaultMatcher creates a new default condition matcher.
func NewDefaultMatcher(logger *slog.Logger, config *EngineConfig) *DefaultMatcher {
	if logger == nil {
		logger = slog.Default()
	}
	if config == nil {
		config = DefaultEngineConfig()
	}
	return &DefaultMatcher{
		logger:        logger,
		businessHours: config.BusinessHours,
		failSafeMode:  config.FailSafeMode,
	}
}

// Match evaluates a condition node and returns whether it matched.
func (m *DefaultMatcher) Match(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	if condition == nil {
		return true, nil // No condition means always match
	}

	switch condition.Type {
	case ast.ConditionTypeSimple:
		return m.matchSimple(ctx, condition, evalCtx)

	case ast.ConditionTypeAll:
		return m.matchAll(ctx, condition, evalCtx)

	case ast.ConditionTypeAny:
		return m.matchAny(ctx, condition, evalCtx)

	case ast.ConditionTypeNot:
		return m.matchNot(ctx, condition, evalCtx)

	case ast.ConditionTypeFunction:
		return m.matchFunction(ctx, condition, evalCtx)

	default:
		return false, fmt.Errorf("unknown condition type: %q", condition.Type)
	}
}

// matchSimple evaluates a simple condition (field operator value).
func (m *DefaultMatcher) matchSimple(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	// Extract field value from evaluation context
	fieldValue, err := extractField(condition.Field, evalCtx)
	if err != nil {
		// Field not found - respect fail-safe mode
		m.logger.Debug("field not found, applying fail-safe mode",
			"field", condition.Field,
			"error", err,
			"fail_safe_mode", m.failSafeMode,
		)

		// Apply fail-safe mode for missing fields
		switch m.failSafeMode {
		case FailOpen:
			// Treat missing field as match (allow)
			return true, nil
		case FailClosed:
			// Treat missing field as error (block)
			return false, &FieldNotFoundError{FieldName: condition.Field}
		case FailSafeDefault:
			// Treat missing field as no match (continue evaluation)
			return false, nil
		default:
			return false, nil
		}
	}

	// Get expected value
	expectedValue := condition.Value.Value

	// Evaluate operator
	matched, err := evaluateOperator(condition.Operator, fieldValue, expectedValue)
	if err != nil {
		return false, fmt.Errorf("operator %q evaluation failed: %w", condition.Operator, err)
	}

	m.logger.Debug("simple condition evaluated",
		"field", condition.Field,
		"operator", condition.Operator,
		"expected", expectedValue,
		"actual", fieldValue,
		"matched", matched,
	)

	return matched, nil
}

// matchAll evaluates an ALL (AND) condition - all children must match.
func (m *DefaultMatcher) matchAll(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	for _, child := range condition.Children {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		matched, err := m.Match(ctx, child, evalCtx)
		if err != nil {
			return false, err
		}

		// Short-circuit: if any child doesn't match, return false
		if !matched {
			return false, nil
		}
	}

	// All children matched
	return true, nil
}

// matchAny evaluates an ANY (OR) condition - at least one child must match.
func (m *DefaultMatcher) matchAny(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	for _, child := range condition.Children {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		matched, err := m.Match(ctx, child, evalCtx)
		if err != nil {
			return false, err
		}

		// Short-circuit: if any child matches, return true
		if matched {
			return true, nil
		}
	}

	// No children matched
	return false, nil
}

// matchNot evaluates a NOT condition - child must not match.
func (m *DefaultMatcher) matchNot(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	if len(condition.Children) != 1 {
		return false, fmt.Errorf("NOT condition must have exactly one child, got %d", len(condition.Children))
	}

	matched, err := m.Match(ctx, condition.Children[0], evalCtx)
	if err != nil {
		return false, err
	}

	// Negate the result
	return !matched, nil
}

// matchFunction evaluates a function condition (e.g., has_pii(), in_business_hours()).
func (m *DefaultMatcher) matchFunction(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	// Get function name
	fnName := condition.Function

	// Dispatch to function handlers
	switch fnName {
	case "has_pii":
		return m.hasPII(condition, evalCtx)

	case "has_injection":
		return m.hasInjection(condition, evalCtx)

	case "in_business_hours":
		return m.inBusinessHours(condition, evalCtx)

	default:
		return false, fmt.Errorf("unknown function: %q", fnName)
	}
}

// hasPII checks if the specified field contains PII.
func (m *DefaultMatcher) hasPII(condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	// Check if enriched request has content analysis
	if evalCtx.Request == nil || evalCtx.Request.ContentAnalysis == nil {
		return false, nil
	}

	// Check PII detection results
	pii := evalCtx.Request.ContentAnalysis.PIIDetection
	if pii == nil {
		return false, nil
	}

	return pii.HasPII, nil
}

// hasInjection checks if the specified field contains prompt injection.
func (m *DefaultMatcher) hasInjection(condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	// Check if enriched request has content analysis
	if evalCtx.Request == nil || evalCtx.Request.ContentAnalysis == nil {
		return false, nil
	}

	// Check prompt injection detection results
	injection := evalCtx.Request.ContentAnalysis.PromptInjection
	if injection == nil {
		return false, nil
	}

	return injection.HasPromptInjection, nil
}

// inBusinessHours checks if the current time is within business hours.
func (m *DefaultMatcher) inBusinessHours(condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error) {
	if m.businessHours == nil {
		// No business hours configured, treat as always in business hours
		m.logger.Debug("no business hours configured, treating as always in business hours")
		return true, nil
	}

	// Use current time (or time from context if available)
	now := evalCtx.StartTime
	if now.IsZero() {
		now = time.Now()
	}

	isBusinessHours := m.businessHours.IsBusinessHours(now)

	m.logger.Debug("business hours check",
		"time", now,
		"is_business_hours", isBusinessHours,
		"timezone", m.businessHours.Timezone,
	)

	return isBusinessHours, nil
}
