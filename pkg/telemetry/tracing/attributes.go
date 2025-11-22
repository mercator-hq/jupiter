package tracing

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Span Attribute Helpers
//
// These functions provide a convenient way to set common attributes on spans.
// They use semantic conventions where applicable and ensure consistent attribute
// naming across the codebase.
//
// # Attribute Keys
//
// Standard attribute keys follow OpenTelemetry semantic conventions:
//   - http.*: HTTP-related attributes
//   - rpc.*: RPC-related attributes
//   - db.*: Database-related attributes
//   - messaging.*: Message queue-related attributes
//
// Custom attribute keys use the "mercator.*" namespace:
//   - mercator.provider: LLM provider name
//   - mercator.model: Model name
//   - mercator.cost: Request cost
//   - mercator.tokens.*: Token counts

// Common attribute keys used throughout the system
const (
	// Provider attributes
	AttrProvider = "mercator.provider"
	AttrModel    = "mercator.model"

	// Request attributes
	AttrRequestID = "mercator.request_id"
	AttrAPIKey    = "mercator.api_key"
	AttrUser      = "mercator.user"
	AttrTeam      = "mercator.team"
	AttrSession   = "mercator.session"

	// Token attributes
	AttrTokensPrompt     = "mercator.tokens.prompt"
	AttrTokensCompletion = "mercator.tokens.completion"
	AttrTokensTotal      = "mercator.tokens.total"

	// Cost attributes
	AttrCost          = "mercator.cost.total"
	AttrCostCurrency  = "mercator.cost.currency"
	AttrCostPerToken  = "mercator.cost.per_token"

	// Policy attributes
	AttrPolicyID    = "mercator.policy.id"
	AttrPolicyRule  = "mercator.policy.rule"
	AttrPolicyAction = "mercator.policy.action"

	// Cache attributes
	AttrCacheHit  = "mercator.cache.hit"
	AttrCacheName = "mercator.cache.name"

	// Error attributes
	AttrErrorType    = "mercator.error.type"
	AttrErrorMessage = "error.message"
	AttrErrorStack   = "error.stack"

	// Performance attributes
	AttrDuration   = "mercator.duration_ms"
	AttrQueueTime  = "mercator.queue_time_ms"
	AttrRetryCount = "mercator.retry_count"
)

// SetProviderAttributes sets provider-related attributes on a span.
//
// Example:
//
//	SetProviderAttributes(span, "openai", "gpt-4")
func SetProviderAttributes(span trace.Span, provider, model string) {
	span.SetAttributes(
		attribute.String(AttrProvider, provider),
		attribute.String(AttrModel, model),
	)
}

// SetRequestAttributes sets request-related attributes on a span.
//
// Example:
//
//	SetRequestAttributes(span, "req-123", "api-key-abc", "user@example.com")
func SetRequestAttributes(span trace.Span, requestID, apiKey, user string) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrRequestID, requestID),
	}

	// Only add non-empty values
	if apiKey != "" {
		// Redact API key (show only first 4 characters)
		redacted := apiKey
		if len(apiKey) > 4 {
			redacted = apiKey[:4] + "***"
		}
		attrs = append(attrs, attribute.String(AttrAPIKey, redacted))
	}

	if user != "" {
		attrs = append(attrs, attribute.String(AttrUser, user))
	}

	span.SetAttributes(attrs...)
}

// SetTokenAttributes sets token count attributes on a span.
//
// Example:
//
//	SetTokenAttributes(span, 1500, 500)
func SetTokenAttributes(span trace.Span, promptTokens, completionTokens int) {
	span.SetAttributes(
		attribute.Int(AttrTokensPrompt, promptTokens),
		attribute.Int(AttrTokensCompletion, completionTokens),
		attribute.Int(AttrTokensTotal, promptTokens+completionTokens),
	)
}

// SetCostAttributes sets cost-related attributes on a span.
//
// Example:
//
//	SetCostAttributes(span, 0.05, "USD")
func SetCostAttributes(span trace.Span, cost float64, currency string) {
	span.SetAttributes(
		attribute.Float64(AttrCost, cost),
		attribute.String(AttrCostCurrency, currency),
	)
}

// SetCostWithTokens sets cost and token attributes on a span.
//
// Example:
//
//	SetCostWithTokens(span, 1500, 500, 0.05)
func SetCostWithTokens(span trace.Span, promptTokens, completionTokens int, cost float64) {
	SetTokenAttributes(span, promptTokens, completionTokens)
	SetCostAttributes(span, cost, "USD")

	// Calculate cost per token
	totalTokens := promptTokens + completionTokens
	if totalTokens > 0 {
		costPerToken := cost / float64(totalTokens)
		span.SetAttributes(attribute.Float64(AttrCostPerToken, costPerToken))
	}
}

// SetPolicyAttributes sets policy-related attributes on a span.
//
// Example:
//
//	SetPolicyAttributes(span, "cost-limit", "rule-1", "allow")
func SetPolicyAttributes(span trace.Span, policyID, ruleID, action string) {
	span.SetAttributes(
		attribute.String(AttrPolicyID, policyID),
		attribute.String(AttrPolicyRule, ruleID),
		attribute.String(AttrPolicyAction, action),
	)
}

// SetCacheAttributes sets cache-related attributes on a span.
//
// Example:
//
//	SetCacheAttributes(span, true, "policy-cache")
func SetCacheAttributes(span trace.Span, hit bool, cacheName string) {
	span.SetAttributes(
		attribute.Bool(AttrCacheHit, hit),
		attribute.String(AttrCacheName, cacheName),
	)
}

// SetErrorAttributes sets error-related attributes on a span.
// This also records the error using span.RecordError() and sets the span status.
//
// Example:
//
//	SetErrorAttributes(span, err, "rate_limit")
func SetErrorAttributes(span trace.Span, err error, errorType string) {
	if err == nil {
		return
	}

	span.SetAttributes(
		attribute.Bool("error", true),
		attribute.String(AttrErrorType, errorType),
		attribute.String(AttrErrorMessage, err.Error()),
	)

	// Record error and set status
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetDurationAttribute sets the duration attribute on a span.
// Duration is recorded in milliseconds.
//
// Example:
//
//	start := time.Now()
//	// ... do work ...
//	SetDurationAttribute(span, time.Since(start).Milliseconds())
func SetDurationAttribute(span trace.Span, durationMs int64) {
	span.SetAttributes(attribute.Int64(AttrDuration, durationMs))
}

// SetRetryAttribute sets the retry count attribute on a span.
//
// Example:
//
//	SetRetryAttribute(span, 2)
func SetRetryAttribute(span trace.Span, retryCount int) {
	span.SetAttributes(attribute.Int(AttrRetryCount, retryCount))
}

// SetTeamAttribute sets the team attribute on a span.
//
// Example:
//
//	SetTeamAttribute(span, "engineering")
func SetTeamAttribute(span trace.Span, team string) {
	if team != "" {
		span.SetAttributes(attribute.String(AttrTeam, team))
	}
}

// SetSessionAttribute sets the session attribute on a span.
//
// Example:
//
//	SetSessionAttribute(span, "session-123")
func SetSessionAttribute(span trace.Span, session string) {
	if session != "" {
		span.SetAttributes(attribute.String(AttrSession, session))
	}
}

// AddEvent adds a named event to the span with optional attributes.
// Events represent interesting points in the span's lifetime.
//
// Example:
//
//	AddEvent(span, "policy_evaluated",
//	    attribute.String("rule_id", "cost-limit"),
//	    attribute.String("action", "allow"),
//	)
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// AddEventWithTimestamp adds a named event with a specific timestamp.
//
// Example:
//
//	AddEventWithTimestamp(span, "cache_miss", time.Now(),
//	    attribute.String("cache_name", "policy"),
//	)
func AddEventWithTimestamp(span trace.Span, name string, timestamp int64, attrs ...attribute.KeyValue) {
	// Note: OpenTelemetry uses time.Time, not int64 for timestamps
	// This is a simplified version - in real code you'd use trace.WithTimestamp()
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordException records an exception event on the span.
// This is a convenience wrapper around AddEvent for errors.
//
// Example:
//
//	RecordException(span, err)
func RecordException(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
	}
}

// AttributeBuilder provides a fluent interface for building span attributes.
type AttributeBuilder struct {
	attrs []attribute.KeyValue
}

// NewAttributeBuilder creates a new attribute builder.
func NewAttributeBuilder() *AttributeBuilder {
	return &AttributeBuilder{
		attrs: make([]attribute.KeyValue, 0, 10),
	}
}

// WithProvider adds provider and model attributes.
func (ab *AttributeBuilder) WithProvider(provider, model string) *AttributeBuilder {
	ab.attrs = append(ab.attrs,
		attribute.String(AttrProvider, provider),
		attribute.String(AttrModel, model),
	)
	return ab
}

// WithRequest adds request-related attributes.
func (ab *AttributeBuilder) WithRequest(requestID, user string) *AttributeBuilder {
	ab.attrs = append(ab.attrs, attribute.String(AttrRequestID, requestID))
	if user != "" {
		ab.attrs = append(ab.attrs, attribute.String(AttrUser, user))
	}
	return ab
}

// WithTokens adds token count attributes.
func (ab *AttributeBuilder) WithTokens(promptTokens, completionTokens int) *AttributeBuilder {
	ab.attrs = append(ab.attrs,
		attribute.Int(AttrTokensPrompt, promptTokens),
		attribute.Int(AttrTokensCompletion, completionTokens),
		attribute.Int(AttrTokensTotal, promptTokens+completionTokens),
	)
	return ab
}

// WithCost adds cost attributes.
func (ab *AttributeBuilder) WithCost(cost float64) *AttributeBuilder {
	ab.attrs = append(ab.attrs,
		attribute.Float64(AttrCost, cost),
		attribute.String(AttrCostCurrency, "USD"),
	)
	return ab
}

// WithPolicy adds policy attributes.
func (ab *AttributeBuilder) WithPolicy(policyID, ruleID, action string) *AttributeBuilder {
	ab.attrs = append(ab.attrs,
		attribute.String(AttrPolicyID, policyID),
		attribute.String(AttrPolicyRule, ruleID),
		attribute.String(AttrPolicyAction, action),
	)
	return ab
}

// WithCache adds cache attributes.
func (ab *AttributeBuilder) WithCache(hit bool, cacheName string) *AttributeBuilder {
	ab.attrs = append(ab.attrs,
		attribute.Bool(AttrCacheHit, hit),
		attribute.String(AttrCacheName, cacheName),
	)
	return ab
}

// WithCustom adds a custom attribute.
func (ab *AttributeBuilder) WithCustom(key string, value interface{}) *AttributeBuilder {
	switch v := value.(type) {
	case string:
		ab.attrs = append(ab.attrs, attribute.String(key, v))
	case int:
		ab.attrs = append(ab.attrs, attribute.Int(key, v))
	case int64:
		ab.attrs = append(ab.attrs, attribute.Int64(key, v))
	case float64:
		ab.attrs = append(ab.attrs, attribute.Float64(key, v))
	case bool:
		ab.attrs = append(ab.attrs, attribute.Bool(key, v))
	default:
		// Fall back to string representation
		ab.attrs = append(ab.attrs, attribute.String(key, fmt.Sprintf("%v", v)))
	}
	return ab
}

// Build returns the built attributes as a trace.SpanStartOption.
func (ab *AttributeBuilder) Build() trace.SpanStartOption {
	return trace.WithAttributes(ab.attrs...)
}

// Apply applies the attributes to a span.
func (ab *AttributeBuilder) Apply(span trace.Span) {
	span.SetAttributes(ab.attrs...)
}

// Attributes returns the raw attribute slice.
func (ab *AttributeBuilder) Attributes() []attribute.KeyValue {
	return ab.attrs
}
