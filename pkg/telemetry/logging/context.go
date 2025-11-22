package logging

import (
	"context"
)

// Context keys for common log fields.
type contextKey string

const (
	// RequestIDKey is the context key for request IDs.
	RequestIDKey contextKey = "request_id"

	// APIKeyKey is the context key for API keys.
	APIKeyKey contextKey = "api_key"

	// UserKey is the context key for user identifiers.
	UserKey contextKey = "user"

	// TeamKey is the context key for team identifiers.
	TeamKey contextKey = "team"

	// ProviderKey is the context key for provider names.
	ProviderKey contextKey = "provider"

	// ModelKey is the context key for model names.
	ModelKey contextKey = "model"

	// SessionKey is the context key for session identifiers.
	SessionKey contextKey = "session"

	// TraceIDKey is the context key for trace IDs.
	TraceIDKey contextKey = "trace_id"

	// SpanIDKey is the context key for span IDs.
	SpanIDKey contextKey = "span_id"
)

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithAPIKey adds an API key to the context.
func WithAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, APIKeyKey, apiKey)
}

// GetAPIKey retrieves the API key from the context.
func GetAPIKey(ctx context.Context) string {
	if apiKey, ok := ctx.Value(APIKeyKey).(string); ok {
		return apiKey
	}
	return ""
}

// WithUser adds a user identifier to the context.
func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

// GetUser retrieves the user identifier from the context.
func GetUser(ctx context.Context) string {
	if user, ok := ctx.Value(UserKey).(string); ok {
		return user
	}
	return ""
}

// WithTeam adds a team identifier to the context.
func WithTeam(ctx context.Context, team string) context.Context {
	return context.WithValue(ctx, TeamKey, team)
}

// GetTeam retrieves the team identifier from the context.
func GetTeam(ctx context.Context) string {
	if team, ok := ctx.Value(TeamKey).(string); ok {
		return team
	}
	return ""
}

// WithProvider adds a provider name to the context.
func WithProvider(ctx context.Context, provider string) context.Context {
	return context.WithValue(ctx, ProviderKey, provider)
}

// GetProvider retrieves the provider name from the context.
func GetProvider(ctx context.Context) string {
	if provider, ok := ctx.Value(ProviderKey).(string); ok {
		return provider
	}
	return ""
}

// WithModel adds a model name to the context.
func WithModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, ModelKey, model)
}

// GetModel retrieves the model name from the context.
func GetModel(ctx context.Context) string {
	if model, ok := ctx.Value(ModelKey).(string); ok {
		return model
	}
	return ""
}

// WithSession adds a session identifier to the context.
func WithSession(ctx context.Context, session string) context.Context {
	return context.WithValue(ctx, SessionKey, session)
}

// GetSession retrieves the session identifier from the context.
func GetSession(ctx context.Context) string {
	if session, ok := ctx.Value(SessionKey).(string); ok {
		return session
	}
	return ""
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithSpanID adds a span ID to the context.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetSpanID retrieves the span ID from the context.
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// extractContextFields extracts common fields from context for logging.
// Returns a slice of key-value pairs suitable for logger.With().
func extractContextFields(ctx context.Context) []any {
	var fields []any

	// Extract request ID
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, "request_id", requestID)
	}

	// Extract API key (will be redacted by logger if PII redaction is enabled)
	if apiKey := GetAPIKey(ctx); apiKey != "" {
		fields = append(fields, "api_key", apiKey)
	}

	// Extract user
	if user := GetUser(ctx); user != "" {
		fields = append(fields, "user", user)
	}

	// Extract team
	if team := GetTeam(ctx); team != "" {
		fields = append(fields, "team", team)
	}

	// Extract provider
	if provider := GetProvider(ctx); provider != "" {
		fields = append(fields, "provider", provider)
	}

	// Extract model
	if model := GetModel(ctx); model != "" {
		fields = append(fields, "model", model)
	}

	// Extract session
	if session := GetSession(ctx); session != "" {
		fields = append(fields, "session", session)
	}

	// Extract trace ID
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}

	// Extract span ID
	if spanID := GetSpanID(ctx); spanID != "" {
		fields = append(fields, "span_id", spanID)
	}

	return fields
}

// ContextLogger is a logger that automatically includes context fields.
type ContextLogger struct {
	logger *Logger
	ctx    context.Context
}

// NewContextLogger creates a logger that automatically includes context fields.
func NewContextLogger(logger *Logger, ctx context.Context) *ContextLogger {
	return &ContextLogger{
		logger: logger.WithContext(ctx),
		ctx:    ctx,
	}
}

// Debug logs a debug message with context fields.
func (cl *ContextLogger) Debug(msg string, args ...any) {
	cl.logger.DebugContext(cl.ctx, msg, args...)
}

// Info logs an info message with context fields.
func (cl *ContextLogger) Info(msg string, args ...any) {
	cl.logger.InfoContext(cl.ctx, msg, args...)
}

// Warn logs a warning message with context fields.
func (cl *ContextLogger) Warn(msg string, args ...any) {
	cl.logger.WarnContext(cl.ctx, msg, args...)
}

// Error logs an error message with context fields.
func (cl *ContextLogger) Error(msg string, args ...any) {
	cl.logger.ErrorContext(cl.ctx, msg, args...)
}

// With creates a new context logger with additional fields.
func (cl *ContextLogger) With(args ...any) *ContextLogger {
	return &ContextLogger{
		logger: cl.logger.With(args...),
		ctx:    cl.ctx,
	}
}
