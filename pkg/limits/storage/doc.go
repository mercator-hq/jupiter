// Package storage provides persistence backends for limit state.
//
// # Overview
//
// The storage package defines the interface for persisting limit state
// (rate limit counters, budget usage) and provides multiple implementations:
//
//   - Memory: Fast in-memory storage (default, no persistence)
//   - SQLite: Lightweight file-based persistence with snapshots
//   - PostgreSQL: Production-grade persistence for distributed deployments
//
// # Usage
//
//	// Create in-memory backend (default)
//	backend := storage.NewMemoryBackend()
//
//	// Save state
//	state := &storage.LimitState{
//	    Identifier: "api-key-123",
//	    Dimension:  "api_key",
//	    RateLimit:  rateLimitData,
//	    Budget:     budgetData,
//	}
//	err := backend.Save(ctx, state)
//
//	// Load state
//	state, err := backend.Load(ctx, "api-key-123", "api_key")
//
// # Thread Safety
//
// All storage backends are thread-safe and support concurrent access
// from multiple goroutines. Locking is handled internally by each backend.
package storage
