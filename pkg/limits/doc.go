// Package limits provides budget tracking and rate limiting for LLM requests.
//
// # Overview
//
// The limits package implements multi-dimensional budget tracking and rate limiting
// to prevent cost overruns and enforce usage quotas. It supports:
//
//   - Budget tracking (per-API key, per-user, per-team)
//   - Rate limiting (request-based, token-based, concurrent)
//   - Rolling time windows (hourly, daily, monthly)
//   - Enforcement actions (block, queue, downgrade, alert)
//
// # Architecture
//
// The package is organized into sub-packages:
//
//   - ratelimit: Token bucket and sliding window rate limiters
//   - budget: Rolling window budget tracking
//   - storage: Persistence backends (memory, SQLite, PostgreSQL)
//   - enforcement: Enforcement action execution
//
// # Usage
//
//	// Initialize manager
//	cfg := config.GetConfig()
//	manager := limits.NewManager(&cfg.Limits)
//
//	// Check limits before request
//	result, err := manager.CheckLimits(ctx, "api-key-123", enriched)
//	if !result.Allowed {
//	    return fmt.Errorf("limit exceeded: %s", result.Reason)
//	}
//
//	// Record usage after request
//	err = manager.RecordUsage(ctx, "api-key-123", usage)
//
// # Performance
//
// The limits package is designed for high throughput:
//
//   - <1ms p99 latency for limit checks
//   - <500µs p99 latency for usage recording
//   - Support for 10,000+ concurrent API keys
//   - Memory-efficient sliding windows
//
// # Thread Safety
//
// All operations are thread-safe and use fine-grained locking to minimize
// contention. The rate limiter and budget tracker can be accessed concurrently
// from multiple goroutines.
package limits
