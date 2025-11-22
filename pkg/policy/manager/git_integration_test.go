package manager

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
	"mercator-hq/jupiter/pkg/policy/git"
)

// TestGitPolicyManager_EndToEnd tests the complete Git integration workflow.
func TestGitPolicyManager_EndToEnd(t *testing.T) {
	// Create test Git repository
	sourceDir := t.TempDir()
	cloneDir := t.TempDir()

	// Initialize Git repo
	repo, err := gogit.PlainInit(sourceDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial policy file
	policyContent := `mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"
description: "Test policy"
author: "Test Author"
created: "2025-11-21T00:00:00Z"
updated: "2025-11-21T00:00:00Z"

rules:
  - name: "test-rule"
    description: "Test rule"
    enabled: true
    priority: 100
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "allow"
`
	policyFile := filepath.Join(sourceDir, "policy.yaml")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Commit initial policy
	worktree, _ := repo.Worktree()
	worktree.Add("policy.yaml")
	firstCommit, err := worktree.Commit("Initial policy", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create policy configuration
	cfg := &config.PolicyConfig{
		Mode: "git",
		Git: config.GitPolicyConfig{
			Enabled:    true,
			Repository: sourceDir,
			Branch:     "master",
			Path:       "",
			Auth: config.GitAuthConfig{
				Type: "none",
			},
			Poll: config.GitPollConfig{
				Enabled:  true,
				Interval: 100 * time.Millisecond,
				Timeout:  5 * time.Second,
			},
			Clone: config.GitCloneConfig{
				Depth:     0,
				LocalPath: cloneDir,
			},
		},
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  true,
		},
	}

	// Create policy manager
	p := parser.NewParser()
	v := validator.NewValidator()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	mgr, err := NewPolicyManager(cfg, p, v, logger)
	if err != nil {
		t.Fatalf("Failed to create policy manager: %v", err)
	}
	defer mgr.Close()

	// Test 1: Initial load
	t.Run("InitialLoad", func(t *testing.T) {
		err := mgr.LoadPolicies()
		if err != nil {
			t.Fatalf("Failed to load policies: %v", err)
		}

		policies := mgr.GetAllPolicies()
		if len(policies) != 1 {
			t.Fatalf("Expected 1 policy, got %d", len(policies))
		}

		if policies[0].Name != "test-policy" {
			t.Errorf("Expected policy name 'test-policy', got '%s'", policies[0].Name)
		}
	})

	// Test 2: Get current commit
	t.Run("GetCurrentCommit", func(t *testing.T) {
		commit, err := mgr.GetCurrentCommit()
		if err != nil {
			t.Fatalf("Failed to get current commit: %v", err)
		}

		if commit.SHA != firstCommit.String() {
			t.Errorf("Expected commit SHA %s, got %s", firstCommit.String(), commit.SHA)
		}

		if commit.Author != "Test Author" {
			t.Errorf("Expected author 'Test Author', got '%s'", commit.Author)
		}
	})

	// Test 3: Commit history
	t.Run("CommitHistory", func(t *testing.T) {
		history, err := mgr.GetCommitHistory(10)
		if err != nil {
			t.Fatalf("Failed to get commit history: %v", err)
		}

		if len(history) != 1 {
			t.Fatalf("Expected 1 commit in history, got %d", len(history))
		}

		if history[0].SHA != firstCommit.String() {
			t.Errorf("Expected commit SHA %s, got %s", firstCommit.String(), history[0].SHA)
		}
	})

	// Test 4: Update policy in Git and sync
	t.Run("UpdateAndSync", func(t *testing.T) {
		// Update policy content
		updatedPolicyContent := `mpl_version: "1.0"
name: "test-policy-v2"
version: "2.0.0"
description: "Updated test policy"
author: "Test Author"
created: "2025-11-21T00:00:00Z"
updated: "2025-11-21T01:00:00Z"

rules:
  - name: "test-rule-updated"
    description: "Updated test rule"
    enabled: true
    priority: 100
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "allow"
`
		if err := os.WriteFile(policyFile, []byte(updatedPolicyContent), 0644); err != nil {
			t.Fatalf("Failed to update policy file: %v", err)
		}

		// Commit update
		worktree.Add("policy.yaml")
		secondCommit, err := worktree.Commit("Update policy", &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			t.Fatalf("Failed to commit update: %v", err)
		}

		// Force sync
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := mgr.ForceSync(ctx); err != nil {
			t.Fatalf("Failed to sync: %v", err)
		}

		// Verify new policy is loaded
		policies := mgr.GetAllPolicies()
		if len(policies) != 1 {
			t.Fatalf("Expected 1 policy after sync, got %d", len(policies))
		}

		if policies[0].Name != "test-policy-v2" {
			t.Errorf("Expected updated policy name 'test-policy-v2', got '%s'", policies[0].Name)
		}

		// Verify commit changed
		commit, err := mgr.GetCurrentCommit()
		if err != nil {
			t.Fatalf("Failed to get current commit: %v", err)
		}

		if commit.SHA != secondCommit.String() {
			t.Errorf("Expected commit SHA %s after sync, got %s", secondCommit.String(), commit.SHA)
		}
	})

	// Test 5: Rollback
	t.Run("Rollback", func(t *testing.T) {
		// Rollback to first commit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := mgr.RollbackToCommit(ctx, firstCommit.String()); err != nil {
			t.Fatalf("Failed to rollback: %v", err)
		}

		// Verify old policy is restored
		policies := mgr.GetAllPolicies()
		if len(policies) != 1 {
			t.Fatalf("Expected 1 policy after rollback, got %d", len(policies))
		}

		if policies[0].Name != "test-policy" {
			t.Errorf("Expected original policy name 'test-policy', got '%s'", policies[0].Name)
		}

		// Verify commit reverted
		commit, err := mgr.GetCurrentCommit()
		if err != nil {
			t.Fatalf("Failed to get current commit: %v", err)
		}

		if commit.SHA != firstCommit.String() {
			t.Errorf("Expected commit SHA %s after rollback, got %s", firstCommit.String(), commit.SHA)
		}
	})

	// Test 6: Git metrics
	t.Run("GitMetrics", func(t *testing.T) {
		metrics := mgr.GetGitMetrics()

		// Check that some operations were performed
		if metrics.SuccessfulPulls == 0 {
			t.Error("Expected at least one successful pull")
		}

		if metrics.CloneDuration == 0 {
			t.Error("Expected non-zero clone duration")
		}
	})
}

// TestGitPolicyManager_ValidationFailureRollback tests automatic rollback on validation failure.
func TestGitPolicyManager_ValidationFailureRollback(t *testing.T) {
	// Create test Git repository
	sourceDir := t.TempDir()
	cloneDir := t.TempDir()

	// Initialize Git repo
	repo, err := gogit.PlainInit(sourceDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create valid initial policy
	validPolicyContent := `mpl_version: "1.0"
name: "valid-policy"
version: "1.0.0"
description: "Valid policy"
author: "Test Author"
created: "2025-11-21T00:00:00Z"
updated: "2025-11-21T00:00:00Z"

rules:
  - name: "test-rule"
    description: "Test rule"
    enabled: true
    priority: 100
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "allow"
`
	policyFile := filepath.Join(sourceDir, "policy.yaml")
	if err := os.WriteFile(policyFile, []byte(validPolicyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Commit valid policy
	worktree, _ := repo.Worktree()
	worktree.Add("policy.yaml")
	validCommit, err := worktree.Commit("Valid policy", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create policy configuration
	cfg := &config.PolicyConfig{
		Mode: "git",
		Git: config.GitPolicyConfig{
			Enabled:    true,
			Repository: sourceDir,
			Branch:     "master",
			Path:       "",
			Auth: config.GitAuthConfig{
				Type: "none",
			},
			Poll: config.GitPollConfig{
				Enabled:  false, // Disable polling for this test
				Interval: 1 * time.Second,
				Timeout:  5 * time.Second,
			},
			Clone: config.GitCloneConfig{
				Depth:     0,
				LocalPath: cloneDir,
			},
		},
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  true,
		},
	}

	// Create policy manager
	p := parser.NewParser()
	v := validator.NewValidator()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	mgr, err := NewPolicyManager(cfg, p, v, logger)
	if err != nil {
		t.Fatalf("Failed to create policy manager: %v", err)
	}
	defer mgr.Close()

	// Load initial valid policy
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatalf("Failed to load policies: %v", err)
	}

	// Create repository handle for testing
	gitRepo, err := git.NewRepository(&cfg.Git)
	if err != nil {
		t.Fatalf("Failed to create git repository: %v", err)
	}

	// Clone the repository
	ctx := context.Background()
	if err := gitRepo.Clone(ctx); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Create invalid policy (malformed YAML)
	invalidPolicyContent := `
name: invalid-policy
version: "1.0"
description: Invalid policy
rules:
  - name: test-rule
    priority: not-a-number
    action: allow
`
	if err := os.WriteFile(policyFile, []byte(invalidPolicyContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid policy file: %v", err)
	}

	// Commit invalid policy
	worktree.Add("policy.yaml")
	_, err = worktree.Commit("Invalid policy", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit invalid policy: %v", err)
	}

	// Attempt to sync (should fail validation and keep old policy)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = mgr.ForceSync(ctx)
	if err == nil {
		t.Fatal("Expected sync to fail due to validation error")
	}

	// Verify old policy is still loaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("Expected 1 policy after failed sync, got %d", len(policies))
	}

	if policies[0].Name != "valid-policy" {
		t.Errorf("Expected original policy 'valid-policy' to be retained, got '%s'", policies[0].Name)
	}

	// Verify commit is still at valid commit
	commit, err := mgr.GetCurrentCommit()
	if err != nil {
		t.Fatalf("Failed to get current commit: %v", err)
	}

	if commit.SHA != validCommit.String() {
		t.Errorf("Expected commit SHA to remain at %s, got %s", validCommit.String(), commit.SHA)
	}
}

// TestGitPolicyManager_HotReload tests hot-reload functionality with watcher.
func TestGitPolicyManager_HotReload(t *testing.T) {
	// Skip if we're in short test mode
	if testing.Short() {
		t.Skip("Skipping hot-reload test in short mode")
	}

	// Create test Git repository
	sourceDir := t.TempDir()
	cloneDir := t.TempDir()

	// Initialize Git repo
	repo, err := gogit.PlainInit(sourceDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial policy
	policyContent := `mpl_version: "1.0"
name: "hot-reload-policy"
version: "1.0.0"
description: "Hot reload test policy"
author: "Test Author"
created: "2025-11-21T00:00:00Z"
updated: "2025-11-21T00:00:00Z"

rules:
  - name: "test-rule"
    description: "Test rule"
    enabled: true
    priority: 100
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "allow"
`
	policyFile := filepath.Join(sourceDir, "policy.yaml")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}

	// Commit initial policy
	worktree, _ := repo.Worktree()
	worktree.Add("policy.yaml")
	_, err = worktree.Commit("Initial policy", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create policy configuration with short poll interval
	cfg := &config.PolicyConfig{
		Mode: "git",
		Git: config.GitPolicyConfig{
			Enabled:    true,
			Repository: sourceDir,
			Branch:     "master",
			Path:       "",
			Auth: config.GitAuthConfig{
				Type: "none",
			},
			Poll: config.GitPollConfig{
				Enabled:  true,
				Interval: 500 * time.Millisecond, // Short interval for testing
				Timeout:  5 * time.Second,
			},
			Clone: config.GitCloneConfig{
				Depth:     0,
				LocalPath: cloneDir,
			},
		},
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  true,
		},
	}

	// Create policy manager
	p := parser.NewParser()
	v := validator.NewValidator()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	mgr, err := NewPolicyManager(cfg, p, v, logger)
	if err != nil {
		t.Fatalf("Failed to create policy manager: %v", err)
	}
	defer mgr.Close()

	// Load initial policies
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatalf("Failed to load policies: %v", err)
	}

	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := mgr.Watch(ctx); err != nil && ctx.Err() == nil {
			t.Logf("Watcher error: %v", err)
		}
	}()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Update policy
	updatedPolicyContent := `mpl_version: "1.0"
name: "hot-reload-policy-v2"
version: "2.0.0"
description: "Updated policy"
author: "Test Author"
created: "2025-11-21T00:00:00Z"
updated: "2025-11-21T01:00:00Z"

rules:
  - name: "test-rule-updated"
    description: "Updated test rule"
    enabled: true
    priority: 100
    conditions:
      field: "request.model"
      operator: "=="
      value: "claude-3"
    actions:
      - type: "allow"
`
	if err := os.WriteFile(policyFile, []byte(updatedPolicyContent), 0644); err != nil {
		t.Fatalf("Failed to update policy file: %v", err)
	}

	// Commit update
	worktree.Add("policy.yaml")
	_, err = worktree.Commit("Update policy", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit update: %v", err)
	}

	// Wait for watcher to detect changes and reload
	// Poll interval is 500ms, so give it 2 seconds to be safe
	time.Sleep(2 * time.Second)

	// Verify policy was reloaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("Expected 1 policy after hot-reload, got %d", len(policies))
	}

	if policies[0].Name != "hot-reload-policy-v2" {
		t.Errorf("Expected updated policy name 'hot-reload-policy-v2', got '%s'", policies[0].Name)
	}

	// Cancel context to stop watcher
	cancel()

	// Give watcher time to stop
	time.Sleep(200 * time.Millisecond)
}
