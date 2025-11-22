package engine

import (
	"errors"
	"fmt"
	"time"
)

// Common sentinel errors
var (
	// ErrNoPoliciesLoaded indicates no policies are loaded in the engine.
	ErrNoPoliciesLoaded = errors.New("no policies loaded")

	// ErrContextCancelled indicates the evaluation context was cancelled.
	ErrContextCancelled = errors.New("evaluation context cancelled")

	// ErrInvalidConfig indicates invalid engine configuration.
	ErrInvalidConfig = errors.New("invalid engine configuration")
)

// EvaluationError is the base error type for all policy evaluation errors.
type EvaluationError struct {
	PolicyID string
	RuleID   string
	Message  string
	Cause    error
}

// Error returns the error message.
func (e *EvaluationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("policy %s rule %s: %s: %v", e.PolicyID, e.RuleID, e.Message, e.Cause)
	}
	return fmt.Sprintf("policy %s rule %s: %s", e.PolicyID, e.RuleID, e.Message)
}

// Unwrap returns the underlying cause.
func (e *EvaluationError) Unwrap() error {
	return e.Cause
}

// TimeoutError indicates a policy evaluation exceeded its timeout.
type TimeoutError struct {
	PolicyID string
	RuleID   string
	Timeout  time.Duration
}

// Error returns the error message.
func (e *TimeoutError) Error() string {
	if e.RuleID != "" {
		return fmt.Sprintf("policy %s rule %s: evaluation timeout after %v", e.PolicyID, e.RuleID, e.Timeout)
	}
	return fmt.Sprintf("policy %s: evaluation timeout after %v", e.PolicyID, e.Timeout)
}

// ConditionError indicates a condition evaluation failure.
type ConditionError struct {
	PolicyID  string
	RuleID    string
	FieldName string
	Cause     error
}

// Error returns the error message.
func (e *ConditionError) Error() string {
	return fmt.Sprintf("policy %s rule %s: condition error on field %q: %v", e.PolicyID, e.RuleID, e.FieldName, e.Cause)
}

// Unwrap returns the underlying cause.
func (e *ConditionError) Unwrap() error {
	return e.Cause
}

// ActionError indicates an action execution failure.
type ActionError struct {
	PolicyID   string
	RuleID     string
	ActionType string
	Cause      error
}

// Error returns the error message.
func (e *ActionError) Error() string {
	return fmt.Sprintf("policy %s rule %s: action %s failed: %v", e.PolicyID, e.RuleID, e.ActionType, e.Cause)
}

// Unwrap returns the underlying cause.
func (e *ActionError) Unwrap() error {
	return e.Cause
}

// ValidationError indicates a policy validation failure.
type ValidationError struct {
	PolicyID string
	Errors   []string
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("policy %s: validation error: %s", e.PolicyID, e.Errors[0])
	}
	return fmt.Sprintf("policy %s: %d validation errors: %v", e.PolicyID, len(e.Errors), e.Errors)
}

// ReloadError indicates a policy reload failure.
type ReloadError struct {
	Path  string
	Cause error
}

// Error returns the error message.
func (e *ReloadError) Error() string {
	return fmt.Sprintf("policy reload failed for %q: %v", e.Path, e.Cause)
}

// Unwrap returns the underlying cause.
func (e *ReloadError) Unwrap() error {
	return e.Cause
}

// ProviderNotFoundError indicates a routing action references a non-existent provider.
type ProviderNotFoundError struct {
	ProviderName string
}

// Error returns the error message.
func (e *ProviderNotFoundError) Error() string {
	return fmt.Sprintf("provider not found: %q", e.ProviderName)
}

// FieldNotFoundError indicates a condition references a non-existent field.
type FieldNotFoundError struct {
	FieldName string
}

// Error returns the error message.
func (e *FieldNotFoundError) Error() string {
	return fmt.Sprintf("field not found: %q", e.FieldName)
}

// TypeMismatchError indicates a type mismatch in condition evaluation.
type TypeMismatchError struct {
	FieldName    string
	ExpectedType string
	ActualType   string
}

// Error returns the error message.
func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("type mismatch for field %q: expected %s, got %s", e.FieldName, e.ExpectedType, e.ActualType)
}
