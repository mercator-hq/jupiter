package metrics

import (
	"fmt"
	"sync"
	"time"

	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector is the main orchestrator for all Prometheus metrics in Mercator Jupiter.
// It manages metric registration, collection, and provides a unified interface
// for recording metrics across all components.
//
// The collector is designed for high-performance with minimal overhead (<50µs per update):
//   - Pre-allocated metric instances
//   - Lock-free counters where possible
//   - Cardinality limits to prevent memory issues
//   - Custom histogram buckets optimized for LLM workloads
type Collector struct {
	config   *config.MetricsConfig
	registry *prometheus.Registry

	// Request metrics
	requestMetrics *RequestMetrics

	// Provider metrics
	providerMetrics *ProviderMetrics

	// Policy metrics
	policyMetrics *PolicyMetrics

	// Cost metrics
	costMetrics *CostMetrics

	// Cache metrics (optional, if caching is implemented)
	cacheMetrics *CacheMetrics

	// Cardinality tracking
	cardinalityLimiter *CardinalityLimiter
}

// NewCollector creates a new metrics collector with the specified configuration
// and Prometheus registry. If registry is nil, the default Prometheus registry
// is used.
//
// Example:
//
//	cfg := &config.MetricsConfig{
//		Enabled:    true,
//		Namespace:  "mercator",
//		Subsystem:  "jupiter",
//	}
//	collector := metrics.NewCollector(cfg, nil)
func NewCollector(cfg *config.MetricsConfig, registry *prometheus.Registry) *Collector {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	// Set defaults if not specified
	if cfg.Namespace == "" {
		cfg.Namespace = "mercator"
	}
	if cfg.Subsystem == "" {
		cfg.Subsystem = "jupiter"
	}
	if len(cfg.RequestDurationBuckets) == 0 {
		// Optimized for LLM request latencies (100ms - 30s)
		cfg.RequestDurationBuckets = []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0}
	}
	if len(cfg.TokenCountBuckets) == 0 {
		// Optimized for token counts (100 - 100K tokens)
		cfg.TokenCountBuckets = []float64{100, 500, 1000, 5000, 10000, 50000, 100000}
	}

	c := &Collector{
		config:             cfg,
		registry:           registry,
		cardinalityLimiter: NewCardinalityLimiter(10000), // Max 10K unique label sets
	}

	// Initialize metric subsystems
	c.requestMetrics = NewRequestMetrics(cfg, registry)
	c.providerMetrics = NewProviderMetrics(cfg, registry)
	c.policyMetrics = NewPolicyMetrics(cfg, registry)
	c.costMetrics = NewCostMetrics(cfg, registry)
	c.cacheMetrics = NewCacheMetrics(cfg, registry)

	return c
}

// RecordRequest records metrics for a completed request.
//
// Parameters:
//   - provider: LLM provider name (e.g., "openai", "anthropic")
//   - model: Model name (e.g., "gpt-4", "claude-3-opus")
//   - status: Request status ("success", "error", "blocked")
//   - duration: Total request duration
//   - tokens: Total token count (prompt + completion)
//   - cost: Total request cost in USD
//
// Example:
//
//	collector.RecordRequest(
//		"openai",
//		"gpt-4",
//		"success",
//		1200*time.Millisecond,
//		1500,
//		0.05,
//	)
func (c *Collector) RecordRequest(provider, model, status string, duration time.Duration, tokens int, cost float64) {
	if !c.config.Enabled {
		return
	}

	// Check cardinality limit
	labelSet := fmt.Sprintf("request:%s:%s:%s", provider, model, status)
	if !c.cardinalityLimiter.Allow(labelSet) {
		// Aggregate into "other" to prevent cardinality explosion
		model = "other"
	}

	c.requestMetrics.RecordRequest(provider, model, status, duration, tokens)
	c.costMetrics.RecordRequestCost(provider, model, cost)
}

// RecordProviderLatency records the latency for a provider API call.
//
// Parameters:
//   - provider: LLM provider name
//   - model: Model name
//   - latency: API call latency in seconds
func (c *Collector) RecordProviderLatency(provider, model string, latency float64) {
	if !c.config.Enabled {
		return
	}

	c.providerMetrics.RecordLatency(provider, model, latency)
}

// UpdateProviderHealth updates the health status of a provider.
//
// Parameters:
//   - provider: LLM provider name
//   - healthy: true if provider is healthy, false otherwise
//
// The health metric is a gauge where 1=healthy, 0=unhealthy.
func (c *Collector) UpdateProviderHealth(provider string, healthy bool) {
	if !c.config.Enabled {
		return
	}

	c.providerMetrics.UpdateHealth(provider, healthy)
}

// RecordProviderError records an error from a provider.
//
// Parameters:
//   - provider: LLM provider name
//   - errorType: Type of error (e.g., "rate_limit", "timeout", "auth", "server_error")
func (c *Collector) RecordProviderError(provider, errorType string) {
	if !c.config.Enabled {
		return
	}

	c.providerMetrics.RecordError(provider, errorType)
}

// RecordPolicyEvaluation records metrics for a policy evaluation.
//
// Parameters:
//   - ruleID: Policy rule identifier
//   - action: Policy action taken ("allow", "deny", "modify")
//   - duration: Evaluation duration
//
// Example:
//
//	collector.RecordPolicyEvaluation(
//		"cost-limit-daily",
//		"allow",
//		2*time.Millisecond,
//	)
func (c *Collector) RecordPolicyEvaluation(ruleID, action string, duration time.Duration) {
	if !c.config.Enabled {
		return
	}

	c.policyMetrics.RecordEvaluation(ruleID, action, duration)
}

// RecordPolicyHit records when a policy rule matched and took action.
//
// Parameters:
//   - ruleID: Policy rule identifier
func (c *Collector) RecordPolicyHit(ruleID string) {
	if !c.config.Enabled {
		return
	}

	c.policyMetrics.RecordHit(ruleID)
}

// RecordPolicyMiss records when a policy rule did not match.
//
// Parameters:
//   - ruleID: Policy rule identifier
func (c *Collector) RecordPolicyMiss(ruleID string) {
	if !c.config.Enabled {
		return
	}

	c.policyMetrics.RecordMiss(ruleID)
}

// RecordCacheHit records a cache hit.
//
// Parameters:
//   - cacheName: Name of the cache (e.g., "policy", "provider_config")
func (c *Collector) RecordCacheHit(cacheName string) {
	if !c.config.Enabled {
		return
	}

	c.cacheMetrics.RecordHit(cacheName)
}

// RecordCacheMiss records a cache miss.
//
// Parameters:
//   - cacheName: Name of the cache
func (c *Collector) RecordCacheMiss(cacheName string) {
	if !c.config.Enabled {
		return
	}

	c.cacheMetrics.RecordMiss(cacheName)
}

// UpdateCacheSize updates the current size of a cache.
//
// Parameters:
//   - cacheName: Name of the cache
//   - size: Current number of entries in the cache
func (c *Collector) UpdateCacheSize(cacheName string, size int) {
	if !c.config.Enabled {
		return
	}

	c.cacheMetrics.UpdateSize(cacheName, size)
}

// Registry returns the Prometheus registry used by this collector.
// This can be used to create an HTTP handler for the /metrics endpoint:
//
//	http.Handle("/metrics", promhttp.HandlerFor(
//		collector.Registry(),
//		promhttp.HandlerOpts{},
//	))
func (c *Collector) Registry() *prometheus.Registry {
	return c.registry
}

// CardinalityLimiter prevents metric cardinality explosion by limiting
// the number of unique label combinations per metric.
type CardinalityLimiter struct {
	maxCardinality int
	current        map[string]struct{}
	mu             sync.RWMutex
}

// NewCardinalityLimiter creates a new cardinality limiter with the specified
// maximum cardinality.
func NewCardinalityLimiter(maxCardinality int) *CardinalityLimiter {
	return &CardinalityLimiter{
		maxCardinality: maxCardinality,
		current:        make(map[string]struct{}),
	}
}

// Allow checks if a label set is allowed. Returns true if the label set
// already exists or if we haven't reached the cardinality limit yet.
// Returns false if adding this label set would exceed the limit.
func (cl *CardinalityLimiter) Allow(labelSet string) bool {
	cl.mu.RLock()
	if _, exists := cl.current[labelSet]; exists {
		cl.mu.RUnlock()
		return true
	}
	cl.mu.RUnlock()

	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Double-check after acquiring write lock
	if _, exists := cl.current[labelSet]; exists {
		return true
	}

	if len(cl.current) >= cl.maxCardinality {
		return false
	}

	cl.current[labelSet] = struct{}{}
	return true
}

// Count returns the current cardinality.
func (cl *CardinalityLimiter) Count() int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return len(cl.current)
}
