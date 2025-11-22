package ratelimit

import (
	"sync"
	"time"
)

// SlidingWindow implements a sliding window counter for rate limiting.
//
// The sliding window tracks requests/tokens over a rolling time period.
// Old entries outside the window are automatically pruned. This provides
// accurate rate limiting without the "reset spike" problem of fixed windows.
//
// # Algorithm
//
//  1. Add value to current bucket
//  2. Prune buckets older than window duration
//  3. Sum all remaining buckets to get current usage
//
// # Memory Efficiency
//
// Uses a circular buffer with fixed granularity to limit memory usage.
// For example, a 1-minute window with 1-second buckets uses 60 buckets.
//
// # Thread Safety
//
// SlidingWindow is thread-safe using sync.RWMutex. Reads are lock-free
// when possible to maximize throughput under read-heavy workloads.
type SlidingWindow struct {
	window     time.Duration  // Window duration (e.g., 1 minute)
	bucketSize time.Duration  // Granularity of each bucket (e.g., 1 second)
	buckets    []bucket       // Circular buffer of buckets
	head       int            // Current write position
	mu         sync.RWMutex
}

// bucket represents a single time-stamped counter bucket.
type bucket struct {
	timestamp time.Time
	value     int64
}

// NewSlidingWindow creates a new sliding window counter.
//
// Parameters:
//   - window: Time window duration (e.g., 1 minute, 1 hour)
//   - bucketSize: Granularity of buckets (e.g., 1 second, 1 minute)
//
// The number of buckets is window/bucketSize. Smaller bucket sizes provide
// more accuracy but use more memory.
//
// Example:
//
//	// 1-minute window with 1-second buckets (60 buckets)
//	sw := NewSlidingWindow(time.Minute, time.Second)
//
//	// 1-hour window with 1-minute buckets (60 buckets)
//	sw := NewSlidingWindow(time.Hour, time.Minute)
func NewSlidingWindow(window time.Duration, bucketSize time.Duration) *SlidingWindow {
	numBuckets := int(window / bucketSize)
	if numBuckets == 0 {
		numBuckets = 1
	}

	return &SlidingWindow{
		window:     window,
		bucketSize: bucketSize,
		buckets:    make([]bucket, numBuckets),
		head:       0,
	}
}

// Add increments the counter by the given value.
// The value is added to the current time bucket.
func (sw *SlidingWindow) Add(value int64) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()

	// Prune old buckets
	sw.pruneLocked(now)

	// Find or create current bucket
	currentBucket := sw.findOrCreateBucketLocked(now)
	currentBucket.value += value
}

// Sum returns the total count across all buckets in the window.
// This automatically prunes expired buckets before summing.
func (sw *SlidingWindow) Sum() int64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()

	// Prune old buckets
	sw.pruneLocked(now)

	// Sum all buckets
	var sum int64
	for i := 0; i < len(sw.buckets); i++ {
		if !sw.buckets[i].timestamp.IsZero() {
			sum += sw.buckets[i].value
		}
	}

	return sum
}

// Reset clears all buckets.
func (sw *SlidingWindow) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	for i := 0; i < len(sw.buckets); i++ {
		sw.buckets[i] = bucket{}
	}
	sw.head = 0
}

// pruneLocked removes buckets older than the window.
// Caller must hold write lock.
func (sw *SlidingWindow) pruneLocked(now time.Time) {
	cutoff := now.Add(-sw.window)

	for i := 0; i < len(sw.buckets); i++ {
		if !sw.buckets[i].timestamp.IsZero() && sw.buckets[i].timestamp.Before(cutoff) {
			sw.buckets[i] = bucket{} // Clear expired bucket
		}
	}
}

// findOrCreateBucketLocked finds the bucket for the current time or creates a new one.
// Caller must hold write lock.
func (sw *SlidingWindow) findOrCreateBucketLocked(now time.Time) *bucket {
	// Round timestamp to bucket boundary
	bucketTime := now.Truncate(sw.bucketSize)

	// Check if current head bucket matches this time
	if sw.buckets[sw.head].timestamp.Equal(bucketTime) {
		return &sw.buckets[sw.head]
	}

	// Search for existing bucket with this timestamp
	for i := 0; i < len(sw.buckets); i++ {
		if sw.buckets[i].timestamp.Equal(bucketTime) {
			return &sw.buckets[i]
		}
	}

	// No existing bucket found, create new one
	// Find next available slot (prefer empty slots, then oldest)
	targetIdx := -1

	// First, try to find an empty slot
	for i := 0; i < len(sw.buckets); i++ {
		if sw.buckets[i].timestamp.IsZero() {
			targetIdx = i
			break
		}
	}

	// If no empty slot, find oldest bucket
	if targetIdx == -1 {
		oldestIdx := 0
		oldestTime := sw.buckets[0].timestamp

		for i := 1; i < len(sw.buckets); i++ {
			if sw.buckets[i].timestamp.Before(oldestTime) {
				oldestIdx = i
				oldestTime = sw.buckets[i].timestamp
			}
		}

		targetIdx = oldestIdx
	}

	// Create new bucket
	sw.buckets[targetIdx] = bucket{
		timestamp: bucketTime,
		value:     0,
	}
	sw.head = targetIdx

	return &sw.buckets[targetIdx]
}
