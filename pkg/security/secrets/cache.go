package secrets

import (
	"sync"
	"time"
)

// CacheConfig configures the secret cache behavior.
type CacheConfig struct {
	Enabled bool          // Enable caching
	TTL     time.Duration // Time to live for cached secrets
	MaxSize int           // Maximum number of secrets to cache
}

// cacheEntry represents a cached secret with expiration.
type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// Cache provides thread-safe caching of secrets with TTL and size limits.
//
// Secrets are cached in memory to reduce backend calls. The cache uses
// a simple LRU eviction policy when the maximum size is reached.
type Cache struct {
	config  CacheConfig
	entries map[string]*cacheEntry
	mu      sync.RWMutex
}

// NewCache creates a new secret cache with the given configuration.
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		config:  config,
		entries: make(map[string]*cacheEntry),
	}
}

// Get retrieves a secret from the cache.
//
// Returns (value, true) if the secret exists and has not expired.
// Returns ("", false) if the secret is not cached or has expired.
func (c *Cache) Get(key string) (string, bool) {
	if !c.config.Enabled {
		return "", false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.value, true
}

// Set stores a secret in the cache with TTL.
//
// If the cache is full (MaxSize reached), the oldest entry is evicted
// to make room for the new entry (simple LRU policy).
func (c *Cache) Set(key, value string) {
	if !c.config.Enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce max size with simple LRU eviction
	if len(c.entries) >= c.config.MaxSize {
		// Find and remove oldest entry
		var oldestKey string
		var oldestTime time.Time
		first := true

		for k, e := range c.entries {
			if first || e.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = e.expiresAt
				first = false
			}
		}

		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.config.TTL),
	}
}

// Clear removes all entries from the cache.
//
// This is typically called when secrets need to be refreshed
// or when the cache needs to be invalidated.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// Delete removes a specific entry from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Size returns the current number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
