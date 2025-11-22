package manager

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
	"mercator-hq/jupiter/pkg/policy/git"
)

// DefaultPolicyManager is the default implementation of PolicyManager.
// It coordinates policy loading, validation, registration, and hot-reload.
type DefaultPolicyManager struct {
	config    *config.PolicyConfig
	loader    *PolicyLoader
	resolver  *IncludeResolver
	registry  *PolicyRegistry
	parser    *parser.Parser
	validator *validator.Validator
	logger    *slog.Logger

	// Git source management
	gitRepo    *git.Repository
	gitWatcher *git.Watcher

	// State management
	mu              sync.RWMutex
	lastLoadTime    time.Time
	lastLoadError   error
	lastGoodPolicies []*ast.Policy // For error recovery

	// Watch management
	watchCtx    context.Context
	watchCancel context.CancelFunc
	watchEvents chan ReloadEvent
	watchMu     sync.Mutex
}

// NewPolicyManager creates a new policy manager.
func NewPolicyManager(
	config *config.PolicyConfig,
	parser *parser.Parser,
	validator *validator.Validator,
	logger *slog.Logger,
) (*DefaultPolicyManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if parser == nil {
		return nil, fmt.Errorf("parser cannot be nil")
	}

	if validator == nil {
		return nil, fmt.Errorf("validator cannot be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	// Create loader configuration
	loaderConfig := DefaultLoaderConfig()

	// Create components
	loader := NewPolicyLoader(loaderConfig, parser)
	registry := NewPolicyRegistry()

	// Determine base path for include resolution
	basePath := config.FilePath
	if basePath == "" {
		basePath = "."
	}
	resolver := NewIncludeResolver(loaderConfig, loader, filepath.Dir(basePath))

	pm := &DefaultPolicyManager{
		config:           config,
		loader:           loader,
		resolver:         resolver,
		registry:         registry,
		parser:           parser,
		validator:        validator,
		logger:           logger,
		watchEvents:      make(chan ReloadEvent, 100),
		lastGoodPolicies: []*ast.Policy{},
	}

	// Initialize Git source if mode is "git"
	if config.Mode == "git" && config.Git.Enabled {
		logger.Info("Initializing Git policy source",
			"repository", config.Git.Repository,
			"branch", config.Git.Branch,
		)

		// Create Git repository manager
		gitRepo, err := git.NewRepository(&config.Git)
		if err != nil {
			return nil, fmt.Errorf("failed to create git repository: %w", err)
		}

		// Clone repository
		ctx, cancel := context.WithTimeout(context.Background(), config.Git.Poll.Timeout)
		defer cancel()

		if err := gitRepo.Clone(ctx); err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}

		pm.gitRepo = gitRepo

		// Start watcher if polling enabled
		if config.Git.Poll.Enabled {
			gitWatcher := git.NewWatcher(
				gitRepo,
				config.Git.Poll.Interval,
				config.Git.Poll.Timeout,
				pm.reloadPoliciesFromGit,
			)

			pm.gitWatcher = gitWatcher
		}
	}

	return pm, nil
}

// LoadPolicies loads all policies from the configured source.
// This performs validation and registration with atomic updates.
func (m *DefaultPolicyManager) LoadPolicies() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	startTime := time.Now()
	m.logger.Info("Loading policies",
		"mode", m.config.Mode,
		"path", m.config.FilePath,
	)

	// Load policies from file system
	policies, err := m.loadPoliciesFromSource()
	if err != nil {
		m.lastLoadError = err
		m.logger.Error("Failed to load policies",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return err
	}

	// Validate all policies before applying
	if err := m.validatePolicies(policies); err != nil {
		m.lastLoadError = err
		m.logger.Error("Policy validation failed",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return err
	}

	// Atomically replace policies in registry
	if err := m.registry.Replace(policies); err != nil {
		m.lastLoadError = err
		m.logger.Error("Failed to register policies",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		return err
	}

	// Update state
	m.lastLoadTime = time.Now()
	m.lastLoadError = nil
	m.lastGoodPolicies = policies

	m.logger.Info("Policies loaded successfully",
		"count", len(policies),
		"version", m.registry.GetVersion(),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// ReloadPolicies reloads all policies from the configured source.
// This is an atomic operation with error recovery.
func (m *DefaultPolicyManager) ReloadPolicies() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	startTime := time.Now()
	m.logger.Info("Reloading policies",
		"mode", m.config.Mode,
		"path", m.config.FilePath,
	)

	// Load policies from file system
	policies, err := m.loadPoliciesFromSource()
	if err != nil {
		m.lastLoadError = err
		m.logger.Error("Failed to reload policies, keeping previous policies",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		// Return error but keep last good policies
		return err
	}

	// Validate all policies before applying
	if err := m.validatePolicies(policies); err != nil {
		m.lastLoadError = err
		m.logger.Error("Policy validation failed during reload, keeping previous policies",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		// Return error but keep last good policies
		return err
	}

	// Atomically replace policies in registry
	if err := m.registry.Replace(policies); err != nil {
		m.lastLoadError = err
		m.logger.Error("Failed to register policies during reload, keeping previous policies",
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
		// Attempt to restore last good policies
		if len(m.lastGoodPolicies) > 0 {
			_ = m.registry.Replace(m.lastGoodPolicies)
		}
		return err
	}

	// Update state
	m.lastLoadTime = time.Now()
	m.lastLoadError = nil
	m.lastGoodPolicies = policies

	m.logger.Info("Policies reloaded successfully",
		"count", len(policies),
		"version", m.registry.GetVersion(),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// GetPolicy retrieves a single policy by ID.
func (m *DefaultPolicyManager) GetPolicy(id string) (*ast.Policy, error) {
	policy, ok := m.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("policy %q not found", id)
	}
	return policy, nil
}

// GetAllPolicies retrieves all loaded policies.
func (m *DefaultPolicyManager) GetAllPolicies() []*ast.Policy {
	return m.registry.GetAll()
}

// GetPolicyVersion returns the version of the currently loaded policies.
func (m *DefaultPolicyManager) GetPolicyVersion() string {
	return m.registry.GetVersion()
}

// Watch starts watching the policy source for changes.
// This implements hot-reload functionality.
func (m *DefaultPolicyManager) Watch(ctx context.Context) error {
	m.watchMu.Lock()
	if m.watchCancel != nil {
		m.watchMu.Unlock()
		return fmt.Errorf("watch already started")
	}

	m.watchCtx, m.watchCancel = context.WithCancel(ctx)
	m.watchMu.Unlock()

	// Git mode uses its own watcher
	if m.config.Mode == "git" && m.gitWatcher != nil {
		m.logger.Info("Starting Git policy watcher",
			"repository", m.config.Git.Repository,
			"branch", m.config.Git.Branch,
			"poll_interval", m.config.Git.Poll.Interval,
		)

		if err := m.gitWatcher.Start(m.watchCtx); err != nil {
			return fmt.Errorf("failed to start git watcher: %w", err)
		}

		// Wait for context cancellation
		<-m.watchCtx.Done()

		return m.gitWatcher.Stop()
	}

	// File mode uses file system watcher
	m.logger.Info("Starting policy watcher",
		"path", m.config.FilePath,
		"watch_enabled", m.config.Watch,
	)

	// Check if watch is enabled in config
	if !m.config.Watch {
		m.logger.Debug("Policy watching disabled in configuration")
		return fmt.Errorf("policy watching is not enabled in configuration")
	}

	// Create file watcher
	watchConfig := DefaultFileWatcherConfig()
	watchConfig.Path = m.config.FilePath

	watcher, err := NewFileWatcher(watchConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Start watching in background
	go func() {
		if err := watcher.Watch(m.watchCtx, func() error {
			// Trigger policy reload
			return m.ReloadPolicies()
		}); err != nil {
			m.logger.Error("File watcher error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-m.watchCtx.Done()

	// Stop watcher
	if err := watcher.Stop(); err != nil {
		m.logger.Error("Failed to stop file watcher", "error", err)
		return err
	}

	return nil
}

// Close performs cleanup and releases resources.
func (m *DefaultPolicyManager) Close() error {
	m.watchMu.Lock()
	if m.watchCancel != nil {
		m.watchCancel()
		m.watchCancel = nil
	}
	m.watchMu.Unlock()

	// Stop Git watcher if running
	if m.gitWatcher != nil {
		if err := m.gitWatcher.Stop(); err != nil {
			m.logger.Error("Failed to stop git watcher", "error", err)
		}
	}

	m.logger.Info("Policy manager closed")
	return nil
}

// ValidatePoliciesDryRun validates policies without applying them to the registry.
// This is useful for testing policy files before deployment or for linting operations.
// It performs all validation steps but does not modify the active policy set.
func (m *DefaultPolicyManager) ValidatePoliciesDryRun() error {
	m.logger.Info("Dry-run validation", "path", m.config.FilePath)

	// Load policies from source
	policies, err := m.loadPoliciesFromSource()
	if err != nil {
		return fmt.Errorf("failed to load policies: %w", err)
	}

	// Validate all policies (includes rule deduplication check)
	if err := m.validatePolicies(policies); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	m.logger.Info("Dry-run validation successful",
		"count", len(policies),
	)

	return nil
}

// loadPoliciesFromSource loads policies from the configured source.
func (m *DefaultPolicyManager) loadPoliciesFromSource() ([]*ast.Policy, error) {
	var policies []*ast.Policy
	var err error

	// Determine policy path based on mode
	policyPath := m.config.FilePath

	// If Git mode, use local cloned repository path
	if m.config.Mode == "git" && m.gitRepo != nil {
		policyPath = m.gitRepo.GetPolicyPath()
	}

	// Check if path is a file or directory
	isDir, err := m.loader.IsDirectory(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access policy path: %w", err)
	}

	if isDir {
		// Load from directory
		policies, err = m.loader.LoadFromDirectory(policyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load policies from directory: %w", err)
		}
	} else {
		// Load single file
		policy, err := m.loader.LoadFromFile(policyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load policy file: %w", err)
		}
		policies = []*ast.Policy{policy}
	}

	// Resolve includes (if any)
	// Check if any policies have includes
	hasIncludes := false
	for _, p := range policies {
		if len(p.Includes) > 0 {
			hasIncludes = true
			break
		}
	}

	// If no includes, return as-is
	if !hasIncludes {
		return policies, nil
	}

	// Extract file paths for resolution
	policyPaths := make([]string, 0, len(policies))
	for _, p := range policies {
		if p.SourceFile != "" {
			policyPaths = append(policyPaths, p.SourceFile)
		}
	}

	// Resolve all includes recursively
	graph, err := m.resolver.ResolveMultiple(policyPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve includes: %w", err)
	}

	// Get policies in topological order (dependencies loaded first)
	sortedPaths := m.resolver.GetSortedPaths()
	resolvedPolicies := make([]*ast.Policy, 0, len(sortedPaths))
	for _, path := range sortedPaths {
		if node, ok := graph.Nodes[path]; ok {
			resolvedPolicies = append(resolvedPolicies, node.Policy)
		}
	}

	return resolvedPolicies, nil
}

// validatePolicies validates all policies before loading.
func (m *DefaultPolicyManager) validatePolicies(policies []*ast.Policy) error {
	if !m.config.Validation.Enabled {
		m.logger.Debug("Policy validation disabled")
		return nil
	}

	errList := &ErrorList{}

	for _, policy := range policies {
		if err := m.validator.Validate(policy); err != nil {
			errList.Add(&ValidationError{
				PolicyID: policy.Name,
				Message:  err.Error(),
				Cause:    err,
			})

			// In strict mode, fail immediately
			if m.config.Validation.Strict {
				return errList.ToError()
			}
		}
	}

	// Check for duplicate policy names
	seen := make(map[string]bool)
	for _, policy := range policies {
		if seen[policy.Name] {
			m.logger.Warn("Duplicate policy name detected",
				"policy", policy.Name,
			)
			// Last policy wins
		}
		seen[policy.Name] = true
	}

	// Check for duplicate rule IDs across all policies
	m.checkDuplicateRuleIDs(policies)

	return errList.ToError()
}

// checkDuplicateRuleIDs detects and warns about duplicate rule IDs across policies.
// This helps identify potential conflicts when multiple policies define rules with the same name.
func (m *DefaultPolicyManager) checkDuplicateRuleIDs(policies []*ast.Policy) {
	// Map from rule ID to list of policies that define it
	ruleLocations := make(map[string][]string)

	for _, policy := range policies {
		for _, rule := range policy.Rules {
			if rule.Name != "" {
				ruleLocations[rule.Name] = append(ruleLocations[rule.Name], policy.Name)
			}
		}
	}

	// Warn about any duplicates
	for ruleID, policyNames := range ruleLocations {
		if len(policyNames) > 1 {
			m.logger.Warn("Duplicate rule ID detected across policies",
				"rule_id", ruleID,
				"policies", policyNames,
				"count", len(policyNames),
			)
		}
	}
}

// reloadPoliciesFromGit is the callback for Git watcher.
// It's called when the watcher detects policy file changes.
// The policyPath parameter is provided by the watcher but not used
// since we determine the path from the Git repository.
// Returns an error if validation fails, triggering automatic rollback.
func (m *DefaultPolicyManager) reloadPoliciesFromGit(policyPath string) error {
	m.logger.Info("Git watcher triggered policy reload",
		"path", policyPath,
	)

	// Use ReloadPolicies for atomic updates and error recovery
	return m.ReloadPolicies()
}

// GetLastLoadTime returns the timestamp of the last successful load.
func (m *DefaultPolicyManager) GetLastLoadTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastLoadTime
}

// GetLastLoadError returns the error from the last load attempt.
func (m *DefaultPolicyManager) GetLastLoadError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastLoadError
}

// GetRegistry returns the underlying policy registry.
// This is useful for testing and introspection.
func (m *DefaultPolicyManager) GetRegistry() *PolicyRegistry {
	return m.registry
}

// PolicySource interface implementation for engine integration

// LoadPolicies implements engine.PolicySource.LoadPolicies.
// This allows the policy engine to load policies through the manager.
func (m *DefaultPolicyManager) LoadPoliciesForEngine(ctx context.Context) ([]*ast.Policy, error) {
	// Load policies if not already loaded
	if m.registry.Count() == 0 {
		if err := m.LoadPolicies(); err != nil {
			return nil, err
		}
	}
	return m.GetAllPolicies(), nil
}

// WatchForEngine implements engine.PolicySource.Watch.
// This allows the policy engine to watch for policy changes.
func (m *DefaultPolicyManager) WatchForEngine(ctx context.Context) (<-chan PolicyEvent, error) {
	events := make(chan PolicyEvent, 100)

	// Start watching in background
	go func() {
		defer close(events)

		if err := m.Watch(ctx); err != nil {
			events <- PolicyEvent{
				Type:  PolicyEventError,
				Error: err,
			}
			return
		}

		// Forward reload events to engine events
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-m.watchEvents:
				if !ok {
					return
				}

				// Convert ReloadEvent to PolicyEvent
				engineEvent := PolicyEvent{
					Path: event.FilePath,
				}

				switch event.Type {
				case ReloadEventCreate:
					engineEvent.Type = PolicyEventCreated
				case ReloadEventModify:
					engineEvent.Type = PolicyEventModified
				case ReloadEventDelete:
					engineEvent.Type = PolicyEventDeleted
				}

				select {
				case events <- engineEvent:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return events, nil
}

// PolicyEvent types for engine integration
type PolicyEventType int

const (
	PolicyEventCreated PolicyEventType = iota
	PolicyEventModified
	PolicyEventDeleted
	PolicyEventError
)

type PolicyEvent struct {
	Type  PolicyEventType
	Path  string
	Error error
}

// Git-specific methods

// GetCurrentCommit returns the current Git commit information.
// Returns nil if not in Git mode or if repository is not initialized.
func (m *DefaultPolicyManager) GetCurrentCommit() (*git.CommitInfo, error) {
	if m.config.Mode != "git" || m.gitRepo == nil {
		return nil, fmt.Errorf("not in git mode")
	}

	return m.gitRepo.GetCurrentCommit()
}

// GetCommitHistory returns the commit history for the policy repository.
// The limit parameter controls how many commits to retrieve.
// Returns nil if not in Git mode or if repository is not initialized.
func (m *DefaultPolicyManager) GetCommitHistory(limit int) ([]*git.CommitInfo, error) {
	if m.config.Mode != "git" || m.gitRepo == nil {
		return nil, fmt.Errorf("not in git mode")
	}

	return m.gitRepo.GetCommitHistory(limit)
}

// RollbackToCommit rolls back policies to a specific Git commit.
// This performs a Git checkout to the target commit, reloads policies,
// and validates them before replacing the active policy set.
// Returns an error if not in Git mode, if the target commit doesn't exist,
// or if the policies at that commit fail validation.
func (m *DefaultPolicyManager) RollbackToCommit(ctx context.Context, commitSHA string) error {
	if m.config.Mode != "git" || m.gitRepo == nil {
		return fmt.Errorf("not in git mode")
	}

	m.logger.Info("Rolling back to commit",
		"commit_sha", commitSHA,
	)

	// Perform Git rollback
	if err := m.gitRepo.Rollback(ctx, commitSHA); err != nil {
		return fmt.Errorf("failed to rollback git repository: %w", err)
	}

	// Reload policies from rolled back commit
	if err := m.ReloadPolicies(); err != nil {
		m.logger.Error("Failed to load policies after rollback",
			"commit_sha", commitSHA,
			"error", err,
		)
		return fmt.Errorf("failed to load policies after rollback: %w", err)
	}

	m.logger.Info("Successfully rolled back to commit",
		"commit_sha", commitSHA,
	)

	return nil
}

// ForceSync forces a Git pull to sync with the remote repository.
// This is useful for manual triggering of policy updates.
// If policy validation fails after pulling, the repository is rolled back
// to the previous commit and the last-known-good policies are retained.
// Returns an error if not in Git mode or if the pull operation fails.
func (m *DefaultPolicyManager) ForceSync(ctx context.Context) error {
	if m.config.Mode != "git" || m.gitRepo == nil {
		return fmt.Errorf("not in git mode")
	}

	m.logger.Info("Forcing Git sync")

	// Pull latest changes
	result, err := m.gitRepo.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	// If no changes, nothing to do
	if !result.HadChanges {
		m.logger.Info("No changes detected")
		return nil
	}

	m.logger.Info("Changes detected, reloading policies",
		"from_sha", result.FromSHA,
		"to_sha", result.ToSHA,
		"changed_files", len(result.ChangedFiles),
	)

	// Reload policies
	if err := m.ReloadPolicies(); err != nil {
		m.logger.Error("Failed to reload policies after sync, rolling back",
			"error", err,
			"from_sha", result.ToSHA,
			"to_sha", result.FromSHA,
		)

		// Rollback to previous commit
		rollbackErr := m.gitRepo.Rollback(ctx, result.FromSHA)
		if rollbackErr != nil {
			m.logger.Error("Failed to rollback after validation failure",
				"error", rollbackErr,
				"target_sha", result.FromSHA,
			)
			// Return both errors
			return fmt.Errorf("failed to reload policies: %w (rollback also failed: %v)", err, rollbackErr)
		}

		m.logger.Info("Successfully rolled back to previous commit",
			"sha", result.FromSHA,
		)

		return fmt.Errorf("failed to reload policies: %w", err)
	}

	return nil
}

// GetGitMetrics returns performance metrics for Git operations.
// Returns zero value if not in Git mode.
func (m *DefaultPolicyManager) GetGitMetrics() git.RepositoryMetrics {
	if m.config.Mode != "git" || m.gitRepo == nil {
		return git.RepositoryMetrics{}
	}

	return m.gitRepo.GetMetrics()
}
