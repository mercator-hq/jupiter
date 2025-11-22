package retention

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/storage"
)

// TestPruner_PruneOldRecords tests pruning records older than retention period.
func TestPruner_PruneOldRecords(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RetentionDays = 7
	config.ArchiveBeforeDelete = false

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Store records with different ages
	records := []*evidence.EvidenceRecord{
		{
			ID:          "old-1",
			RequestID:   "req-old-1",
			RequestTime: now.AddDate(0, 0, -10), // 10 days old
			Model:       "gpt-4",
		},
		{
			ID:          "old-2",
			RequestID:   "req-old-2",
			RequestTime: now.AddDate(0, 0, -8), // 8 days old
			Model:       "gpt-4",
		},
		{
			ID:          "recent-1",
			RequestID:   "req-recent-1",
			RequestTime: now.AddDate(0, 0, -5), // 5 days old
			Model:       "gpt-4",
		},
		{
			ID:          "recent-2",
			RequestID:   "req-recent-2",
			RequestTime: now.AddDate(0, 0, -3), // 3 days old
			Model:       "gpt-4",
		},
	}

	for _, record := range records {
		if err := store.Store(ctx, record); err != nil {
			t.Fatalf("Store() failed: %v", err)
		}
	}

	// Verify all records are stored
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 4 {
		t.Fatalf("Expected 4 records, got %d", count)
	}

	// Run pruner
	deleted, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	// Should delete 2 old records
	if deleted != 2 {
		t.Errorf("Expected 2 deleted records, got %d", deleted)
	}

	// Verify remaining records
	count, _ = store.Count(ctx, &evidence.Query{})
	if count != 2 {
		t.Errorf("Expected 2 remaining records, got %d", count)
	}

	// Verify only recent records remain
	results, _ := store.Query(ctx, &evidence.Query{})
	for _, r := range results {
		if r.ID == "old-1" || r.ID == "old-2" {
			t.Errorf("Old record %s should have been deleted", r.ID)
		}
	}
}

// TestPruner_RetentionDisabled tests that pruning is skipped when retention is 0.
func TestPruner_RetentionDisabled(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RetentionDays = 0 // Disabled

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Store an old record
	record := &evidence.EvidenceRecord{
		ID:          "old-record",
		RequestID:   "req-old",
		RequestTime: now.AddDate(0, 0, -100), // Very old
		Model:       "gpt-4",
	}

	_ = store.Store(ctx, record)

	// Run pruner
	deleted, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	// Should delete nothing
	if deleted != 0 {
		t.Errorf("Expected 0 deleted records when retention disabled, got %d", deleted)
	}

	// Verify record still exists
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 1 {
		t.Errorf("Expected 1 record to remain, got %d", count)
	}
}

// TestPruner_ArchiveBeforeDelete tests archiving records before deletion.
func TestPruner_ArchiveBeforeDelete(t *testing.T) {
	store := storage.NewMemoryStorage()

	// Create temp archive directory
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.RetentionDays = 7
	config.ArchiveBeforeDelete = true
	config.ArchivePath = tmpDir

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Store old records
	records := []*evidence.EvidenceRecord{
		{
			ID:          "old-1",
			RequestID:   "req-old-1",
			RequestTime: now.AddDate(0, 0, -10),
			Model:       "gpt-4",
		},
		{
			ID:          "old-2",
			RequestID:   "req-old-2",
			RequestTime: now.AddDate(0, 0, -8),
			Model:       "gpt-4",
		},
	}

	for _, record := range records {
		_ = store.Store(ctx, record)
	}

	// Run pruner
	deleted, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted records, got %d", deleted)
	}

	// Verify archive file was created
	files, err := filepath.Glob(filepath.Join(tmpDir, "evidence-*.json"))
	if err != nil {
		t.Fatalf("Failed to list archive files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 archive file, got %d", len(files))
	}

	// Verify archive file exists and has content
	if len(files) > 0 {
		stat, err := os.Stat(files[0])
		if err != nil {
			t.Fatalf("Failed to stat archive file: %v", err)
		}

		if stat.Size() == 0 {
			t.Error("Archive file is empty")
		}

		t.Logf("Archive file created: %s (size: %d bytes)", files[0], stat.Size())
	}
}

// TestPruner_NoRecordsToDelete tests pruning when no records match.
func TestPruner_NoRecordsToDelete(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RetentionDays = 7

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Store only recent records
	records := []*evidence.EvidenceRecord{
		{
			ID:          "recent-1",
			RequestID:   "req-recent-1",
			RequestTime: now.AddDate(0, 0, -1),
			Model:       "gpt-4",
		},
		{
			ID:          "recent-2",
			RequestID:   "req-recent-2",
			RequestTime: now.AddDate(0, 0, -2),
			Model:       "gpt-4",
		},
	}

	for _, record := range records {
		_ = store.Store(ctx, record)
	}

	// Run pruner
	deleted, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	// Should delete nothing
	if deleted != 0 {
		t.Errorf("Expected 0 deleted records, got %d", deleted)
	}

	// Verify all records remain
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 2 {
		t.Errorf("Expected 2 records to remain, got %d", count)
	}
}

// TestPruner_EmptyStorage tests pruning empty storage.
func TestPruner_EmptyStorage(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RetentionDays = 7

	pruner := NewPruner(store, config)

	ctx := context.Background()

	// Run pruner on empty storage
	deleted, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	if deleted != 0 {
		t.Errorf("Expected 0 deleted records from empty storage, got %d", deleted)
	}
}

// TestPruner_CustomRetentionPeriod tests various retention periods.
func TestPruner_CustomRetentionPeriod(t *testing.T) {
	tests := []struct {
		name          string
		retentionDays int
		recordAge     int
		shouldDelete  bool
	}{
		{
			name:          "30 day retention - 35 days old",
			retentionDays: 30,
			recordAge:     35,
			shouldDelete:  true,
		},
		{
			name:          "30 day retention - 25 days old",
			retentionDays: 30,
			recordAge:     25,
			shouldDelete:  false,
		},
		{
			name:          "90 day retention - 100 days old",
			retentionDays: 90,
			recordAge:     100,
			shouldDelete:  true,
		},
		{
			name:          "1 day retention - 2 days old",
			retentionDays: 1,
			recordAge:     2,
			shouldDelete:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storage.NewMemoryStorage()
			config := DefaultConfig()
			config.RetentionDays = tt.retentionDays

			pruner := NewPruner(store, config)

			ctx := context.Background()
			now := time.Now()

			// Store a record with specified age
			record := &evidence.EvidenceRecord{
				ID:          "test-record",
				RequestID:   "req-test",
				RequestTime: now.AddDate(0, 0, -tt.recordAge),
				Model:       "gpt-4",
			}

			_ = store.Store(ctx, record)

			// Run pruner
			deleted, err := pruner.Prune(ctx)
			if err != nil {
				t.Fatalf("Prune() failed: %v", err)
			}

			if tt.shouldDelete && deleted != 1 {
				t.Errorf("Expected record to be deleted, but got deleted count: %d", deleted)
			}

			if !tt.shouldDelete && deleted != 0 {
				t.Errorf("Expected record to remain, but got deleted count: %d", deleted)
			}
		})
	}
}

// TestPruner_ArchiveDirectoryCreation tests that archive directory is created if missing.
func TestPruner_ArchiveDirectoryCreation(t *testing.T) {
	store := storage.NewMemoryStorage()

	// Use a nested temp directory that doesn't exist yet
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "nested", "archives")

	config := DefaultConfig()
	config.RetentionDays = 7
	config.ArchiveBeforeDelete = true
	config.ArchivePath = archivePath

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Store an old record
	record := &evidence.EvidenceRecord{
		ID:          "old-record",
		RequestID:   "req-old",
		RequestTime: now.AddDate(0, 0, -10),
		Model:       "gpt-4",
	}

	_ = store.Store(ctx, record)

	// Run pruner (should create directory)
	_, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive directory was not created")
	}
}

// TestPruner_NoArchiveWhenNoRecords tests that no archive is created when no records match.
func TestPruner_NoArchiveWhenNoRecords(t *testing.T) {
	store := storage.NewMemoryStorage()

	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.RetentionDays = 7
	config.ArchiveBeforeDelete = true
	config.ArchivePath = tmpDir

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Store only recent records
	record := &evidence.EvidenceRecord{
		ID:          "recent-record",
		RequestID:   "req-recent",
		RequestTime: now.AddDate(0, 0, -1),
		Model:       "gpt-4",
	}

	_ = store.Store(ctx, record)

	// Run pruner
	_, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	// Verify no archive file was created
	files, _ := filepath.Glob(filepath.Join(tmpDir, "evidence-*.json"))
	if len(files) != 0 {
		t.Errorf("Expected no archive files, got %d", len(files))
	}
}

// BenchmarkPruner_Prune benchmarks the pruning operation.
func BenchmarkPruner_Prune(b *testing.B) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RetentionDays = 7
	config.ArchiveBeforeDelete = false

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Pre-populate with 1000 records (500 old, 500 recent)
	for i := 0; i < 1000; i++ {
		age := -5 // Recent
		if i < 500 {
			age = -10 // Old
		}

		record := &evidence.EvidenceRecord{
			ID:          "record-" + string(rune(i)),
			RequestID:   "req-" + string(rune(i)),
			RequestTime: now.AddDate(0, 0, age),
			Model:       "gpt-4",
		}

		_ = store.Store(ctx, record)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Restore deleted records for next iteration
		for j := 0; j < 500; j++ {
			record := &evidence.EvidenceRecord{
				ID:          "record-" + string(rune(j)),
				RequestID:   "req-" + string(rune(j)),
				RequestTime: now.AddDate(0, 0, -10),
				Model:       "gpt-4",
			}
			_ = store.Store(ctx, record)
		}
		b.StartTimer()

		_, _ = pruner.Prune(ctx)
	}
}

// Test Pruner_PruneByCount tests count-based pruning.
func TestPruner_PruneByCount(t *testing.T) {
	tests := []struct {
		name           string
		maxRecords     int64
		existingCount  int
		expectedDelete int64
	}{
		{
			name:           "within limit - no deletion",
			maxRecords:     100,
			existingCount:  50,
			expectedDelete: 0,
		},
		{
			name:           "at limit - no deletion",
			maxRecords:     100,
			existingCount:  100,
			expectedDelete: 0,
		},
		{
			name:           "exceeds by 1 - delete oldest",
			maxRecords:     100,
			existingCount:  101,
			expectedDelete: 1,
		},
		{
			name:           "exceeds by many - delete oldest batch",
			maxRecords:     100,
			existingCount:  150,
			expectedDelete: 50,
		},
		{
			name:           "unlimited - no deletion",
			maxRecords:     0, // 0 = unlimited
			existingCount:  1000,
			expectedDelete: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storage.NewMemoryStorage()
			config := DefaultConfig()
			config.RetentionDays = 0 // Disable age-based pruning
			config.MaxRecords = tt.maxRecords
			config.ArchiveBeforeDelete = false

			pruner := NewPruner(store, config)

			ctx := context.Background()
			now := time.Now()

			// Insert test records with incrementing timestamps
			for i := 0; i < tt.existingCount; i++ {
				record := &evidence.EvidenceRecord{
					ID:          "test-" + string(rune(i)),
					RequestID:   "req-" + string(rune(i)),
					RequestTime: now.Add(time.Duration(i) * time.Second),
					Model:       "gpt-4",
				}
				if err := store.Store(ctx, record); err != nil {
					t.Fatalf("failed to store record: %v", err)
				}
			}

			// Run pruning
			deleted, err := pruner.Prune(ctx)
			if err != nil {
				t.Fatalf("Prune() failed: %v", err)
			}

			if deleted != tt.expectedDelete {
				t.Errorf("deleted = %d, want %d", deleted, tt.expectedDelete)
			}

			// Verify remaining count
			remaining, err := store.Count(ctx, &evidence.Query{})
			if err != nil {
				t.Fatalf("Count() failed: %v", err)
			}

			expectedRemaining := int64(tt.existingCount) - tt.expectedDelete
			if tt.maxRecords > 0 && remaining > tt.maxRecords {
				t.Errorf("remaining count %d exceeds max %d", remaining, tt.maxRecords)
			}
			if remaining != expectedRemaining {
				t.Errorf("remaining = %d, want %d", remaining, expectedRemaining)
			}
		})
	}
}

// TestPruner_BothAgeAndCount tests that both age-based and count-based pruning work together.
func TestPruner_BothAgeAndCount(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RetentionDays = 90 // Delete >90 days old
	config.MaxRecords = 80    // Keep max 80 records
	config.ArchiveBeforeDelete = false

	pruner := NewPruner(store, config)

	ctx := context.Background()
	now := time.Now()

	// Insert 50 records that are 100 days old (should be deleted by age)
	for i := 0; i < 50; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "old-" + string(rune(i)),
			RequestID:   "req-old-" + string(rune(i)),
			RequestTime: now.AddDate(0, 0, -100), // 100 days old
			Model:       "gpt-4",
		}
		if err := store.Store(ctx, record); err != nil {
			t.Fatalf("failed to store record: %v", err)
		}
	}

	// Insert 100 recent records (should be kept, but 20 deleted by count limit)
	for i := 0; i < 100; i++ {
		record := &evidence.EvidenceRecord{
			ID:          "recent-" + string(rune(i)),
			RequestID:   "req-recent-" + string(rune(i)),
			RequestTime: now.Add(time.Duration(i) * time.Second),
			Model:       "gpt-4",
		}
		if err := store.Store(ctx, record); err != nil {
			t.Fatalf("failed to store record: %v", err)
		}
	}

	// Verify initial count
	initialCount, _ := store.Count(ctx, &evidence.Query{})
	if initialCount != 150 {
		t.Fatalf("Expected 150 initial records, got %d", initialCount)
	}

	// Run full prune
	deleted, err := pruner.Prune(ctx)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}

	// Should delete:
	// - 50 old records (age-based)
	// - 20 recent records (count-based: 100 - 80 = 20)
	// Total: 70 deleted
	expectedDeleted := int64(70)
	if deleted != expectedDeleted {
		t.Errorf("deleted = %d, want %d", deleted, expectedDeleted)
	}

	// Verify final count is at max_records
	remaining, _ := store.Count(ctx, &evidence.Query{})
	if remaining != 80 {
		t.Errorf("remaining = %d, want 80", remaining)
	}

	// Verify no old records remain
	allRecords, _ := store.Query(ctx, &evidence.Query{})
	for _, r := range allRecords {
		age := now.Sub(r.RequestTime).Hours() / 24
		if age > 90 {
			t.Errorf("Record %s is %f days old, should have been deleted", r.ID, age)
		}
	}
}

// BenchmarkPruner_PruneWithArchive benchmarks pruning with archiving enabled.
func BenchmarkPruner_PruneWithArchive(b *testing.B) {
	tmpDir := b.TempDir()

	config := DefaultConfig()
	config.RetentionDays = 7
	config.ArchiveBeforeDelete = true
	config.ArchivePath = tmpDir

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store := storage.NewMemoryStorage()
		pruner := NewPruner(store, config)

		// Add 100 old records
		for j := 0; j < 100; j++ {
			record := &evidence.EvidenceRecord{
				ID:          "record-" + string(rune(j)),
				RequestID:   "req-" + string(rune(j)),
				RequestTime: now.AddDate(0, 0, -10),
				Model:       "gpt-4",
			}
			_ = store.Store(ctx, record)
		}
		b.StartTimer()

		_, _ = pruner.Prune(ctx)
	}
}
