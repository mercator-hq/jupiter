package routing

import (
	"sync"
	"sync/atomic"
	"time"
)

// AtomicRoutingStats implements thread-safe routing statistics using atomic operations.
// All counters are updated atomically for lock-free performance.
type AtomicRoutingStats struct {
	// totalRequests is the total number of routing requests processed
	totalRequests atomic.Int64

	// requestsPerProvider tracks requests routed to each provider
	// Uses sync.Map for thread-safe concurrent access
	requestsPerProvider sync.Map // map[string]*atomic.Int64

	// strategyUseCount tracks how many times each strategy was used
	strategyUseCount sync.Map // map[string]*atomic.Int64

	// healthFilteredCount is the number of requests where unhealthy providers were filtered
	healthFilteredCount atomic.Int64

	// manualOverrideCount is the number of manual provider selections
	manualOverrideCount atomic.Int64

	// policyOverrideCount is the number of policy-driven routing decisions
	policyOverrideCount atomic.Int64

	// errors is the total number of routing errors
	errors atomic.Int64

	// lastResetTime is when statistics were last reset
	lastResetTime time.Time

	// mu protects lastResetTime
	mu sync.RWMutex
}

// NewAtomicRoutingStats creates a new atomic routing statistics tracker.
func NewAtomicRoutingStats() *AtomicRoutingStats {
	return &AtomicRoutingStats{
		lastResetTime: time.Now(),
	}
}

// IncrementTotal increments the total request counter.
func (s *AtomicRoutingStats) IncrementTotal() {
	s.totalRequests.Add(1)
}

// IncrementProvider increments the counter for a specific provider.
func (s *AtomicRoutingStats) IncrementProvider(providerName string) {
	// Get or create counter for this provider
	val, _ := s.requestsPerProvider.LoadOrStore(providerName, &atomic.Int64{})
	counter := val.(*atomic.Int64)
	counter.Add(1)
}

// IncrementStrategy increments the counter for a specific strategy.
func (s *AtomicRoutingStats) IncrementStrategy(strategyName string) {
	// Get or create counter for this strategy
	val, _ := s.strategyUseCount.LoadOrStore(strategyName, &atomic.Int64{})
	counter := val.(*atomic.Int64)
	counter.Add(1)
}

// IncrementHealthFiltered increments the health filtered counter.
func (s *AtomicRoutingStats) IncrementHealthFiltered() {
	s.healthFilteredCount.Add(1)
}

// IncrementManualOverride increments the manual override counter.
func (s *AtomicRoutingStats) IncrementManualOverride() {
	s.manualOverrideCount.Add(1)
}

// IncrementPolicyOverride increments the policy override counter.
func (s *AtomicRoutingStats) IncrementPolicyOverride() {
	s.policyOverrideCount.Add(1)
}

// IncrementErrors increments the error counter.
func (s *AtomicRoutingStats) IncrementErrors() {
	s.errors.Add(1)
}

// Snapshot returns a point-in-time snapshot of the statistics.
// The returned RoutingStats struct is safe to read without locks.
func (s *AtomicRoutingStats) Snapshot() *RoutingStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build provider request map
	providerRequests := make(map[string]int64)
	s.requestsPerProvider.Range(func(key, value interface{}) bool {
		providerRequests[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	// Build strategy use count map
	strategyUse := make(map[string]int64)
	s.strategyUseCount.Range(func(key, value interface{}) bool {
		strategyUse[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	return &RoutingStats{
		TotalRequests:       s.totalRequests.Load(),
		RequestsPerProvider: providerRequests,
		StrategyUseCount:    strategyUse,
		HealthFilteredCount: s.healthFilteredCount.Load(),
		ManualOverrideCount: s.manualOverrideCount.Load(),
		PolicyOverrideCount: s.policyOverrideCount.Load(),
		Errors:              s.errors.Load(),
		LastResetTime:       s.lastResetTime,
	}
}

// Reset resets all statistics to zero.
func (s *AtomicRoutingStats) Reset() {
	s.totalRequests.Store(0)
	s.healthFilteredCount.Store(0)
	s.manualOverrideCount.Store(0)
	s.policyOverrideCount.Store(0)
	s.errors.Store(0)

	// Clear all provider counters
	s.requestsPerProvider.Range(func(key, value interface{}) bool {
		s.requestsPerProvider.Delete(key)
		return true
	})

	// Clear all strategy counters
	s.strategyUseCount.Range(func(key, value interface{}) bool {
		s.strategyUseCount.Delete(key)
		return true
	})

	s.mu.Lock()
	s.lastResetTime = time.Now()
	s.mu.Unlock()
}
