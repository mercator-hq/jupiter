package git

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"
)

// ReloadCallback is called when policies need reloading.
// It receives the full path to the policy directory and should
// load and validate all policies from that path.
// If validation fails, it should return an error to trigger rollback.
type ReloadCallback func(policyPath string) error

// Watcher monitors a Git repository for changes and triggers policy reloads.
// It uses polling to periodically check for new commits and intelligently
// reloads policies only when policy files (.mpl, .yaml, .yml) are changed.
//
// The watcher implements debouncing to prevent reload storms from multiple
// rapid commits, and provides error recovery by keeping last-known-good
// policies active if validation fails.
//
// Basic usage:
//
//	watcher := git.NewWatcher(repo, 30*time.Second, 10*time.Second, reloadFn)
//	if err := watcher.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer watcher.Stop()
type Watcher struct {
	repo          *Repository
	pollInterval  time.Duration
	pollTimeout   time.Duration
	stopCh        chan struct{}
	reloadFn      ReloadCallback
	lastCommitSHA string
	mu            sync.RWMutex
	running       bool
	debounceTimer *time.Timer
	debounceMu    sync.Mutex
	logger        *slog.Logger
	metrics       *WatcherMetrics
}

// WatcherMetrics tracks watcher operation metrics.
type WatcherMetrics struct {
	PollCount         int64
	SuccessfulReloads int64
	FailedReloads     int64
	LastReloadTime    time.Time
	LastReloadDur     time.Duration
	SkippedPolls      int64 // Non-policy file changes
}

// NewWatcher creates a new change watcher for the given repository.
// The watcher will poll for changes at the specified interval and use
// the timeout for Git operations. The reloadFn callback is called when
// policy files change.
func NewWatcher(repo *Repository, interval, timeout time.Duration, reloadFn ReloadCallback) *Watcher {
	return &Watcher{
		repo:         repo,
		pollInterval: interval,
		pollTimeout:  timeout,
		reloadFn:     reloadFn,
		stopCh:       make(chan struct{}),
		logger:       slog.Default(),
		metrics:      &WatcherMetrics{},
	}
}

// SetLogger sets a custom logger for the watcher.
func (w *Watcher) SetLogger(logger *slog.Logger) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.logger = logger
}

// Start begins watching for changes in the repository.
// It starts a background goroutine that polls for changes at the configured interval.
// The context is used for cancellation - when the context is cancelled, the watcher stops.
// Returns an error if the watcher is already running or if the initial commit cannot be read.
func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher already running")
	}

	// Get initial commit SHA
	commit, err := w.repo.GetCurrentCommit()
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to get initial commit: %w", err)
	}
	w.lastCommitSHA = commit.SHA
	w.running = true
	w.mu.Unlock()

	w.logger.Info("watcher started",
		"poll_interval", w.pollInterval,
		"initial_commit", w.lastCommitSHA[:8])

	// Start polling loop in background
	go w.pollLoop(ctx)

	return nil
}

// Stop gracefully stops the watcher.
// It signals the polling loop to stop and waits for it to exit.
// Returns an error if the watcher is not running.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return fmt.Errorf("watcher not running")
	}

	w.logger.Info("stopping watcher")
	close(w.stopCh)
	w.running = false

	// Cancel any pending debounce timer
	w.debounceMu.Lock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceMu.Unlock()

	return nil
}

// IsRunning returns true if the watcher is currently running.
func (w *Watcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// pollLoop runs the main change detection loop.
// It checks for changes at regular intervals and triggers reloads when needed.
func (w *Watcher) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("watcher stopped by context cancellation")
			return
		case <-w.stopCh:
			w.logger.Info("watcher stopped by Stop()")
			return
		case <-ticker.C:
			if err := w.checkForChanges(ctx); err != nil {
				w.logger.Error("error checking for changes",
					"error", err)
			}
		}
	}
}

// checkForChanges checks for new commits and reloads if needed.
// It implements smart filtering to only reload when policy files change,
// and uses debouncing to handle multiple rapid commits.
func (w *Watcher) checkForChanges(ctx context.Context) error {
	w.metrics.PollCount++

	// Create timeout context for pull operation
	pullCtx, cancel := context.WithTimeout(ctx, w.pollTimeout)
	defer cancel()

	// Pull latest changes
	result, err := w.repo.Pull(pullCtx)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	// No changes
	if !result.HadChanges {
		return nil
	}

	w.logger.Info("detected changes",
		"from_sha", result.FromSHA[:8],
		"to_sha", result.ToSHA[:8],
		"changed_files", len(result.ChangedFiles))

	// Check if policy files changed
	hasPolicyChanges := w.hasPolicyFileChanges(result.ChangedFiles)

	if !hasPolicyChanges {
		w.metrics.SkippedPolls++
		w.logger.Info("non-policy files changed, skipping reload",
			"changed_files", result.ChangedFiles)
		// Update last commit SHA even though we're not reloading
		// This prevents checking the same commit repeatedly
		w.mu.Lock()
		w.lastCommitSHA = result.ToSHA
		w.mu.Unlock()
		return nil
	}

	// Debounce: wait a bit to see if more changes are coming
	w.debounceReload(ctx, result.ToSHA)

	return nil
}

// hasPolicyFileChanges checks if any policy files changed.
func (w *Watcher) hasPolicyFileChanges(files []string) bool {
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext == ".mpl" || ext == ".yaml" || ext == ".yml" {
			return true
		}
	}
	return false
}

// debounceReload implements debouncing for reload operations.
// If multiple changes happen within 100ms, only the last one triggers a reload.
func (w *Watcher) debounceReload(ctx context.Context, newSHA string) {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	// Cancel previous timer if exists
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	// Create new timer
	w.debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
		if err := w.performReload(ctx, newSHA); err != nil {
			w.logger.Error("reload failed", "error", err)
		}
	})
}

// performReload executes the actual reload operation.
// It calls the reload callback and handles errors with rollback.
func (w *Watcher) performReload(ctx context.Context, newSHA string) error {
	start := time.Now()
	defer func() {
		w.metrics.LastReloadDur = time.Since(start)
		w.metrics.LastReloadTime = time.Now()
	}()

	w.logger.Info("reloading policies", "commit_sha", newSHA[:8])

	// Get policy path
	policyPath := w.repo.GetPolicyPath()

	// Call reload callback
	if err := w.reloadFn(policyPath); err != nil {
		w.metrics.FailedReloads++
		w.logger.Error("policy validation failed, attempting rollback",
			"error", err,
			"current_sha", newSHA[:8],
			"rollback_to", w.lastCommitSHA[:8])

		// Attempt rollback to previous commit
		if rollbackErr := w.rollbackToPrevious(ctx, w.lastCommitSHA); rollbackErr != nil {
			w.logger.Error("rollback failed",
				"error", rollbackErr,
				"target_sha", w.lastCommitSHA[:8])
			return fmt.Errorf("validation failed and rollback failed: %w (rollback: %v)", err, rollbackErr)
		}

		w.logger.Info("successfully rolled back to previous commit",
			"sha", w.lastCommitSHA[:8])
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Success - update last commit SHA
	w.mu.Lock()
	oldSHA := w.lastCommitSHA
	w.lastCommitSHA = newSHA
	w.mu.Unlock()

	w.metrics.SuccessfulReloads++
	w.logger.Info("successfully reloaded policies",
		"from_sha", oldSHA[:8],
		"to_sha", newSHA[:8],
		"duration", w.metrics.LastReloadDur)

	return nil
}

// rollbackToPrevious reverts repository to previous commit on error.
func (w *Watcher) rollbackToPrevious(ctx context.Context, sha string) error {
	// Rollback the repository
	if err := w.repo.Rollback(ctx, sha); err != nil {
		return fmt.Errorf("failed to rollback repository: %w", err)
	}

	// Reload policies from rolled-back commit
	policyPath := w.repo.GetPolicyPath()
	if err := w.reloadFn(policyPath); err != nil {
		return fmt.Errorf("failed to reload policies after rollback: %w", err)
	}

	return nil
}

// ForceCheck immediately checks for changes without waiting for the next poll interval.
// This is useful for CLI commands that want to trigger an immediate sync.
func (w *Watcher) ForceCheck(ctx context.Context) error {
	w.mu.RLock()
	if !w.running {
		w.mu.RUnlock()
		return fmt.Errorf("watcher not running")
	}
	w.mu.RUnlock()

	w.logger.Info("force checking for changes")
	return w.checkForChanges(ctx)
}

// GetLastCommitSHA returns the SHA of the currently active commit.
// This is the commit that policies were last successfully loaded from.
func (w *Watcher) GetLastCommitSHA() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastCommitSHA
}

// GetMetrics returns current watcher metrics.
// This returns a copy of the metrics for thread-safe access.
func (w *Watcher) GetMetrics() WatcherMetrics {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return *w.metrics
}
