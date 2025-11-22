package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"mercator-hq/jupiter/pkg/config"
)

// Repository manages Git operations for policy repos.
type Repository struct {
	config    *config.GitPolicyConfig
	localPath string
	auth      AuthProvider
	repo      *gogit.Repository
	mu        sync.RWMutex
	metrics   *RepositoryMetrics
}

// NewRepository creates a new Git repository manager.
// The config parameter must be non-nil and contain valid Git configuration.
// Returns an error if authentication provider creation fails.
func NewRepository(cfg *config.GitPolicyConfig) (*Repository, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.Repository == "" {
		return nil, fmt.Errorf("repository URL cannot be empty")
	}

	if cfg.Branch == "" {
		return nil, fmt.Errorf("branch cannot be empty")
	}

	auth, err := NewAuthProvider(&cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider: %w", err)
	}

	localPath := cfg.Clone.LocalPath
	if localPath == "" {
		// Default to temp directory if not specified
		localPath = filepath.Join(os.TempDir(), "mercator-policies")
	}

	return &Repository{
		config:    cfg,
		localPath: localPath,
		auth:      auth,
		metrics:   &RepositoryMetrics{},
	}, nil
}

// Clone initializes the repository by cloning it locally.
// If the repository already exists and CleanOnStart is false, it opens the existing repo.
// If CleanOnStart is true, it removes any existing repository before cloning.
// Returns an error if cloning fails.
func (r *Repository) Clone(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()
	defer func() {
		r.metrics.CloneDuration = time.Since(start)
	}()

	// Clean existing directory if configured
	if r.config.Clone.CleanOnStart {
		if err := os.RemoveAll(r.localPath); err != nil {
			return fmt.Errorf("failed to clean existing repository: %w", err)
		}
	}

	// Check if repo already exists
	gitDir := filepath.Join(r.localPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Open existing repo
		repo, err := gogit.PlainOpen(r.localPath)
		if err != nil {
			return fmt.Errorf("failed to open existing repo: %w", err)
		}
		r.repo = repo
		return nil
	}

	// Create parent directory
	if err := os.MkdirAll(r.localPath, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Clone options
	cloneOpts := &gogit.CloneOptions{
		URL:           r.config.Repository,
		ReferenceName: plumbing.NewBranchReferenceName(r.config.Branch),
		SingleBranch:  r.config.Clone.Depth > 0, // Only single branch for shallow clones
		Depth:         r.config.Clone.Depth,
		Progress:      nil, // Can add progress reporting if needed
	}

	// Add auth if configured
	auth, err := r.auth.GetAuth()
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	cloneOpts.Auth = auth

	// Clone repository with timeout
	cloneCtx, cancel := context.WithTimeout(ctx, r.config.Poll.Timeout)
	defer cancel()

	repo, err := gogit.PlainCloneContext(cloneCtx, r.localPath, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	r.repo = repo
	return nil
}

// Pull fetches latest changes from the remote repository.
// It returns a PullResult indicating whether changes were found and what files changed.
// This method is thread-safe and can be called concurrently.
// Returns an error if the pull operation fails.
func (r *Repository) Pull(ctx context.Context) (*PullResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()
	defer func() {
		r.metrics.PullDuration = time.Since(start)
		r.metrics.LastPullTime = time.Now()
	}()

	if r.repo == nil {
		return nil, fmt.Errorf("repository not initialized, call Clone() first")
	}

	// Get current HEAD before pull
	ref, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	fromSHA := ref.Hash().String()

	// Pull changes
	worktree, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	pullOpts := &gogit.PullOptions{
		RemoteName: "origin",
		Force:      false, // Never force pull (fail-safe)
	}

	// Add auth
	auth, err := r.auth.GetAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth: %w", err)
	}
	pullOpts.Auth = auth

	// Pull with timeout
	pullCtx, cancel := context.WithTimeout(ctx, r.config.Poll.Timeout)
	defer cancel()

	err = worktree.PullContext(pullCtx, pullOpts)
	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		r.metrics.FailedPulls++
		return nil, fmt.Errorf("failed to pull: %w", err)
	}

	r.metrics.SuccessfulPulls++

	// Get new HEAD
	newRef, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get new HEAD: %w", err)
	}
	toSHA := newRef.Hash().String()

	result := &PullResult{
		FromSHA:    fromSHA,
		ToSHA:      toSHA,
		HadChanges: fromSHA != toSHA,
	}

	// Get changed files if there were changes
	if result.HadChanges {
		changedFiles, err := r.getChangedFilesInternal(fromSHA, toSHA)
		if err != nil {
			return nil, fmt.Errorf("failed to get changed files: %w", err)
		}
		result.ChangedFiles = changedFiles
		r.metrics.LastCommitSHA = toSHA
	}

	return result, nil
}

// GetCurrentCommit returns metadata about the current HEAD commit.
// This includes commit SHA, author, timestamp, message, and branch information.
// This method is thread-safe and can be called concurrently.
// Returns an error if the repository is not initialized or HEAD cannot be read.
func (r *Repository) GetCurrentCommit() (*CommitInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.repo == nil {
		return nil, fmt.Errorf("repository not initialized, call Clone() first")
	}

	ref, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return &CommitInfo{
		SHA:        commit.Hash.String(),
		Author:     commit.Author.Name,
		Email:      commit.Author.Email,
		Timestamp:  commit.Author.When,
		Message:    commit.Message,
		Branch:     r.config.Branch,
		Repository: r.config.Repository,
	}, nil
}

// ListPolicyFiles returns all policy files (.mpl, .yaml, .yml) in the configured path.
// It recursively walks the directory tree looking for policy files.
// Hidden files (starting with .) are excluded.
// This method is thread-safe and can be called concurrently.
// Returns an error if the policy directory cannot be accessed or walked.
func (r *Repository) ListPolicyFiles() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	policyPath := filepath.Join(r.localPath, r.config.Path)

	// Check if policy path exists
	if _, err := os.Stat(policyPath); err != nil {
		return nil, fmt.Errorf("policy path does not exist: %w", err)
	}

	var files []string
	err := filepath.Walk(policyPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files
		if len(info.Name()) > 0 && info.Name()[0] == '.' {
			return nil
		}

		// Check for policy file extensions
		ext := filepath.Ext(path)
		if ext == ".mpl" || ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk policy directory: %w", err)
	}

	return files, nil
}

// GetChangedFiles returns files changed between two commits.
// It uses git diff to identify files that were added, modified, or deleted.
// Only the file paths relative to the repository root are returned.
// This method is thread-safe and can be called concurrently.
// Returns an error if either commit cannot be found or the diff fails.
func (r *Repository) GetChangedFiles(fromSHA, toSHA string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.getChangedFilesInternal(fromSHA, toSHA)
}

// getChangedFilesInternal is the internal implementation of GetChangedFiles
// that doesn't acquire locks. This is used by methods that already hold locks.
func (r *Repository) getChangedFilesInternal(fromSHA, toSHA string) ([]string, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	// Get from commit
	fromHash := plumbing.NewHash(fromSHA)
	fromCommit, err := r.repo.CommitObject(fromHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit: %w", err)
	}

	// Get to commit
	toHash := plumbing.NewHash(toSHA)
	toCommit, err := r.repo.CommitObject(toHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get to commit: %w", err)
	}

	// Get trees
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get from tree: %w", err)
	}

	toTree, err := toCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get to tree: %w", err)
	}

	// Get diff
	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	// Extract file paths
	var files []string
	for _, change := range changes {
		// Get the "to" path (file after change)
		if change.To.Name != "" {
			files = append(files, change.To.Name)
		} else if change.From.Name != "" {
			// File was deleted, use "from" path
			files = append(files, change.From.Name)
		}
	}

	return files, nil
}

// SwitchBranch changes the active branch (useful for rollback).
// This checks out the specified branch and updates the working tree.
// The branch must exist in the repository.
// This method is NOT safe to call during concurrent operations.
// Returns an error if the branch cannot be checked out.
func (r *Repository) SwitchBranch(branch string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}

	// Update config to track new branch
	r.config.Branch = branch

	return nil
}

// Rollback reverts repository to specific commit SHA.
// This performs a hard checkout to the target commit.
// The commit must exist and be reachable in the repository history.
// This method is NOT safe to call during concurrent operations.
// Returns an error if the target commit cannot be checked out.
func (r *Repository) Rollback(ctx context.Context, targetSHA string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	// Validate target commit exists
	targetHash := plumbing.NewHash(targetSHA)
	_, err := r.repo.CommitObject(targetHash)
	if err != nil {
		return fmt.Errorf("target commit not found: %w", err)
	}

	// Checkout target commit
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Checkout(&gogit.CheckoutOptions{
		Hash: targetHash,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", targetSHA, err)
	}

	return nil
}

// GetMetrics returns current repository metrics.
// This includes clone/pull durations, success/failure counts, and last commit info.
// This method is thread-safe and returns a copy of the metrics.
func (r *Repository) GetMetrics() RepositoryMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return *r.metrics
}

// GetCommitHistory returns a list of recent commits.
// The limit parameter specifies the maximum number of commits to return.
// This method is thread-safe and can be called concurrently.
// Returns an error if the commit log cannot be accessed.
func (r *Repository) GetCommitHistory(limit int) ([]*CommitInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	ref, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit log
	iter, err := r.repo.Log(&gogit.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	var history []*CommitInfo
	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if count >= limit {
			return fmt.Errorf("limit reached") // Stop iteration
		}

		history = append(history, &CommitInfo{
			SHA:        c.Hash.String(),
			Author:     c.Author.Name,
			Email:      c.Author.Email,
			Timestamp:  c.Author.When,
			Message:    c.Message,
			Branch:     r.config.Branch,
			Repository: r.config.Repository,
		})

		count++
		return nil
	})

	// Ignore "limit reached" error as it's expected
	if err != nil && err.Error() != "limit reached" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return history, nil
}

// GetLocalPath returns the local filesystem path where the repository is cloned.
func (r *Repository) GetLocalPath() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.localPath
}

// GetPolicyPath returns the full path to the policy directory within the repository.
func (r *Repository) GetPolicyPath() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return filepath.Join(r.localPath, r.config.Path)
}
