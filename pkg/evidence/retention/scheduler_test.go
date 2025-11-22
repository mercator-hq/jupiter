package retention

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/storage"
)

func TestScheduler_Start(t *testing.T) {
	tests := []struct {
		name        string
		schedule    string
		wantRunning bool
		wantError   bool
	}{
		{
			name:        "valid daily schedule",
			schedule:    "0 3 * * *",
			wantRunning: true,
			wantError:   false,
		},
		{
			name:        "valid hourly schedule",
			schedule:    "0 * * * *",
			wantRunning: true,
			wantError:   false,
		},
		{
			name:        "empty schedule - no error, not running",
			schedule:    "",
			wantRunning: false,
			wantError:   false,
		},
		{
			name:        "invalid schedule",
			schedule:    "invalid cron",
			wantRunning: false,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memStorage := storage.NewMemoryStorage()

			pruner := &Pruner{
				storage: memStorage,
				config: &Config{
					PruneSchedule: tt.schedule,
					RetentionDays: 90,
				},
				logger: slog.Default(),
			}

			scheduler := NewScheduler(pruner)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := scheduler.Start(ctx)

			if (err != nil) != tt.wantError {
				t.Errorf("Start() error = %v, wantError %v", err, tt.wantError)
			}

			if scheduler.IsRunning() != tt.wantRunning {
				t.Errorf("IsRunning() = %v, want %v",
					scheduler.IsRunning(), tt.wantRunning)
			}

			if tt.wantRunning {
				next := scheduler.NextRun()
				if next == nil {
					t.Error("NextRun() returned nil for running scheduler")
				} else {
					t.Logf("Next run: %s", next)
				}
			}

			scheduler.Stop()

			if scheduler.IsRunning() {
				t.Error("scheduler still running after Stop()")
			}
		})
	}
}

func TestScheduler_ActualPruning(t *testing.T) {
	// Integration test: verify pruning actually runs
	// Use very short interval for testing

	memStorage := storage.NewMemoryStorage()

	// Insert some old records
	oldTime := time.Now().AddDate(0, 0, -100)
	for i := 0; i < 10; i++ {
		record := &evidence.EvidenceRecord{
			ID:             fmt.Sprintf("old-%d", i),
			RequestTime:    oldTime,
			RequestID:      fmt.Sprintf("req-%d", i),
			PolicyDecision: "allow",
		}
		if err := memStorage.Store(context.Background(), record); err != nil {
			t.Fatalf("failed to store record: %v", err)
		}
	}

	// Create pruner with 90-day retention
	pruner := &Pruner{
		storage: memStorage,
		config: &Config{
			RetentionDays:       90,
			PruneSchedule:       "*/1 * * * *", // Every minute (for testing)
			ArchiveBeforeDelete: false,
		},
		logger: slog.Default(),
	}

	scheduler := NewScheduler(pruner)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Start scheduler
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer scheduler.Stop()

	// Wait for next run (max 70 seconds)
	next := scheduler.NextRun()
	if next == nil {
		t.Fatal("NextRun() returned nil")
	}

	waitDuration := time.Until(*next) + 5*time.Second
	if waitDuration > 70*time.Second {
		t.Skip("Next run too far in future for test")
	}

	t.Logf("Waiting %s for pruning to run...", waitDuration)
	time.Sleep(waitDuration)

	// Verify records were pruned
	count, err := memStorage.Count(context.Background(), &evidence.Query{})
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 records after pruning, got %d", count)
	}
}

func TestScheduler_GracefulShutdown(t *testing.T) {
	memStorage := storage.NewMemoryStorage()

	pruner := &Pruner{
		storage: memStorage,
		config: &Config{
			PruneSchedule: "0 3 * * *",
			RetentionDays: 90,
		},
		logger: slog.Default(),
	}

	scheduler := NewScheduler(pruner)

	ctx, cancel := context.WithCancel(context.Background())

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Cancel context - should trigger shutdown
	cancel()

	// Wait a bit for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	if scheduler.IsRunning() {
		t.Error("scheduler still running after context cancelled")
	}
}

func TestScheduler_NextRun(t *testing.T) {
	memStorage := storage.NewMemoryStorage()

	pruner := &Pruner{
		storage: memStorage,
		config: &Config{
			PruneSchedule: "0 3 * * *", // Daily at 3 AM
			RetentionDays: 90,
		},
		logger: slog.Default(),
	}

	scheduler := NewScheduler(pruner)

	// Before starting, NextRun should return nil
	if next := scheduler.NextRun(); next != nil {
		t.Errorf("NextRun() before start = %v, want nil", next)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer scheduler.Stop()

	// After starting, NextRun should return a time
	next := scheduler.NextRun()
	if next == nil {
		t.Fatal("NextRun() after start returned nil")
	}

	// Verify it's in the future
	if !next.After(time.Now()) {
		t.Errorf("NextRun() = %v, want time in future", next)
	}

	t.Logf("Next scheduled run: %s", next)
}

func TestScheduler_MultipleStartStop(t *testing.T) {
	memStorage := storage.NewMemoryStorage()

	pruner := &Pruner{
		storage: memStorage,
		config: &Config{
			PruneSchedule: "0 * * * *",
			RetentionDays: 90,
		},
		logger: slog.Default(),
	}

	scheduler := NewScheduler(pruner)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start and stop multiple times
	for i := 0; i < 3; i++ {
		if err := scheduler.Start(ctx); err != nil {
			t.Fatalf("Start() iteration %d failed: %v", i, err)
		}

		if !scheduler.IsRunning() {
			t.Errorf("IsRunning() = false after Start() iteration %d", i)
		}

		scheduler.Stop()

		if scheduler.IsRunning() {
			t.Errorf("IsRunning() = true after Stop() iteration %d", i)
		}

		// Give it time to clean up
		time.Sleep(50 * time.Millisecond)
	}
}

func TestPruner_StartStop(t *testing.T) {
	// Test the Pruner's Start/Stop methods (which delegate to scheduler)
	memStorage := storage.NewMemoryStorage()

	pruner := NewPruner(memStorage, &Config{
		PruneSchedule: "0 3 * * *",
		RetentionDays: 90,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pruner
	if err := pruner.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify scheduler is running
	if !pruner.scheduler.IsRunning() {
		t.Error("scheduler not running after Pruner.Start()")
	}

	// Check next pruning time
	next := pruner.NextPruning()
	if next == nil {
		t.Error("NextPruning() returned nil")
	} else {
		t.Logf("Next pruning: %s", next)
	}

	// Stop pruner
	pruner.Stop()

	// Verify scheduler is stopped
	if pruner.scheduler.IsRunning() {
		t.Error("scheduler still running after Pruner.Stop()")
	}
}
