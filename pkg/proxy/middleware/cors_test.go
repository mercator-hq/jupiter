package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	t.Run("adds CORS headers for allowed origin", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
			MaxAge:         3600,
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header to be set")
		}
	})

	t.Run("allows all origins with wildcard", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST"},
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://any-origin.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		got := w.Header().Get("Access-Control-Allow-Origin")
		// When origin is present and wildcard is in allowed list,
		// it can return either the origin or "*" depending on middleware logic
		if got != "*" && got != "https://any-origin.com" {
			t.Errorf("Expected Access-Control-Allow-Origin to be '*' or matching origin, got: %s", got)
		}
	})

	t.Run("handles preflight OPTIONS request", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         3600,
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Preflight should return 204, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Access-Control-Allow-Methods should be set for preflight")
		}

		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("Access-Control-Allow-Headers should be set for preflight")
		}

		if w.Header().Get("Access-Control-Max-Age") != "3600" {
			t.Errorf("Access-Control-Max-Age = %v, want 3600", w.Header().Get("Access-Control-Max-Age"))
		}
	})

	t.Run("blocks disallowed origin", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		// Should not set CORS headers for disallowed origin
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("Should not set CORS headers for disallowed origin")
		}
	})

	t.Run("skips CORS when disabled", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        false,
			AllowedOrigins: []string{"*"},
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("Should not set CORS headers when disabled")
		}
	})

	t.Run("sets credentials header when enabled", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"https://example.com"},
			AllowCredentials: true,
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("Should set Access-Control-Allow-Credentials when enabled")
		}
	})

	t.Run("exposes headers", func(t *testing.T) {
		config := &CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			ExposedHeaders: []string{"X-Request-ID", "X-Total-Count"},
		}

		wrapped := CORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		exposed := w.Header().Get("Access-Control-Expose-Headers")
		if exposed == "" {
			t.Error("Should set Access-Control-Expose-Headers")
		}
	})
}

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	if !config.Enabled {
		t.Error("Default CORS should be enabled")
	}

	if len(config.AllowedOrigins) == 0 {
		t.Error("Default CORS should have allowed origins")
	}

	if len(config.AllowedMethods) == 0 {
		t.Error("Default CORS should have allowed methods")
	}

	if config.MaxAge == 0 {
		t.Error("Default CORS should have max age set")
	}
}

func BenchmarkCORSMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := DefaultCORSConfig()
	wrapped := CORSMiddleware(config)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
	}
}
