package storage

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestSQLiteBackend_SaveAndLoad tests basic save and load operations.
func TestSQLiteBackend_SaveAndLoad(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create test state
	state := &LimitState{
		Identifier: "api-key-123",
		Dimension:  "api_key",
		RateLimit: &RateLimitState{
			TokenBucket: &TokenBucketState{
				Capacity:   100,
				Tokens:     50,
				RefillRate: 10,
				LastRefill: time.Now(),
			},
		},
		Budget: &BudgetState{
			TotalSpent: 25.50,
		},
	}

	// Save state
	err := backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load state
	loaded, err := backend.Load(ctx, "api-key-123", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected state, got nil")
	}

	// Verify loaded state
	if loaded.Identifier != state.Identifier {
		t.Errorf("Expected identifier %s, got %s", state.Identifier, loaded.Identifier)
	}
	if loaded.Dimension != state.Dimension {
		t.Errorf("Expected dimension %s, got %s", state.Dimension, loaded.Dimension)
	}
	if loaded.RateLimit.TokenBucket.Capacity != 100 {
		t.Errorf("Expected capacity 100, got %d", loaded.RateLimit.TokenBucket.Capacity)
	}
	if loaded.Budget.TotalSpent != 25.50 {
		t.Errorf("Expected total spent 25.50, got %.2f", loaded.Budget.TotalSpent)
	}
}

// TestSQLiteBackend_LoadNonExistent tests loading a non-existent state.
func TestSQLiteBackend_LoadNonExistent(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Load non-existent state
	loaded, err := backend.Load(ctx, "nonexistent", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded != nil {
		t.Errorf("Expected nil for non-existent state, got %v", loaded)
	}
}

// TestSQLiteBackend_Update tests updating existing state.
func TestSQLiteBackend_Update(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial state
	state := &LimitState{
		Identifier: "api-key-123",
		Dimension:  "api_key",
		RateLimit: &RateLimitState{
			TokenBucket: &TokenBucketState{
				Tokens: 50,
			},
		},
	}

	// Save initial state
	err := backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Update state
	state.RateLimit.TokenBucket.Tokens = 75
	err = backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	loaded, err := backend.Load(ctx, "api-key-123", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.RateLimit.TokenBucket.Tokens != 75 {
		t.Errorf("Expected tokens 75, got %d", loaded.RateLimit.TokenBucket.Tokens)
	}
}

// TestSQLiteBackend_Delete tests deleting state.
func TestSQLiteBackend_Delete(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create and save state
	state := &LimitState{
		Identifier: "api-key-123",
		Dimension:  "api_key",
	}
	err := backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify it exists
	loaded, err := backend.Load(ctx, "api-key-123", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected state to exist")
	}

	// Delete state
	err = backend.Delete(ctx, "api-key-123", "api_key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	loaded, err = backend.Load(ctx, "api-key-123", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded != nil {
		t.Error("Expected state to be deleted")
	}
}

// TestSQLiteBackend_List tests listing states by dimension.
func TestSQLiteBackend_List(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple states for different dimensions
	states := []*LimitState{
		{Identifier: "key-1", Dimension: "api_key"},
		{Identifier: "key-2", Dimension: "api_key"},
		{Identifier: "user-1", Dimension: "user"},
		{Identifier: "team-1", Dimension: "team"},
	}

	for _, state := range states {
		err := backend.Save(ctx, state)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// List API key dimension
	apiKeyStates, err := backend.List(ctx, "api_key")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(apiKeyStates) != 2 {
		t.Errorf("Expected 2 API key states, got %d", len(apiKeyStates))
	}

	// List user dimension
	userStates, err := backend.List(ctx, "user")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(userStates) != 1 {
		t.Errorf("Expected 1 user state, got %d", len(userStates))
	}

	// List non-existent dimension
	emptyStates, err := backend.List(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(emptyStates) != 0 {
		t.Errorf("Expected 0 states for nonexistent dimension, got %d", len(emptyStates))
	}
}

// TestSQLiteBackend_Cleanup tests cleaning up old entries.
func TestSQLiteBackend_Cleanup(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create states with different timestamps
	oldState := &LimitState{
		Identifier:  "old-key",
		Dimension:   "api_key",
		LastUpdated: time.Now().Add(-48 * time.Hour), // 2 days ago
	}

	recentState := &LimitState{
		Identifier:  "recent-key",
		Dimension:   "api_key",
		LastUpdated: time.Now().Add(-1 * time.Hour), // 1 hour ago
	}

	err := backend.Save(ctx, oldState)
	if err != nil {
		t.Fatalf("Save old state failed: %v", err)
	}

	err = backend.Save(ctx, recentState)
	if err != nil {
		t.Fatalf("Save recent state failed: %v", err)
	}

	// Cleanup entries older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	deleted, err := backend.Cleanup(ctx, cutoff)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected to delete 1 entry, deleted %d", deleted)
	}

	// Verify old state is gone
	loaded, err := backend.Load(ctx, "old-key", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded != nil {
		t.Error("Expected old state to be cleaned up")
	}

	// Verify recent state still exists
	loaded, err = backend.Load(ctx, "recent-key", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Error("Expected recent state to still exist")
	}
}

// TestSQLiteBackend_Persistence tests that data persists across backend restarts.
func TestSQLiteBackend_Persistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persistence.db")

	// Create backend and save state
	backend1, err := NewSQLiteBackend(dbPath)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()
	state := &LimitState{
		Identifier: "persistent-key",
		Dimension:  "api_key",
		RateLimit: &RateLimitState{
			TokenBucket: &TokenBucketState{
				Tokens: 42,
			},
		},
	}

	err = backend1.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Close first backend
	err = backend1.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Open new backend with same database
	backend2, err := NewSQLiteBackend(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen backend: %v", err)
	}
	defer backend2.Close()

	// Verify state persisted
	loaded, err := backend2.Load(ctx, "persistent-key", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected persisted state, got nil")
	}
	if loaded.RateLimit.TokenBucket.Tokens != 42 {
		t.Errorf("Expected tokens 42, got %d", loaded.RateLimit.TokenBucket.Tokens)
	}
}

// TestSQLiteBackend_Concurrent tests concurrent access.
func TestSQLiteBackend_Concurrent(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()
	const numGoroutines = 10
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent saves
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				state := &LimitState{
					Identifier: "concurrent-key",
					Dimension:  "api_key",
					RateLimit: &RateLimitState{
						TokenBucket: &TokenBucketState{
							Tokens: int64(id*numOperations + j),
						},
					},
				}
				if err := backend.Save(ctx, state); err != nil {
					t.Errorf("Concurrent save failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state exists
	loaded, err := backend.Load(ctx, "concurrent-key", "api_key")
	if err != nil {
		t.Fatalf("Load after concurrent operations failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected state after concurrent operations")
	}
}

// TestSQLiteBackend_Validation tests input validation.
func TestSQLiteBackend_Validation(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name      string
		operation func() error
		wantErr   bool
	}{
		{
			name: "nil state",
			operation: func() error {
				return backend.Save(ctx, nil)
			},
			wantErr: true,
		},
		{
			name: "empty identifier",
			operation: func() error {
				return backend.Save(ctx, &LimitState{
					Identifier: "",
					Dimension:  "api_key",
				})
			},
			wantErr: true,
		},
		{
			name: "empty dimension",
			operation: func() error {
				return backend.Save(ctx, &LimitState{
					Identifier: "key",
					Dimension:  "",
				})
			},
			wantErr: true,
		},
		{
			name: "valid state",
			operation: func() error {
				return backend.Save(ctx, &LimitState{
					Identifier: "valid-key",
					Dimension:  "api_key",
				})
			},
			wantErr: false,
		},
		{
			name: "load with empty identifier",
			operation: func() error {
				_, err := backend.Load(ctx, "", "api_key")
				return err
			},
			wantErr: true,
		},
		{
			name: "load with empty dimension",
			operation: func() error {
				_, err := backend.Load(ctx, "key", "")
				return err
			},
			wantErr: true,
		},
		{
			name: "delete with empty identifier",
			operation: func() error {
				return backend.Delete(ctx, "", "api_key")
			},
			wantErr: true,
		},
		{
			name: "delete with empty dimension",
			operation: func() error {
				return backend.Delete(ctx, "key", "")
			},
			wantErr: true,
		},
		{
			name: "list with empty dimension",
			operation: func() error {
				_, err := backend.List(ctx, "")
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestSQLiteBackend_EmptyPath tests creating backend with empty path.
func TestSQLiteBackend_EmptyPath(t *testing.T) {
	backend, err := NewSQLiteBackend("")
	if err == nil {
		backend.Close()
		t.Fatal("Expected error for empty path, got nil")
	}
}

// TestSQLiteBackend_ComplexState tests saving and loading complex nested state.
func TestSQLiteBackend_ComplexState(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// Create complex state with all fields populated
	now := time.Now()
	state := &LimitState{
		Identifier:  "complex-key",
		Dimension:   "api_key",
		LastUpdated: now,
		CreatedAt:   now.Add(-1 * time.Hour),
		RateLimit: &RateLimitState{
			TokenBucket: &TokenBucketState{
				Capacity:   1000,
				Tokens:     750,
				RefillRate: 100,
				LastRefill: now.Add(-5 * time.Minute),
			},
			SlidingWindow: &SlidingWindowState{
				Window:     60 * time.Second,
				BucketSize: 1 * time.Second,
				Buckets: []WindowBucket{
					{Timestamp: now, Value: 10},
					{Timestamp: now.Add(-10 * time.Second), Value: 15},
					{Timestamp: now.Add(-20 * time.Second), Value: 17},
				},
			},
			Concurrent:    15,
			MaxConcurrent: 50,
		},
		Budget: &BudgetState{
			TotalSpent: 125.75,
			HourlyBuckets: []BudgetBucket{
				{Timestamp: now, Amount: 10.50},
			},
			DailyBuckets: []BudgetBucket{
				{Timestamp: now, Amount: 75.25},
			},
			MonthlyBuckets: []BudgetBucket{
				{Timestamp: now, Amount: 105.00},
			},
		},
	}

	// Save complex state
	err := backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save complex state failed: %v", err)
	}

	// Load and verify all fields
	loaded, err := backend.Load(ctx, "complex-key", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected loaded state, got nil")
	}

	// Verify rate limit state
	if loaded.RateLimit == nil {
		t.Fatal("Expected rate limit state, got nil")
	}
	if loaded.RateLimit.TokenBucket.Capacity != 1000 {
		t.Errorf("Expected capacity 1000, got %d", loaded.RateLimit.TokenBucket.Capacity)
	}
	if loaded.RateLimit.TokenBucket.Tokens != 750 {
		t.Errorf("Expected tokens 750, got %d", loaded.RateLimit.TokenBucket.Tokens)
	}
	if loaded.RateLimit.SlidingWindow == nil {
		t.Fatal("Expected sliding window state, got nil")
	}
	if len(loaded.RateLimit.SlidingWindow.Buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(loaded.RateLimit.SlidingWindow.Buckets))
	}
	if loaded.RateLimit.Concurrent != 15 {
		t.Errorf("Expected concurrent 15, got %d", loaded.RateLimit.Concurrent)
	}

	// Verify budget state
	if loaded.Budget == nil {
		t.Fatal("Expected budget state, got nil")
	}
	if loaded.Budget.TotalSpent != 125.75 {
		t.Errorf("Expected total spent 125.75, got %.2f", loaded.Budget.TotalSpent)
	}
	if len(loaded.Budget.HourlyBuckets) != 1 {
		t.Errorf("Expected 1 hourly bucket, got %d", len(loaded.Budget.HourlyBuckets))
	}
	if len(loaded.Budget.DailyBuckets) != 1 {
		t.Errorf("Expected 1 daily bucket, got %d", len(loaded.Budget.DailyBuckets))
	}
	if len(loaded.Budget.MonthlyBuckets) != 1 {
		t.Errorf("Expected 1 monthly bucket, got %d", len(loaded.Budget.MonthlyBuckets))
	}
}

// TestSQLiteBackend_PartialState tests saving state with only rate limit or budget.
func TestSQLiteBackend_PartialState(t *testing.T) {
	backend, cleanup := newTestSQLiteBackend(t)
	defer cleanup()

	ctx := context.Background()

	// State with only rate limit
	rateLimitOnly := &LimitState{
		Identifier: "rate-only",
		Dimension:  "api_key",
		RateLimit: &RateLimitState{
			TokenBucket: &TokenBucketState{Tokens: 100},
		},
	}

	err := backend.Save(ctx, rateLimitOnly)
	if err != nil {
		t.Fatalf("Save rate limit only failed: %v", err)
	}

	loaded, err := backend.Load(ctx, "rate-only", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.RateLimit == nil {
		t.Error("Expected rate limit state")
	}
	if loaded.Budget != nil {
		t.Error("Expected nil budget state")
	}

	// State with only budget
	budgetOnly := &LimitState{
		Identifier: "budget-only",
		Dimension:  "api_key",
		Budget: &BudgetState{
			TotalSpent: 50.00,
		},
	}

	err = backend.Save(ctx, budgetOnly)
	if err != nil {
		t.Fatalf("Save budget only failed: %v", err)
	}

	loaded, err = backend.Load(ctx, "budget-only", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.RateLimit != nil {
		t.Error("Expected nil rate limit state")
	}
	if loaded.Budget == nil {
		t.Error("Expected budget state")
	}
}

// TestSQLiteBackend_CustomConfig tests creating backend with custom config.
func TestSQLiteBackend_CustomConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "custom.db")

	backend, err := NewSQLiteBackendWithConfig(SQLiteBackendConfig{
		DBPath:           dbPath,
		SnapshotInterval: 100 * time.Millisecond,
		BusyTimeout:      1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create backend with custom config: %v", err)
	}
	defer backend.Close()

	// Verify backend is functional
	ctx := context.Background()
	state := &LimitState{
		Identifier: "test-key",
		Dimension:  "api_key",
	}

	err = backend.Save(ctx, state)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := backend.Load(ctx, "test-key", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected state, got nil")
	}

	// Wait for at least one checkpoint
	time.Sleep(150 * time.Millisecond)
}

// TestSQLiteBackend_Close tests proper cleanup on close.
func TestSQLiteBackend_Close(t *testing.T) {
	backend, _ := newTestSQLiteBackend(t)

	// Close should not error
	err := backend.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Double close should not panic
	err = backend.Close()
	if err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

// newTestSQLiteBackend creates a new SQLite backend for testing with a temporary database.
func newTestSQLiteBackend(t *testing.T) (*SQLiteBackend, func()) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	backend, err := NewSQLiteBackendWithConfig(SQLiteBackendConfig{
		DBPath:           dbPath,
		SnapshotInterval: 1 * time.Hour, // Disable checkpointing for most tests
		BusyTimeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}

	cleanup := func() {
		backend.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}

	return backend, cleanup
}
