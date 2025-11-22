package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

// createTempDB creates a temporary SQLite database for testing.
func createTempDB(t *testing.T) (*SQLiteStorage, string) {
	t.Helper()

	// Create temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := &SQLiteConfig{
		Path:         dbPath,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}

	return storage, dbPath
}

// TestSQLiteStorage_Initialize tests database initialization.
func TestSQLiteStorage_Initialize(t *testing.T) {
	storage, dbPath := createTempDB(t)
	defer storage.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify WAL files exist (if WAL mode enabled)
	walPath := dbPath + "-wal"
	if _, err := os.Stat(walPath); err == nil {
		// WAL file exists, which is good
		t.Logf("WAL mode enabled, found %s", walPath)
	}
}

// TestSQLiteStorage_StoreAndQuery tests storing and querying records.
func TestSQLiteStorage_StoreAndQuery(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store a record
	now := time.Now().UTC().Truncate(time.Millisecond)
	record := &evidence.EvidenceRecord{
		ID:             "test-id-1",
		RequestID:      "req-1",
		RequestTime:    now,
		RecordedTime:   now,
		Model:          "gpt-4",
		Provider:       "openai",
		PolicyDecision: "allow",
		TotalTokens:    100,
		ActualCost:     0.01,
	}

	err := storage.Store(ctx, record)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Query all records
	query := &evidence.Query{}
	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(results))
	}

	if results[0].ID != "test-id-1" {
		t.Errorf("Expected ID 'test-id-1', got '%s'", results[0].ID)
	}
	if results[0].Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", results[0].Model)
	}
}

// TestSQLiteStorage_StoreComplexRecord tests storing records with complex nested fields.
func TestSQLiteStorage_StoreComplexRecord(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	record := &evidence.EvidenceRecord{
		ID:          "complex-record",
		RequestID:   "req-complex",
		RequestTime: now,
		RequestHeaders: map[string]string{
			"user-agent":   "test-agent/1.0",
			"content-type": "application/json",
		},
		ToolsUsed: []string{"web_search", "calculator", "code_interpreter"},
		PIITypes:  []string{"email", "phone_number"},
		MatchedRules: []evidence.MatchedRuleRecord{
			{
				PolicyID:       "policy-1",
				RuleID:         "rule-1",
				Action:         "allow",
				Reason:         "Within rate limit",
				EvaluationTime: 5 * time.Millisecond,
			},
		},
		Model:           "gpt-4",
		Provider:        "openai",
		ProviderLatency: 250 * time.Millisecond,
	}

	err := storage.Store(ctx, record)
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Query and verify
	query := &evidence.Query{}
	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(results))
	}

	r := results[0]

	// Verify headers were stored and retrieved
	if len(r.RequestHeaders) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(r.RequestHeaders))
	}
	if r.RequestHeaders["user-agent"] != "test-agent/1.0" {
		t.Error("Header 'user-agent' not preserved")
	}

	// Verify tools array
	if len(r.ToolsUsed) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(r.ToolsUsed))
	}

	// Verify PII types
	if len(r.PIITypes) != 2 {
		t.Errorf("Expected 2 PII types, got %d", len(r.PIITypes))
	}

	// Verify matched rules
	if len(r.MatchedRules) != 1 {
		t.Errorf("Expected 1 matched rule, got %d", len(r.MatchedRules))
	}

	// Verify duration fields
	if r.ProviderLatency != 250*time.Millisecond {
		t.Errorf("Expected provider latency 250ms, got %v", r.ProviderLatency)
	}
}

// TestSQLiteStorage_QueryWithTimeRange tests time range filtering.
func TestSQLiteStorage_QueryWithTimeRange(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store records with different timestamps
	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []*evidence.EvidenceRecord{
		{
			ID:          "old-record",
			RequestID:   "req-old",
			RequestTime: now.Add(-2 * time.Hour),
			Model:       "gpt-4",
		},
		{
			ID:          "recent-record",
			RequestID:   "req-recent",
			RequestTime: now.Add(-30 * time.Minute),
			Model:       "gpt-4",
		},
		{
			ID:          "new-record",
			RequestID:   "req-new",
			RequestTime: now,
			Model:       "gpt-4",
		},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Query records from last hour
	startTime := now.Add(-1 * time.Hour)
	query := &evidence.Query{
		StartTime: &startTime,
	}

	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should only get recent and new records
	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}

	// Verify old record is not included
	for _, r := range results {
		if r.ID == "old-record" {
			t.Error("Old record should not be in results")
		}
	}
}

// TestSQLiteStorage_QueryWithFilters tests various filter combinations.
func TestSQLiteStorage_QueryWithFilters(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store records with different attributes
	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []*evidence.EvidenceRecord{
		{
			ID:             "record-1",
			RequestID:      "req-1",
			RequestTime:    now,
			Model:          "gpt-4",
			Provider:       "openai",
			UserID:         "user-alice",
			PolicyDecision: "allow",
			TotalTokens:    100,
			ActualCost:     0.01,
		},
		{
			ID:             "record-2",
			RequestID:      "req-2",
			RequestTime:    now,
			Model:          "claude-3-opus",
			Provider:       "anthropic",
			UserID:         "user-bob",
			PolicyDecision: "block",
			TotalTokens:    200,
			ActualCost:     0.02,
		},
		{
			ID:             "record-3",
			RequestID:      "req-3",
			RequestTime:    now,
			Model:          "gpt-4",
			Provider:       "openai",
			UserID:         "user-alice",
			PolicyDecision: "allow",
			TotalTokens:    150,
			ActualCost:     0.015,
		},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	tests := []struct {
		name          string
		query         *evidence.Query
		expectedCount int
	}{
		{
			name: "filter by user",
			query: &evidence.Query{
				UserID: "user-alice",
			},
			expectedCount: 2,
		},
		{
			name: "filter by provider",
			query: &evidence.Query{
				Provider: "anthropic",
			},
			expectedCount: 1,
		},
		{
			name: "filter by model",
			query: &evidence.Query{
				Model: "gpt-4",
			},
			expectedCount: 2,
		},
		{
			name: "filter by policy decision",
			query: &evidence.Query{
				PolicyDecision: "block",
			},
			expectedCount: 1,
		},
		{
			name: "combined filters",
			query: &evidence.Query{
				UserID:   "user-alice",
				Provider: "openai",
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.Query(ctx, tt.query)
			if err != nil {
				t.Fatalf("Query() failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d records, got %d", tt.expectedCount, len(results))
			}
		})
	}
}

// TestSQLiteStorage_QueryWithCostThresholds tests cost filtering.
func TestSQLiteStorage_QueryWithCostThresholds(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store records with different costs
	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []*evidence.EvidenceRecord{
		{ID: "cheap", RequestID: "req-1", RequestTime: now, ActualCost: 0.001, Model: "gpt-4"},
		{ID: "medium", RequestID: "req-2", RequestTime: now, ActualCost: 0.01, Model: "gpt-4"},
		{ID: "expensive", RequestID: "req-3", RequestTime: now, ActualCost: 0.1, Model: "gpt-4"},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Query with min and max cost
	minCost := 0.005
	maxCost := 0.05
	query := &evidence.Query{
		MinCost: &minCost,
		MaxCost: &maxCost,
	}

	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should only get medium
	if len(results) != 1 {
		t.Errorf("Expected 1 record, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != "medium" {
		t.Errorf("Expected 'medium' record, got '%s'", results[0].ID)
	}
}

// TestSQLiteStorage_QueryWithStatus tests status filtering.
func TestSQLiteStorage_QueryWithStatus(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store records with different statuses
	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []*evidence.EvidenceRecord{
		{
			ID:             "success-1",
			RequestID:      "req-1",
			RequestTime:    now,
			PolicyDecision: "allow",
			Error:          "",
			Model:          "gpt-4",
		},
		{
			ID:             "error-1",
			RequestID:      "req-2",
			RequestTime:    now,
			PolicyDecision: "allow",
			Error:          "connection timeout",
			Model:          "gpt-4",
		},
		{
			ID:             "blocked-1",
			RequestID:      "req-3",
			RequestTime:    now,
			PolicyDecision: "block",
			Error:          "",
			Model:          "gpt-4",
		},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	tests := []struct {
		name          string
		status        string
		expectedCount int
	}{
		{
			name:          "success status",
			status:        "success",
			expectedCount: 2, // success-1 and blocked-1 (both have no error)
		},
		{
			name:          "error status",
			status:        "error",
			expectedCount: 1,
		},
		{
			name:          "blocked status",
			status:        "blocked",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &evidence.Query{
				Status: tt.status,
			}

			results, err := storage.Query(ctx, query)
			if err != nil {
				t.Fatalf("Query() failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d records, got %d", tt.expectedCount, len(results))
			}
		})
	}
}

// TestSQLiteStorage_QueryWithPagination tests limit and offset.
func TestSQLiteStorage_QueryWithPagination(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store 10 records
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 10; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune('0'+i)),
			RequestID:   "req-" + string(rune('0'+i)),
			RequestTime: now.Add(time.Duration(i) * time.Second),
			Model:       "gpt-4",
		}
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Query with limit
	query := &evidence.Query{
		Limit: 5,
	}

	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 records, got %d", len(results))
	}

	// Query with limit and offset
	query = &evidence.Query{
		Limit:  3,
		Offset: 5,
	}

	results, err = storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 records, got %d", len(results))
	}
}

// TestSQLiteStorage_QueryWithSorting tests sorting options.
func TestSQLiteStorage_QueryWithSorting(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store records with different costs and tokens
	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []*evidence.EvidenceRecord{
		{ID: "low", RequestID: "req-1", RequestTime: now, ActualCost: 0.01, TotalTokens: 100, Model: "gpt-4"},
		{ID: "high", RequestID: "req-2", RequestTime: now.Add(1 * time.Second), ActualCost: 0.1, TotalTokens: 1000, Model: "gpt-4"},
		{ID: "medium", RequestID: "req-3", RequestTime: now.Add(2 * time.Second), ActualCost: 0.05, TotalTokens: 500, Model: "gpt-4"},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Sort by cost descending
	query := &evidence.Query{
		SortBy:    "actual_cost",
		SortOrder: "DESC",
	}

	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(results))
	}

	// Verify order: high, medium, low
	if results[0].ID != "high" {
		t.Errorf("Expected first record to be 'high', got '%s'", results[0].ID)
	}
	if results[2].ID != "low" {
		t.Errorf("Expected last record to be 'low', got '%s'", results[2].ID)
	}

	// Sort by tokens ascending
	query = &evidence.Query{
		SortBy:    "total_tokens",
		SortOrder: "ASC",
	}

	results, err = storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Verify order: low, medium, high
	if results[0].ID != "low" {
		t.Errorf("Expected first record to be 'low', got '%s'", results[0].ID)
	}
	if results[2].ID != "high" {
		t.Errorf("Expected last record to be 'high', got '%s'", results[2].ID)
	}
}

// TestSQLiteStorage_Count tests counting records.
func TestSQLiteStorage_Count(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Initially empty
	count, err := storage.Count(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Store records
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 5; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune('0'+i)),
			RequestID:   "req-" + string(rune('0'+i)),
			RequestTime: now,
			Provider:    "openai",
			Model:       "gpt-4",
		}
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Count all
	count, err = storage.Count(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}

	// Count with filter
	query := &evidence.Query{
		Provider: "openai",
	}
	count, err = storage.Count(ctx, query)
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

// TestSQLiteStorage_Delete tests deleting records.
func TestSQLiteStorage_Delete(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Store records
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 5; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune('0'+i)),
			RequestID:   "req-" + string(rune('0'+i)),
			RequestTime: now,
			Provider:    "openai",
			Model:       "gpt-4",
		}
		if i >= 3 {
			record.Provider = "anthropic"
		}
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Delete records with provider=openai
	query := &evidence.Query{
		Provider: "openai",
	}

	deleted, err := storage.Delete(ctx, query)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if deleted != 3 {
		t.Errorf("Expected 3 deleted, got %d", deleted)
	}

	// Verify remaining records
	count, err := storage.Count(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 remaining records, got %d", count)
	}
}

// TestSQLiteStorage_ConcurrentWrites tests concurrent write operations.
func TestSQLiteStorage_ConcurrentWrites(t *testing.T) {
	storage, _ := createTempDB(t)
	defer storage.Close()

	ctx := context.Background()

	// Launch 10 goroutines that write concurrently
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			record := &evidence.EvidenceRecord{
				ID:          "record-" + string(rune('0'+id)),
				RequestID:   "req-" + string(rune('0'+id)),
				RequestTime: time.Now().UTC().Truncate(time.Millisecond),
				Model:       "gpt-4",
			}

			if err := storage.Store(ctx, record); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify all records were stored
	count, err := storage.Count(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 10 {
		t.Errorf("Expected 10 records after concurrent writes, got %d", count)
	}
}

// TestSQLiteStorage_Close tests closing the storage.
func TestSQLiteStorage_Close(t *testing.T) {
	storage, _ := createTempDB(t)

	// Close storage
	if err := storage.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify subsequent operations fail gracefully
	ctx := context.Background()
	record := &evidence.EvidenceRecord{
		ID:          "test-record",
		RequestID:   "req-1",
		RequestTime: time.Now(),
		Model:       "gpt-4",
	}

	err := storage.Store(ctx, record)
	if err == nil {
		t.Error("Expected error after Close(), got nil")
	}
}

// BenchmarkSQLiteStorage_Store benchmarks storing records.
func BenchmarkSQLiteStorage_Store(b *testing.B) {
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
			ID:          "record-" + string(rune(i)),
			RequestID:   "req-" + string(rune(i)),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
		}
		_ = storage.Store(ctx, record)
	}
}

// BenchmarkSQLiteStorage_Query benchmarks querying records.
func BenchmarkSQLiteStorage_Query(b *testing.B) {
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

	// Pre-populate with 1000 records
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune(i)),
			RequestID:   "req-" + string(rune(i)),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
		}
		_ = storage.Store(ctx, record)
	}

	query := &evidence.Query{
		Provider: "openai",
		Limit:    100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Query(ctx, query)
	}
}

// BenchmarkSQLiteStorage_Count benchmarks counting records.
func BenchmarkSQLiteStorage_Count(b *testing.B) {
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

	// Pre-populate with 1000 records
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune(i)),
			RequestID:   "req-" + string(rune(i)),
			RequestTime: now,
			Provider:    "openai",
			Model:       "gpt-4",
		}
		_ = storage.Store(ctx, record)
	}

	query := &evidence.Query{
		Provider: "openai",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Count(ctx, query)
	}
}
