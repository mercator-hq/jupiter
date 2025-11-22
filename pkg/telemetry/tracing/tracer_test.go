package tracing

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/config"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TestNew tests the creation of a new tracer
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.TracingConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "disabled tracing",
			config: &config.TracingConfig{
				Enabled:     false,
				ServiceName: "test-service",
			},
			wantErr: false,
		},
		{
			name: "enabled with always sampler",
			config: &config.TracingConfig{
				Enabled:     true,
				Sampler:     "always",
				Exporter:    "otlp",
				Endpoint:    "localhost:4317",
				ServiceName: "test-service",
				OTLP: config.OTLPConfig{
					Insecure: true,
					Timeout:  10 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "enabled with never sampler",
			config: &config.TracingConfig{
				Enabled:     true,
				Sampler:     "never",
				Exporter:    "otlp",
				Endpoint:    "localhost:4317",
				ServiceName: "test-service",
				OTLP: config.OTLPConfig{
					Insecure: true,
					Timeout:  10 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "enabled with ratio sampler",
			config: &config.TracingConfig{
				Enabled:     true,
				Sampler:     "ratio",
				SampleRatio: 0.5,
				Exporter:    "otlp",
				Endpoint:    "localhost:4317",
				ServiceName: "test-service",
				OTLP: config.OTLPConfig{
					Insecure: true,
					Timeout:  10 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid sampler",
			config: &config.TracingConfig{
				Enabled:     true,
				Sampler:     "invalid",
				Exporter:    "otlp",
				Endpoint:    "localhost:4317",
				ServiceName: "test-service",
			},
			wantErr: true,
		},
		{
			name: "jaeger exporter (not implemented)",
			config: &config.TracingConfig{
				Enabled:     true,
				Sampler:     "always",
				Exporter:    "jaeger",
				ServiceName: "test-service",
			},
			wantErr: true,
		},
		{
			name: "zipkin exporter (not implemented)",
			config: &config.TracingConfig{
				Enabled:     true,
				Sampler:     "always",
				Exporter:    "zipkin",
				Endpoint:    "http://localhost:9411",
				ServiceName: "test-service",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify tracer is not nil
				if tracer == nil {
					t.Error("New() returned nil tracer without error")
					return
				}

				// Verify enabled state
				if tracer.Enabled() != tt.config.Enabled {
					t.Errorf("tracer.Enabled() = %v, want %v", tracer.Enabled(), tt.config.Enabled)
				}

				// Clean up
				if err := tracer.Shutdown(context.Background()); err != nil {
					t.Errorf("Shutdown() error = %v", err)
				}
			}
		})
	}
}

// TestTracer_Start tests span creation
func TestTracer_Start(t *testing.T) {
	// Create disabled tracer (returns noop spans)
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()

	// Test basic span creation
	ctx, span := tracer.Start(ctx, "test-operation")
	if span == nil {
		t.Error("Start() returned nil span")
	}
	span.End()

	// Test span with attributes
	ctx, span = tracer.Start(ctx, "test-operation-with-attrs",
		trace.WithAttributes(
			attribute.String("test.key", "test.value"),
		),
	)
	if span == nil {
		t.Error("Start() returned nil span")
	}
	span.End()

	// Test nested spans
	ctx, parentSpan := tracer.Start(ctx, "parent-operation")
	ctx, childSpan := tracer.Start(ctx, "child-operation")
	childSpan.End()
	parentSpan.End()
}

// TestTracer_Shutdown tests graceful shutdown
func TestTracer_Shutdown(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		wantErr bool
	}{
		{
			name:    "shutdown disabled tracer",
			enabled: false,
			wantErr: false,
		},
		{
			name:    "shutdown enabled tracer",
			enabled: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.TracingConfig{
				Enabled:     tt.enabled,
				ServiceName: "test-service",
			}

			if tt.enabled {
				cfg.Sampler = "always"
				cfg.Exporter = "otlp"
				cfg.Endpoint = "localhost:4317"
				cfg.OTLP = config.OTLPConfig{
					Insecure: true,
					Timeout:  10 * time.Second,
				}
			}

			tracer, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create tracer: %v", err)
			}

			// Create a span before shutdown
			ctx, span := tracer.Start(context.Background(), "test-operation")
			span.End()

			// Shutdown
			if err := tracer.Shutdown(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSpanFromContext tests retrieving span from context
func TestSpanFromContext(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()

	// Test with no span in context
	span := SpanFromContext(ctx)
	if span == nil {
		t.Error("SpanFromContext() returned nil")
	}

	// Test with span in context
	ctx, createdSpan := tracer.Start(ctx, "test-operation")
	retrievedSpan := SpanFromContext(ctx)
	if retrievedSpan == nil {
		t.Error("SpanFromContext() returned nil")
	}
	createdSpan.End()
}

// TestContextWithSpan tests adding span to context
func TestContextWithSpan(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-operation")
	defer span.End()

	// Add span to new context
	newCtx := ContextWithSpan(context.Background(), span)

	// Verify span is in new context
	retrievedSpan := SpanFromContext(newCtx)
	if retrievedSpan == nil {
		t.Error("SpanFromContext() returned nil after ContextWithSpan()")
	}
}

// TestSpanContext tests retrieving span context
func TestSpanContext(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()

	// Test with no span
	sc := SpanContext(ctx)
	if sc.IsValid() {
		t.Error("SpanContext() returned valid context with no span")
	}

	// Test with span
	ctx, span := tracer.Start(ctx, "test-operation")
	defer span.End()

	sc = SpanContext(ctx)
	// For noop tracer, span context may or may not be valid
	// Just verify it doesn't panic
}

// TestTraceID tests retrieving trace ID
func TestTraceID(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()

	// Test with no span
	traceID := TraceID(ctx)
	if traceID != "" {
		t.Errorf("TraceID() = %q, want empty string", traceID)
	}

	// Test with span
	ctx, span := tracer.Start(ctx, "test-operation")
	defer span.End()

	traceID = TraceID(ctx)
	// For noop tracer, trace ID will be empty
}

// TestSpanID tests retrieving span ID
func TestSpanID(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()

	// Test with no span
	spanID := SpanID(ctx)
	if spanID != "" {
		t.Errorf("SpanID() = %q, want empty string", spanID)
	}

	// Test with span
	ctx, span := tracer.Start(ctx, "test-operation")
	defer span.End()

	spanID = SpanID(ctx)
	// For noop tracer, span ID will be empty
}

// TestIsSampled tests checking if trace is sampled
func TestIsSampled(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()

	// Test with no span
	if IsSampled(ctx) {
		t.Error("IsSampled() = true, want false with no span")
	}

	// Test with span
	ctx, span := tracer.Start(ctx, "test-operation")
	defer span.End()

	// For noop tracer, sampling will be false
	// Just verify it doesn't panic
	_ = IsSampled(ctx)
}

// TestSetError tests setting error on span
func TestSetError(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Test with nil error
	SetError(span, nil)

	// Test with actual error
	testErr := context.DeadlineExceeded
	SetError(span, testErr)

	// Verify it doesn't panic
}

// TestSetStatus tests setting span status
func TestSetStatus(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Test OK status
	SetStatus(span, nil)

	// Test Error status
	testErr := context.DeadlineExceeded
	SetStatus(span, testErr)

	// Verify it doesn't panic
}

// TestTracer_SpanAttributes tests setting attributes on spans
func TestTracer_SpanAttributes(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Set various attribute types
	span.SetAttributes(
		attribute.String("string.key", "value"),
		attribute.Int("int.key", 42),
		attribute.Int64("int64.key", 1234567890),
		attribute.Float64("float64.key", 3.14),
		attribute.Bool("bool.key", true),
	)

	// Verify it doesn't panic
}

// TestTracer_SpanEvents tests adding events to spans
func TestTracer_SpanEvents(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Add event without attributes
	span.AddEvent("test-event")

	// Add event with attributes
	span.AddEvent("test-event-with-attrs",
		trace.WithAttributes(
			attribute.String("event.key", "event.value"),
		),
	)

	// Verify it doesn't panic
}

// TestTracer_RecordError tests recording errors
func TestTracer_RecordError(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Record an error
	testErr := context.DeadlineExceeded
	span.RecordError(testErr)

	// Verify it doesn't panic
}

// TestTracer_SetStatus tests setting span status with codes
func TestTracer_SetStatus(t *testing.T) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer tracer.Shutdown(context.Background())

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// Set OK status
	span.SetStatus(codes.Ok, "success")

	// Set Error status
	span.SetStatus(codes.Error, "failed")

	// Verify it doesn't panic
}
