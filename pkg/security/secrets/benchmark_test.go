package secrets

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkEnvProvider_GetSecret(b *testing.B) {
	os.Setenv("MERCATOR_SECRET_BENCH_KEY", "benchmark-value")
	provider := NewEnvProvider("MERCATOR_SECRET_")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetSecret(ctx, "bench-key")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileProvider_GetSecret_Cached(b *testing.B) {
	tmpDir := b.TempDir()
	secretFile := filepath.Join(tmpDir, "test-secret")
	err := os.WriteFile(secretFile, []byte("secret-value"), 0600)
	if err != nil {
		b.Fatalf("failed to create secret file: %v", err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		b.Fatalf("failed to create file provider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Prime cache
	_, _ = provider.GetSecret(ctx, "test-secret")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetSecret(ctx, "test-secret")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileProvider_GetSecret_Uncached(b *testing.B) {
	tmpDir := b.TempDir()
	secretFile := filepath.Join(tmpDir, "test-secret")
	err := os.WriteFile(secretFile, []byte("secret-value"), 0600)
	if err != nil {
		b.Fatalf("failed to create secret file: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		provider, err := NewFileProvider(tmpDir, false)
		if err != nil {
			b.Fatalf("failed to create file provider: %v", err)
		}
		b.StartTimer()

		_, err = provider.GetSecret(ctx, "test-secret")
		if err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		provider.Close()
		b.StartTimer()
	}
}

func BenchmarkManager_GetSecret_CacheHit(b *testing.B) {
	os.Setenv("MERCATOR_SECRET_BENCH", "value")
	provider := NewEnvProvider("MERCATOR_SECRET_")

	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	}

	manager := NewManager([]SecretProvider{provider}, cacheConfig)
	ctx := context.Background()

	// Prime cache
	_, _ = manager.GetSecret(ctx, "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GetSecret(ctx, "bench")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManager_GetSecret_CacheMiss(b *testing.B) {
	os.Setenv("MERCATOR_SECRET_BENCH", "value")
	provider := NewEnvProvider("MERCATOR_SECRET_")

	cacheConfig := CacheConfig{
		Enabled: false, // Disable cache
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	}

	manager := NewManager([]SecretProvider{provider}, cacheConfig)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GetSecret(ctx, "bench")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManager_ResolveReferences(b *testing.B) {
	os.Setenv("MERCATOR_SECRET_KEY1", "value1")
	os.Setenv("MERCATOR_SECRET_KEY2", "value2")
	provider := NewEnvProvider("MERCATOR_SECRET_")

	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	}

	manager := NewManager([]SecretProvider{provider}, cacheConfig)
	ctx := context.Background()

	input := "api_key: ${secret:key1}, password: ${secret:key2}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ResolveReferences(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManager_ResolveReferences_10Refs(b *testing.B) {
	// Set up 10 secrets
	for i := 0; i < 10; i++ {
		os.Setenv("MERCATOR_SECRET_KEY"+string(rune('0'+i)), "value")
	}
	provider := NewEnvProvider("MERCATOR_SECRET_")

	cacheConfig := CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	}

	manager := NewManager([]SecretProvider{provider}, cacheConfig)
	ctx := context.Background()

	input := ""
	for i := 0; i < 10; i++ {
		if i > 0 {
			input += ", "
		}
		input += "${secret:key" + string(rune('0'+i)) + "}"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ResolveReferences(ctx, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache_Get(b *testing.B) {
	cache := NewCache(CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	})

	cache.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, ok := cache.Get("key")
		if !ok {
			b.Fatal("cache miss")
		}
	}
}

func BenchmarkCache_Set(b *testing.B) {
	cache := NewCache(CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", "value")
	}
}

func BenchmarkCache_GetSet_Concurrent(b *testing.B) {
	cache := NewCache(CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.Set("key", "value")
			} else {
				_, _ = cache.Get("key")
			}
			i++
		}
	})
}

func BenchmarkEnvProvider_ListSecrets(b *testing.B) {
	// Set up multiple secrets
	for i := 0; i < 10; i++ {
		os.Setenv("MERCATOR_SECRET_KEY"+string(rune('0'+i)), "value")
	}
	provider := NewEnvProvider("MERCATOR_SECRET_")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.ListSecrets(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileProvider_ListSecrets(b *testing.B) {
	tmpDir := b.TempDir()

	// Create multiple secret files
	for i := 0; i < 10; i++ {
		secretFile := filepath.Join(tmpDir, "secret-"+string(rune('0'+i)))
		err := os.WriteFile(secretFile, []byte("value"), 0600)
		if err != nil {
			b.Fatalf("failed to create secret file: %v", err)
		}
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		b.Fatalf("failed to create file provider: %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.ListSecrets(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
