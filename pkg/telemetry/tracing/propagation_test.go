package tracing

import (
	"context"
	"net/http"
	"testing"
)

// TestValidateTraceParent tests traceparent header validation
func TestValidateTraceParent(t *testing.T) {
	tests := []struct {
		name        string
		traceparent string
		want        bool
	}{
		{
			name:        "valid traceparent",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        true,
		},
		{
			name:        "valid traceparent - not sampled",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00",
			want:        true,
		},
		{
			name:        "invalid - wrong number of parts",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7",
			want:        false,
		},
		{
			name:        "invalid - version wrong length",
			traceparent: "0-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "invalid - trace ID wrong length",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e473-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "invalid - parent ID wrong length",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902-01",
			want:        false,
		},
		{
			name:        "invalid - flags wrong length",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-1",
			want:        false,
		},
		{
			name:        "invalid - non-hex characters in trace ID",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e473g-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "invalid - non-hex characters in parent ID",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902bz-01",
			want:        false,
		},
		{
			name:        "invalid - all-zeros trace ID",
			traceparent: "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
			want:        false,
		},
		{
			name:        "invalid - all-zeros parent ID",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01",
			want:        false,
		},
		{
			name:        "empty string",
			traceparent: "",
			want:        false,
		},
		{
			name:        "invalid format",
			traceparent: "invalid",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateTraceParent(tt.traceparent); got != tt.want {
				t.Errorf("ValidateTraceParent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseTraceParent tests traceparent header parsing
func TestParseTraceParent(t *testing.T) {
	tests := []struct {
		name          string
		traceparent   string
		wantVersion   string
		wantTraceID   string
		wantParentID  string
		wantFlags     string
		wantValid     bool
	}{
		{
			name:          "valid traceparent",
			traceparent:   "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantVersion:   "00",
			wantTraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
			wantParentID:  "00f067aa0ba902b7",
			wantFlags:     "01",
			wantValid:     true,
		},
		{
			name:          "valid traceparent - not sampled",
			traceparent:   "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00",
			wantVersion:   "00",
			wantTraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
			wantParentID:  "00f067aa0ba902b7",
			wantFlags:     "00",
			wantValid:     true,
		},
		{
			name:          "invalid traceparent",
			traceparent:   "invalid",
			wantVersion:   "",
			wantTraceID:   "",
			wantParentID:  "",
			wantFlags:     "",
			wantValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, traceID, parentID, flags, valid := ParseTraceParent(tt.traceparent)
			if valid != tt.wantValid {
				t.Errorf("ParseTraceParent() valid = %v, want %v", valid, tt.wantValid)
			}
			if version != tt.wantVersion {
				t.Errorf("ParseTraceParent() version = %v, want %v", version, tt.wantVersion)
			}
			if traceID != tt.wantTraceID {
				t.Errorf("ParseTraceParent() traceID = %v, want %v", traceID, tt.wantTraceID)
			}
			if parentID != tt.wantParentID {
				t.Errorf("ParseTraceParent() parentID = %v, want %v", parentID, tt.wantParentID)
			}
			if flags != tt.wantFlags {
				t.Errorf("ParseTraceParent() flags = %v, want %v", flags, tt.wantFlags)
			}
		})
	}
}

// TestIsSampledFromTraceParent tests sampling flag extraction
func TestIsSampledFromTraceParent(t *testing.T) {
	tests := []struct {
		name        string
		traceparent string
		want        bool
	}{
		{
			name:        "sampled (01)",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			want:        true,
		},
		{
			name:        "not sampled (00)",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00",
			want:        false,
		},
		{
			name:        "sampled with other flags (03)",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-03",
			want:        true,
		},
		{
			name:        "not sampled with other flags (02)",
			traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-02",
			want:        false,
		},
		{
			name:        "invalid traceparent",
			traceparent: "invalid",
			want:        false,
		},
		{
			name:        "empty string",
			traceparent: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSampledFromTraceParent(tt.traceparent); got != tt.want {
				t.Errorf("IsSampledFromTraceParent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsHexString tests hex string validation
func TestIsHexString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "valid lowercase hex",
			s:    "4bf92f3577b34da6a3ce929d0e0e4736",
			want: true,
		},
		{
			name: "valid uppercase hex",
			s:    "4BF92F3577B34DA6A3CE929D0E0E4736",
			want: true,
		},
		{
			name: "valid mixed case hex",
			s:    "4BF92f3577b34DA6a3CE929d0e0e4736",
			want: true,
		},
		{
			name: "invalid - contains g",
			s:    "4bf92f3577b34da6a3ce929d0e0e473g",
			want: false,
		},
		{
			name: "invalid - contains z",
			s:    "4bf92f3577b34da6a3ce929d0e0e473z",
			want: false,
		},
		{
			name: "invalid - contains space",
			s:    "4bf92f35 77b34da6a3ce929d0e0e4736",
			want: false,
		},
		{
			name: "empty string",
			s:    "",
			want: true, // Empty string is technically all hex
		},
		{
			name: "valid - all zeros",
			s:    "00000000000000000000000000000000",
			want: true,
		},
		{
			name: "valid - all f's",
			s:    "ffffffffffffffffffffffffffffffff",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHexString(tt.s); got != tt.want {
				t.Errorf("isHexString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtract tests trace context extraction from HTTP headers
func TestExtract(t *testing.T) {
	ctx := context.Background()

	// Test with valid traceparent
	headers := http.Header{}
	headers.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	extractedCtx := Extract(ctx, headers)
	if extractedCtx == nil {
		t.Error("Extract() returned nil context")
	}

	// Test with no traceparent
	headers = http.Header{}
	extractedCtx = Extract(ctx, headers)
	if extractedCtx == nil {
		t.Error("Extract() returned nil context")
	}

	// Test with invalid traceparent
	headers = http.Header{}
	headers.Set("traceparent", "invalid")
	extractedCtx = Extract(ctx, headers)
	if extractedCtx == nil {
		t.Error("Extract() returned nil context")
	}
}

// TestInject tests trace context injection into HTTP headers
func TestInject(t *testing.T) {
	ctx := context.Background()
	headers := http.Header{}

	// Inject should not panic even with no span
	Inject(ctx, headers)

	// Headers may or may not contain traceparent depending on context
	// Just verify it doesn't panic
}

// TestExtractFromMap tests trace context extraction from map
func TestExtractFromMap(t *testing.T) {
	ctx := context.Background()

	// Test with valid traceparent
	carrier := map[string]string{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}

	extractedCtx := ExtractFromMap(ctx, carrier)
	if extractedCtx == nil {
		t.Error("ExtractFromMap() returned nil context")
	}

	// Test with no traceparent
	carrier = map[string]string{}
	extractedCtx = ExtractFromMap(ctx, carrier)
	if extractedCtx == nil {
		t.Error("ExtractFromMap() returned nil context")
	}
}

// TestInjectToMap tests trace context injection into map
func TestInjectToMap(t *testing.T) {
	ctx := context.Background()
	carrier := map[string]string{}

	// Inject should not panic even with no span
	InjectToMap(ctx, carrier)

	// Carrier may or may not contain traceparent depending on context
	// Just verify it doesn't panic
}

// TestPropagationDebugInfo tests debug info generation
func TestPropagationDebugInfo(t *testing.T) {
	tests := []struct {
		name       string
		setupHeaders func() http.Header
		wantKeys   []string
	}{
		{
			name: "with valid traceparent",
			setupHeaders: func() http.Header {
				h := http.Header{}
				h.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
				return h
			},
			wantKeys: []string{"traceparent", "version", "trace_id", "parent_id", "flags", "sampled", "tracestate"},
		},
		{
			name: "with invalid traceparent",
			setupHeaders: func() http.Header {
				h := http.Header{}
				h.Set("traceparent", "invalid")
				return h
			},
			wantKeys: []string{"traceparent", "error", "tracestate"},
		},
		{
			name: "with no headers",
			setupHeaders: func() http.Header {
				return http.Header{}
			},
			wantKeys: []string{"traceparent", "tracestate"},
		},
		{
			name: "with tracestate",
			setupHeaders: func() http.Header {
				h := http.Header{}
				h.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
				h.Set("tracestate", "congo=t61rcWkgMzE")
				return h
			},
			wantKeys: []string{"traceparent", "version", "trace_id", "parent_id", "flags", "sampled", "tracestate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := tt.setupHeaders()
			info := PropagationDebugInfo(headers)

			// Verify expected keys are present
			for _, key := range tt.wantKeys {
				if _, ok := info[key]; !ok {
					t.Errorf("PropagationDebugInfo() missing key %q", key)
				}
			}
		})
	}
}

// TestHTTPMiddleware tests the HTTP middleware
func TestHTTPMiddleware(t *testing.T) {
	// Create a test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify context is passed through
		if r.Context() == nil {
			t.Error("HTTPMiddleware() handler received nil context")
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	middleware := HTTPMiddleware(testHandler)

	// Create test request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add traceparent header
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	// Create response recorder
	rr := &testResponseWriter{header: make(http.Header)}

	// Call middleware
	middleware.ServeHTTP(rr, req)

	// Verify handler was called
	if !handlerCalled {
		t.Error("HTTPMiddleware() did not call handler")
	}
}

// testResponseWriter is a simple ResponseWriter for testing
type testResponseWriter struct {
	header http.Header
	code   int
	body   []byte
}

func (w *testResponseWriter) Header() http.Header {
	return w.header
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *testResponseWriter) WriteHeader(code int) {
	w.code = code
}
