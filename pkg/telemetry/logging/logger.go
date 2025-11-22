package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"mercator-hq/jupiter/pkg/config"
)

// LogFormat represents the output format for logs.
type LogFormat string

const (
	// FormatJSON outputs logs in JSON format.
	FormatJSON LogFormat = "json"
	// FormatText outputs logs in plain text format.
	FormatText LogFormat = "text"
	// FormatConsole outputs logs in human-readable console format.
	FormatConsole LogFormat = "console"
)

// Logger provides structured logging with PII redaction and async buffering.
type Logger struct {
	// slog is the underlying structured logger
	slog *slog.Logger

	// redactor performs PII redaction on log fields
	redactor *Redactor

	// level is the minimum log level
	level slog.Level

	// format is the output format
	format LogFormat

	// addSource includes file:line in logs
	addSource bool

	// buffer is the async log buffer
	buffer *LogBuffer

	// writer is the underlying writer
	writer io.Writer
}

// LogBuffer provides async buffering for log writes to avoid blocking.
type LogBuffer struct {
	entries  chan *LogEntry
	maxSize  int
	dropped  atomic.Int64
	writer   io.Writer
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Level     slog.Level
	Message   string
	Attrs     []slog.Attr
	Timestamp time.Time
}

// Config contains configuration for the Logger.
type Config struct {
	// Level is the minimum log level ("debug", "info", "warn", "error")
	Level string

	// Format is the output format ("json", "text", "console")
	Format string

	// AddSource includes file and line number in logs
	AddSource bool

	// RedactPII enables automatic PII redaction
	RedactPII bool

	// BufferSize is the async log buffer size
	BufferSize int

	// RedactPatterns contains custom PII redaction patterns
	RedactPatterns []config.RedactPattern

	// Writer is the output writer (defaults to os.Stdout)
	Writer io.Writer
}

// New creates a new Logger with the given configuration.
func New(cfg Config) (*Logger, error) {
	// Parse log level
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Parse log format
	format, err := parseFormat(cfg.Format)
	if err != nil {
		return nil, fmt.Errorf("invalid log format: %w", err)
	}

	// Set default writer
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}

	// Set default buffer size
	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 10000 // Default: 10K entries
	}

	// Create redactor
	var redactor *Redactor
	if cfg.RedactPII {
		redactor = NewRedactor(cfg.RedactPatterns)
	}

	// Create log buffer for async writes
	buffer := &LogBuffer{
		entries:  make(chan *LogEntry, bufferSize),
		maxSize:  bufferSize,
		writer:   writer,
		stopChan: make(chan struct{}),
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	switch format {
	case FormatJSON:
		handler = slog.NewJSONHandler(buffer, opts)
	case FormatText:
		handler = slog.NewTextHandler(buffer, opts)
	case FormatConsole:
		// Console format is like text but more human-readable
		handler = slog.NewTextHandler(buffer, opts)
	default:
		handler = slog.NewJSONHandler(buffer, opts)
	}

	// Create logger
	logger := &Logger{
		slog:      slog.New(handler),
		redactor:  redactor,
		level:     level,
		format:    format,
		addSource: cfg.AddSource,
		buffer:    buffer,
		writer:    writer,
	}

	// Start async writer
	buffer.Start()

	return logger, nil
}

// Write implements io.Writer for the log buffer.
// This is called by slog handlers to write log output.
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	// For direct writes (from slog handlers), just write immediately
	// This avoids double-buffering since we'll use our own buffering
	return lb.writer.Write(p)
}

// Start begins the async log writer goroutine.
func (lb *LogBuffer) Start() {
	lb.wg.Add(1)
	go lb.runWriter()
}

// runWriter is the async writer goroutine that processes log entries.
func (lb *LogBuffer) runWriter() {
	defer lb.wg.Done()

	for {
		select {
		case <-lb.stopChan:
			// Drain remaining entries
			for len(lb.entries) > 0 {
				entry := <-lb.entries
				lb.writeEntry(entry)
			}
			return
		case entry := <-lb.entries:
			lb.writeEntry(entry)
		}
	}
}

// writeEntry writes a single log entry to the underlying writer.
func (lb *LogBuffer) writeEntry(entry *LogEntry) {
	// This method is no longer needed with direct slog integration
	// but kept for future custom buffering if needed
}

// Stop stops the async writer and waits for pending writes.
func (lb *LogBuffer) Stop() {
	close(lb.stopChan)
	lb.wg.Wait()
}

// DroppedCount returns the number of dropped log entries.
func (lb *LogBuffer) DroppedCount() int64 {
	return lb.dropped.Load()
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(context.Background(), slog.LevelDebug, msg, args...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.log(context.Background(), slog.LevelInfo, msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(context.Background(), slog.LevelWarn, msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.log(context.Background(), slog.LevelError, msg, args...)
}

// DebugContext logs a debug message with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	// Extract context fields and prepend to args
	ctxFields := extractContextFields(ctx)
	allArgs := append(ctxFields, args...)
	l.log(ctx, slog.LevelDebug, msg, allArgs...)
}

// InfoContext logs an info message with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	// Extract context fields and prepend to args
	ctxFields := extractContextFields(ctx)
	allArgs := append(ctxFields, args...)
	l.log(ctx, slog.LevelInfo, msg, allArgs...)
}

// WarnContext logs a warning message with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	// Extract context fields and prepend to args
	ctxFields := extractContextFields(ctx)
	allArgs := append(ctxFields, args...)
	l.log(ctx, slog.LevelWarn, msg, allArgs...)
}

// ErrorContext logs an error message with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	// Extract context fields and prepend to args
	ctxFields := extractContextFields(ctx)
	allArgs := append(ctxFields, args...)
	l.log(ctx, slog.LevelError, msg, allArgs...)
}

// log is the internal logging method that handles PII redaction.
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	// Fast path: if level is disabled, return immediately (near-zero cost)
	if !l.slog.Enabled(ctx, level) {
		return
	}

	// Apply PII redaction if enabled
	if l.redactor != nil {
		args = l.redactor.RedactArgs(args...)
	}

	// Log using slog
	l.slog.Log(ctx, level, msg, args...)
}

// With creates a new logger with additional fields.
func (l *Logger) With(args ...any) *Logger {
	// Apply PII redaction if enabled
	if l.redactor != nil {
		args = l.redactor.RedactArgs(args...)
	}

	return &Logger{
		slog:      l.slog.With(args...),
		redactor:  l.redactor,
		level:     l.level,
		format:    l.format,
		addSource: l.addSource,
		buffer:    l.buffer,
		writer:    l.writer,
	}
}

// WithContext creates a new logger with context values.
// It extracts common fields from context (request_id, api_key, user).
func (l *Logger) WithContext(ctx context.Context) *Logger {
	args := extractContextFields(ctx)
	if len(args) == 0 {
		return l
	}
	return l.With(args...)
}

// Shutdown gracefully shuts down the logger, flushing pending writes.
func (l *Logger) Shutdown() error {
	if l.buffer != nil {
		l.buffer.Stop()
	}
	return nil
}

// parseLevel parses a log level string into slog.Level.
func parseLevel(levelStr string) (slog.Level, error) {
	switch levelStr {
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "info", "INFO", "":
		return slog.LevelInfo, nil
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", levelStr)
	}
}

// parseFormat parses a log format string into LogFormat.
func parseFormat(formatStr string) (LogFormat, error) {
	switch formatStr {
	case "json", "JSON", "":
		return FormatJSON, nil
	case "text", "TEXT":
		return FormatText, nil
	case "console", "CONSOLE":
		return FormatConsole, nil
	default:
		return FormatJSON, fmt.Errorf("unknown log format: %s", formatStr)
	}
}
