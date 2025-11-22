package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryBackend implements Backend using in-memory storage.
// This is the default backend and provides fast access with no persistence.
// All data is lost when the process exits.
//
// MemoryBackend is thread-safe and supports concurrent access using sync.RWMutex.
type MemoryBackend struct {
	// states maps composite key (dimension:identifier) to limit state.
	states map[string]*LimitState

	// mu protects access to states map.
	mu sync.RWMutex

	// maxEntries is the maximum number of entries before eviction (LRU).
	maxEntries int

	// cleanupInterval is how often to run cleanup.
	cleanupInterval time.Duration

	// done signals the cleanup goroutine to stop.
	done chan struct{}
}

// MemoryBackendConfig configures the memory backend.
type MemoryBackendConfig struct {
	// MaxEntries is the maximum number of state entries to store.
	// Oldest entries are evicted when this limit is reached.
	// Default: 100,000
	MaxEntries int

	// CleanupInterval is how often to cleanup expired entries.
	// Default: 1 minute
	CleanupInterval time.Duration

	// RetentionPeriod is how long to keep inactive entries.
	// Entries not updated within this period are eligible for cleanup.
	// Default: 24 hours
	RetentionPeriod time.Duration
}

// NewMemoryBackend creates a new in-memory storage backend with default settings.
func NewMemoryBackend() *MemoryBackend {
	return NewMemoryBackendWithConfig(MemoryBackendConfig{
		MaxEntries:      100000,
		CleanupInterval: time.Minute,
		RetentionPeriod: 24 * time.Hour,
	})
}

// NewMemoryBackendWithConfig creates a new in-memory backend with custom configuration.
func NewMemoryBackendWithConfig(cfg MemoryBackendConfig) *MemoryBackend {
	// Apply defaults
	if cfg.MaxEntries == 0 {
		cfg.MaxEntries = 100000
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = time.Minute
	}
	if cfg.RetentionPeriod == 0 {
		cfg.RetentionPeriod = 24 * time.Hour
	}

	backend := &MemoryBackend{
		states:          make(map[string]*LimitState),
		maxEntries:      cfg.MaxEntries,
		cleanupInterval: cfg.CleanupInterval,
		done:            make(chan struct{}),
	}

	// Start background cleanup goroutine
	go backend.cleanupLoop(cfg.RetentionPeriod)

	return backend
}

// Save persists the limit state for an identifier.
func (m *MemoryBackend) Save(ctx context.Context, state *LimitState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if state.Dimension == "" {
		return fmt.Errorf("dimension cannot be empty")
	}

	// Create composite key
	key := m.makeKey(state.Identifier, state.Dimension)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we need to evict entries
	if len(m.states) >= m.maxEntries {
		m.evictOldestLocked()
	}

	// Update timestamps
	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	// Only update LastUpdated if it's zero (not explicitly set by caller)
	if state.LastUpdated.IsZero() {
		state.LastUpdated = now
	}

	// Store state
	m.states[key] = state

	return nil
}

// Load retrieves the limit state for an identifier and dimension.
func (m *MemoryBackend) Load(ctx context.Context, identifier string, dimension string) (*LimitState, error) {
	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}
	if dimension == "" {
		return nil, fmt.Errorf("dimension cannot be empty")
	}

	key := m.makeKey(identifier, dimension)

	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[key]
	if !exists {
		return nil, nil
	}

	return state, nil
}

// Delete removes the limit state for an identifier and dimension.
func (m *MemoryBackend) Delete(ctx context.Context, identifier string, dimension string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if dimension == "" {
		return fmt.Errorf("dimension cannot be empty")
	}

	key := m.makeKey(identifier, dimension)

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, key)
	return nil
}

// List returns all limit states for a dimension.
func (m *MemoryBackend) List(ctx context.Context, dimension string) ([]*LimitState, error) {
	if dimension == "" {
		return nil, fmt.Errorf("dimension cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var states []*LimitState
	for _, state := range m.states {
		if state.Dimension == dimension {
			states = append(states, state)
		}
	}

	return states, nil
}

// Cleanup removes expired state entries based on retention policy.
func (m *MemoryBackend) Cleanup(ctx context.Context, olderThan time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := 0
	for key, state := range m.states {
		if state.LastUpdated.Before(olderThan) {
			delete(m.states, key)
			deleted++
		}
	}

	return deleted, nil
}

// Close releases any resources held by the backend.
func (m *MemoryBackend) Close() error {
	// Signal cleanup goroutine to stop
	close(m.done)
	return nil
}

// Size returns the current number of stored states.
// This is useful for monitoring and testing.
func (m *MemoryBackend) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.states)
}

// makeKey creates a composite key from identifier and dimension.
func (m *MemoryBackend) makeKey(identifier string, dimension string) string {
	return fmt.Sprintf("%s:%s", dimension, identifier)
}

// evictOldestLocked evicts the oldest entry to make room for new entries.
// Caller must hold write lock.
func (m *MemoryBackend) evictOldestLocked() {
	var (
		oldestKey   string
		oldestTime  time.Time
		foundOldest bool
	)

	// Find oldest entry
	for key, state := range m.states {
		if !foundOldest || state.LastUpdated.Before(oldestTime) {
			oldestKey = key
			oldestTime = state.LastUpdated
			foundOldest = true
		}
	}

	// Evict oldest
	if foundOldest {
		delete(m.states, oldestKey)
	}
}

// cleanupLoop runs periodic cleanup of expired entries.
func (m *MemoryBackend) cleanupLoop(retentionPeriod time.Duration) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-retentionPeriod)
			_, _ = m.Cleanup(context.Background(), cutoff)
		case <-m.done:
			return
		}
	}
}
