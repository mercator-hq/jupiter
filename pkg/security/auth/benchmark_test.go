package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkAPIKeyValidator_Validate(b *testing.B) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-benchmark-key-1234567890",
			UserID:    "user-1",
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.Validate("sk-benchmark-key-1234567890")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAPIKeyValidator_ValidateMultipleKeys(b *testing.B) {
	// Simulate 1000 keys
	keys := make([]*APIKeyInfo, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = &APIKeyInfo{
			Key:       fmt.Sprintf("sk-key-%d", i),
			UserID:    fmt.Sprintf("user-%d", i),
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		}
	}

	validator := NewAPIKeyValidator(keys)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Validate a key in the middle
		_, err := validator.Validate("sk-key-500")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAPIKeyValidator_ValidateInvalid(b *testing.B) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-valid-key",
			UserID:    "user-1",
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.Validate("sk-invalid-key")
		if err == nil {
			b.Fatal("expected error for invalid key")
		}
	}
}

func BenchmarkAPIKeyMiddleware_Handle(b *testing.B) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-benchmark-key",
			UserID:    "user-1",
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)
	sources := []APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
	}

	middleware := NewAPIKeyMiddleware(validator, sources)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Handle(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer sk-benchmark-key")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", w.Code)
		}
	}
}

func BenchmarkAPIKeyMiddleware_HandleUnauthorized(b *testing.B) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-valid-key",
			UserID:    "user-1",
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)
	sources := []APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
	}

	middleware := NewAPIKeyMiddleware(validator, sources)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Handle(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer sk-invalid-key")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			b.Fatalf("expected 401, got: %d", w.Code)
		}
	}
}

func BenchmarkExtractAPIKey_Bearer(b *testing.B) {
	sources := []APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
		{Type: "header", Name: "X-API-Key", Scheme: ""},
	}

	middleware := &APIKeyMiddleware{
		validator: nil,
		sources:   sources,
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer sk-test-key-1234567890")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := middleware.extractAPIKey(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExtractAPIKey_CustomHeader(b *testing.B) {
	sources := []APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
		{Type: "header", Name: "X-API-Key", Scheme: ""},
	}

	middleware := &APIKeyMiddleware{
		validator: nil,
		sources:   sources,
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "sk-test-key-1234567890")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := middleware.extractAPIKey(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAPIKeyInfo(b *testing.B) {
	keyInfo := &APIKeyInfo{
		Key:       "sk-test-key",
		UserID:    "user-1",
		TeamID:    "team-eng",
		Enabled:   true,
		RateLimit: "1000/hour",
		CreatedAt: time.Now(),
	}

	ctx := context.WithValue(context.Background(), apiKeyInfoKey, keyInfo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, ok := GetAPIKeyInfo(ctx)
		if !ok {
			b.Fatal("key info not found")
		}
	}
}

func BenchmarkAPIKeyValidator_Add(b *testing.B) {
	validator := NewAPIKeyValidator(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := fmt.Sprintf("sk-key-%d", i)
		b.StartTimer()

		validator.Add(&APIKeyInfo{
			Key:       key,
			UserID:    fmt.Sprintf("user-%d", i),
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		})
	}
}

func BenchmarkAPIKeyValidator_Remove(b *testing.B) {
	// Pre-populate with keys
	keys := make([]*APIKeyInfo, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = &APIKeyInfo{
			Key:       fmt.Sprintf("sk-key-%d", i),
			UserID:    fmt.Sprintf("user-%d", i),
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		}
	}

	validator := NewAPIKeyValidator(keys)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("sk-key-%d", i%1000)
		validator.Remove(key)
	}
}

func BenchmarkAPIKeyValidator_List(b *testing.B) {
	// Pre-populate with keys
	keys := make([]*APIKeyInfo, 100)
	for i := 0; i < 100; i++ {
		keys[i] = &APIKeyInfo{
			Key:       fmt.Sprintf("sk-key-%d", i),
			UserID:    fmt.Sprintf("user-%d", i),
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		}
	}

	validator := NewAPIKeyValidator(keys)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := validator.List()
		if len(list) != 100 {
			b.Fatalf("expected 100 keys, got %d", len(list))
		}
	}
}

func BenchmarkAPIKeyValidator_Concurrent(b *testing.B) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-benchmark-key",
			UserID:    "user-1",
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := validator.Validate("sk-benchmark-key")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
