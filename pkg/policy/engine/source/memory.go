package source

import (
	"context"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/policy/engine"
)

// MemorySource is an in-memory policy source for testing.
type MemorySource struct {
	policies []*ast.Policy
}

// NewMemorySource creates a new in-memory policy source.
func NewMemorySource(policies ...*ast.Policy) *MemorySource {
	return &MemorySource{
		policies: policies,
	}
}

// LoadPolicies returns the policies stored in memory.
func (s *MemorySource) LoadPolicies(ctx context.Context) ([]*ast.Policy, error) {
	// Return a copy to prevent external modification
	policies := make([]*ast.Policy, len(s.policies))
	copy(policies, s.policies)
	return policies, nil
}

// Watch returns a channel that never sends events (for testing).
func (s *MemorySource) Watch(ctx context.Context) (<-chan engine.PolicyEvent, error) {
	eventCh := make(chan engine.PolicyEvent)

	// Close channel when context is cancelled
	go func() {
		<-ctx.Done()
		close(eventCh)
	}()

	return eventCh, nil
}

// SetPolicies updates the policies in memory (for testing).
func (s *MemorySource) SetPolicies(policies []*ast.Policy) {
	s.policies = policies
}
