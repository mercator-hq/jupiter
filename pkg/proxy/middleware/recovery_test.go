package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		wrapped := RecoveryMiddleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		// Should not panic
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("passes through normal requests", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})

		wrapped := RecoveryMiddleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
		}

		if w.Body.String() != "OK" {
			t.Errorf("Body = %v, want OK", w.Body.String())
		}
	})

	t.Run("recovers from panic with string", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("string panic")
		})

		wrapped := RecoveryMiddleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("recovers from panic with error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(http.ErrAbortHandler)
		})

		wrapped := RecoveryMiddleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Status code = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RecoveryMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
	}
}
