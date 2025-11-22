package engine

import (
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/processing"
)

// PolicyAction represents the final decision after policy evaluation.
type PolicyAction string

const (
	// ActionAllow explicitly allows the request to proceed.
	ActionAllow PolicyAction = "allow"

	// ActionBlock blocks the request with an error message.
	ActionBlock PolicyAction = "block"

	// ActionTransform allows the request with transformations applied.
	ActionTransform PolicyAction = "transform"

	// ActionRoute routes the request to a specific provider/model.
	ActionRoute PolicyAction = "route"
)

// PolicyDecision represents the result of evaluating policies against a request or response.
// It contains the final action, all matched rules, and metadata about the evaluation.
type PolicyDecision struct {
	// Action is the final policy action (allow, block, transform, route).
	Action PolicyAction

	// MatchedRules contains all rules that matched during evaluation.
	MatchedRules []*MatchedRule

	// BlockReason explains why the request was blocked (if Action is ActionBlock).
	BlockReason string

	// BlockStatusCode is the HTTP status code to return (if Action is ActionBlock).
	BlockStatusCode int

	// Transformations contains request transformations to apply.
	Transformations []Transformation

	// Redactions contains content redactions to apply.
	Redactions []Redaction

	// RoutingTarget specifies the provider/model to route to (if Action is ActionRoute).
	RoutingTarget *RoutingTarget

	// Tags contains metadata tags accumulated during evaluation.
	Tags map[string]string

	// Notifications contains notifications to send (webhooks, alerts).
	Notifications []Notification

	// EvaluationTime is the total time taken to evaluate all policies.
	EvaluationTime time.Duration

	// Trace contains detailed evaluation trace (if enabled).
	Trace *EvaluationTrace
}

// MatchedRule represents a single rule that matched during evaluation.
type MatchedRule struct {
	// PolicyID is the unique identifier of the policy containing this rule.
	PolicyID string

	// PolicyName is the human-readable name of the policy.
	PolicyName string

	// RuleID is the unique identifier of the rule within the policy.
	RuleID string

	// RuleName is the human-readable name of the rule.
	RuleName string

	// ConditionResult indicates whether the rule's condition matched.
	ConditionResult bool

	// ActionsExecuted contains the results of executing each action in the rule.
	ActionsExecuted []*ActionResult

	// EvaluationTime is the time taken to evaluate this rule.
	EvaluationTime time.Duration

	// Error contains any error that occurred during rule evaluation.
	Error error
}

// ActionResult represents the result of executing a single action.
type ActionResult struct {
	// ActionType is the type of action executed (e.g., "deny", "redact", "route").
	ActionType ast.ActionType

	// Success indicates whether the action executed successfully.
	Success bool

	// Error contains any error that occurred during action execution.
	Error error

	// Details contains action-specific details (e.g., redacted content, routing target).
	Details map[string]interface{}
}

// Transformation represents a modification to be applied to a request.
type Transformation struct {
	// Field is the request field to modify (e.g., "temperature", "max_tokens").
	Field string

	// Value is the new value to set.
	Value interface{}

	// Operation is the type of transformation ("set", "append", "multiply").
	Operation string
}

// Redaction represents content redaction to be applied to a request or response.
type Redaction struct {
	// Field is the field to redact from (e.g., "prompt", "messages", "response").
	Field string

	// Strategy is the redaction strategy ("mask", "remove", "replace").
	Strategy string

	// Pattern is the regex pattern to match for redaction (optional).
	Pattern string

	// Replacement is the replacement text (for "replace" and "mask" strategies).
	Replacement string

	// MatchCount is the number of matches that were redacted.
	MatchCount int
}

// RoutingTarget specifies the provider and model to route a request to.
type RoutingTarget struct {
	// Provider is the provider name (e.g., "openai", "anthropic").
	Provider string

	// Model is the model name (e.g., "gpt-4", "claude-3-opus").
	Model string

	// Fallback contains fallback providers if the primary is unavailable.
	Fallback []string
}

// Notification represents an external notification to send (webhook, alert).
type Notification struct {
	// Type is the notification type ("webhook", "email", "slack").
	Type string

	// Destination is the notification destination (URL, email address, etc.).
	Destination string

	// Payload contains the notification payload.
	Payload map[string]interface{}

	// Async indicates whether the notification should be sent asynchronously.
	Async bool
}

// EvaluationContext contains the context for a single policy evaluation.
// It is passed through all evaluation steps and accumulates state.
type EvaluationContext struct {
	// RequestID is the unique identifier for this request.
	RequestID string

	// Request contains the enriched request metadata (for pre-request evaluation).
	Request *processing.EnrichedRequest

	// Response contains the enriched response metadata (for post-response evaluation).
	Response *processing.EnrichedResponse

	// MatchedRules accumulates rules that have matched so far.
	MatchedRules []*MatchedRule

	// Tags accumulates metadata tags during evaluation.
	Tags map[string]string

	// Transformations accumulates request transformations.
	Transformations []Transformation

	// Redactions accumulates content redactions.
	Redactions []Redaction

	// Notifications accumulates notifications to send.
	Notifications []Notification

	// RoutingTarget is the routing target set by routing actions.
	RoutingTarget *RoutingTarget

	// BlockReason is set by blocking actions.
	BlockReason string

	// BlockStatusCode is the HTTP status code for blocking actions.
	BlockStatusCode int

	// Trace records evaluation steps (if tracing is enabled).
	Trace *EvaluationTrace

	// StartTime is when evaluation started.
	StartTime time.Time

	// Stopped indicates whether evaluation should stop (short-circuit).
	Stopped bool
}

// EvaluationTrace records detailed steps during policy evaluation for debugging.
type EvaluationTrace struct {
	// Steps contains individual trace steps.
	Steps []*TraceStep

	// TotalTime is the total evaluation time.
	TotalTime time.Duration
}

// TraceStep represents a single step in the evaluation trace.
type TraceStep struct {
	// StepType identifies the type of step ("policy_start", "rule_eval", "condition_match", "action_exec").
	StepType string

	// PolicyID is the policy being evaluated.
	PolicyID string

	// RuleID is the rule being evaluated.
	RuleID string

	// Details contains step-specific details.
	Details string

	// Timestamp is when this step occurred.
	Timestamp time.Time

	// Duration is how long this step took.
	Duration time.Duration
}

// ConditionMatch represents the result of evaluating a single condition.
type ConditionMatch struct {
	// Matched indicates whether the condition was satisfied.
	Matched bool

	// FieldName is the field being evaluated.
	FieldName string

	// ExpectedValue is the expected value from the policy.
	ExpectedValue interface{}

	// ActualValue is the actual value from the request/response.
	ActualValue interface{}

	// MatchType describes the type of match ("exact", "pattern", "threshold", "time").
	MatchType string

	// Error contains any error that occurred during evaluation.
	Error error
}

// AddTag adds a metadata tag to the evaluation context.
func (ctx *EvaluationContext) AddTag(key, value string) {
	if ctx.Tags == nil {
		ctx.Tags = make(map[string]string)
	}
	ctx.Tags[key] = value
}

// AddTransformation adds a request transformation to the evaluation context.
func (ctx *EvaluationContext) AddTransformation(field, operation string, value interface{}) {
	ctx.Transformations = append(ctx.Transformations, Transformation{
		Field:     field,
		Value:     value,
		Operation: operation,
	})
}

// AddRedaction adds a content redaction to the evaluation context.
func (ctx *EvaluationContext) AddRedaction(field, strategy, pattern, replacement string, matchCount int) {
	ctx.Redactions = append(ctx.Redactions, Redaction{
		Field:       field,
		Strategy:    strategy,
		Pattern:     pattern,
		Replacement: replacement,
		MatchCount:  matchCount,
	})
}

// AddNotification adds a notification to the evaluation context.
func (ctx *EvaluationContext) AddNotification(notifType, destination string, payload map[string]interface{}, async bool) {
	ctx.Notifications = append(ctx.Notifications, Notification{
		Type:        notifType,
		Destination: destination,
		Payload:     payload,
		Async:       async,
	})
}

// SetRouting sets the routing target in the evaluation context.
func (ctx *EvaluationContext) SetRouting(provider, model string, fallback []string) {
	ctx.RoutingTarget = &RoutingTarget{
		Provider: provider,
		Model:    model,
		Fallback: fallback,
	}
}

// SetBlock sets the block reason and status code in the evaluation context.
func (ctx *EvaluationContext) SetBlock(reason string, statusCode int) {
	ctx.BlockReason = reason
	ctx.BlockStatusCode = statusCode
	ctx.Stopped = true
}

// Stop stops further evaluation (short-circuit).
func (ctx *EvaluationContext) Stop() {
	ctx.Stopped = true
}

// AddTraceStep adds a step to the evaluation trace (if tracing is enabled).
func (ctx *EvaluationContext) AddTraceStep(stepType, policyID, ruleID, details string, duration time.Duration) {
	if ctx.Trace == nil {
		return
	}
	ctx.Trace.Steps = append(ctx.Trace.Steps, &TraceStep{
		StepType:  stepType,
		PolicyID:  policyID,
		RuleID:    ruleID,
		Details:   details,
		Timestamp: time.Now(),
		Duration:  duration,
	})
}
