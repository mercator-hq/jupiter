package secrets

import (
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		MaxSize: 10,
	}

	cache := NewCache(config)

	// Set a value
	cache.Set("test-key", "test-value")

	// Get the value
	value, ok := cache.Get("test-key")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}

	if value != "test-value" {
		t.Errorf("expected value 'test-value', got '%s'", value)
	}
}

func TestCache_Miss(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		MaxSize: 10,
	}

	cache := NewCache(config)

	// Try to get non-existent key
	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("expected cache miss, got hit")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     100 * time.Millisecond,
		MaxSize: 10,
	}

	cache := NewCache(config)

	// Set a value
	cache.Set("test-key", "test-value")

	// Value should be available immediately
	_, ok := cache.Get("test-key")
	if !ok {
		t.Error("expected cache hit immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Value should be expired
	_, ok = cache.Get("test-key")
	if ok {
		t.Error("expected cache miss after TTL expiration")
	}
}

func TestCache_MaxSize(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		MaxSize: 3,
	}

	cache := NewCache(config)

	// Add 4 values (one more than max size)
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	cache.Set("key4", "value4")

	// Cache size should be limited to MaxSize
	size := cache.Size()
	if size > config.MaxSize {
		t.Errorf("cache size %d exceeds max size %d", size, config.MaxSize)
	}

	// At least one of the first keys should have been evicted
	hits := 0
	for _, key := range []string{"key1", "key2", "key3", "key4"} {
		if _, ok := cache.Get(key); ok {
			hits++
		}
	}

	if hits > config.MaxSize {
		t.Errorf("expected at most %d hits, got %d", config.MaxSize, hits)
	}
}

func TestCache_Clear(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		MaxSize: 10,
	}

	cache := NewCache(config)

	// Add some values
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Clear cache
	cache.Clear()

	// All values should be gone
	if _, ok := cache.Get("key1"); ok {
		t.Error("expected cache miss after clear")
	}
	if _, ok := cache.Get("key2"); ok {
		t.Error("expected cache miss after clear")
	}
	if _, ok := cache.Get("key3"); ok {
		t.Error("expected cache miss after clear")
	}

	// Size should be 0
	if size := cache.Size(); size != 0 {
		t.Errorf("expected size 0 after clear, got %d", size)
	}
}

func TestCache_Delete(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		MaxSize: 10,
	}

	cache := NewCache(config)

	// Add some values
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Delete one key
	cache.Delete("key1")

	// key1 should be gone
	if _, ok := cache.Get("key1"); ok {
		t.Error("expected cache miss after delete")
	}

	// key2 should still exist
	if _, ok := cache.Get("key2"); !ok {
		t.Error("expected cache hit for non-deleted key")
	}
}

func TestCache_Disabled(t *testing.T) {
	config := CacheConfig{
		Enabled: false,
		TTL:     1 * time.Minute,
		MaxSize: 10,
	}

	cache := NewCache(config)

	// Set a value
	cache.Set("test-key", "test-value")

	// Get should always return false when cache is disabled
	_, ok := cache.Get("test-key")
	if ok {
		t.Error("expected cache miss when cache is disabled")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	config := CacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		MaxSize: 100,
	}

	cache := NewCache(config)

	// Run concurrent set operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := "key"
				value := "value"
				cache.Set(key, value)
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should still be functional
	cache.Set("final-key", "final-value")
	if _, ok := cache.Get("final-key"); !ok {
		t.Error("cache corrupted after concurrent access")
	}
}
