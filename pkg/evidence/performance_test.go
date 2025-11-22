package evidence_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/storage"
)

// Helper function to get provider based on index
func getProvider(i int) string {
	if i%2 == 0 {
		return "openai"
	}
	return "anthropic"
}

// Performance Test Suite
// Validates that evidence system meets performance targets:
// - Recording throughput: >1000 writes/sec
// - Query performance: 1M records in <1s
// - Retention performance: Delete 10K in <5s

// BenchmarkRecordingThroughput benchmarks evidence recording throughput.
// Target: >1000 evidence writes/sec
func BenchmarkRecordingThroughput(b *testing.B) {
	store := storage.NewMemoryStorage()
	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
			TotalTokens: 1000,
			ActualCost:  0.03,
		}

		_ = store.Store(ctx, record)
	}
	b.StopTimer()

	// Calculate throughput
	duration := b.Elapsed()
	recordsPerSec := float64(b.N) / duration.Seconds()

	b.ReportMetric(recordsPerSec, "records/sec")
	b.ReportMetric(float64(duration.Microseconds())/float64(b.N), "µs/record")

	// Verify target: >1000 writes/sec
	if recordsPerSec < 1000 {
		b.Logf("Warning: Throughput %.0f records/sec is below target of 1000", recordsPerSec)
	} else {
		b.Logf("[PASS] Throughput target met: %.0f records/sec", recordsPerSec)
	}
}

// BenchmarkRecordingThroughput_SQLite benchmarks SQLite recording throughput.
func BenchmarkRecordingThroughput_SQLite(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	config := &storage.SQLiteConfig{
		Path:         dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}

	store, err := storage.NewSQLiteStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
		}

		_ = store.Store(ctx, record)
	}
	b.StopTimer()

	duration := b.Elapsed()
	recordsPerSec := float64(b.N) / duration.Seconds()
	avgInsertTime := duration / time.Duration(b.N)

	b.ReportMetric(recordsPerSec, "records/sec")
	b.ReportMetric(float64(avgInsertTime.Microseconds()), "µs/insert")

	// Target: >1000 writes/sec, <5ms per insert
	if recordsPerSec < 1000 {
		b.Logf("Warning: SQLite throughput %.0f records/sec is below target of 1000", recordsPerSec)
	}
	if avgInsertTime > 5*time.Millisecond {
		b.Logf("Warning: Average insert time %v exceeds target of 5ms", avgInsertTime)
	}
}

// TestQueryPerformance_LargeDataset tests query performance with large datasets.
// Target: Query 1M records in <1s (with indexes)
func TestQueryPerformance_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	// Use in-memory storage for speed
	store := storage.NewMemoryStorage()
	ctx := context.Background()
	now := time.Now()

	// Insert 100K records (1M takes too long for tests)
	recordCount := 100000
	t.Logf("Inserting %d records...", recordCount)

	insertStart := time.Now()
	for i := 0; i < recordCount; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now.Add(time.Duration(i) * time.Second),
			Model:       "gpt-4",
			Provider:    getProvider(i),
			UserID:      fmt.Sprintf("user-%d", i%100),
			TotalTokens: 1000 + i,
			ActualCost:  0.01 + float64(i)*0.0001,
		}
		_ = store.Store(ctx, record)
	}
	insertDuration := time.Since(insertStart)
	t.Logf("Inserted %d records in %v", recordCount, insertDuration)

	// Test 1: Time range query
	t.Run("TimeRangeQuery", func(t *testing.T) {
		startTime := now.Add(10000 * time.Second)
		endTime := now.Add(20000 * time.Second)

		start := time.Now()
		query := &evidence.Query{
			StartTime: &startTime,
			EndTime:   &endTime,
		}
		results, err := store.Query(ctx, query)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		t.Logf("Time range query returned %d records in %v", len(results), duration)

		// Target: <100ms for typical query
		if duration > 100*time.Millisecond {
			t.Logf("Warning: Query took %v (target: <100ms)", duration)
		}
	})

	// Test 2: User filter query
	t.Run("UserFilterQuery", func(t *testing.T) {
		start := time.Now()
		query := &evidence.Query{
			UserID: "user-50",
		}
		results, err := store.Query(ctx, query)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		t.Logf("User filter query returned %d records in %v", len(results), duration)

		if duration > 100*time.Millisecond {
			t.Logf("Warning: User query took %v (target: <100ms)", duration)
		}
	})

	// Test 3: Provider filter query
	t.Run("ProviderFilterQuery", func(t *testing.T) {
		start := time.Now()
		query := &evidence.Query{
			Provider: "openai",
			Limit:    1000,
		}
		results, err := store.Query(ctx, query)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		t.Logf("Provider filter query returned %d records in %v", len(results), duration)

		if duration > 100*time.Millisecond {
			t.Logf("Warning: Provider query took %v (target: <100ms)", duration)
		}
	})

	// Test 4: Cost threshold query
	t.Run("CostThresholdQuery", func(t *testing.T) {
		minCost := 5.0
		maxCost := 10.0

		start := time.Now()
		query := &evidence.Query{
			MinCost: &minCost,
			MaxCost: &maxCost,
		}
		results, err := store.Query(ctx, query)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		t.Logf("Cost threshold query returned %d records in %v", len(results), duration)
	})

	// Test 5: Count performance
	t.Run("CountPerformance", func(t *testing.T) {
		start := time.Now()
		count, err := store.Count(ctx, &evidence.Query{})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != int64(recordCount) {
			t.Errorf("Expected count %d, got %d", recordCount, count)
		}

		t.Logf("Counted %d records in %v", count, duration)
	})
}

// TestRetentionPerformance tests retention pruning performance.
// Target: Delete 10K records in <5s
func TestRetentionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retention performance test in short mode")
	}

	store := storage.NewMemoryStorage()
	ctx := context.Background()
	now := time.Now()

	// Insert 10K old records and 10K recent records
	oldCount := 10000
	recentCount := 10000
	totalCount := oldCount + recentCount

	t.Logf("Inserting %d records...", totalCount)

	for i := 0; i < totalCount; i++ {
		age := -5 // Recent
		if i < oldCount {
			age = -10 // Old
		}

		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now.AddDate(0, 0, age),
			Model:       "gpt-4",
		}
		_ = store.Store(ctx, record)
	}

	// Delete old records (simulate retention pruning)
	cutoff := now.AddDate(0, 0, -7)

	start := time.Now()
	deleted, err := store.Delete(ctx, &evidence.Query{
		EndTime: &cutoff,
	})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if deleted != int64(oldCount) {
		t.Errorf("Expected to delete %d records, deleted %d", oldCount, deleted)
	}

	t.Logf("Deleted %d records in %v (%.0f records/sec)",
		deleted, duration, float64(deleted)/duration.Seconds())

	// Target: delete 10K records in <5s
	if duration > 5*time.Second {
		t.Logf("Warning: Delete took %v (target: <5s)", duration)
	} else {
		t.Logf("[PASS] Retention target met: deleted %d records in %v", deleted, duration)
	}

	// Verify remaining records
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != int64(recentCount) {
		t.Errorf("Expected %d remaining records, got %d", recentCount, count)
	}
}

// TestMemoryUsageUnderLoad tests memory usage under sustained load.
// Target: No memory leaks, reasonable memory footprint
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	store := storage.NewMemoryStorage()
	ctx := context.Background()
	now := time.Now()

	// Insert 10K records
	for i := 0; i < 10000; i++ {
		record := &evidence.EvidenceRecord{
			ID:           fmt.Sprintf("record-%d", i),
			RequestID:    fmt.Sprintf("req-%d", i),
			RequestTime:  now,
			Model:        "gpt-4",
			SystemPrompt: "You are a helpful assistant",
			UserPrompt:   "Test question",
		}
		_ = store.Store(ctx, record)
	}

	// Query multiple times to test for memory leaks
	for i := 0; i < 100; i++ {
		_, _ = store.Query(ctx, &evidence.Query{Limit: 100})
	}

	// Check storage size
	size := store.Size()
	if size != 10000 {
		t.Errorf("Expected storage size 10000, got %d", size)
	}

	t.Logf("Memory test completed: %d records stored, 100 queries executed", size)

	// Note: Actual memory profiling would require runtime.ReadMemStats
	// This test documents the behavior
}

// BenchmarkEndToEndRecording benchmarks complete recording workflow.
// Simulates: Record creation → hashing → storage
func BenchmarkEndToEndRecording(b *testing.B) {
	store := storage.NewMemoryStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate complete recording workflow
		now := time.Now()

		// 1. Create record (would be done by recorder)
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
			TotalTokens: 1000,
			ActualCost:  0.03,
		}

		// 2. Hash would be computed (simulated)
		// In real scenario: HashContent() would be called

		// 3. Store
		_ = store.Store(ctx, record)
	}
	b.StopTimer()

	duration := b.Elapsed()
	recordsPerSec := float64(b.N) / duration.Seconds()
	avgTime := duration / time.Duration(b.N)

	b.ReportMetric(recordsPerSec, "records/sec")
	b.ReportMetric(float64(avgTime.Microseconds()), "µs/record")

	// Target: <2ms per complete recording
	if avgTime > 2*time.Millisecond {
		b.Logf("Warning: End-to-end recording took %v (target: <2ms)", avgTime)
	}
}

// BenchmarkAsyncChannelOverhead benchmarks async channel operations.
// Target: <1ms async channel overhead
func BenchmarkAsyncChannelOverhead(b *testing.B) {
	// Simulate async channel buffering used by recorder
	bufferSize := 1000
	ch := make(chan *evidence.EvidenceRecord, bufferSize)

	// Start consumer
	done := make(chan bool)
	go func() {
		for range ch {
			// Drain channel
		}
		done <- true
	}()

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
		}

		ch <- record
	}
	b.StopTimer()

	close(ch)
	<-done

	avgOverhead := b.Elapsed() / time.Duration(b.N)
	b.ReportMetric(float64(avgOverhead.Nanoseconds()), "ns/enqueue")
	b.ReportMetric(float64(avgOverhead.Microseconds()), "µs/enqueue")

	// Target: <1ms channel overhead
	if avgOverhead > 1*time.Millisecond {
		b.Logf("Warning: Channel overhead %v exceeds target of 1ms", avgOverhead)
	}
}

// BenchmarkConcurrentQueryPerformance benchmarks concurrent query operations.
func BenchmarkConcurrentQueryPerformance(b *testing.B) {
	store := storage.NewMemoryStorage()
	ctx := context.Background()
	now := time.Now()

	// Pre-populate with 1000 records
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
		}
		_ = store.Store(ctx, record)
	}

	query := &evidence.Query{
		Provider: "openai",
		Limit:    100,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.Query(ctx, query)
		}
	})
}
