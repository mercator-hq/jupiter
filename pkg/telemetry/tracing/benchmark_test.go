package tracing

import (
	"context"
	"net/http"
	"testing"

	"mercator-hq/jupiter/pkg/config"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BenchmarkTracer_Start_Disabled benchmarks span creation with disabled tracing
// Target: <1µs (noop overhead)
func BenchmarkTracer_Start_Disabled(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "test-operation")
		span.End()
	}
}

// BenchmarkTracer_Start_Enabled benchmarks span creation with enabled tracing
// Target: <100µs per span
func BenchmarkTracer_Start_Enabled(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     true,
		Sampler:     "always",
		SampleRatio: 1.0,
		Exporter:    "otlp",
		Endpoint:    "localhost:4317",
		ServiceName: "test-service",
		OTLP: config.OTLPConfig{
			Insecure: true,
		},
	})
	if err != nil {
		b.Skip("OTLP endpoint not available, skipping benchmark")
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "test-operation")
		span.End()
	}
}

// BenchmarkTracer_Start_WithAttributes benchmarks span creation with attributes
// Target: <100µs per span
func BenchmarkTracer_Start_WithAttributes(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := tracer.Start(ctx, "test-operation",
			trace.WithAttributes(
				attribute.String("provider", "openai"),
				attribute.String("model", "gpt-4"),
				attribute.Int("tokens", 1500),
				attribute.Float64("cost", 0.05),
			),
		)
		span.End()
	}
}

// BenchmarkTracer_NestedSpans benchmarks nested span creation
// Target: <200µs for parent + child (100µs each)
func BenchmarkTracer_NestedSpans(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, parentSpan := tracer.Start(ctx, "parent-operation")
		_, childSpan := tracer.Start(ctx, "child-operation")
		childSpan.End()
		parentSpan.End()
	}
}

// BenchmarkSetProviderAttributes benchmarks setting provider attributes
// Target: <10µs
func BenchmarkSetProviderAttributes(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		SetProviderAttributes(span, "openai", "gpt-4")
	}
}

// BenchmarkSetRequestAttributes benchmarks setting request attributes
// Target: <10µs
func BenchmarkSetRequestAttributes(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		SetRequestAttributes(span, "req-123", "api-key-abc", "user@example.com")
	}
}

// BenchmarkSetCostWithTokens benchmarks setting cost and token attributes
// Target: <10µs
func BenchmarkSetCostWithTokens(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		SetCostWithTokens(span, 1500, 500, 0.05)
	}
}

// BenchmarkAttributeBuilder benchmarks the fluent attribute builder
// Target: <20µs
func BenchmarkAttributeBuilder(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		builder := NewAttributeBuilder().
			WithProvider("openai", "gpt-4").
			WithRequest("req-123", "user@example.com").
			WithTokens(1500, 500).
			WithCost(0.05)
		builder.Apply(span)
	}
}

// BenchmarkExtract benchmarks trace context extraction
// Target: <10µs
func BenchmarkExtract(b *testing.B) {
	headers := http.Header{}
	headers.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Extract(ctx, headers)
	}
}

// BenchmarkInject benchmarks trace context injection
// Target: <10µs
func BenchmarkInject(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		headers := http.Header{}
		Inject(ctx, headers)
	}
}

// BenchmarkValidateTraceParent benchmarks traceparent validation
// Target: <1µs
func BenchmarkValidateTraceParent(b *testing.B) {
	traceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ValidateTraceParent(traceparent)
	}
}

// BenchmarkParseTraceParent benchmarks traceparent parsing
// Target: <1µs
func BenchmarkParseTraceParent(b *testing.B) {
	traceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _, _, _ = ParseTraceParent(traceparent)
	}
}

// BenchmarkIsSampledFromTraceParent benchmarks sampling flag check
// Target: <1µs
func BenchmarkIsSampledFromTraceParent(b *testing.B) {
	traceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = IsSampledFromTraceParent(traceparent)
	}
}

// BenchmarkSpanFromContext benchmarks retrieving span from context
// Target: <1µs
func BenchmarkSpanFromContext(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = SpanFromContext(ctx)
	}
}

// BenchmarkTraceID benchmarks trace ID extraction
// Target: <1µs
func BenchmarkTraceID(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = TraceID(ctx)
	}
}

// BenchmarkSetError benchmarks setting error on span
// Target: <10µs
func BenchmarkSetError(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	_, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	testErr := context.DeadlineExceeded

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		SetError(span, testErr)
	}
}

// BenchmarkCreateSampler benchmarks sampler creation
// Target: <1µs
func BenchmarkCreateSampler(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = createSampler("ratio", 0.1)
	}
}

// BenchmarkFullRequestTrace benchmarks a complete request trace scenario
// Target: <100µs total
func BenchmarkFullRequestTrace(b *testing.B) {
	tracer, err := New(&config.TracingConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})
	if err != nil {
		b.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() { _ = tracer.Shutdown(context.Background()) }()

	headers := http.Header{}
	headers.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Extract context from headers
		ctx := Extract(context.Background(), headers)

		// Create request span
		ctx, requestSpan := tracer.Start(ctx, "mercator.proxy.request")
		SetRequestAttributes(requestSpan, "req-123", "api-key-abc", "user@example.com")

		// Create processing span
		ctx, processingSpan := tracer.Start(ctx, "mercator.processing.request")
		processingSpan.End()

		// Create provider span
		ctx, providerSpan := tracer.Start(ctx, "mercator.provider.call")
		SetProviderAttributes(providerSpan, "openai", "gpt-4")
		SetCostWithTokens(providerSpan, 1500, 500, 0.05)
		providerSpan.End()

		// End request span
		requestSpan.End()

		// Inject context into response headers
		responseHeaders := http.Header{}
		Inject(ctx, responseHeaders)
	}
}
