package parser

import (
	"fmt"
	"path/filepath"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

// Composer handles policy composition through includes/imports.
// It supports loading multiple policy files and merging them into a single policy.
type Composer struct {
	parser  *Parser
	visited map[string]bool // Tracks visited files to detect cycles
	stack   []string        // Current import stack for cycle detection
}

// NewComposer creates a new policy composer.
func NewComposer(parser *Parser) *Composer {
	return &Composer{
		parser:  parser,
		visited: make(map[string]bool),
		stack:   make([]string, 0),
	}
}

// ComposeFromIncludes loads a policy and all its includes.
// It detects circular imports and returns an error if found.
func (c *Composer) ComposeFromIncludes(mainPath string, includes []string) (*ast.Policy, error) {
	// Reset state
	c.visited = make(map[string]bool)
	c.stack = make([]string, 0)

	// Parse main policy
	policy, err := c.loadPolicyWithTracking(mainPath)
	if err != nil {
		return nil, err
	}

	// Process includes
	for _, includePath := range includes {
		// Resolve relative path
		absPath := includePath
		if !filepath.IsAbs(includePath) {
			baseDir := filepath.Dir(mainPath)
			absPath = filepath.Join(baseDir, includePath)
		}

		// Load included policy
		includedPolicy, err := c.loadPolicyWithTracking(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load include %q: %w", includePath, err)
		}

		// Merge included policy
		c.mergePolicy(policy, includedPolicy)
	}

	return policy, nil
}

// loadPolicyWithTracking loads a policy file while tracking for circular imports.
func (c *Composer) loadPolicyWithTracking(path string) (*ast.Policy, error) {
	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Check for circular import
	if c.isInStack(absPath) {
		cycle := append(c.stack, absPath)
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeSemantic,
			Message: fmt.Sprintf("Circular import detected: %v", cycle),
			Location: ast.Location{
				File: absPath,
			},
			Suggestion: "Remove the circular import dependency",
		}
	}

	// Check if already visited (but not in current stack - this is OK for DAG)
	if c.visited[absPath] {
		// Already loaded in another branch, skip
		return nil, nil
	}

	// Mark as visited and add to stack
	c.visited[absPath] = true
	c.stack = append(c.stack, absPath)

	// Parse policy
	policy, err := c.parser.Parse(absPath)
	if err != nil {
		return nil, err
	}

	// Remove from stack (backtrack)
	c.stack = c.stack[:len(c.stack)-1]

	return policy, nil
}

// isInStack checks if a path is currently in the import stack.
func (c *Composer) isInStack(path string) bool {
	for _, p := range c.stack {
		if p == path {
			return true
		}
	}
	return false
}

// mergePolicy merges an included policy into the main policy.
// Variables from the included policy override variables in the main policy.
// Rules from the included policy are appended to the main policy's rules.
func (c *Composer) mergePolicy(main, included *ast.Policy) {
	if included == nil {
		return
	}

	// Merge variables (later includes override earlier ones)
	for name, variable := range included.Variables {
		main.Variables[name] = variable
	}

	// Append rules (order matters - included rules come after main rules)
	main.Rules = append(main.Rules, included.Rules...)
}

// ComposeFromDirectory loads all policy files in a directory.
// It supports glob patterns and merges all policies into one.
func (c *Composer) ComposeFromDirectory(pattern string) (*ast.Policy, error) {
	// Find matching files
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, &mplErrors.Error{
			Type:    mplErrors.ErrorTypeIO,
			Message: fmt.Sprintf("No policy files found matching pattern %q", pattern),
		}
	}

	// Parse first file as base
	policy, err := c.parser.Parse(matches[0])
	if err != nil {
		return nil, err
	}

	// Merge remaining files
	for _, path := range matches[1:] {
		includedPolicy, err := c.parser.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q: %w", path, err)
		}

		c.mergePolicy(policy, includedPolicy)
	}

	return policy, nil
}
