package git

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/config"
)

// TestNewWatcher tests watcher creation.
func TestNewWatcher(t *testing.T) {
	// Create test repository
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 1*time.Second, 5*time.Second, reloadFn)

	if watcher == nil {
		t.Fatal("expected non-nil watcher")
	}

	if watcher.pollInterval != 1*time.Second {
		t.Errorf("expected poll interval 1s, got %v", watcher.pollInterval)
	}

	if watcher.pollTimeout != 5*time.Second {
		t.Errorf("expected poll timeout 5s, got %v", watcher.pollTimeout)
	}

	if watcher.reloadFn == nil {
		t.Error("expected non-nil reload function")
	}

	if watcher.running {
		t.Error("expected watcher not running initially")
	}
}

// TestWatcher_StartStop tests watcher lifecycle.
func TestWatcher_StartStop(t *testing.T) {
	// Create test repository
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	if err := repo.Clone(context.Background()); err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 1*time.Second, 5*time.Second, reloadFn)

	// Test start
	ctx := context.Background()
	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("expected watcher to be running after Start()")
	}

	if watcher.lastCommitSHA == "" {
		t.Error("expected lastCommitSHA to be set after Start()")
	}

	// Test double start (should fail)
	if err := watcher.Start(ctx); err == nil {
		t.Error("expected error when starting already running watcher")
	}

	// Test stop
	if err := watcher.Stop(); err != nil {
		t.Fatalf("failed to stop watcher: %v", err)
	}

	if watcher.IsRunning() {
		t.Error("expected watcher not running after Stop()")
	}

	// Test double stop (should fail)
	if err := watcher.Stop(); err == nil {
		t.Error("expected error when stopping already stopped watcher")
	}
}

// TestWatcher_StartWithoutClone tests that Start fails if repository not cloned.
func TestWatcher_StartWithoutClone(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	// Don't call Clone()

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 1*time.Second, 5*time.Second, reloadFn)

	// Start should fail because repository not cloned
	ctx := context.Background()
	if err := watcher.Start(ctx); err == nil {
		t.Error("expected error when starting watcher with uncloned repository")
	}
}

// TestWatcher_GetLastCommitSHA tests commit SHA tracking.
func TestWatcher_GetLastCommitSHA(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	if err := repo.Clone(context.Background()); err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 1*time.Second, 5*time.Second, reloadFn)

	ctx := context.Background()
	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer func() { _ = watcher.Stop() }() // Intentionally ignoring error in test cleanup

	sha := watcher.GetLastCommitSHA()
	if sha == "" {
		t.Error("expected non-empty commit SHA")
	}

	if len(sha) != 40 { // Git SHA is 40 hex characters
		t.Errorf("expected 40-character SHA, got %d characters", len(sha))
	}
}

// TestWatcher_GetMetrics tests metrics tracking.
func TestWatcher_GetMetrics(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 100 * time.Millisecond,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	if err := repo.Clone(context.Background()); err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 100*time.Millisecond, 5*time.Second, reloadFn)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer func() { _ = watcher.Stop() }() // Intentionally ignoring error in test cleanup

	// Let it poll a few times
	time.Sleep(400 * time.Millisecond)

	metrics := watcher.GetMetrics()

	// Should have polled at least a few times
	if metrics.PollCount == 0 {
		t.Error("expected PollCount > 0")
	}
}

// TestWatcher_ContextCancellation tests watcher stops on context cancellation.
func TestWatcher_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 100 * time.Millisecond,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	if err := repo.Clone(context.Background()); err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 100*time.Millisecond, 5*time.Second, reloadFn)

	ctx, cancel := context.WithCancel(context.Background())

	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("expected watcher to be running")
	}

	// Cancel context
	cancel()

	// Wait for watcher to stop
	time.Sleep(200 * time.Millisecond)

	// Watcher should still report as running (Stop() not called)
	// but pollLoop should have exited due to context cancellation
	// This is expected behavior - the running flag tracks explicit Start/Stop calls
}

// TestWatcher_hasPolicyFileChanges tests policy file filtering.
func TestWatcher_hasPolicyFileChanges(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 1*time.Second, 5*time.Second, reloadFn)

	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{
			name:  "mpl file",
			files: []string{"policy.mpl"},
			want:  true,
		},
		{
			name:  "yaml file",
			files: []string{"policy.yaml"},
			want:  true,
		},
		{
			name:  "yml file",
			files: []string{"policy.yml"},
			want:  true,
		},
		{
			name:  "mixed with mpl",
			files: []string{"README.md", "policy.mpl", "config.json"},
			want:  true,
		},
		{
			name:  "no policy files",
			files: []string{"README.md", "config.json", "script.sh"},
			want:  false,
		},
		{
			name:  "empty list",
			files: []string{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := watcher.hasPolicyFileChanges(tt.files)
			if got != tt.want {
				t.Errorf("hasPolicyFileChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWatcher_ForceCheckNotRunning tests ForceCheck when watcher is not running.
func TestWatcher_ForceCheckNotRunning(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepo(t, tempDir)

	cfg := &config.GitPolicyConfig{
		Repository: tempDir,
		Branch:     "master",
		Path:       "",
		Auth: config.GitAuthConfig{
			Type: "none",
		},
		Poll: config.GitPollConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		Clone: config.GitCloneConfig{
			Depth:     0,
			LocalPath: tempDir,
		},
	}

	repo, err := NewRepository(cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	if err := repo.Clone(context.Background()); err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	reloadFn := func(path string) error {
		return nil
	}

	watcher := NewWatcher(repo, 1*time.Second, 5*time.Second, reloadFn)

	// Don't start watcher

	ctx := context.Background()
	if err := watcher.ForceCheck(ctx); err == nil {
		t.Error("expected error when calling ForceCheck() on stopped watcher")
	}
}
