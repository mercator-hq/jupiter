package manager

import (
	"context"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// PolicyManager is the main interface for policy management operations.
// It coordinates policy loading, validation, registration, and hot-reload.
type PolicyManager interface {
	// LoadPolicies loads all policies from the configured source.
	// This performs initial validation and registration with the policy engine.
	// Returns an error if loading fails.
	LoadPolicies() error

	// ReloadPolicies reloads all policies from the configured source.
	// This is an atomic operation - all policies are validated before any
	// are applied. If validation fails, the previous policies remain active.
	// Returns an error if reload fails.
	ReloadPolicies() error

	// GetPolicy retrieves a single policy by ID.
	// Returns nil if the policy is not found.
	GetPolicy(id string) (*ast.Policy, error)

	// GetAllPolicies retrieves all loaded policies.
	// The returned slice is a snapshot and will not be modified by the manager.
	GetAllPolicies() []*ast.Policy

	// GetPolicyVersion returns the version of the currently loaded policies.
	// This is typically a hash of all policy file contents or a timestamp.
	GetPolicyVersion() string

	// Watch starts watching the policy source for changes.
	// When changes are detected, policies are automatically reloaded.
	// This is a blocking operation that runs until the context is cancelled.
	// Returns an error if watching fails to start.
	Watch(ctx context.Context) error

	// Close performs cleanup and releases resources.
	// This stops any active file watchers and cleans up internal state.
	Close() error
}

// PolicyMetadata contains metadata extracted from a policy file.
// This is used for tracking and reporting policy information.
type PolicyMetadata struct {
	// ID is the unique policy identifier
	ID string

	// Name is the human-readable policy name
	Name string

	// Version is the policy version (semantic version or custom)
	Version string

	// Author is the policy author
	Author string

	// Description is the policy description
	Description string

	// CreatedAt is the policy creation timestamp
	CreatedAt time.Time

	// ModifiedAt is the last modification timestamp
	ModifiedAt time.Time

	// FilePath is the path to the policy file
	FilePath string

	// Includes is the list of files included by this policy
	Includes []string

	// RuleCount is the number of rules in the policy
	RuleCount int

	// EnabledRuleCount is the number of enabled rules
	EnabledRuleCount int
}

// LoadResult contains the results of a policy loading operation.
// This includes loaded policies, errors, warnings, and timing information.
type LoadResult struct {
	// Policies is the list of successfully loaded policies
	Policies []*ast.Policy

	// Errors is the list of errors encountered during loading
	Errors []error

	// Warnings is the list of warnings encountered during loading
	Warnings []string

	// LoadTime is the duration of the load operation
	LoadTime time.Duration

	// Version is the version identifier for this load
	Version string

	// FileCount is the number of files processed
	FileCount int
}

// DependencyGraph represents the dependency relationships between policies.
// This is used for topological sorting and cycle detection.
type DependencyGraph struct {
	// Nodes maps policy file paths to their dependency nodes
	Nodes map[string]*PolicyNode

	// Edges maps a file path to the list of files it includes
	Edges map[string][]string
}

// PolicyNode represents a node in the dependency graph.
type PolicyNode struct {
	// Policy is the parsed policy
	Policy *ast.Policy

	// FilePath is the path to the policy file
	FilePath string

	// Includes is the list of files this policy includes
	Includes []string

	// IncludedBy is the list of files that include this policy
	IncludedBy []string

	// Depth is the include depth (0 for root policies)
	Depth int
}

// ReloadEvent represents a file system change event that triggers a reload.
type ReloadEvent struct {
	// Type is the event type (create, modify, delete)
	Type ReloadEventType

	// FilePath is the path to the file that changed
	FilePath string

	// Timestamp is when the event occurred
	Timestamp time.Time
}

// ReloadEventType represents the type of file system change.
type ReloadEventType int

const (
	// ReloadEventCreate indicates a new file was created
	ReloadEventCreate ReloadEventType = iota

	// ReloadEventModify indicates an existing file was modified
	ReloadEventModify

	// ReloadEventDelete indicates a file was deleted
	ReloadEventDelete
)

// String returns a string representation of the event type.
func (t ReloadEventType) String() string {
	switch t {
	case ReloadEventCreate:
		return "create"
	case ReloadEventModify:
		return "modify"
	case ReloadEventDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// PolicyLoaderConfig contains configuration for the policy loader.
type PolicyLoaderConfig struct {
	// MaxFileSize is the maximum file size in bytes (default: 10MB)
	MaxFileSize int64

	// MaxIncludeDepth is the maximum include nesting depth (default: 10)
	MaxIncludeDepth int

	// AllowedExtensions is the list of allowed file extensions (default: [".yaml", ".yml"])
	AllowedExtensions []string

	// FollowSymlinks controls whether to follow symbolic links (default: true)
	FollowSymlinks bool

	// SkipHidden controls whether to skip hidden files/directories (default: true)
	SkipHidden bool
}

// DefaultLoaderConfig returns the default loader configuration.
func DefaultLoaderConfig() *PolicyLoaderConfig {
	return &PolicyLoaderConfig{
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		MaxIncludeDepth:   10,
		AllowedExtensions: []string{".yaml", ".yml"},
		FollowSymlinks:    true,
		SkipHidden:        true,
	}
}
