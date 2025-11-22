package metrics

import (
	"mercator-hq/jupiter/pkg/config"

	"github.com/prometheus/client_golang/prometheus"
)

// CacheMetrics tracks cache performance metrics.
//
// Metrics:
//   - mercator_cache_hits_total: Total cache hits by cache name
//   - mercator_cache_misses_total: Total cache misses by cache name
//   - mercator_cache_entries: Current number of entries in cache
//   - mercator_cache_evictions_total: Total cache evictions
//
// These metrics are optional and only used if caching is implemented.
type CacheMetrics struct {
	// Cache hit counter
	hitsTotal *prometheus.CounterVec

	// Cache miss counter
	missesTotal *prometheus.CounterVec

	// Current cache size (entries)
	entries *prometheus.GaugeVec

	// Cache evictions counter
	evictionsTotal *prometheus.CounterVec
}

// NewCacheMetrics creates and registers cache metrics with the provided registry.
func NewCacheMetrics(cfg *config.MetricsConfig, registry *prometheus.Registry) *CacheMetrics {
	cm := &CacheMetrics{
		hitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache"},
		),

		missesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache"},
		),

		entries: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cache_entries",
				Help:      "Current number of entries in cache",
			},
			[]string{"cache"},
		),

		evictionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "cache_evictions_total",
				Help:      "Total number of cache evictions",
			},
			[]string{"cache"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		cm.hitsTotal,
		cm.missesTotal,
		cm.entries,
		cm.evictionsTotal,
	)

	return cm
}

// RecordHit records a cache hit.
//
// Parameters:
//   - cacheName: Name of the cache (e.g., "policy", "provider_config", "token_estimate")
//
// Example:
//
//	cm.RecordHit("policy")
func (cm *CacheMetrics) RecordHit(cacheName string) {
	cm.hitsTotal.WithLabelValues(cacheName).Inc()
}

// RecordMiss records a cache miss.
//
// Parameters:
//   - cacheName: Name of the cache
//
// Example:
//
//	cm.RecordMiss("policy")
func (cm *CacheMetrics) RecordMiss(cacheName string) {
	cm.missesTotal.WithLabelValues(cacheName).Inc()
}

// UpdateSize updates the current size of a cache.
//
// Parameters:
//   - cacheName: Name of the cache
//   - size: Current number of entries in the cache
//
// Example:
//
//	cm.UpdateSize("policy", 42)
func (cm *CacheMetrics) UpdateSize(cacheName string, size int) {
	cm.entries.WithLabelValues(cacheName).Set(float64(size))
}

// RecordEviction records a cache eviction.
//
// Parameters:
//   - cacheName: Name of the cache
//
// An eviction occurs when a cache entry is removed due to:
//   - Cache is full and needs space for new entry
//   - Entry expired (TTL)
//   - Manual invalidation
//
// Example:
//
//	cm.RecordEviction("policy")
func (cm *CacheMetrics) RecordEviction(cacheName string) {
	cm.evictionsTotal.WithLabelValues(cacheName).Inc()
}

// GetHitRate calculates the cache hit rate for a given cache.
// This is a utility function and does not directly modify metrics.
//
// Note: This function cannot be implemented here as it requires
// reading metric values, which Prometheus doesn't support directly.
// Use PromQL queries instead:
//
//	rate(mercator_cache_hits_total{cache="policy"}[5m]) /
//	(rate(mercator_cache_hits_total{cache="policy"}[5m]) +
//	 rate(mercator_cache_misses_total{cache="policy"}[5m]))
