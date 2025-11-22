// Package manager provides policy management capabilities for loading, validating,
// and managing Mercator Policy Language (MPL) policies from the file system.
//
// The package supports single-file policies, multi-file directory structures,
// an include/import system for policy composition, validation on load, and
// hot-reload capabilities for zero-downtime policy updates. This provides the
// foundation for GitOps-based policy management workflows.
//
// # Core Components
//
// PolicyManager is the main orchestrator that coordinates all policy management
// operations including loading, validation, registration, and hot-reload.
//
// PolicyLoader handles file system operations and YAML parsing, supporting both
// single files and directory structures.
//
// IncludeResolver resolves policy dependencies and detects circular includes
// to enable policy composition.
//
// PolicyRegistry provides thread-safe in-memory storage for loaded policies
// with copy-on-write semantics for atomic updates.
//
// FileWatcher monitors the file system for changes and triggers hot-reload
// with debouncing to prevent reload storms.
//
// # Basic Usage
//
// Loading a single policy file:
//
//	cfg := &config.PolicyConfig{
//	    Mode:     "file",
//	    FilePath: "policies.yaml",
//	    Watch:    false,
//	}
//
//	mgr, err := manager.NewPolicyManager(cfg, parser, validator)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := mgr.LoadPolicies(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get loaded policies
//	policies := mgr.GetAllPolicies()
//	fmt.Printf("Loaded %d policies\n", len(policies))
//
// # Loading Policies with Hot-Reload
//
// Enable file watching for automatic policy reloading:
//
//	cfg := &config.PolicyConfig{
//	    Mode:     "file",
//	    FilePath: "policies/",
//	    Watch:    true,
//	}
//
//	mgr, err := manager.NewPolicyManager(cfg, parser, validator)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := mgr.LoadPolicies(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start watching for changes
//	ctx := context.Background()
//	go func() {
//	    if err := mgr.Watch(ctx); err != nil {
//	        log.Printf("Watcher error: %v", err)
//	    }
//	}()
//
//	// Policies will auto-reload on file changes
//	// ...
//
//	// Graceful shutdown
//	mgr.Close()
//
// # Policy Organization
//
// Policies can be organized in multiple ways:
//
// Single file: All policies in one YAML file
//
//	policies.yaml
//
// Multi-file: Policies split across multiple files
//
//	policies/
//	├── base.yaml
//	├── rules.yaml
//	└── actions.yaml
//
// With includes: Policies can reference shared policy fragments
//
//	policies/
//	├── main.yaml       # includes: [shared/common.yaml]
//	└── shared/
//	    └── common.yaml
//
// # Error Handling
//
// The package provides detailed error types for different failure scenarios:
//
// LoadError: File system and loading errors (file not found, permission denied)
//
// ParseError: YAML parsing errors with line numbers and context
//
// ValidationError: Policy validation errors from the MPL validator
//
// IncludeError: Include resolution errors (missing files, circular dependencies)
//
// All errors implement the standard error interface and provide context
// for troubleshooting.
//
// # Thread Safety
//
// All policy operations are thread-safe. Multiple goroutines can safely:
//
//   - Read policies from the registry
//   - Trigger policy reloads
//   - Access policy metadata
//
// The registry uses copy-on-write semantics to ensure atomic updates without
// blocking concurrent reads during reload operations.
//
// # Performance
//
// The policy manager is designed for high performance:
//
//   - Initial load: <100ms for 100 policy files
//   - Hot-reload: <50ms for 10 modified files
//   - Registry access: <1µs for Get() operations
//   - Memory: <10KB per policy in-memory representation
//
// File changes are debounced (100ms) to prevent reload storms when multiple
// files are modified in quick succession.
//
// # Security
//
// The policy manager implements several security measures:
//
//   - File size limits (10MB) to prevent DoS attacks
//   - Include depth limits (10 levels) to prevent infinite recursion
//   - Path validation to prevent directory traversal attacks
//   - Symlink loop detection to prevent infinite recursion
//   - UTF-8 validation to prevent encoding attacks
//
// Policies are sandboxed to their root directory - includes cannot reference
// files outside the policy directory structure.
package manager
