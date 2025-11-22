package tracing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mercator-hq/jupiter/pkg/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Tracer wraps the OpenTelemetry tracer and provides simplified span creation
// with automatic attribute handling and context propagation.
type Tracer struct {
	config   *config.TracingConfig
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	sampler  sdktrace.Sampler
	enabled  bool
}

// New creates a new Tracer with the given configuration.
// It initializes the OpenTelemetry SDK, sets up the configured exporter,
// and returns a ready-to-use tracer.
//
// If tracing is disabled in the config, a noop tracer is returned that
// adds minimal overhead (<1µs per operation).
//
// The tracer must be shut down when no longer needed:
//
//	defer tracer.Shutdown(context.Background())
func New(cfg *config.TracingConfig) (*Tracer, error) {
	if cfg == nil {
		return nil, errors.New("tracing config is nil")
	}

	t := &Tracer{
		config:  cfg,
		enabled: cfg.Enabled,
	}

	// If tracing is disabled, return noop tracer
	if !cfg.Enabled {
		t.tracer = trace.NewNoopTracerProvider().Tracer("mercator-jupiter")
		return t, nil
	}

	// Create sampler
	sampler, err := createSampler(cfg.Sampler, cfg.SampleRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to create sampler: %w", err)
	}
	t.sampler = sampler

	// Create exporter
	exporter, err := createExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("0.1.0"), // TODO: Get from build info
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	t.provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global trace provider
	otel.SetTracerProvider(t.provider)

	// Set global propagator for W3C Trace Context
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// Create tracer
	t.tracer = t.provider.Tracer("mercator-jupiter")

	return t, nil
}

// Start creates a new span with the given name and options.
// The span is automatically linked to the parent span from the context.
//
// The returned span must be ended when the operation completes:
//
//	ctx, span := tracer.Start(ctx, "operation")
//	defer span.End()
//
// If tracing is disabled, a noop span is returned with minimal overhead.
func (t *Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// Shutdown flushes any pending spans and shuts down the tracer.
// It should be called before application exit, typically with defer:
//
//	defer tracer.Shutdown(context.Background())
func (t *Tracer) Shutdown(ctx context.Context) error {
	if !t.enabled || t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

// Enabled returns whether tracing is enabled.
func (t *Tracer) Enabled() bool {
	return t.enabled
}

// createExporter creates a trace exporter based on the configuration.
func createExporter(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "otlp":
		return createOTLPExporter(cfg)
	case "jaeger":
		return createJaegerExporter(cfg)
	case "zipkin":
		return createZipkinExporter(cfg)
	default:
		return nil, fmt.Errorf("unsupported exporter: %s", cfg.Exporter)
	}
}

// createOTLPExporter creates an OTLP gRPC exporter.
func createOTLPExporter(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	// Add insecure option if configured
	if cfg.OTLP.Insecure {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	// Add timeout if configured
	if cfg.OTLP.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(cfg.OTLP.Timeout))
	}

	// Add dial option to handle connection
	opts = append(opts, otlptracegrpc.WithDialOption(
		grpc.WithBlock(),
	))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := otlptracegrpc.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	return exporter, nil
}

// createJaegerExporter creates a Jaeger exporter.
// Note: This requires the jaeger exporter package which may need to be added.
func createJaegerExporter(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	// For now, return an error indicating Jaeger support needs to be added
	// In a real implementation, you would use:
	// import "go.opentelemetry.io/otel/exporters/jaeger"
	// return jaeger.New(jaeger.WithAgentEndpoint(
	//     jaeger.WithAgentHost(cfg.Jaeger.AgentHost),
	//     jaeger.WithAgentPort(fmt.Sprintf("%d", cfg.Jaeger.AgentPort)),
	// ))
	return nil, errors.New("Jaeger exporter not yet implemented - use OTLP exporter with Jaeger collector")
}

// createZipkinExporter creates a Zipkin exporter.
// Note: This requires the zipkin exporter package which may need to be added.
func createZipkinExporter(cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	// For now, return an error indicating Zipkin support needs to be added
	// In a real implementation, you would use:
	// import "go.opentelemetry.io/otel/exporters/zipkin"
	// return zipkin.New(cfg.Endpoint)
	return nil, errors.New("Zipkin exporter not yet implemented - use OTLP exporter with Zipkin collector")
}

// SpanFromContext returns the current span from the context.
// If no span exists, a noop span is returned.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the given span.
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// SpanContext returns the span context from the given context.
// Returns an invalid span context if no span exists.
func SpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanFromContext(ctx).SpanContext()
}

// TraceID returns the trace ID from the context as a string.
// Returns empty string if no trace context exists.
func TraceID(ctx context.Context) string {
	sc := SpanContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// SpanID returns the span ID from the context as a string.
// Returns empty string if no span context exists.
func SpanID(ctx context.Context) string {
	sc := SpanContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}

// IsSampled returns whether the current trace is sampled.
func IsSampled(ctx context.Context) bool {
	return SpanContext(ctx).IsSampled()
}

// SetError marks the span as failed and records the error.
func SetError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.SetAttributes(
		attribute.Bool("error", true),
		attribute.String("error.message", err.Error()),
	)
	span.RecordError(err)
}

// SetStatus sets the span status based on an error.
// If err is nil, status is set to OK, otherwise to Error.
func SetStatus(span trace.Span, err error) {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
}
