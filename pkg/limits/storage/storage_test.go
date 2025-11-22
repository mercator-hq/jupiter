package storage

import (
	"context"
	"testing"
	"time"
)

func TestMemoryBackend_SaveAndLoad(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

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

func TestMemoryBackend_LoadNonExistent(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

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

func TestMemoryBackend_Update(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

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

func TestMemoryBackend_Delete(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

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

func TestMemoryBackend_List(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

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

func TestMemoryBackend_Cleanup(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

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

func TestMemoryBackend_MaxEntries(t *testing.T) {
	// Create backend with small max entries
	backend := NewMemoryBackendWithConfig(MemoryBackendConfig{
		MaxEntries:      3,
		CleanupInterval: time.Hour, // Disable cleanup for this test
		RetentionPeriod: time.Hour,
	})
	defer backend.Close()

	ctx := context.Background()

	// Add 5 entries (exceeds max of 3)
	for i := 1; i <= 5; i++ {
		state := &LimitState{
			Identifier: "key-" + string(rune('0'+i)),
			Dimension:  "api_key",
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
		err := backend.Save(ctx, state)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// Should have at most 3 entries (max entries)
	size := backend.Size()
	if size > 3 {
		t.Errorf("Expected at most 3 entries, got %d", size)
	}
}

func TestMemoryBackend_Concurrent(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	const numGoroutines = 10
	const numOperations = 100

	// Run concurrent save/load operations
	done := make(chan bool)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
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

				// Save
				_ = backend.Save(ctx, state)

				// Load
				_, _ = backend.Load(ctx, "concurrent-key", "api_key")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify state still exists and is valid
	loaded, err := backend.Load(ctx, "concurrent-key", "api_key")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("Expected state to exist after concurrent operations")
	}
}

func TestMemoryBackend_Validation(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		state       *LimitState
		identifier  string
		dimension   string
		expectError bool
	}{
		{
			name:        "nil state",
			state:       nil,
			expectError: true,
		},
		{
			name: "empty identifier",
			state: &LimitState{
				Identifier: "",
				Dimension:  "api_key",
			},
			expectError: true,
		},
		{
			name: "empty dimension",
			state: &LimitState{
				Identifier: "key-123",
				Dimension:  "",
			},
			expectError: true,
		},
		{
			name: "valid state",
			state: &LimitState{
				Identifier: "key-123",
				Dimension:  "api_key",
			},
			expectError: false,
		},
		{
			name:        "load with empty identifier",
			identifier:  "",
			dimension:   "api_key",
			expectError: true,
		},
		{
			name:        "load with empty dimension",
			identifier:  "key-123",
			dimension:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state != nil {
				err := backend.Save(ctx, tt.state)
				if tt.expectError && err == nil {
					t.Error("Expected error, got nil")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if tt.identifier != "" || tt.dimension != "" {
				_, err := backend.Load(ctx, tt.identifier, tt.dimension)
				if tt.expectError && err == nil {
					t.Error("Expected error, got nil")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func BenchmarkMemoryBackend_Save(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	state := &LimitState{
		Identifier: "bench-key",
		Dimension:  "api_key",
		RateLimit: &RateLimitState{
			TokenBucket: &TokenBucketState{
				Capacity: 100,
				Tokens:   50,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backend.Save(ctx, state)
	}
}

func BenchmarkMemoryBackend_Load(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	state := &LimitState{
		Identifier: "bench-key",
		Dimension:  "api_key",
	}
	_ = backend.Save(ctx, state)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.Load(ctx, "bench-key", "api_key")
	}
}

func BenchmarkMemoryBackend_Concurrent(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	ctx := context.Background()
	state := &LimitState{
		Identifier: "bench-key",
		Dimension:  "api_key",
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = backend.Save(ctx, state)
			_, _ = backend.Load(ctx, "bench-key", "api_key")
		}
	})
}
