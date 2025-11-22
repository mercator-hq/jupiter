package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// IncludeResolver resolves policy include dependencies and detects circular references.
// It builds a dependency graph and provides topological sorting for load order.
type IncludeResolver struct {
	config      *PolicyLoaderConfig
	loader      *PolicyLoader
	basePath    string
	graph       *DependencyGraph
	visited     map[string]bool
	visiting    map[string]bool
	sortedPaths []string
}

// NewIncludeResolver creates a new include resolver.
func NewIncludeResolver(config *PolicyLoaderConfig, loader *PolicyLoader, basePath string) *IncludeResolver {
	// Normalize the base path to resolve symlinks (e.g., /var -> /private/var on macOS)
	normalizedBasePath := basePath
	if absPath, err := filepath.Abs(basePath); err == nil {
		if realPath, err := filepath.EvalSymlinks(absPath); err == nil {
			normalizedBasePath = realPath
		}
	}

	return &IncludeResolver{
		config:   config,
		loader:   loader,
		basePath: normalizedBasePath,
		graph: &DependencyGraph{
			Nodes: make(map[string]*PolicyNode),
			Edges: make(map[string][]string),
		},
		visited:  make(map[string]bool),
		visiting: make(map[string]bool),
	}
}

// ResolveIncludes resolves all includes for a policy file and returns the dependency graph.
// It loads included policies recursively and detects circular dependencies.
func (r *IncludeResolver) ResolveIncludes(rootPath string) (*DependencyGraph, error) {
	// Reset state
	r.graph = &DependencyGraph{
		Nodes: make(map[string]*PolicyNode),
		Edges: make(map[string][]string),
	}
	r.visited = make(map[string]bool)
	r.visiting = make(map[string]bool)
	r.sortedPaths = nil

	// Resolve the root policy and its includes
	if err := r.resolveFile(rootPath, 0); err != nil {
		return nil, err
	}

	// Perform topological sort
	if err := r.topologicalSort(); err != nil {
		return nil, err
	}

	return r.graph, nil
}

// ResolveMultiple resolves includes for multiple policy files and returns a single dependency graph.
func (r *IncludeResolver) ResolveMultiple(paths []string) (*DependencyGraph, error) {
	// Reset state
	r.graph = &DependencyGraph{
		Nodes: make(map[string]*PolicyNode),
		Edges: make(map[string][]string),
	}
	r.visited = make(map[string]bool)
	r.visiting = make(map[string]bool)
	r.sortedPaths = nil

	// Resolve all root policies
	for _, path := range paths {
		if err := r.resolveFile(path, 0); err != nil {
			return nil, err
		}
	}

	// Perform topological sort
	if err := r.topologicalSort(); err != nil {
		return nil, err
	}

	return r.graph, nil
}

// resolveFile recursively resolves a policy file and its includes.
func (r *IncludeResolver) resolveFile(path string, depth int) error {
	// Normalize path
	normalizedPath, err := r.normalizePath(path)
	if err != nil {
		return &IncludeError{
			FilePath: path,
			Message:  "failed to normalize path",
			Cause:    err,
		}
	}

	// Check if already visited
	if r.visited[normalizedPath] {
		return nil
	}

	// Check for circular dependency
	if r.visiting[normalizedPath] {
		cycle := r.buildCycle(normalizedPath)
		return &IncludeError{
			FilePath: normalizedPath,
			Cycle:    cycle,
			Message:  "circular include detected",
		}
	}

	// Check include depth
	if depth > r.config.MaxIncludeDepth {
		return &IncludeError{
			FilePath: normalizedPath,
			Message:  fmt.Sprintf("include depth %d exceeds maximum %d", depth, r.config.MaxIncludeDepth),
		}
	}

	// Mark as visiting (for cycle detection)
	r.visiting[normalizedPath] = true

	// Load the policy
	policy, err := r.loader.LoadFromFile(normalizedPath)
	if err != nil {
		return err
	}

	// Create policy node
	node := &PolicyNode{
		Policy:     policy,
		FilePath:   normalizedPath,
		Includes:   []string{},
		IncludedBy: []string{},
		Depth:      depth,
	}

	// Extract includes from policy metadata
	// Note: The current AST doesn't have an Includes field, so we'll need to handle this
	// For now, we'll look for includes in the policy metadata or add it to the AST later
	includes := r.extractIncludes(policy)

	// Resolve each include
	for _, includePath := range includes {
		// Resolve relative to current file
		resolvedPath, err := r.resolveIncludePath(normalizedPath, includePath)
		if err != nil {
			return &IncludeError{
				FilePath:    normalizedPath,
				IncludePath: includePath,
				Message:     "failed to resolve include path",
				Cause:       err,
			}
		}

		// Validate the include path (prevent directory traversal)
		if err := r.validateIncludePath(resolvedPath); err != nil {
			return err
		}

		node.Includes = append(node.Includes, resolvedPath)

		// Recursively resolve the included file
		if err := r.resolveFile(resolvedPath, depth+1); err != nil {
			return err
		}

		// Update included-by relationship
		if includedNode, ok := r.graph.Nodes[resolvedPath]; ok {
			includedNode.IncludedBy = append(includedNode.IncludedBy, normalizedPath)
		}
	}

	// Add node to graph
	r.graph.Nodes[normalizedPath] = node
	r.graph.Edges[normalizedPath] = node.Includes

	// Mark as visited
	r.visiting[normalizedPath] = false
	r.visited[normalizedPath] = true

	return nil
}

// extractIncludes extracts include paths from a policy.
func (r *IncludeResolver) extractIncludes(policy *ast.Policy) []string {
	if policy.Includes == nil {
		return []string{}
	}
	return policy.Includes
}

// resolveIncludePath resolves an include path relative to the current file.
func (r *IncludeResolver) resolveIncludePath(currentPath, includePath string) (string, error) {
	// Get the directory of the current file
	currentDir := filepath.Dir(currentPath)

	// If include path is absolute, use it as-is (but validate later)
	if filepath.IsAbs(includePath) {
		return filepath.Clean(includePath), nil
	}

	// Resolve relative to current file's directory
	resolvedPath := filepath.Join(currentDir, includePath)
	return filepath.Clean(resolvedPath), nil
}

// normalizePath normalizes a file path to its absolute canonical form.
func (r *IncludeResolver) normalizePath(path string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If file doesn't exist, that's okay - return the absolute path
		if os.IsNotExist(err) {
			return absPath, nil
		}
		return "", err
	}

	return realPath, nil
}

// validateIncludePath validates that an include path is safe (prevents directory traversal).
func (r *IncludeResolver) validateIncludePath(includePath string) error {
	// Get absolute base path
	absBasePath, err := filepath.Abs(r.basePath)
	if err != nil {
		return &IncludeError{
			FilePath:    includePath,
			IncludePath: includePath,
			Message:     "failed to resolve base path",
			Cause:       err,
		}
	}

	// Get absolute include path
	absIncludePath, err := filepath.Abs(includePath)
	if err != nil {
		return &IncludeError{
			FilePath:    includePath,
			IncludePath: includePath,
			Message:     "failed to resolve include path",
			Cause:       err,
		}
	}

	// Check if include path is within base path (prevent directory traversal)
	relPath, err := filepath.Rel(absBasePath, absIncludePath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return &IncludeError{
			FilePath:    includePath,
			IncludePath: includePath,
			Message:     "include path outside of policy directory (directory traversal prevented)",
		}
	}

	return nil
}

// buildCycle builds the cycle path for error reporting.
func (r *IncludeResolver) buildCycle(startPath string) []string {
	// Find the cycle by traversing the visiting set
	cycle := []string{startPath}

	// This is a simplified implementation
	// In a real implementation, we'd track the path during DFS traversal
	for path := range r.visiting {
		if path != startPath && r.visiting[path] {
			cycle = append(cycle, path)
		}
	}

	cycle = append(cycle, startPath) // Close the cycle
	return cycle
}

// topologicalSort performs topological sorting on the dependency graph.
// This determines the order in which policies should be loaded.
func (r *IncludeResolver) topologicalSort() error {
	r.sortedPaths = nil
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(path string) error
	visit = func(path string) error {
		if visited[path] {
			return nil
		}

		if visiting[path] {
			return &IncludeError{
				FilePath: path,
				Message:  "circular dependency detected during topological sort",
			}
		}

		visiting[path] = true

		// Visit all dependencies first
		for _, dep := range r.graph.Edges[path] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[path] = false
		visited[path] = true
		r.sortedPaths = append(r.sortedPaths, path)

		return nil
	}

	// Visit all nodes
	for path := range r.graph.Nodes {
		if err := visit(path); err != nil {
			return err
		}
	}

	return nil
}

// GetSortedPaths returns the topologically sorted list of policy paths.
// Policies are ordered such that dependencies are loaded before dependents.
func (r *IncludeResolver) GetSortedPaths() []string {
	return r.sortedPaths
}

// GetDependencyGraph returns the resolved dependency graph.
func (r *IncludeResolver) GetDependencyGraph() *DependencyGraph {
	return r.graph
}

// DetectCycles checks if the dependency graph contains any cycles.
func (r *IncludeResolver) DetectCycles() error {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(path string, pathStack []string) error
	visit = func(path string, pathStack []string) error {
		if visited[path] {
			return nil
		}

		if visiting[path] {
			// Found a cycle
			cycle := append(pathStack, path)
			return &IncludeError{
				FilePath: path,
				Cycle:    cycle,
				Message:  "circular dependency detected",
			}
		}

		visiting[path] = true
		pathStack = append(pathStack, path)

		// Visit all dependencies
		for _, dep := range r.graph.Edges[path] {
			if err := visit(dep, pathStack); err != nil {
				return err
			}
		}

		visiting[path] = false
		visited[path] = true

		return nil
	}

	// Check all nodes
	for path := range r.graph.Nodes {
		if err := visit(path, []string{}); err != nil {
			return err
		}
	}

	return nil
}
