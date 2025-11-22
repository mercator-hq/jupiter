package logging

import (
	"context"
	"testing"
)

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	// Test RequestID
	ctx = WithRequestID(ctx, "req-123")
	if got := GetRequestID(ctx); got != "req-123" {
		t.Errorf("GetRequestID() = %q, want %q", got, "req-123")
	}

	// Test APIKey
	ctx = WithAPIKey(ctx, "sk-abc123")
	if got := GetAPIKey(ctx); got != "sk-abc123" {
		t.Errorf("GetAPIKey() = %q, want %q", got, "sk-abc123")
	}

	// Test User
	ctx = WithUser(ctx, "user@example.com")
	if got := GetUser(ctx); got != "user@example.com" {
		t.Errorf("GetUser() = %q, want %q", got, "user@example.com")
	}

	// Test Team
	ctx = WithTeam(ctx, "team-alpha")
	if got := GetTeam(ctx); got != "team-alpha" {
		t.Errorf("GetTeam() = %q, want %q", got, "team-alpha")
	}

	// Test Provider
	ctx = WithProvider(ctx, "openai")
	if got := GetProvider(ctx); got != "openai" {
		t.Errorf("GetProvider() = %q, want %q", got, "openai")
	}

	// Test Model
	ctx = WithModel(ctx, "gpt-4")
	if got := GetModel(ctx); got != "gpt-4" {
		t.Errorf("GetModel() = %q, want %q", got, "gpt-4")
	}

	// Test Session
	ctx = WithSession(ctx, "session-xyz")
	if got := GetSession(ctx); got != "session-xyz" {
		t.Errorf("GetSession() = %q, want %q", got, "session-xyz")
	}

	// Test TraceID
	ctx = WithTraceID(ctx, "trace-abc")
	if got := GetTraceID(ctx); got != "trace-abc" {
		t.Errorf("GetTraceID() = %q, want %q", got, "trace-abc")
	}

	// Test SpanID
	ctx = WithSpanID(ctx, "span-def")
	if got := GetSpanID(ctx); got != "span-def" {
		t.Errorf("GetSpanID() = %q, want %q", got, "span-def")
	}
}

func TestContextKeys_Empty(t *testing.T) {
	ctx := context.Background()

	// Test that getters return empty strings for missing values
	tests := []struct {
		name string
		get  func(context.Context) string
	}{
		{"RequestID", GetRequestID},
		{"APIKey", GetAPIKey},
		{"User", GetUser},
		{"Team", GetTeam},
		{"Provider", GetProvider},
		{"Model", GetModel},
		{"Session", GetSession},
		{"TraceID", GetTraceID},
		{"SpanID", GetSpanID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.get(ctx); got != "" {
				t.Errorf("Get%s() = %q, want empty string", tt.name, got)
			}
		})
	}
}

func TestExtractContextFields(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(context.Context) context.Context
		wantFields map[string]string
	}{
		{
			name: "empty context",
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			wantFields: map[string]string{},
		},
		{
			name: "request ID only",
			setupCtx: func(ctx context.Context) context.Context {
				return WithRequestID(ctx, "req-123")
			},
			wantFields: map[string]string{
				"request_id": "req-123",
			},
		},
		{
			name: "multiple fields",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = WithRequestID(ctx, "req-456")
				ctx = WithUser(ctx, "user@example.com")
				ctx = WithProvider(ctx, "openai")
				ctx = WithModel(ctx, "gpt-4")
				return ctx
			},
			wantFields: map[string]string{
				"request_id": "req-456",
				"user":       "user@example.com",
				"provider":   "openai",
				"model":      "gpt-4",
			},
		},
		{
			name: "all fields",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = WithRequestID(ctx, "req-789")
				ctx = WithAPIKey(ctx, "sk-abc")
				ctx = WithUser(ctx, "user1")
				ctx = WithTeam(ctx, "team1")
				ctx = WithProvider(ctx, "anthropic")
				ctx = WithModel(ctx, "claude-3")
				ctx = WithSession(ctx, "sess-1")
				ctx = WithTraceID(ctx, "trace-1")
				ctx = WithSpanID(ctx, "span-1")
				return ctx
			},
			wantFields: map[string]string{
				"request_id": "req-789",
				"api_key":    "sk-abc",
				"user":       "user1",
				"team":       "team1",
				"provider":   "anthropic",
				"model":      "claude-3",
				"session":    "sess-1",
				"trace_id":   "trace-1",
				"span_id":    "span-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx(context.Background())
			fields := extractContextFields(ctx)

			// Convert []any to map for easier checking
			fieldsMap := make(map[string]string)
			for i := 0; i < len(fields); i += 2 {
				key := fields[i].(string)
				value := fields[i+1].(string)
				fieldsMap[key] = value
			}

			// Check expected fields are present
			for key, expectedValue := range tt.wantFields {
				if gotValue, ok := fieldsMap[key]; !ok {
					t.Errorf("Expected field %q not found", key)
				} else if gotValue != expectedValue {
					t.Errorf("Field %q = %q, want %q", key, gotValue, expectedValue)
				}
			}

			// Check no extra fields
			if len(fieldsMap) != len(tt.wantFields) {
				t.Errorf("Got %d fields, want %d. Fields: %v",
					len(fieldsMap), len(tt.wantFields), fieldsMap)
			}
		})
	}
}

func TestContextLogger(t *testing.T) {
	// This test verifies that ContextLogger properly wraps the logger
	// Actual logging is tested in logger_test.go

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-cl-1")
	ctx = WithUser(ctx, "testuser")

	// Create a basic logger (using nil config to test error handling is in logger_test)
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	// Create context logger
	ctxLogger := NewContextLogger(logger, ctx)
	if ctxLogger == nil {
		t.Fatal("NewContextLogger returned nil")
	}

	// Test that methods don't panic
	ctxLogger.Debug("debug message")
	ctxLogger.Info("info message")
	ctxLogger.Warn("warn message")
	ctxLogger.Error("error message")

	// Test With
	childLogger := ctxLogger.With("extra", "value")
	if childLogger == nil {
		t.Fatal("ContextLogger.With returned nil")
	}

	childLogger.Info("child message")
}

func TestContextLogger_With(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-with-1")

	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	ctxLogger := NewContextLogger(logger, ctx)

	// Create child logger with additional fields
	childLogger := ctxLogger.With("key1", "value1", "key2", 42)
	if childLogger == nil {
		t.Fatal("ContextLogger.With returned nil")
	}

	// Verify it doesn't panic
	childLogger.Info("test message")
}

func TestContextChaining(t *testing.T) {
	// Test that context values can be added incrementally
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-chain-1")
	ctx = WithUser(ctx, "user1")
	ctx = WithProvider(ctx, "provider1")

	// Verify all values are present
	if got := GetRequestID(ctx); got != "req-chain-1" {
		t.Errorf("After chaining, GetRequestID() = %q, want %q", got, "req-chain-1")
	}
	if got := GetUser(ctx); got != "user1" {
		t.Errorf("After chaining, GetUser() = %q, want %q", got, "user1")
	}
	if got := GetProvider(ctx); got != "provider1" {
		t.Errorf("After chaining, GetProvider() = %q, want %q", got, "provider1")
	}

	// Add more values
	ctx = WithModel(ctx, "model1")
	ctx = WithTeam(ctx, "team1")

	if got := GetModel(ctx); got != "model1" {
		t.Errorf("After more chaining, GetModel() = %q, want %q", got, "model1")
	}
	if got := GetTeam(ctx); got != "team1" {
		t.Errorf("After more chaining, GetTeam() = %q, want %q", got, "team1")
	}

	// Verify original values still present
	if got := GetRequestID(ctx); got != "req-chain-1" {
		t.Errorf("Original value changed: GetRequestID() = %q, want %q", got, "req-chain-1")
	}
}

func TestContextOverwrite(t *testing.T) {
	// Test that context values can be overwritten
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-old")

	if got := GetRequestID(ctx); got != "req-old" {
		t.Errorf("Initial GetRequestID() = %q, want %q", got, "req-old")
	}

	// Overwrite with new value
	ctx = WithRequestID(ctx, "req-new")

	if got := GetRequestID(ctx); got != "req-new" {
		t.Errorf("After overwrite, GetRequestID() = %q, want %q", got, "req-new")
	}
}

func BenchmarkExtractContextFields(b *testing.B) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-bench")
	ctx = WithUser(ctx, "user@example.com")
	ctx = WithProvider(ctx, "openai")
	ctx = WithModel(ctx, "gpt-4")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractContextFields(ctx)
	}
}

func BenchmarkWithRequestID(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithRequestID(ctx, "req-123")
	}
}

func BenchmarkGetRequestID(b *testing.B) {
	ctx := WithRequestID(context.Background(), "req-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRequestID(ctx)
	}
}
