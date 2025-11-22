package retention

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages automatic retention pruning on a schedule.
// It runs the pruner at scheduled intervals (e.g., daily at 3 AM)
// using cron syntax.
type Scheduler struct {
	pruner  *Pruner
	cron    *cron.Cron
	mu      sync.Mutex
	logger  *slog.Logger
	running bool
}

// NewScheduler creates a new retention scheduler.
func NewScheduler(pruner *Pruner) *Scheduler {
	return &Scheduler{
		pruner: pruner,
		cron:   cron.New(),
		logger: slog.Default().With("component", "evidence.scheduler"),
	}
}

// Start begins the scheduled pruning based on the cron expression.
// The cron expression is read from pruner.config.PruneSchedule.
//
// Common cron expressions:
//   - "0 3 * * *"    - Daily at 3 AM
//   - "0 */6 * * *"  - Every 6 hours
//   - "0 0 * * 0"    - Weekly on Sunday at midnight
//
// If PruneSchedule is empty, the scheduler does nothing.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pruner.config.PruneSchedule == "" {
		s.logger.Info("prune schedule not configured, skipping scheduler")
		return nil
	}

	// Validate cron expression
	_, err := cron.ParseStandard(s.pruner.config.PruneSchedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule %q: %w",
			s.pruner.config.PruneSchedule, err)
	}

	// Add cron job
	_, err = s.cron.AddFunc(s.pruner.config.PruneSchedule, func() {
		s.runPruning(ctx)
	})

	if err != nil {
		return fmt.Errorf("failed to schedule pruning: %w", err)
	}

	// Start cron scheduler
	s.cron.Start()
	s.running = true

	s.logger.Info("retention scheduler started",
		"schedule", s.pruner.config.PruneSchedule,
		"retention_days", s.pruner.config.RetentionDays,
		"max_records", s.pruner.config.MaxRecords,
	)

	// Wait for context cancellation in background
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// runPruning executes a pruning cycle.
func (s *Scheduler) runPruning(ctx context.Context) {
	s.logger.Info("starting scheduled evidence pruning")

	deleted, err := s.pruner.Prune(ctx)
	if err != nil {
		s.logger.Error("scheduled pruning failed",
			"error", err,
		)
		return
	}

	if deleted > 0 {
		s.logger.Info("scheduled pruning completed",
			"deleted_count", deleted,
		)
	} else {
		s.logger.Debug("scheduled pruning completed, no records deleted")
	}
}

// Stop stops the scheduler and waits for any running jobs to complete.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil && s.running {
		ctx := s.cron.Stop()
		<-ctx.Done() // Wait for running jobs to finish
		s.running = false
		s.logger.Info("retention scheduler stopped")
	}
}

// IsRunning returns true if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.running
}

// NextRun returns the next scheduled pruning time.
func (s *Scheduler) NextRun() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron == nil {
		return nil
	}

	entries := s.cron.Entries()
	if len(entries) == 0 {
		return nil
	}

	next := entries[0].Next
	return &next
}
