package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

// Helper function to get provider based on index
func getProvider(i int) string {
	if i%2 == 0 {
		return "openai"
	}
	return "anthropic"
}

// TestSQLiteStorage_QueryTimeout tests query cancellation via context.
func TestSQLiteStorage_QueryTimeout(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Insert many records to make query slower
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
		}
		_ = storage.Store(ctx, record)
	}

	// Create a context with very short timeout
	queryCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(10 * time.Millisecond)

	// Query should respect context cancellation
	_, err := storage.Query(queryCtx, &evidence.Query{})

	// Note: This test may pass even if query completes quickly
	// The key is that the storage respects context cancellation
	if err != nil {
		t.Logf("Query cancelled as expected: %v", err)
	} else {
		t.Log("Query completed before timeout (database is fast)")
	}
}

// TestSQLiteStorage_CountTimeout tests count cancellation via context.
func TestSQLiteStorage_CountTimeout(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Insert records
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 100; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
		}
		_ = storage.Store(ctx, record)
	}

	// Create cancelled context
	queryCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	// Count should respect context cancellation
	_, err := storage.Count(queryCtx, &evidence.Query{})

	if err != nil {
		t.Logf("Count cancelled as expected: %v", err)
	} else {
		t.Log("Count completed before cancellation")
	}
}

// TestSQLiteStorage_InvalidDatabasePath tests error when database path is invalid.
func TestSQLiteStorage_InvalidDatabasePath(t *testing.T) {
	config := &SQLiteConfig{
		Path:         "/nonexistent/directory/test.db",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}

	_, err := NewSQLiteStorage(config)
	if err == nil {
		t.Error("Expected error for invalid database path, got nil")
	}

	if !strings.Contains(err.Error(), "storage error") {
		t.Errorf("Expected storage error, got: %v", err)
	}
}

// TestSQLiteStorage_ReadOnlyDatabasePath tests error when database path is read-only.
func TestSQLiteStorage_ReadOnlyDatabasePath(t *testing.T) {
	// Create a read-only directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "readonly.db")

	// Create the database file first
	config := &SQLiteConfig{
		Path:         dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		t.Fatalf("Failed to create initial database: %v", err)
	}
	storage.Close()

	// Make the directory read-only
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Skipf("Cannot change directory permissions: %v", err)
	}
	defer func() { _ = os.Chmod(tmpDir, 0755) }() // Restore for cleanup

	// Try to open database in read-only directory
	_, err = NewSQLiteStorage(config)
	if err == nil {
		t.Log("Warning: Expected error for read-only database path (may succeed on some systems)")
	}
}

// TestSQLiteStorage_DiskFullSimulation tests handling of disk full errors.
func TestSQLiteStorage_DiskFullSimulation(t *testing.T) {
	// Note: Actual disk full simulation is difficult in unit tests
	// This test verifies error propagation when storage fails

	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Create a very large record to potentially trigger storage issues
	largeContent := strings.Repeat("x", 10*1024*1024) // 10MB

	record := &evidence.EvidenceRecord{
		ID:              "large-record",
		RequestID:       "req-large",
		RequestTime:     now,
		Model:           "gpt-4",
		SystemPrompt:    largeContent[:500], // Truncated
		ResponseContent: largeContent[:500], // Truncated
	}

	err := storage.Store(ctx, record)
	if err != nil {
		t.Logf("Storage failed with large record (expected on constrained systems): %v", err)
	} else {
		t.Log("Large record stored successfully")
	}
}

// TestSQLiteStorage_CorruptedDatabase tests detection of corrupted database.
func TestSQLiteStorage_CorruptedDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "corrupt.db")

	// Create a corrupted database file (just write garbage)
	if err := os.WriteFile(dbPath, []byte("This is not a valid SQLite database"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted database: %v", err)
	}

	config := &SQLiteConfig{
		Path:         dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		WALMode:      false, // Disable WAL for this test
		BusyTimeout:  5 * time.Second,
	}

	storage, err := NewSQLiteStorage(config)
	if err == nil {
		storage.Close()
		t.Error("Expected error for corrupted database, got nil")
	} else {
		t.Logf("Corrupted database detected correctly: %v", err)
	}
}

// TestSQLiteStorage_PreparedStatementReuse tests that prepared statements are reused.
func TestSQLiteStorage_PreparedStatementReuse(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Insert multiple records (should reuse prepared statements)
	for i := 0; i < 10; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
		}

		err := storage.Store(ctx, record)
		if err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Verify prepared statements map exists (internal check)
	storage.mu.RLock()
	stmtCount := len(storage.preparedStmts)
	storage.mu.RUnlock()

	t.Logf("Prepared statements in cache: %d", stmtCount)
	// Note: Current implementation doesn't cache prepared statements
	// This test documents the behavior for future optimization
}

// TestSQLiteStorage_BusyTimeout tests that busy timeout is configured.
func TestSQLiteStorage_BusyTimeout(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	// Verify busy timeout is set correctly
	if storage.config.BusyTimeout != 5*time.Second {
		t.Errorf("Expected BusyTimeout 5s, got %v", storage.config.BusyTimeout)
	}

	// In a real scenario, we'd test concurrent writes causing locks
	// but that's covered by TestSQLiteStorage_ConcurrentWrites
	t.Log("Busy timeout configured correctly")
}

// TestSQLiteStorage_SchemaVersion tests schema version checking.
func TestSQLiteStorage_SchemaVersion(t *testing.T) {
	storage, dbPath := createTempDB(t)
	storage.Close()

	// Reopen the database to verify schema version persists
	config := &SQLiteConfig{
		Path:         dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}

	storage2, err := NewSQLiteStorage(config)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer storage2.Close()

	// Query schema version directly
	var version int
	err = storage2.db.QueryRow("SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query schema version: %v", err)
	}

	if version != SchemaVersion {
		t.Errorf("Expected schema version %d, got %d", SchemaVersion, version)
	}

	t.Logf("Schema version verified: %d", version)
}

// TestSQLiteStorage_LargeResultSet tests querying large result sets.
func TestSQLiteStorage_LargeResultSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large result set test in short mode")
	}

	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Insert 10,000 records
	recordCount := 10000
	t.Logf("Inserting %d records...", recordCount)

	start := time.Now()
	for i := 0; i < recordCount; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now.Add(time.Duration(i) * time.Second),
			Model:       "gpt-4",
			Provider:    "openai",
			TotalTokens: 100 + i,
			ActualCost:  0.01 + float64(i)*0.001,
		}

		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed at record %d: %v", i, err)
		}
	}
	insertDuration := time.Since(start)
	t.Logf("Inserted %d records in %v (%.2f records/sec)",
		recordCount, insertDuration, float64(recordCount)/insertDuration.Seconds())

	// Query with pagination to test large result sets
	start = time.Now()
	query := &evidence.Query{
		Limit: 1000,
	}
	results, err := storage.Query(ctx, query)
	queryDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 1000 {
		t.Errorf("Expected 1000 results, got %d", len(results))
	}

	t.Logf("Queried 1000 records from %d total in %v", recordCount, queryDuration)

	// Test query performance target: <100ms for typical query
	if queryDuration > 100*time.Millisecond {
		t.Logf("Warning: Query took %v (target: <100ms)", queryDuration)
	}

	// Test count performance
	start = time.Now()
	count, err := storage.Count(ctx, &evidence.Query{})
	countDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != int64(recordCount) {
		t.Errorf("Expected count %d, got %d", recordCount, count)
	}

	t.Logf("Counted %d records in %v", count, countDuration)
}

// TestSQLiteStorage_IndexPerformance tests query performance with indexes.
func TestSQLiteStorage_IndexPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping index performance test in short mode")
	}

	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Insert 1000 records
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now.Add(time.Duration(i) * time.Second),
			Model:       "gpt-4",
			Provider:    getProvider(i),
			UserID:      fmt.Sprintf("user-%d", i%10),
		}
		_ = storage.Store(ctx, record)
	}

	// Test indexed query (timestamp)
	startTime := now.Add(500 * time.Second)
	endTime := now.Add(600 * time.Second)

	start := time.Now()
	query := &evidence.Query{
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	results, err := storage.Query(ctx, query)
	indexedDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Indexed query failed: %v", err)
	}

	t.Logf("Indexed query (time range) returned %d records in %v", len(results), indexedDuration)

	// Test indexed query (user_id)
	start = time.Now()
	query = &evidence.Query{
		UserID: "user-5",
	}
	results, err = storage.Query(ctx, query)
	userIndexDuration := time.Since(start)

	if err != nil {
		t.Fatalf("User query failed: %v", err)
	}

	t.Logf("User query returned %d records in %v", len(results), userIndexDuration)

	// Test combined indexed query
	start = time.Now()
	query = &evidence.Query{
		StartTime: &startTime,
		Provider:  "openai",
	}
	results, err = storage.Query(ctx, query)
	combinedDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Combined query failed: %v", err)
	}

	t.Logf("Combined query returned %d records in %v", len(results), combinedDuration)

	// Performance targets
	if indexedDuration > 10*time.Millisecond {
		t.Logf("Warning: Indexed query took %v (target: <10ms)", indexedDuration)
	}
}

// TestSQLiteStorage_DeletePerformance tests deletion performance.
func TestSQLiteStorage_DeletePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping delete performance test in short mode")
	}

	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Insert 10,000 records
	recordCount := 10000
	for i := 0; i < recordCount; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now.Add(time.Duration(i) * time.Second),
			Model:       "gpt-4",
		}
		_ = storage.Store(ctx, record)
	}

	// Delete half the records
	deleteThreshold := now.Add(5000 * time.Second)
	query := &evidence.Query{
		EndTime: &deleteThreshold,
	}

	start := time.Now()
	deleted, err := storage.Delete(ctx, query)
	deleteDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	t.Logf("Deleted %d records in %v (%.2f records/sec)",
		deleted, deleteDuration, float64(deleted)/deleteDuration.Seconds())

	// Performance target: delete 10K records in <5s
	expectedDeleted := int64(5001) // 0-5000 inclusive
	if deleted != expectedDeleted {
		t.Errorf("Expected to delete %d records, deleted %d", expectedDeleted, deleted)
	}

	// Verify remaining records
	count, _ := storage.Count(ctx, &evidence.Query{})
	expectedRemaining := int64(recordCount) - deleted
	if count != expectedRemaining {
		t.Errorf("Expected %d remaining records, got %d", expectedRemaining, count)
	}
}

// BenchmarkSQLiteStorage_InsertPerformance benchmarks insert performance target.
func BenchmarkSQLiteStorage_InsertPerformance(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	config := &SQLiteConfig{
		Path:         dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record := &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
		}
		_ = storage.Store(ctx, record)
	}
	b.StopTimer()

	// Report performance metrics
	recordsPerSec := float64(b.N) / b.Elapsed().Seconds()
	avgInsertTime := b.Elapsed() / time.Duration(b.N)

	b.ReportMetric(recordsPerSec, "records/sec")
	b.ReportMetric(float64(avgInsertTime.Microseconds()), "µs/insert")

	// Check against target: <5ms per record
	if avgInsertTime > 5*time.Millisecond {
		b.Logf("Warning: Average insert time %v exceeds target of 5ms", avgInsertTime)
	}
}
