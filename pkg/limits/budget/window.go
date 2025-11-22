package budget

import (
	"sync"
	"time"
)

// RollingWindow tracks spending over a rolling time window.
//
// The window is divided into fixed-size buckets for memory efficiency.
// Old buckets are automatically pruned when they fall outside the window.
//
// # Implementation
//
// Uses a circular buffer of buckets where each bucket tracks spending
// for a specific time interval. Bucket granularity affects accuracy:
//
//   - Hourly window: 1-minute buckets (60 buckets)
//   - Daily window: 1-hour buckets (24 buckets)
//   - Monthly window: 1-day buckets (30 buckets)
//
// # Thread Safety
//
// RollingWindow is thread-safe using sync.RWMutex.
type RollingWindow struct {
	window     time.Duration  // Total window duration
	bucketSize time.Duration  // Granularity of each bucket
	buckets    []bucket       // Circular buffer of buckets
	mu         sync.RWMutex
}

// bucket represents spending in a specific time interval.
type bucket struct {
	timestamp time.Time
	amount    float64
}

// NewRollingWindow creates a new rolling window for budget tracking.
//
// Parameters:
//   - window: Time window duration (e.g., 1 hour, 24 hours, 30 days)
//   - bucketSize: Granularity of each bucket (smaller = more accurate, more memory)
//
// Example:
//
//	// Hourly window with 1-minute buckets
//	rw := NewRollingWindow(time.Hour, time.Minute)
//
//	// Daily window with 1-hour buckets
//	rw := NewRollingWindow(24*time.Hour, time.Hour)
//
//	// Monthly window with 1-day buckets
//	rw := NewRollingWindow(30*24*time.Hour, 24*time.Hour)
func NewRollingWindow(window time.Duration, bucketSize time.Duration) *RollingWindow {
	numBuckets := int(window / bucketSize)
	if numBuckets == 0 {
		numBuckets = 1
	}

	return &RollingWindow{
		window:     window,
		bucketSize: bucketSize,
		buckets:    make([]bucket, numBuckets),
	}
}

// Add adds spending to the current time bucket.
//
// The amount is added to the bucket corresponding to the current time.
// Old buckets outside the window are automatically pruned.
func (rw *RollingWindow) Add(amount float64) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	now := time.Now()

	// Prune old buckets
	rw.pruneLocked(now)

	// Find or create current bucket
	currentBucket := rw.findOrCreateBucketLocked(now)
	currentBucket.amount += amount
}

// Sum returns the total spending across all buckets in the window.
//
// This automatically prunes expired buckets before summing.
func (rw *RollingWindow) Sum() float64 {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	now := time.Now()

	// Prune old buckets
	rw.pruneLocked(now)

	// Sum all buckets
	var sum float64
	for i := 0; i < len(rw.buckets); i++ {
		if !rw.buckets[i].timestamp.IsZero() {
			sum += rw.buckets[i].amount
		}
	}

	return sum
}

// Reset clears all buckets.
func (rw *RollingWindow) Reset() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	for i := 0; i < len(rw.buckets); i++ {
		rw.buckets[i] = bucket{}
	}
}

// OldestTimestamp returns the timestamp of the oldest bucket in the window.
// This is useful for determining when the window will reset.
func (rw *RollingWindow) OldestTimestamp() time.Time {
	rw.mu.RLock()
	defer rw.mu.RUnlock()

	var oldest time.Time
	for i := 0; i < len(rw.buckets); i++ {
		if !rw.buckets[i].timestamp.IsZero() {
			if oldest.IsZero() || rw.buckets[i].timestamp.Before(oldest) {
				oldest = rw.buckets[i].timestamp
			}
		}
	}

	return oldest
}

// pruneLocked removes buckets older than the window.
// Caller must hold write lock.
func (rw *RollingWindow) pruneLocked(now time.Time) {
	cutoff := now.Add(-rw.window)

	for i := 0; i < len(rw.buckets); i++ {
		if !rw.buckets[i].timestamp.IsZero() && rw.buckets[i].timestamp.Before(cutoff) {
			rw.buckets[i] = bucket{} // Clear expired bucket
		}
	}
}

// findOrCreateBucketLocked finds the bucket for the current time or creates a new one.
// Caller must hold write lock.
func (rw *RollingWindow) findOrCreateBucketLocked(now time.Time) *bucket {
	// Round timestamp to bucket boundary
	bucketTime := now.Truncate(rw.bucketSize)

	// Search for existing bucket with this timestamp
	for i := 0; i < len(rw.buckets); i++ {
		if rw.buckets[i].timestamp.Equal(bucketTime) {
			return &rw.buckets[i]
		}
	}

	// No existing bucket found, create new one
	// Find next available slot (prefer empty slots, then oldest)
	targetIdx := -1

	// First, try to find an empty slot
	for i := 0; i < len(rw.buckets); i++ {
		if rw.buckets[i].timestamp.IsZero() {
			targetIdx = i
			break
		}
	}

	// If no empty slot, find oldest bucket
	if targetIdx == -1 {
		oldestIdx := 0
		oldestTime := rw.buckets[0].timestamp

		for i := 1; i < len(rw.buckets); i++ {
			if rw.buckets[i].timestamp.Before(oldestTime) {
				oldestIdx = i
				oldestTime = rw.buckets[i].timestamp
			}
		}

		targetIdx = oldestIdx
	}

	// Create new bucket
	rw.buckets[targetIdx] = bucket{
		timestamp: bucketTime,
		amount:    0,
	}

	return &rw.buckets[targetIdx]
}
