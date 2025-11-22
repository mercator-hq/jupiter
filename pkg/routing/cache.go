package routing

import (
	"sync"
	"time"
)

// StickyCache implements a thread-safe cache for sticky routing with TTL and LRU eviction.
// It maps keys (user/session/API key) to provider names and automatically expires
// entries after the configured TTL. When the cache reaches max capacity, it evicts
// the least recently accessed entry (LRU).
type StickyCache struct {
	// entries maps cache keys to sticky entries
	entries map[string]*StickyEntry

	// ttl is the time-to-live for cache entries (0 = no expiry)
	ttl time.Duration

	// maxEntries is the maximum number of entries (0 = unlimited)
	maxEntries int

	// mu protects concurrent access to the cache
	mu sync.RWMutex

	// stopCh signals the cleanup goroutine to stop
	stopCh chan struct{}

	// cleanupInterval is how often to run expiry cleanup
	cleanupInterval time.Duration
}

// NewStickyCache creates a new sticky cache with the specified TTL and max entries.
// If ttl is 0, entries never expire.
// If maxEntries is 0, the cache has unlimited size.
// cleanup Interval defaults to ttl/2, or 1 minute if TTL is 0.
func NewStickyCache(ttl time.Duration, maxEntries int) *StickyCache {
	cleanupInterval := time.Minute
	if ttl > 0 {
		cleanupInterval = ttl / 2
		if cleanupInterval < 10*time.Second {
			cleanupInterval = 10 * time.Second
		}
	}

	cache := &StickyCache{
		entries:         make(map[string]*StickyEntry),
		ttl:             ttl,
		maxEntries:      maxEntries,
		stopCh:          make(chan struct{}),
		cleanupInterval: cleanupInterval,
	}

	// Start background cleanup goroutine if TTL is configured
	if ttl > 0 {
		go cache.cleanupExpired()
	}

	return cache
}

// Get retrieves a provider name from the cache.
// Returns (provider, true) if found and not expired.
// Returns ("", false) if not found or expired.
func (c *StickyCache) Get(key string) (string, bool) {
	// First check with read lock
	c.mu.RLock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.RUnlock()
		return "", false
	}

	// Check if entry has expired
	if c.ttl > 0 && time.Now().After(entry.ExpiresAt) {
		c.mu.RUnlock()
		return "", false
	}
	providerName := entry.ProviderName
	c.mu.RUnlock()

	// Update access time and count with write lock
	c.mu.Lock()
	// Re-check entry exists (might have been deleted between locks)
	if entry, ok := c.entries[key]; ok {
		entry.LastAccessedAt = time.Now()
		entry.AccessCount++
	}
	c.mu.Unlock()

	return providerName, true
}

// Set stores a provider name in the cache with the configured TTL.
// If the cache is full, it evicts the least recently used entry.
func (c *StickyCache) Set(key string, providerName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict an entry
	if c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		// Only evict if the key doesn't already exist
		if _, exists := c.entries[key]; !exists {
			c.evictLRU()
		}
	}

	now := time.Now()
	expiresAt := time.Time{} // Zero time = no expiry
	if c.ttl > 0 {
		expiresAt = now.Add(c.ttl)
	}

	c.entries[key] = &StickyEntry{
		ProviderName:   providerName,
		ExpiresAt:      expiresAt,
		CreatedAt:      now,
		LastAccessedAt: now,
		AccessCount:    1,
	}
}

// Delete removes an entry from the cache.
func (c *StickyCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Size returns the current number of entries in the cache.
func (c *StickyCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Clear removes all entries from the cache.
func (c *StickyCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*StickyEntry)
}

// Close stops the background cleanup goroutine.
// After calling Close, the cache should not be used.
func (c *StickyCache) Close() {
	close(c.stopCh)
}

// evictLRU evicts the least recently used entry from the cache.
// Must be called with write lock held.
func (c *StickyCache) evictLRU() {
	if len(c.entries) == 0 {
		return
	}

	// Find the least recently accessed entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.LastAccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccessedAt
		}
	}

	// Evict the oldest entry
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// cleanupExpired runs periodically to remove expired entries.
// Runs in a background goroutine until Close() is called.
func (c *StickyCache) cleanupExpired() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopCh:
			return
		}
	}
}

// removeExpired removes all expired entries from the cache.
func (c *StickyCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl == 0 {
		return // No expiry configured
	}

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}
