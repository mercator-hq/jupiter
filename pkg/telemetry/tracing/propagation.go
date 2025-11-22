package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// W3C Trace Context Propagation
//
// The W3C Trace Context specification (https://www.w3.org/TR/trace-context/)
// defines standard HTTP headers for propagating trace context across service
// boundaries.
//
// # Headers
//
// traceparent: Required header containing trace context
// Format: version-trace_id-parent_id-trace_flags
// Example: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//
// tracestate: Optional header containing vendor-specific trace context
// Format: key1=value1,key2=value2
// Example: congo=t61rcWkgMzE,rojo=00f067aa0ba902b7
//
// # Trace Flags
//
// The trace flags byte contains:
//   - Bit 0: Sampled (1 = sampled, 0 = not sampled)
//   - Bits 1-7: Reserved for future use
//
// # Example Flow
//
// Service A → Service B → Service C
//
// Service A creates trace:
//   trace_id: 4bf92f3577b34da6a3ce929d0e0e4736
//   span_id:  00f067aa0ba902b7
//   sampled:  true
//
// Service A calls Service B with header:
//   traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//
// Service B extracts context, creates child span:
//   trace_id: 4bf92f3577b34da6a3ce929d0e0e4736 (same)
//   parent_id: 00f067aa0ba902b7 (from Service A)
//   span_id: 5e107e4a0ba902c8 (new)
//
// Service B calls Service C with header:
//   traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-5e107e4a0ba902c8-01

// Propagator returns the configured text map propagator.
// This is typically a composite propagator that handles both
// W3C Trace Context and W3C Baggage.
func Propagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// Extract extracts trace context from HTTP headers and returns a context
// with the extracted trace context.
//
// This should be called on the server side when receiving an HTTP request:
//
//	ctx := propagation.Extract(r.Context(), r.Header)
//	ctx, span := tracer.Start(ctx, "handle_request")
//	defer span.End()
//
// If no trace context is found in the headers, the original context is returned.
func Extract(ctx context.Context, headers http.Header) context.Context {
	return Propagator().Extract(ctx, propagation.HeaderCarrier(headers))
}

// Inject injects trace context into HTTP headers.
//
// This should be called on the client side before making an HTTP request:
//
//	req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
//	propagation.Inject(ctx, req.Header)
//	resp, err := client.Do(req)
//
// The trace context from ctx is serialized into traceparent and tracestate headers.
func Inject(ctx context.Context, headers http.Header) {
	Propagator().Inject(ctx, propagation.HeaderCarrier(headers))
}

// ExtractFromMap extracts trace context from a string map.
// This is useful for extracting context from non-HTTP sources.
func ExtractFromMap(ctx context.Context, carrier map[string]string) context.Context {
	return Propagator().Extract(ctx, propagation.MapCarrier(carrier))
}

// InjectToMap injects trace context into a string map.
// This is useful for injecting context into non-HTTP destinations.
func InjectToMap(ctx context.Context, carrier map[string]string) {
	Propagator().Inject(ctx, propagation.MapCarrier(carrier))
}

// HTTPMiddleware returns an HTTP middleware that automatically extracts
// trace context from incoming requests and injects it into outgoing responses.
//
// Usage:
//
//	http.Handle("/", propagation.HTTPMiddleware(handler))
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from request headers
		ctx := Extract(r.Context(), r.Header)

		// Inject trace context into response headers (for debugging)
		// Note: Typically you don't inject into responses, but it can be useful
		// for debugging to see the trace ID in response headers
		if span := SpanFromContext(ctx); span.SpanContext().IsValid() {
			w.Header().Set("X-Trace-ID", span.SpanContext().TraceID().String())
			w.Header().Set("X-Span-ID", span.SpanContext().SpanID().String())
		}

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateTraceParent validates the traceparent header format.
// Returns true if the header is valid according to W3C Trace Context spec.
//
// Format: version-trace_id-parent_id-trace_flags
//   - version: 2 hex digits (00)
//   - trace_id: 32 hex digits (128-bit)
//   - parent_id: 16 hex digits (64-bit)
//   - trace_flags: 2 hex digits (8-bit)
//
// Example: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
func ValidateTraceParent(traceparent string) bool {
	// Split into parts
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return false
	}

	// Validate version (2 hex digits)
	if len(parts[0]) != 2 || !isHexString(parts[0]) {
		return false
	}

	// Validate trace ID (32 hex digits)
	if len(parts[1]) != 32 || !isHexString(parts[1]) {
		return false
	}

	// Validate parent ID (16 hex digits)
	if len(parts[2]) != 16 || !isHexString(parts[2]) {
		return false
	}

	// Validate trace flags (2 hex digits)
	if len(parts[3]) != 2 || !isHexString(parts[3]) {
		return false
	}

	// Check for all-zeros trace ID (invalid)
	if parts[1] == "00000000000000000000000000000000" {
		return false
	}

	// Check for all-zeros parent ID (invalid)
	if parts[2] == "0000000000000000" {
		return false
	}

	return true
}

// isHexString checks if a string contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ParseTraceParent parses a traceparent header into its components.
// Returns empty strings if the header is invalid.
func ParseTraceParent(traceparent string) (version, traceID, parentID, flags string, valid bool) {
	if !ValidateTraceParent(traceparent) {
		return "", "", "", "", false
	}

	parts := strings.Split(traceparent, "-")
	return parts[0], parts[1], parts[2], parts[3], true
}

// IsSampledFromTraceParent checks if a trace is sampled based on the
// traceparent header's trace flags.
func IsSampledFromTraceParent(traceparent string) bool {
	_, _, _, flags, valid := ParseTraceParent(traceparent)
	if !valid {
		return false
	}

	// Check if the sampled bit (bit 0) is set
	// flags is a 2-character hex string representing 8 bits
	// We need to check if the last bit is 1
	if len(flags) != 2 {
		return false
	}

	// Parse the flags as hex
	var flagsByte byte
	if _, err := fmt.Sscanf(flags, "%02x", &flagsByte); err != nil {
		return false
	}

	// Check if bit 0 is set (sampled)
	return (flagsByte & 0x01) == 0x01
}

// PropagationDebugInfo returns debug information about trace propagation
// from HTTP headers.
func PropagationDebugInfo(headers http.Header) map[string]string {
	info := make(map[string]string)

	// Check for traceparent header
	if traceparent := headers.Get("traceparent"); traceparent != "" {
		info["traceparent"] = traceparent
		version, traceID, parentID, flags, valid := ParseTraceParent(traceparent)
		if valid {
			info["version"] = version
			info["trace_id"] = traceID
			info["parent_id"] = parentID
			info["flags"] = flags
			info["sampled"] = fmt.Sprintf("%t", IsSampledFromTraceParent(traceparent))
		} else {
			info["error"] = "invalid traceparent format"
		}
	} else {
		info["traceparent"] = "not present"
	}

	// Check for tracestate header
	if tracestate := headers.Get("tracestate"); tracestate != "" {
		info["tracestate"] = tracestate
	} else {
		info["tracestate"] = "not present"
	}

	return info
}
