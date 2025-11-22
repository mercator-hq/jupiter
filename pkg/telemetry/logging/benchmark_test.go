package logging

import (
	"bytes"
	"context"
	"testing"
)

// BenchmarkLogger_Info_Enabled measures logging performance when enabled.
// Target: <10µs per log entry
func BenchmarkLogger_Info_Enabled(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key", "value", "count", i)
	}
}

// BenchmarkLogger_Debug_Disabled measures logging performance when level is disabled.
// Target: <1µs per call (should be near-zero cost)
func BenchmarkLogger_Debug_Disabled(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info", // Debug is disabled
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Debug("test message", "key", "value", "count", i)
	}
}

// BenchmarkLogger_WithRedaction measures logging with PII redaction enabled.
func BenchmarkLogger_WithRedaction(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  true,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("User logged in",
			"email", "user@example.com",
			"api_key", "sk-abc123xyz789",
		)
	}
}

// BenchmarkLogger_WithoutRedaction measures logging without PII redaction.
func BenchmarkLogger_WithoutRedaction(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("User logged in",
			"email", "user@example.com",
			"api_key", "sk-abc123xyz789",
		)
	}
}

// BenchmarkLogger_With measures creating child loggers.
func BenchmarkLogger_With(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = logger.With("request_id", "req-123", "user", "testuser")
	}
}

// BenchmarkLogger_WithContext measures creating context loggers.
func BenchmarkLogger_WithContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUser(ctx, "testuser")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = logger.WithContext(ctx)
	}
}

// BenchmarkLogger_InfoContext measures logging with context.
func BenchmarkLogger_InfoContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	ctx := WithRequestID(context.Background(), "req-123")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "test message", "key", "value")
	}
}

// BenchmarkRedactor_RedactString measures PII redaction performance.
func BenchmarkRedactor_RedactString(b *testing.B) {
	redactor := NewRedactor(nil)
	input := "User user@example.com with API key sk-abc123xyz789 logged in from 192.168.1.100"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = redactor.RedactString(input)
	}
}

// BenchmarkRedactor_RedactArgs measures argument redaction performance.
func BenchmarkRedactor_RedactArgs(b *testing.B) {
	redactor := NewRedactor(nil)
	args := []any{
		"email", "user@example.com",
		"api_key", "sk-abc123xyz789",
		"count", 42,
		"message", "test message",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = redactor.RedactArgs(args...)
	}
}

// BenchmarkLogger_JSON measures JSON format performance.
func BenchmarkLogger_JSON(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test", "k1", "v1", "k2", 42)
	}
}

// BenchmarkLogger_Text measures text format performance.
func BenchmarkLogger_Text(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "text",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test", "k1", "v1", "k2", 42)
	}
}

// BenchmarkLogger_Parallel measures concurrent logging performance.
func BenchmarkLogger_Parallel(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info("test message", "iteration", i)
			i++
		}
	})
}

// BenchmarkLogger_ManyFields measures logging with many fields.
func BenchmarkLogger_ManyFields(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("test",
			"f1", "v1", "f2", "v2", "f3", "v3", "f4", "v4",
			"f5", "v5", "f6", "v6", "f7", "v7", "f8", "v8",
			"f9", "v9", "f10", "v10",
		)
	}
}

// BenchmarkContextLogger_Info measures ContextLogger performance.
func BenchmarkContextLogger_Info(b *testing.B) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100000,
		Writer:     buf,
	})
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	ctx := WithRequestID(context.Background(), "req-bench")
	ctxLogger := NewContextLogger(logger, ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctxLogger.Info("test message", "key", "value")
	}
}

// Benchmark comparison: disabled vs enabled logging
func BenchmarkLogger_Compare(b *testing.B) {
	b.Run("Info_Enabled", func(b *testing.B) {
		buf := &bytes.Buffer{}
		logger, _ := New(Config{
			Level:      "info",
			Format:     "json",
			RedactPII:  false,
			BufferSize: 100000,
			Writer:     buf,
		})
		defer logger.Shutdown()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("test", "k", "v")
		}
	})

	b.Run("Info_Disabled", func(b *testing.B) {
		buf := &bytes.Buffer{}
		logger, _ := New(Config{
			Level:      "error", // Info is disabled
			Format:     "json",
			RedactPII:  false,
			BufferSize: 100000,
			Writer:     buf,
		})
		defer logger.Shutdown()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("test", "k", "v")
		}
	})

	b.Run("WithRedaction", func(b *testing.B) {
		buf := &bytes.Buffer{}
		logger, _ := New(Config{
			Level:      "info",
			Format:     "json",
			RedactPII:  true,
			BufferSize: 100000,
			Writer:     buf,
		})
		defer logger.Shutdown()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("test", "api_key", "sk-abc123")
		}
	})

	b.Run("WithoutRedaction", func(b *testing.B) {
		buf := &bytes.Buffer{}
		logger, _ := New(Config{
			Level:      "info",
			Format:     "json",
			RedactPII:  false,
			BufferSize: 100000,
			Writer:     buf,
		})
		defer logger.Shutdown()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("test", "api_key", "sk-abc123")
		}
	})
}
