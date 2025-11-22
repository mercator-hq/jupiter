package routing

import (
	"sync"
	"testing"
	"time"
)

func TestNewStickyCache(t *testing.T) {
	tests := []struct {
		name       string
		ttl        time.Duration
		maxEntries int
	}{
		{
			name:       "with TTL and max entries",
			ttl:        time.Hour,
			maxEntries: 100,
		},
		{
			name:       "with zero TTL (no expiry)",
			ttl:        0,
			maxEntries: 100,
		},
		{
			name:       "with zero max entries (unlimited)",
			ttl:        time.Hour,
			maxEntries: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewStickyCache(tt.ttl, tt.maxEntries)
			defer cache.Close()

			if cache == nil {
				t.Fatal("NewStickyCache() returned nil")
			}
			if cache.ttl != tt.ttl {
				t.Errorf("cache.ttl = %v, want %v", cache.ttl, tt.ttl)
			}
			if cache.maxEntries != tt.maxEntries {
				t.Errorf("cache.maxEntries = %d, want %d", cache.maxEntries, tt.maxEntries)
			}
		})
	}
}

func TestStickyCache_SetAndGet(t *testing.T) {
	cache := NewStickyCache(time.Hour, 100)
	defer cache.Close()

	// Set a value
	cache.Set("user-123", "openai")

	// Get the value
	provider, ok := cache.Get("user-123")
	if !ok {
		t.Error("Get() returned false for existing key")
	}
	if provider != "openai" {
		t.Errorf("Get() = %s, want openai", provider)
	}

	// Get non-existent key
	_, ok = cache.Get("user-nonexistent")
	if ok {
		t.Error("Get() returned true for non-existent key")
	}
}

func TestStickyCache_Expiry(t *testing.T) {
	// Use a short TTL for testing
	cache := NewStickyCache(100*time.Millisecond, 100)
	defer cache.Close()

	// Set a value
	cache.Set("user-123", "openai")

	// Immediately get should succeed
	provider, ok := cache.Get("user-123")
	if !ok || provider != "openai" {
		t.Error("Get() failed immediately after Set()")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Get should fail (expired)
	_, ok = cache.Get("user-123")
	if ok {
		t.Error("Get() returned true for expired key")
	}
}

func TestStickyCache_NoExpiry(t *testing.T) {
	// TTL = 0 means no expiry
	cache := NewStickyCache(0, 100)
	defer cache.Close()

	cache.Set("user-123", "openai")

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Should still be there
	provider, ok := cache.Get("user-123")
	if !ok || provider != "openai" {
		t.Error("Get() failed for non-expiring cache")
	}
}

func TestStickyCache_LRUEviction(t *testing.T) {
	// Small cache to test eviction
	cache := NewStickyCache(time.Hour, 3)
	defer cache.Close()

	// Fill cache
	cache.Set("user-1", "provider-1")
	cache.Set("user-2", "provider-2")
	cache.Set("user-3", "provider-3")

	// Access user-1 to make it recently used
	cache.Get("user-1")

	// Sleep a bit to ensure different access times
	time.Sleep(10 * time.Millisecond)

	// Access user-2
	cache.Get("user-2")

	// Add one more entry - should evict user-3 (least recently used)
	cache.Set("user-4", "provider-4")

	// user-1 and user-2 should still be there
	if _, ok := cache.Get("user-1"); !ok {
		t.Error("user-1 was evicted but should have been kept")
	}
	if _, ok := cache.Get("user-2"); !ok {
		t.Error("user-2 was evicted but should have been kept")
	}

	// user-3 should be evicted
	if _, ok := cache.Get("user-3"); ok {
		t.Error("user-3 should have been evicted")
	}

	// user-4 should be there
	if _, ok := cache.Get("user-4"); !ok {
		t.Error("user-4 should be in cache")
	}
}

func TestStickyCache_UpdateExisting(t *testing.T) {
	cache := NewStickyCache(time.Hour, 10)
	defer cache.Close()

	// Set initial value
	cache.Set("user-123", "openai")

	// Update to different provider
	cache.Set("user-123", "anthropic")

	// Should get updated value
	provider, ok := cache.Get("user-123")
	if !ok {
		t.Error("Get() failed for updated key")
	}
	if provider != "anthropic" {
		t.Errorf("Get() = %s, want anthropic", provider)
	}
}

func TestStickyCache_Delete(t *testing.T) {
	cache := NewStickyCache(time.Hour, 100)
	defer cache.Close()

	cache.Set("user-123", "openai")

	// Verify it exists
	if _, ok := cache.Get("user-123"); !ok {
		t.Error("Get() failed before Delete()")
	}

	// Delete it
	cache.Delete("user-123")

	// Should no longer exist
	if _, ok := cache.Get("user-123"); ok {
		t.Error("Get() succeeded after Delete()")
	}
}

func TestStickyCache_Size(t *testing.T) {
	cache := NewStickyCache(time.Hour, 100)
	defer cache.Close()

	if cache.Size() != 0 {
		t.Errorf("Size() = %d, want 0 for empty cache", cache.Size())
	}

	cache.Set("user-1", "provider-1")
	if cache.Size() != 1 {
		t.Errorf("Size() = %d, want 1", cache.Size())
	}

	cache.Set("user-2", "provider-2")
	if cache.Size() != 2 {
		t.Errorf("Size() = %d, want 2", cache.Size())
	}

	cache.Delete("user-1")
	if cache.Size() != 1 {
		t.Errorf("Size() = %d, want 1 after Delete()", cache.Size())
	}
}

func TestStickyCache_Clear(t *testing.T) {
	cache := NewStickyCache(time.Hour, 100)
	defer cache.Close()

	// Add some entries
	cache.Set("user-1", "provider-1")
	cache.Set("user-2", "provider-2")
	cache.Set("user-3", "provider-3")

	if cache.Size() != 3 {
		t.Errorf("Size() = %d, want 3 before Clear()", cache.Size())
	}

	// Clear cache
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Size() = %d, want 0 after Clear()", cache.Size())
	}

	// Entries should be gone
	if _, ok := cache.Get("user-1"); ok {
		t.Error("Get() succeeded after Clear()")
	}
}

func TestStickyCache_AccessCount(t *testing.T) {
	cache := NewStickyCache(time.Hour, 100)
	defer cache.Close()

	cache.Set("user-123", "openai")

	// Access multiple times
	for i := 0; i < 5; i++ {
		cache.Get("user-123")
	}

	cache.mu.RLock()
	entry := cache.entries["user-123"]
	cache.mu.RUnlock()

	// Initial Set counts as 1, plus 5 Gets = 6 total
	if entry.AccessCount != 6 {
		t.Errorf("AccessCount = %d, want 6", entry.AccessCount)
	}
}

func TestStickyCache_ConcurrentAccess(t *testing.T) {
	cache := NewStickyCache(time.Hour, 1000)
	defer cache.Close()

	concurrency := 100
	opsPerGoroutine := 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := string(rune('A' + (id%26)))
				cache.Set(key, "provider")
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := string(rune('A' + (id%26)))
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be functional
	cache.Set("test", "provider")
	if _, ok := cache.Get("test"); !ok {
		t.Error("Cache broken after concurrent access")
	}
}

func TestStickyCache_RemoveExpired(t *testing.T) {
	// Use short TTL for testing
	cache := NewStickyCache(50*time.Millisecond, 100)
	defer cache.Close()

	// Add entries
	cache.Set("user-1", "provider-1")
	cache.Set("user-2", "provider-2")

	if cache.Size() != 2 {
		t.Errorf("Size() = %d, want 2", cache.Size())
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup (testing the cleanup method directly)
	cache.removeExpired()

	// Entries should be removed
	if cache.Size() != 0 {
		t.Errorf("Size() = %d after removeExpired(), want 0", cache.Size())
	}
}

func TestStickyCache_NoEvictionWhenUpdating(t *testing.T) {
	cache := NewStickyCache(time.Hour, 2)
	defer cache.Close()

	// Fill cache
	cache.Set("user-1", "provider-1")
	cache.Set("user-2", "provider-2")

	// Update existing key should not trigger eviction
	cache.Set("user-1", "provider-updated")

	// Both should still be there
	if _, ok := cache.Get("user-1"); !ok {
		t.Error("user-1 should still be in cache")
	}
	if _, ok := cache.Get("user-2"); !ok {
		t.Error("user-2 should still be in cache")
	}

	if cache.Size() != 2 {
		t.Errorf("Size() = %d, want 2", cache.Size())
	}
}
