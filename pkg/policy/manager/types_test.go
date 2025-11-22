package manager

import (
	"testing"
)

func TestReloadEventType_String(t *testing.T) {
	tests := []struct {
		name  string
		event ReloadEventType
		want  string
	}{
		{
			name:  "create event",
			event: ReloadEventCreate,
			want:  "create",
		},
		{
			name:  "modify event",
			event: ReloadEventModify,
			want:  "modify",
		},
		{
			name:  "delete event",
			event: ReloadEventDelete,
			want:  "delete",
		},
		{
			name:  "unknown event",
			event: ReloadEventType(999),
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReloadEvent(t *testing.T) {
	event := ReloadEvent{
		Type:     ReloadEventModify,
		FilePath: "/test/policy.yaml",
	}

	if event.Type != ReloadEventModify {
		t.Errorf("Type = %v, want %v", event.Type, ReloadEventModify)
	}

	if event.FilePath != "/test/policy.yaml" {
		t.Errorf("FilePath = %q, want %q", event.FilePath, "/test/policy.yaml")
	}
}

func TestPolicyMetadata(t *testing.T) {
	meta := &PolicyMetadata{
		ID:          "test-policy-id",
		Name:        "test-policy",
		Version:     "1.0.0",
		Author:      "Test Author",
		Description: "Test policy description",
	}

	if meta.ID != "test-policy-id" {
		t.Errorf("ID = %q, want %q", meta.ID, "test-policy-id")
	}

	if meta.Name != "test-policy" {
		t.Errorf("Name = %q, want %q", meta.Name, "test-policy")
	}

	if meta.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.0.0")
	}

	if meta.Author != "Test Author" {
		t.Errorf("Author = %q, want %q", meta.Author, "Test Author")
	}
}

func TestDependencyGraph(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*PolicyNode),
		Edges: make(map[string][]string),
	}

	// Add nodes
	graph.Nodes["policy1"] = &PolicyNode{
		FilePath: "/test/policy1.yaml",
		Includes: []string{"policy2"},
	}
	graph.Nodes["policy2"] = &PolicyNode{
		FilePath: "/test/policy2.yaml",
		Includes: []string{},
	}

	// Add edges
	graph.Edges["policy1"] = []string{"policy2"}

	if len(graph.Nodes) != 2 {
		t.Errorf("Nodes count = %d, want 2", len(graph.Nodes))
	}

	if len(graph.Edges) != 1 {
		t.Errorf("Edges count = %d, want 1", len(graph.Edges))
	}

	if len(graph.Edges["policy1"]) != 1 {
		t.Errorf("Edges[policy1] count = %d, want 1", len(graph.Edges["policy1"]))
	}
}

func TestPolicyNode(t *testing.T) {
	node := &PolicyNode{
		FilePath: "/test/policy.yaml",
		Includes: []string{"/test/include1.yaml", "/test/include2.yaml"},
	}

	if node.FilePath != "/test/policy.yaml" {
		t.Errorf("FilePath = %q, want %q", node.FilePath, "/test/policy.yaml")
	}

	if len(node.Includes) != 2 {
		t.Errorf("Includes count = %d, want 2", len(node.Includes))
	}
}

func TestLoadResult(t *testing.T) {
	result := &LoadResult{
		Policies: nil,
		Errors:   nil,
		Warnings: nil,
	}

	if result.Policies != nil {
		t.Error("Policies should be nil initially")
	}

	if result.Errors != nil {
		t.Error("Errors should be nil initially")
	}

	if result.Warnings != nil {
		t.Error("Warnings should be nil initially")
	}
}
