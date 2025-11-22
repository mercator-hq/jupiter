package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get request ID from context
		requestID := GetRequestID(r.Context())

		if requestID == "" {
			t.Error("Request ID should not be empty")
		}

		// Write request ID to header for verification
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrapped := RequestIDMiddleware(handler)

	t.Run("generates request ID when not provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("Request ID should be set in response header")
		}

		if len(requestID) < 10 {
			t.Errorf("Request ID seems too short: %s", requestID)
		}
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		customID := "custom-request-id-12345"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", customID)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID != customID {
			t.Errorf("Request ID = %v, want %v", requestID, customID)
		}
	})

	t.Run("generates unique IDs for different requests", func(t *testing.T) {
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		wrapped.ServeHTTP(w1, req1)

		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		wrapped.ServeHTTP(w2, req2)

		id1 := w1.Header().Get("X-Request-ID")
		id2 := w2.Header().Get("X-Request-ID")

		if id1 == id2 {
			t.Errorf("Request IDs should be unique, got %s for both", id1)
		}
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns empty string for context without request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestID := GetRequestID(req.Context())

		if requestID != "" {
			t.Errorf("Expected empty string, got %s", requestID)
		}
	})

	t.Run("returns request ID from context", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := GetRequestID(r.Context())
			if requestID == "" {
				t.Error("Request ID should not be empty in handler")
			}
			w.WriteHeader(http.StatusOK)
		})

		wrapped := RequestIDMiddleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)
	})
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestIDMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
	}
}
