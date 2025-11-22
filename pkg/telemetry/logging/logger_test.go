package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid JSON config",
			config: Config{
				Level:      "info",
				Format:     "json",
				RedactPII:  true,
				BufferSize: 100,
			},
			wantErr: false,
		},
		{
			name: "valid text config",
			config: Config{
				Level:      "debug",
				Format:     "text",
				RedactPII:  false,
				BufferSize: 100,
			},
			wantErr: false,
		},
		{
			name: "valid console config",
			config: Config{
				Level:      "warn",
				Format:     "console",
				RedactPII:  true,
				BufferSize: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: Config{
				Level:      "invalid",
				Format:     "json",
				BufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: Config{
				Level:      "info",
				Format:     "invalid",
				BufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "default buffer size",
			config: Config{
				Level:      "info",
				Format:     "json",
				BufferSize: 0, // Should use default
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			tt.config.Writer = buf

			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if logger != nil {
				defer logger.Shutdown()
			}
		})
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logMethod func(*Logger, string)
		wantLog   bool
	}{
		{
			name:      "debug level logs debug",
			logLevel:  "debug",
			logMethod: func(l *Logger, msg string) { l.Debug(msg) },
			wantLog:   true,
		},
		{
			name:      "debug level logs info",
			logLevel:  "debug",
			logMethod: func(l *Logger, msg string) { l.Info(msg) },
			wantLog:   true,
		},
		{
			name:      "info level filters debug",
			logLevel:  "info",
			logMethod: func(l *Logger, msg string) { l.Debug(msg) },
			wantLog:   false,
		},
		{
			name:      "info level logs info",
			logLevel:  "info",
			logMethod: func(l *Logger, msg string) { l.Info(msg) },
			wantLog:   true,
		},
		{
			name:      "warn level filters info",
			logLevel:  "warn",
			logMethod: func(l *Logger, msg string) { l.Info(msg) },
			wantLog:   false,
		},
		{
			name:      "warn level logs warn",
			logLevel:  "warn",
			logMethod: func(l *Logger, msg string) { l.Warn(msg) },
			wantLog:   true,
		},
		{
			name:      "error level filters warn",
			logLevel:  "error",
			logMethod: func(l *Logger, msg string) { l.Warn(msg) },
			wantLog:   false,
		},
		{
			name:      "error level logs error",
			logLevel:  "error",
			logMethod: func(l *Logger, msg string) { l.Error(msg) },
			wantLog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger, err := New(Config{
				Level:      tt.logLevel,
				Format:     "json",
				RedactPII:  false,
				BufferSize: 100,
				Writer:     buf,
			})
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}
			defer logger.Shutdown()

			testMsg := "test message"
			tt.logMethod(logger, testMsg)

			// Give async writer time to flush
			time.Sleep(10 * time.Millisecond)

			output := buf.String()
			hasLog := strings.Contains(output, testMsg)

			if hasLog != tt.wantLog {
				t.Errorf("Log filtering failed: got log=%v, want log=%v, output=%s",
					hasLog, tt.wantLog, output)
			}
		})
	}
}

func TestLogger_StructuredFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	logger.Info("test message",
		"string_field", "value",
		"int_field", 42,
		"float_field", 3.14,
		"bool_field", true,
	)

	time.Sleep(10 * time.Millisecond)
	output := buf.String()

	// Check that all fields are present in JSON output
	expectedFields := []string{
		"test message",
		"string_field",
		"value",
		"int_field",
		"42",
		"float_field",
		"3.14",
		"bool_field",
		"true",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected field %q not found in output: %s", field, output)
		}
	}
}

func TestLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	// Create logger with additional fields
	childLogger := logger.With("request_id", "req-123", "user", "testuser")
	childLogger.Info("test message")

	time.Sleep(10 * time.Millisecond)
	output := buf.String()

	// Check that child logger fields are present
	expectedFields := []string{"request_id", "req-123", "user", "testuser", "test message"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected field %q not found in output: %s", field, output)
		}
	}
}

func TestLogger_WithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	// Create context with fields
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-456")
	ctx = WithUser(ctx, "user@example.com")
	ctx = WithProvider(ctx, "openai")

	// Create logger from context
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("test message")

	time.Sleep(10 * time.Millisecond)
	output := buf.String()

	// Check that context fields are present
	expectedFields := []string{"request_id", "req-456", "user", "user@example.com", "provider", "openai"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected field %q not found in output: %s", field, output)
		}
	}
}

func TestLogger_PIIRedaction(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  true,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	// Log message with PII
	logger.Info("User login",
		"email", "user@example.com",
		"api_key", "sk-abc123xyz789",
		"ssn", "123-45-6789",
	)

	time.Sleep(10 * time.Millisecond)
	output := buf.String()

	// Original PII should NOT be present
	piiValues := []string{
		"user@example.com",
		"sk-abc123xyz789",
		"123-45-6789",
	}

	for _, pii := range piiValues {
		if strings.Contains(output, pii) {
			t.Errorf("PII value %q was not redacted in output: %s", pii, output)
		}
	}
}

func TestLogger_ContextMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "debug",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	ctx := WithRequestID(context.Background(), "req-789")

	tests := []struct {
		name   string
		method func()
		level  string
	}{
		{
			name:   "DebugContext",
			method: func() { logger.DebugContext(ctx, "debug message") },
			level:  "DEBUG",
		},
		{
			name:   "InfoContext",
			method: func() { logger.InfoContext(ctx, "info message") },
			level:  "INFO",
		},
		{
			name:   "WarnContext",
			method: func() { logger.WarnContext(ctx, "warn message") },
			level:  "WARN",
		},
		{
			name:   "ErrorContext",
			method: func() { logger.ErrorContext(ctx, "error message") },
			level:  "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.method()
			time.Sleep(10 * time.Millisecond)

			output := buf.String()
			if !strings.Contains(output, "req-789") {
				t.Errorf("Context request_id not found in %s output: %s", tt.name, output)
			}
		})
	}
}

func TestLogger_Formats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"JSON format", "json"},
		{"Text format", "text"},
		{"Console format", "console"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger, err := New(Config{
				Level:      "info",
				Format:     tt.format,
				RedactPII:  false,
				BufferSize: 100,
				Writer:     buf,
			})
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}
			defer logger.Shutdown()

			logger.Info("test message", "key", "value")
			time.Sleep(10 * time.Millisecond)

			output := buf.String()
			if output == "" {
				t.Errorf("No output for format %s", tt.format)
			}

			// All formats should include the message
			if !strings.Contains(output, "test message") {
				t.Errorf("Message not found in %s output: %s", tt.format, output)
			}
		})
	}
}

func TestLogger_AddSource(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		AddSource:  true,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Shutdown()

	logger.Info("test message")
	time.Sleep(10 * time.Millisecond)

	output := buf.String()

	// Should include source field with file and line information
	if !strings.Contains(output, "source") {
		t.Errorf("Source field not found in output: %s", output)
	}
	if !strings.Contains(output, "logger.go") {
		t.Errorf("Source file not found in output: %s", output)
	}
}

func TestLogger_Shutdown(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(Config{
		Level:      "info",
		Format:     "json",
		RedactPII:  false,
		BufferSize: 100,
		Writer:     buf,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Info("message before shutdown")

	err = logger.Shutdown()
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	output := buf.String()

	if !strings.Contains(output, "message before shutdown") {
		t.Errorf("Message logged before shutdown not found: %s", output)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"debug", false},
		{"DEBUG", false},
		{"info", false},
		{"INFO", false},
		{"", false}, // Default to info
		{"warn", false},
		{"WARN", false},
		{"warning", false},
		{"error", false},
		{"ERROR", false},
		{"invalid", true},
		{"trace", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"json", false},
		{"JSON", false},
		{"", false}, // Default to JSON
		{"text", false},
		{"TEXT", false},
		{"console", false},
		{"CONSOLE", false},
		{"invalid", true},
		{"xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
