package storage

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

// TestMemoryStorage_StoreAndQuery tests storing and querying records.
func TestMemoryStorage_StoreAndQuery(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store a record
	now := time.Now()
	record := &evidence.EvidenceRecord{
		ID:             "test-id-1",
		RequestID:      "req-1",
		RequestTime:    now,
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
}

// TestMemoryStorage_QueryWithTimeRange tests time range filtering.
func TestMemoryStorage_QueryWithTimeRange(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store records with different timestamps
	now := time.Now()
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

// TestMemoryStorage_QueryWithFilters tests various filter combinations.
func TestMemoryStorage_QueryWithFilters(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store records with different attributes
	now := time.Now()
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
		expectedIDs   []string
	}{
		{
			name: "filter by user",
			query: &evidence.Query{
				UserID: "user-alice",
			},
			expectedCount: 2,
			expectedIDs:   []string{"record-1", "record-3"},
		},
		{
			name: "filter by provider",
			query: &evidence.Query{
				Provider: "anthropic",
			},
			expectedCount: 1,
			expectedIDs:   []string{"record-2"},
		},
		{
			name: "filter by model",
			query: &evidence.Query{
				Model: "gpt-4",
			},
			expectedCount: 2,
			expectedIDs:   []string{"record-1", "record-3"},
		},
		{
			name: "filter by policy decision",
			query: &evidence.Query{
				PolicyDecision: "block",
			},
			expectedCount: 1,
			expectedIDs:   []string{"record-2"},
		},
		{
			name: "combined filters",
			query: &evidence.Query{
				UserID:   "user-alice",
				Provider: "openai",
				Model:    "gpt-4",
			},
			expectedCount: 2,
			expectedIDs:   []string{"record-1", "record-3"},
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

			// Verify expected IDs are present
			foundIDs := make(map[string]bool)
			for _, r := range results {
				foundIDs[r.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("Expected to find record %s", expectedID)
				}
			}
		})
	}
}

// TestMemoryStorage_QueryWithCostThresholds tests cost filtering.
func TestMemoryStorage_QueryWithCostThresholds(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store records with different costs
	now := time.Now()
	records := []*evidence.EvidenceRecord{
		{ID: "cheap", RequestID: "req-1", RequestTime: now, ActualCost: 0.001},
		{ID: "medium", RequestID: "req-2", RequestTime: now, ActualCost: 0.01},
		{ID: "expensive", RequestID: "req-3", RequestTime: now, ActualCost: 0.1},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Query with min cost threshold
	minCost := 0.005
	query := &evidence.Query{
		MinCost: &minCost,
	}

	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should get medium and expensive
	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}

	// Query with max cost threshold
	maxCost := 0.05
	query = &evidence.Query{
		MaxCost: &maxCost,
	}

	results, err = storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should get cheap and medium
	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}

	// Query with both min and max
	minCost = 0.005
	maxCost = 0.05
	query = &evidence.Query{
		MinCost: &minCost,
		MaxCost: &maxCost,
	}

	results, err = storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should only get medium
	if len(results) != 1 {
		t.Errorf("Expected 1 record, got %d", len(results))
	}
	if results[0].ID != "medium" {
		t.Errorf("Expected 'medium' record, got '%s'", results[0].ID)
	}
}

// TestMemoryStorage_QueryWithTokenThresholds tests token filtering.
func TestMemoryStorage_QueryWithTokenThresholds(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store records with different token counts
	now := time.Now()
	records := []*evidence.EvidenceRecord{
		{ID: "small", RequestID: "req-1", RequestTime: now, TotalTokens: 100},
		{ID: "medium", RequestID: "req-2", RequestTime: now, TotalTokens: 1000},
		{ID: "large", RequestID: "req-3", RequestTime: now, TotalTokens: 10000},
	}

	for _, record := range records {
		if err := storage.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Query with min tokens
	minTokens := 500
	query := &evidence.Query{
		MinTokens: &minTokens,
	}

	results, err := storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should get medium and large
	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}

	// Query with max tokens
	maxTokens := 5000
	query = &evidence.Query{
		MaxTokens: &maxTokens,
	}

	results, err = storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Should get small and medium
	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}
}

// TestMemoryStorage_QueryWithStatus tests status filtering.
func TestMemoryStorage_QueryWithStatus(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store records with different statuses
	// Note: "success" means no error (Error field is empty)
	// "blocked" means policy decision is block
	// A blocked request can have Error="" (it was blocked, not errored)
	now := time.Now()
	records := []*evidence.EvidenceRecord{
		{
			ID:             "success-allow",
			RequestID:      "req-1",
			RequestTime:    now,
			PolicyDecision: "allow",
			Error:          "",
		},
		{
			ID:             "error-1",
			RequestID:      "req-2",
			RequestTime:    now,
			PolicyDecision: "allow",
			Error:          "connection timeout",
		},
		{
			ID:             "blocked-1",
			RequestID:      "req-3",
			RequestTime:    now,
			PolicyDecision: "block",
			Error:          "",
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
		expectedIDs   []string
	}{
		{
			name:   "success status",
			status: "success",
			// Success means no error - both success-allow and blocked-1 have Error=""
			// But blocked should not be counted as success per the storage implementation
			// Let me check the actual storage implementation...
			// Looking at memory.go line 171-176:
			// case "success": if record.Error != "" { return false }
			// case "blocked": if record.PolicyDecision != "block" { return false }
			// So "success" returns records with Error == ""
			// And "blocked" returns records with PolicyDecision == "block"
			// Therefore "success-allow" and "blocked-1" both have Error=""
			expectedCount: 2,
			expectedIDs:   []string{"success-allow", "blocked-1"},
		},
		{
			name:          "error status",
			status:        "error",
			expectedCount: 1,
			expectedIDs:   []string{"error-1"},
		},
		{
			name:          "blocked status",
			status:        "blocked",
			expectedCount: 1,
			expectedIDs:   []string{"blocked-1"},
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

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, r := range results {
					if r.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find record %s", expectedID)
				}
			}
		})
	}
}

// TestMemoryStorage_QueryWithPagination tests limit and offset.
func TestMemoryStorage_QueryWithPagination(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store 10 records
	now := time.Now()
	for i := 0; i < 10; i++ {
		record := &evidence.EvidenceRecord{
			ID:          string(rune('A' + i)),
			RequestID:   "req-" + string(rune('0'+i)),
			RequestTime: now,
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

	// Query with offset beyond available records
	query = &evidence.Query{
		Offset: 100,
	}

	results, err = storage.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 records, got %d", len(results))
	}
}

// TestMemoryStorage_Count tests counting records.
func TestMemoryStorage_Count(t *testing.T) {
	storage := NewMemoryStorage()
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
	now := time.Now()
	for i := 0; i < 5; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune('0'+i)),
			RequestID:   "req-" + string(rune('0'+i)),
			RequestTime: now,
			Provider:    "openai",
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

	// Count with non-matching filter
	query = &evidence.Query{
		Provider: "anthropic",
	}
	count, err = storage.Count(ctx, query)
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

// TestMemoryStorage_Delete tests deleting records.
func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store records
	now := time.Now()
	for i := 0; i < 5; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune('0'+i)),
			RequestID:   "req-" + string(rune('0'+i)),
			RequestTime: now,
			Provider:    "openai",
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

	// Verify only anthropic records remain
	results, err := storage.Query(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	for _, r := range results {
		if r.Provider != "anthropic" {
			t.Errorf("Expected only anthropic records, found %s", r.Provider)
		}
	}
}

// TestMemoryStorage_Close tests closing the storage.
func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store a record
	record := &evidence.EvidenceRecord{
		ID:          "test-record",
		RequestID:   "req-1",
		RequestTime: time.Now(),
	}

	if err := storage.Store(ctx, record); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Close storage
	if err := storage.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify storage is cleared
	if storage.Size() != 0 {
		t.Errorf("Expected storage to be cleared after Close(), got %d records", storage.Size())
	}
}

// TestMemoryStorage_ThreadSafety tests concurrent access.
func TestMemoryStorage_ThreadSafety(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Use channels to coordinate goroutines
	done := make(chan bool, 10)

	// Launch 10 goroutines that write concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			record := &evidence.EvidenceRecord{
				ID:          "record-" + string(rune('0'+id)),
				RequestID:   "req-" + string(rune('0'+id)),
				RequestTime: time.Now(),
			}

			if err := storage.Store(ctx, record); err != nil {
				t.Errorf("Store() failed: %v", err)
			}
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all records were stored
	count, err := storage.Count(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 10 {
		t.Errorf("Expected 10 records after concurrent writes, got %d", count)
	}

	// Launch 10 goroutines that read concurrently
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			_, err := storage.Query(ctx, &evidence.Query{})
			if err != nil {
				t.Errorf("Query() failed: %v", err)
			}
		}()
	}

	// Wait for all reads to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestMemoryStorage_RecordIsolation tests that stored records are isolated from mutations.
func TestMemoryStorage_RecordIsolation(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Store a record
	original := &evidence.EvidenceRecord{
		ID:          "isolation-test",
		RequestID:   "req-1",
		RequestTime: time.Now(),
		Model:       "gpt-4",
	}

	if err := storage.Store(ctx, original); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Mutate the original record
	original.Model = "mutated-model"

	// Query the record back
	results, err := storage.Query(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(results))
	}

	// Verify the stored record was not mutated
	if results[0].Model != "gpt-4" {
		t.Errorf("Expected stored record to be isolated from mutations, got model=%s", results[0].Model)
	}

	// Mutate the queried record
	results[0].Model = "another-mutation"

	// Query again
	results2, err := storage.Query(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	// Verify the stored record was not mutated
	if results2[0].Model != "gpt-4" {
		t.Errorf("Expected stored record to be isolated from query result mutations, got model=%s", results2[0].Model)
	}
}

// BenchmarkMemoryStorage_Store benchmarks storing records.
func BenchmarkMemoryStorage_Store(b *testing.B) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	record := &evidence.EvidenceRecord{
		ID:          "benchmark-record",
		RequestID:   "req-bench",
		RequestTime: time.Now(),
		Model:       "gpt-4",
		Provider:    "openai",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Store(ctx, record)
	}
}

// BenchmarkMemoryStorage_Query benchmarks querying records.
func BenchmarkMemoryStorage_Query(b *testing.B) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Pre-populate with 1000 records
	now := time.Now()
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune(i)),
			RequestID:   "req-" + string(rune(i)),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
		}
		storage.Store(ctx, record)
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

// BenchmarkMemoryStorage_Count benchmarks counting records.
func BenchmarkMemoryStorage_Count(b *testing.B) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Pre-populate with 1000 records
	now := time.Now()
	for i := 0; i < 1000; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune(i)),
			RequestID:   "req-" + string(rune(i)),
			RequestTime: now,
			Provider:    "openai",
		}
		storage.Store(ctx, record)
	}

	query := &evidence.Query{
		Provider: "openai",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Count(ctx, query)
	}
}

// BenchmarkMemoryStorage_Delete benchmarks deleting records.
func BenchmarkMemoryStorage_Delete(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create fresh storage for each iteration
		storage := NewMemoryStorage()
		now := time.Now()
		for j := 0; j < 100; j++ {
			record := &evidence.EvidenceRecord{
				ID:          "record-" + string(rune(j)),
				RequestID:   "req-" + string(rune(j)),
				RequestTime: now,
				Provider:    "openai",
			}
			storage.Store(ctx, record)
		}
		b.StartTimer()

		query := &evidence.Query{
			Provider: "openai",
		}
		_, _ = storage.Delete(ctx, query)
	}
}
