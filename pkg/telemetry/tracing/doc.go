// Package tracing provides OpenTelemetry distributed tracing for Mercator Jupiter.
//
// # Overview
//
// The tracing package implements W3C Trace Context propagation, span creation,
// and trace export to OTLP, Jaeger, and Zipkin collectors. It provides visibility
// into request flows across system boundaries with minimal overhead (<100µs per span).
//
// # Distributed Tracing
//
// Distributed tracing tracks requests as they flow through multiple services,
// creating a hierarchy of spans that represent operations. Each span records:
//   - Operation name and duration
//   - Attributes (key-value pairs)
//   - Events (timestamped logs within the span)
//   - Trace context (trace ID, span ID, sampling decision)
//
// # Trace Context Propagation
//
// The package implements W3C Trace Context (https://www.w3.org/TR/trace-context/)
// for propagating trace context across HTTP boundaries:
//
//	traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//	tracestate: congo=t61rcWkgMzE
//
// # Sampling Strategies
//
// Three sampling strategies are supported:
//   - always: Sample all traces (development/debugging)
//   - never: Sample no traces (tracing disabled)
//   - ratio: Sample a percentage of traces (production)
//
// # Usage
//
//	// Initialize tracer
//	cfg := &config.TracingConfig{
//	    Enabled:     true,
//	    Sampler:     "ratio",
//	    SampleRatio: 0.1,
//	    Exporter:    "otlp",
//	    Endpoint:    "localhost:4317",
//	    ServiceName: "mercator-jupiter",
//	}
//	tracer, err := tracing.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tracer.Shutdown(context.Background())
//
//	// Create span
//	ctx, span := tracer.Start(ctx, "mercator.proxy.request")
//	defer span.End()
//
//	// Add attributes
//	span.SetAttributes(
//	    attribute.String("provider", "openai"),
//	    attribute.String("model", "gpt-4"),
//	    attribute.Int("tokens", 1500),
//	    attribute.Float64("cost", 0.05),
//	)
//
//	// Add event
//	span.AddEvent("policy_evaluated", trace.WithAttributes(
//	    attribute.String("rule_id", "cost-limit"),
//	    attribute.String("action", "allow"),
//	))
//
// # Span Hierarchy
//
// Spans form a hierarchy representing the call tree:
//
//	mercator.proxy.request (10s)
//	├── mercator.processing.request (5ms)
//	├── mercator.policy.evaluate (2ms)
//	├── mercator.provider.call (9.9s)
//	│   ├── mercator.provider.connect (100ms)
//	│   ├── mercator.provider.send (50ms)
//	│   └── mercator.provider.receive (9.75s)
//	└── mercator.evidence.generate (10ms)
//
// # HTTP Integration
//
// Extract trace context from incoming HTTP requests:
//
//	ctx := propagation.Extract(r.Context(), r.Header)
//	ctx, span := tracer.Start(ctx, "handle_request")
//	defer span.End()
//
// Inject trace context into outgoing HTTP requests:
//
//	req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
//	propagation.Inject(ctx, req.Header)
//
// # Performance
//
// The tracing package is designed for minimal overhead:
//   - Span creation: <100µs per span
//   - Context propagation: <10µs
//   - Sampling decision: <1µs
//   - When disabled: <1µs (noop span)
//
// # Trace Exporters
//
// Three trace exporters are supported:
//
// OTLP (OpenTelemetry Protocol):
//
//	telemetry:
//	  tracing:
//	    exporter: otlp
//	    endpoint: localhost:4317
//	    otlp:
//	      insecure: true
//	      timeout: 10s
//
// Jaeger:
//
//	telemetry:
//	  tracing:
//	    exporter: jaeger
//	    jaeger:
//	      agent_host: localhost
//	      agent_port: 6831
//
// Zipkin:
//
//	telemetry:
//	  tracing:
//	    exporter: zipkin
//	    endpoint: http://localhost:9411/api/v2/spans
//
// # Attribute Helpers
//
// Common attributes can be set using helper functions:
//
//	// Provider attributes
//	tracing.SetProviderAttributes(span, "openai", "gpt-4")
//
//	// Request attributes
//	tracing.SetRequestAttributes(span, requestID, apiKey, user)
//
//	// Cost attributes
//	tracing.SetCostAttributes(span, 1500, 500, 0.05)
//
//	// Error attributes
//	tracing.SetErrorAttributes(span, err, "rate_limit")
package tracing
