package manager

import (
	"path/filepath"
	"strings"
	"testing"

	"mercator-hq/jupiter/pkg/mpl/parser"
)

func TestIncludeResolver_New(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	if resolver == nil {
		t.Fatal("NewIncludeResolver() returned nil")
	}

	if resolver.config != config {
		t.Error("resolver.config not set correctly")
	}

	if resolver.loader != loader {
		t.Error("resolver.loader not set correctly")
	}

	// Base path should be normalized to absolute path
	expectedBasePath, _ := filepath.Abs("testdata")
	if expectedRealPath, err := filepath.EvalSymlinks(expectedBasePath); err == nil {
		expectedBasePath = expectedRealPath
	}
	if resolver.basePath != expectedBasePath {
		t.Errorf("resolver.basePath = %q, want %q", resolver.basePath, expectedBasePath)
	}

	if resolver.graph == nil {
		t.Error("resolver.graph is nil")
	}

	if resolver.visited == nil {
		t.Error("resolver.visited is nil")
	}

	if resolver.visiting == nil {
		t.Error("resolver.visiting is nil")
	}
}

func TestIncludeResolver_ResolveIncludePath(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	tests := []struct {
		name        string
		currentPath string
		includePath string
		want        string
	}{
		{
			name:        "relative include",
			currentPath: "/path/to/policies/main.yaml",
			includePath: "shared/common.yaml",
			want:        "/path/to/policies/shared/common.yaml",
		},
		{
			name:        "parent directory include",
			currentPath: "/path/to/policies/subdir/policy.yaml",
			includePath: "../shared/common.yaml",
			want:        "/path/to/policies/shared/common.yaml",
		},
		{
			name:        "absolute include",
			currentPath: "/path/to/policies/main.yaml",
			includePath: "/other/path/policy.yaml",
			want:        "/other/path/policy.yaml",
		},
		{
			name:        "current directory include",
			currentPath: "/path/to/policies/main.yaml",
			includePath: "./common.yaml",
			want:        "/path/to/policies/common.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.resolveIncludePath(tt.currentPath, tt.includePath)
			if err != nil {
				t.Fatalf("resolveIncludePath() error = %v, want nil", err)
			}

			if got != tt.want {
				t.Errorf("resolveIncludePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIncludeResolver_NormalizePath(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	// Test with an existing file
	path := filepath.Join("testdata", "valid", "simple.yaml")
	normalized, err := resolver.normalizePath(path)

	if err != nil {
		t.Fatalf("normalizePath() error = %v, want nil", err)
	}

	if !filepath.IsAbs(normalized) {
		t.Errorf("normalizePath() = %q, want absolute path", normalized)
	}

	if !strings.Contains(normalized, "simple.yaml") {
		t.Errorf("normalizePath() = %q, want to contain 'simple.yaml'", normalized)
	}
}

func TestIncludeResolver_ValidateIncludePath(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())

	// Get absolute testdata path
	testdataPath, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}

	resolver := NewIncludeResolver(config, loader, testdataPath)

	tests := []struct {
		name        string
		includePath string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid path within base",
			includePath: filepath.Join(testdataPath, "valid", "simple.yaml"),
			wantErr:     false,
		},
		{
			name:        "directory traversal attempt",
			includePath: filepath.Join(testdataPath, "..", "other", "file.yaml"),
			wantErr:     true,
			errContains: "directory traversal prevented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.validateIncludePath(tt.includePath)

			if tt.wantErr && err == nil {
				t.Error("validateIncludePath() error = nil, want error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("validateIncludePath() error = %v, want nil", err)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateIncludePath() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestIncludeResolver_ResolveIncludes_NoIncludes(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	path := filepath.Join("testdata", "valid", "simple.yaml")
	graph, err := resolver.ResolveIncludes(path)

	if err != nil {
		t.Fatalf("ResolveIncludes() error = %v, want nil", err)
	}

	if graph == nil {
		t.Fatal("ResolveIncludes() returned nil graph")
	}

	if len(graph.Nodes) != 1 {
		t.Errorf("graph.Nodes count = %d, want 1", len(graph.Nodes))
	}

	// Check that the policy was loaded
	normalizedPath, _ := resolver.normalizePath(path)
	if _, ok := graph.Nodes[normalizedPath]; !ok {
		t.Error("graph.Nodes missing the policy")
	}
}

func TestIncludeResolver_ResolveMultiple(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	paths := []string{
		filepath.Join("testdata", "multi", "policy1.yaml"),
		filepath.Join("testdata", "multi", "policy2.yaml"),
	}

	graph, err := resolver.ResolveMultiple(paths)

	if err != nil {
		t.Fatalf("ResolveMultiple() error = %v, want nil", err)
	}

	if graph == nil {
		t.Fatal("ResolveMultiple() returned nil graph")
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("graph.Nodes count = %d, want 2", len(graph.Nodes))
	}
}

func TestIncludeResolver_GetSortedPaths(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	path := filepath.Join("testdata", "valid", "simple.yaml")
	_, err := resolver.ResolveIncludes(path)

	if err != nil {
		t.Fatalf("ResolveIncludes() error = %v, want nil", err)
	}

	sortedPaths := resolver.GetSortedPaths()

	if len(sortedPaths) == 0 {
		t.Error("GetSortedPaths() returned empty slice")
	}

	if len(sortedPaths) != 1 {
		t.Errorf("GetSortedPaths() count = %d, want 1", len(sortedPaths))
	}
}

func TestIncludeResolver_GetDependencyGraph(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	path := filepath.Join("testdata", "valid", "simple.yaml")
	graph1, err := resolver.ResolveIncludes(path)

	if err != nil {
		t.Fatalf("ResolveIncludes() error = %v, want nil", err)
	}

	graph2 := resolver.GetDependencyGraph()

	if graph1 != graph2 {
		t.Error("GetDependencyGraph() returned different graph")
	}
}

func TestIncludeResolver_DetectCycles_NoCycles(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	path := filepath.Join("testdata", "valid", "simple.yaml")
	_, err := resolver.ResolveIncludes(path)

	if err != nil {
		t.Fatalf("ResolveIncludes() error = %v, want nil", err)
	}

	err = resolver.DetectCycles()

	if err != nil {
		t.Errorf("DetectCycles() error = %v, want nil", err)
	}
}

func TestIncludeResolver_TopologicalSort(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	// Create a simple graph manually for testing
	resolver.graph = &DependencyGraph{
		Nodes: make(map[string]*PolicyNode),
		Edges: make(map[string][]string),
	}

	// Add nodes (no cycles)
	resolver.graph.Nodes["a"] = &PolicyNode{FilePath: "a"}
	resolver.graph.Nodes["b"] = &PolicyNode{FilePath: "b"}
	resolver.graph.Nodes["c"] = &PolicyNode{FilePath: "c"}

	// Add edges: a -> b, b -> c
	resolver.graph.Edges["a"] = []string{"b"}
	resolver.graph.Edges["b"] = []string{"c"}
	resolver.graph.Edges["c"] = []string{}

	err := resolver.topologicalSort()

	if err != nil {
		t.Fatalf("topologicalSort() error = %v, want nil", err)
	}

	sortedPaths := resolver.GetSortedPaths()

	if len(sortedPaths) != 3 {
		t.Errorf("GetSortedPaths() count = %d, want 3", len(sortedPaths))
	}

	// Verify order: c should come before b, b before a
	cIndex := indexOf(sortedPaths, "c")
	bIndex := indexOf(sortedPaths, "b")
	aIndex := indexOf(sortedPaths, "a")

	if cIndex < 0 || bIndex < 0 || aIndex < 0 {
		t.Error("Missing nodes in sorted paths")
	}

	if cIndex > bIndex {
		t.Error("Topological sort incorrect: c should come before b")
	}

	if bIndex > aIndex {
		t.Error("Topological sort incorrect: b should come before a")
	}
}

func TestIncludeResolver_ExtractIncludes(t *testing.T) {
	config := DefaultLoaderConfig()
	loader := NewPolicyLoader(config, parser.NewParser())
	resolver := NewIncludeResolver(config, loader, "testdata")

	path := filepath.Join("testdata", "valid", "simple.yaml")
	policy, err := loader.LoadFromFile(path)

	if err != nil {
		t.Fatalf("LoadFromFile() error = %v, want nil", err)
	}

	includes := resolver.extractIncludes(policy)

	// Current implementation returns empty since AST doesn't have includes yet
	if len(includes) != 0 {
		t.Errorf("extractIncludes() count = %d, want 0 (not implemented yet)", len(includes))
	}
}

// Helper function to find index of string in slice
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
