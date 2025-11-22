package manager

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// PolicyRegistry is a thread-safe in-memory storage for loaded policies.
// It uses copy-on-write semantics for atomic updates.
type PolicyRegistry struct {
	mu       sync.RWMutex
	policies map[string]*ast.Policy
	version  string
	loadTime time.Time
}

// NewPolicyRegistry creates a new empty policy registry.
func NewPolicyRegistry() *PolicyRegistry {
	return &PolicyRegistry{
		policies: make(map[string]*ast.Policy),
		version:  "",
		loadTime: time.Now(),
	}
}

// Register adds a policy to the registry.
// If a policy with the same name already exists, it will be replaced.
func (r *PolicyRegistry) Register(policy *ast.Policy) error {
	if policy == nil {
		return &RegistryError{
			Operation: "register",
			Message:   "policy cannot be nil",
		}
	}

	if policy.Name == "" {
		return &RegistryError{
			Operation: "register",
			Message:   "policy name cannot be empty",
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.policies[policy.Name] = policy
	r.updateVersion()

	return nil
}

// RegisterMultiple adds multiple policies to the registry.
// This is more efficient than calling Register multiple times.
func (r *PolicyRegistry) RegisterMultiple(policies []*ast.Policy) error {
	if len(policies) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate all policies first
	for _, policy := range policies {
		if policy == nil {
			return &RegistryError{
				Operation: "register_multiple",
				Message:   "policy cannot be nil",
			}
		}
		if policy.Name == "" {
			return &RegistryError{
				Operation: "register_multiple",
				Message:   "policy name cannot be empty",
			}
		}
	}

	// Register all policies
	for _, policy := range policies {
		r.policies[policy.Name] = policy
	}

	r.updateVersion()
	return nil
}

// Unregister removes a policy from the registry by name.
func (r *PolicyRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.policies[name]; !ok {
		return &RegistryError{
			PolicyID:  name,
			Operation: "unregister",
			Message:   "policy not found",
		}
	}

	delete(r.policies, name)
	r.updateVersion()

	return nil
}

// Get retrieves a policy by name.
// Returns nil if the policy is not found.
func (r *PolicyRegistry) Get(name string) (*ast.Policy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	policy, ok := r.policies[name]
	return policy, ok
}

// GetAll retrieves all policies in the registry.
// The returned slice is a copy and will not be modified by the registry.
func (r *PolicyRegistry) GetAll() []*ast.Policy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	policies := make([]*ast.Policy, 0, len(r.policies))
	for _, policy := range r.policies {
		policies = append(policies, policy)
	}

	return policies
}

// GetAllSorted retrieves all policies sorted by name.
func (r *PolicyRegistry) GetAllSorted() []*ast.Policy {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all policy names and sort them
	names := make([]string, 0, len(r.policies))
	for name := range r.policies {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build sorted slice
	policies := make([]*ast.Policy, 0, len(r.policies))
	for _, name := range names {
		policies = append(policies, r.policies[name])
	}

	return policies
}

// Count returns the number of policies in the registry.
func (r *PolicyRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.policies)
}

// Clear removes all policies from the registry.
func (r *PolicyRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.policies = make(map[string]*ast.Policy)
	r.updateVersion()
}

// Clone creates a deep copy of the registry.
// This is used for copy-on-write updates.
func (r *PolicyRegistry) Clone() *PolicyRegistry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	newRegistry := &PolicyRegistry{
		policies: make(map[string]*ast.Policy, len(r.policies)),
		version:  r.version,
		loadTime: r.loadTime,
	}

	// Copy all policies
	for name, policy := range r.policies {
		newRegistry.policies[name] = policy
	}

	return newRegistry
}

// Replace atomically replaces the entire policy set with a new set.
// This is used for atomic hot-reload operations.
func (r *PolicyRegistry) Replace(policies []*ast.Policy) error {
	if policies == nil {
		return &RegistryError{
			Operation: "replace",
			Message:   "policies cannot be nil",
		}
	}

	// Validate all policies first
	for _, policy := range policies {
		if policy == nil {
			return &RegistryError{
				Operation: "replace",
				Message:   "policy cannot be nil",
			}
		}
		if policy.Name == "" {
			return &RegistryError{
				Operation: "replace",
				Message:   "policy name cannot be empty",
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Create new policy map
	newPolicies := make(map[string]*ast.Policy, len(policies))
	for _, policy := range policies {
		newPolicies[policy.Name] = policy
	}

	// Atomic swap
	r.policies = newPolicies
	r.loadTime = time.Now()
	r.updateVersion()

	return nil
}

// GetVersion returns the current version of the registry.
// The version changes whenever policies are added, removed, or replaced.
func (r *PolicyRegistry) GetVersion() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.version
}

// GetLoadTime returns the timestamp when policies were last loaded or updated.
func (r *PolicyRegistry) GetLoadTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.loadTime
}

// GetMetadata returns metadata for all policies in the registry.
func (r *PolicyRegistry) GetMetadata() []PolicyMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make([]PolicyMetadata, 0, len(r.policies))
	for _, policy := range r.policies {
		metadata = append(metadata, PolicyMetadata{
			ID:               policy.Name,
			Name:             policy.Name,
			Version:          policy.Version,
			Author:           policy.Author,
			Description:      policy.Description,
			CreatedAt:        policy.Created,
			ModifiedAt:       policy.Updated,
			FilePath:         policy.SourceFile,
			RuleCount:        len(policy.Rules),
			EnabledRuleCount: len(policy.EnabledRules()),
		})
	}

	return metadata
}

// HasPolicy checks if a policy with the given name exists in the registry.
func (r *PolicyRegistry) HasPolicy(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.policies[name]
	return ok
}

// GetPolicyNames returns a sorted list of all policy names in the registry.
func (r *PolicyRegistry) GetPolicyNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.policies))
	for name := range r.policies {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetStats returns statistics about the policies in the registry.
func (r *PolicyRegistry) GetStats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		PolicyCount: len(r.policies),
		LoadTime:    r.loadTime,
		Version:     r.version,
	}

	for _, policy := range r.policies {
		stats.TotalRules += len(policy.Rules)
		stats.EnabledRules += len(policy.EnabledRules())
	}

	return stats
}

// updateVersion updates the registry version based on the current state.
// This should be called with the write lock held.
func (r *PolicyRegistry) updateVersion() {
	// Generate version hash based on policy names and versions
	h := sha256.New()

	// Get sorted policy names for deterministic hashing
	names := make([]string, 0, len(r.policies))
	for name := range r.policies {
		names = append(names, name)
	}
	sort.Strings(names)

	// Hash each policy's name and version
	for _, name := range names {
		policy := r.policies[name]
		h.Write([]byte(policy.Name))
		h.Write([]byte(policy.Version))
		h.Write([]byte(policy.SourceFile))
	}

	r.version = fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// RegistryStats contains statistics about the policy registry.
type RegistryStats struct {
	PolicyCount  int
	TotalRules   int
	EnabledRules int
	LoadTime     time.Time
	Version      string
}
