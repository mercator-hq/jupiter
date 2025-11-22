package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
	"mercator-hq/jupiter/pkg/policy/manager"
)

var policyFlags struct {
	configFile string
	limit      int
	to         string
	format     string
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policy lifecycle",
	Long: `Manage policy lifecycle for Git-based policy management.

The policy command provides tools for managing policies stored in Git repositories,
including version tracking, synchronization, and rollback capabilities.

Subcommands:
  version  - Show current policy version (commit info)
  sync     - Force pull latest policies from Git
  history  - Show policy commit history
  rollback - Rollback policies to a specific commit

Examples:
  # Show current policy version
  mercator policy version

  # Force sync with Git remote
  mercator policy sync

  # Show last 10 commits
  mercator policy history --limit 10

  # Rollback to specific commit
  mercator policy rollback --to a1b2c3d4`,
}

var policyVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current policy version",
	Long: `Show current policy version information.

For Git-based policy management, this displays the active commit SHA,
author, timestamp, and commit message.

Examples:
  # Show version info
  mercator policy version

  # Output as JSON
  mercator policy version --format json`,
	RunE: showPolicyVersion,
}

var policySyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Force pull latest policies",
	Long: `Force pull latest policies from Git repository.

This command manually triggers a Git pull operation to sync with the
remote repository. If changes are detected, policies are automatically
reloaded and validated.

Examples:
  # Sync with remote
  mercator policy sync`,
	RunE: syncPolicies,
}

var policyHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show policy commit history",
	Long: `Show policy commit history.

Displays the commit history for the policy repository, including
commit SHA, author, timestamp, and message.

Examples:
  # Show last 10 commits
  mercator policy history --limit 10

  # Show last 50 commits
  mercator policy history --limit 50

  # Output as JSON
  mercator policy history --limit 10 --format json`,
	RunE: showPolicyHistory,
}

var policyRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback policies to a specific commit",
	Long: `Rollback policies to a specific Git commit.

This command performs a Git checkout to the specified commit and reloads
the policies. The policies at the target commit are validated before
being activated. If validation fails, the rollback is aborted.

Examples:
  # Rollback to commit
  mercator policy rollback --to a1b2c3d4e5f6

  # Rollback to short SHA
  mercator policy rollback --to a1b2c3d`,
	RunE: rollbackPolicies,
}

func init() {
	rootCmd.AddCommand(policyCmd)
	policyCmd.AddCommand(policyVersionCmd, policySyncCmd, policyHistoryCmd, policyRollbackCmd)

	// Flags for policy commands
	policyCmd.PersistentFlags().StringVar(&policyFlags.configFile, "config", "", "config file (default is ./config.yaml)")
	policyCmd.PersistentFlags().StringVar(&policyFlags.format, "format", "text", "output format: text, json")

	// Flags for history command
	policyHistoryCmd.Flags().IntVar(&policyFlags.limit, "limit", 10, "number of commits to show")

	// Flags for rollback command
	policyRollbackCmd.Flags().StringVar(&policyFlags.to, "to", "", "target commit SHA")
	_ = policyRollbackCmd.MarkFlagRequired("to")
}

func showPolicyVersion(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadPolicyConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if in Git mode
	if cfg.Mode != "git" || !cfg.Git.Enabled {
		return fmt.Errorf("policy version command requires Git mode (set policy.mode: git)")
	}

	// Create policy manager
	mgr, err := createPolicyManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create policy manager: %w", err)
	}
	defer mgr.Close()

	// Get current commit
	commit, err := mgr.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	// Output based on format
	switch policyFlags.format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(commit)
	default:
		fmt.Printf("Current Policy Version:\n")
		fmt.Printf("  Commit:     %s\n", commit.SHA)
		fmt.Printf("  Branch:     %s\n", commit.Branch)
		fmt.Printf("  Author:     %s\n", commit.Author)
		fmt.Printf("  Timestamp:  %s\n", commit.Timestamp.Format(time.RFC3339))
		fmt.Printf("  Repository: %s\n", commit.Repository)
		if commit.Message != "" {
			fmt.Printf("  Message:    %s\n", commit.Message)
		}
	}

	return nil
}

func syncPolicies(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadPolicyConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if in Git mode
	if cfg.Mode != "git" || !cfg.Git.Enabled {
		return fmt.Errorf("policy sync command requires Git mode (set policy.mode: git)")
	}

	// Create policy manager
	mgr, err := createPolicyManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create policy manager: %w", err)
	}
	defer mgr.Close()

	fmt.Println("Syncing with Git remote...")

	// Force sync
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mgr.ForceSync(ctx); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	// Get new commit
	commit, err := mgr.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	fmt.Printf("✓ Synced successfully to commit %s\n", commit.SHA[:8])
	fmt.Printf("  Author: %s\n", commit.Author)
	fmt.Printf("  Date:   %s\n", commit.Timestamp.Format(time.RFC3339))

	return nil
}

func showPolicyHistory(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadPolicyConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if in Git mode
	if cfg.Mode != "git" || !cfg.Git.Enabled {
		return fmt.Errorf("policy history command requires Git mode (set policy.mode: git)")
	}

	// Create policy manager
	mgr, err := createPolicyManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create policy manager: %w", err)
	}
	defer mgr.Close()

	// Get commit history
	commits, err := mgr.GetCommitHistory(policyFlags.limit)
	if err != nil {
		return fmt.Errorf("failed to get commit history: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println("No commits found")
		return nil
	}

	// Output based on format
	switch policyFlags.format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(commits)
	default:
		fmt.Printf("Policy Commit History (last %d commits):\n\n", policyFlags.limit)
		for i, commit := range commits {
			fmt.Printf("%d. %s\n", i+1, commit.SHA[:8])
			fmt.Printf("   Author:    %s\n", commit.Author)
			fmt.Printf("   Date:      %s\n", commit.Timestamp.Format(time.RFC3339))
			fmt.Printf("   Branch:    %s\n", commit.Branch)
			if commit.Message != "" {
				// Truncate message to first line
				message := commit.Message
				if idx := len(message); idx > 60 {
					message = message[:60] + "..."
				}
				fmt.Printf("   Message:   %s\n", message)
			}
			fmt.Println()
		}
	}

	return nil
}

func rollbackPolicies(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadPolicyConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if in Git mode
	if cfg.Mode != "git" || !cfg.Git.Enabled {
		return fmt.Errorf("policy rollback command requires Git mode (set policy.mode: git)")
	}

	// Create policy manager
	mgr, err := createPolicyManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create policy manager: %w", err)
	}
	defer mgr.Close()

	// Get current commit before rollback
	currentCommit, err := mgr.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	fmt.Printf("Current commit: %s\n", currentCommit.SHA[:8])
	fmt.Printf("Rolling back to: %s\n", policyFlags.to)

	// Perform rollback
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mgr.RollbackToCommit(ctx, policyFlags.to); err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	// Get new commit
	newCommit, err := mgr.GetCurrentCommit()
	if err != nil {
		return fmt.Errorf("failed to get new commit: %w", err)
	}

	fmt.Printf("✓ Successfully rolled back to commit %s\n", newCommit.SHA[:8])
	fmt.Printf("  Author: %s\n", newCommit.Author)
	fmt.Printf("  Date:   %s\n", newCommit.Timestamp.Format(time.RFC3339))

	return nil
}

// Helper functions

func loadPolicyConfig() (*config.PolicyConfig, error) {
	// Determine config file path
	configFile := policyFlags.configFile
	if configFile == "" {
		configFile = "config.yaml"
	}

	// Initialize config system
	if err := config.Initialize(configFile); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	cfg := config.GetConfig()
	return &cfg.Policy, nil
}

func createPolicyManager(cfg *config.PolicyConfig) (*manager.DefaultPolicyManager, error) {
	// Create parser and validator
	p := parser.NewParser()
	v := validator.NewValidator()

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create policy manager
	mgr, err := manager.NewPolicyManager(cfg, p, v, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy manager: %w", err)
	}

	// Load policies
	if err := mgr.LoadPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return mgr, nil
}
