package manager

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/mpl/parser"
)

func TestNewPolicyRegistry(t *testing.T) {
	registry := NewPolicyRegistry()

	if registry == nil {
		t.Fatal("NewPolicyRegistry() returned nil")
	}

	if registry.policies == nil {
		t.Error("registry.policies is nil")
	}

	if registry.Count() != 0 {
		t.Errorf("registry.Count() = %d, want 0", registry.Count())
	}
}

func TestPolicyRegistry_Register(t *testing.T) {
	registry := NewPolicyRegistry()
	policy := createTestPolicy("test-policy", "1.0.0")

	err := registry.Register(policy)

	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	if registry.Count() != 1 {
		t.Errorf("registry.Count() = %d, want 1", registry.Count())
	}

	// Verify policy can be retrieved
	retrieved, ok := registry.Get("test-policy")
	if !ok {
		t.Error("Get() returned false, want true")
	}

	if retrieved.Name != "test-policy" {
		t.Errorf("retrieved policy name = %q, want %q", retrieved.Name, "test-policy")
	}
}

func TestPolicyRegistry_Register_NilPolicy(t *testing.T) {
	registry := NewPolicyRegistry()

	err := registry.Register(nil)

	if err == nil {
		t.Fatal("Register(nil) error = nil, want error")
	}

	_, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("Register(nil) error type = %T, want *RegistryError", err)
	}
}

func TestPolicyRegistry_Register_EmptyName(t *testing.T) {
	registry := NewPolicyRegistry()
	policy := &ast.Policy{Name: ""}

	err := registry.Register(policy)

	if err == nil {
		t.Fatal("Register(empty name) error = nil, want error")
	}

	_, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("Register(empty name) error type = %T, want *RegistryError", err)
	}
}

func TestPolicyRegistry_Register_ReplaceExisting(t *testing.T) {
	registry := NewPolicyRegistry()

	policy1 := createTestPolicy("test-policy", "1.0.0")
	err := registry.Register(policy1)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	policy2 := createTestPolicy("test-policy", "2.0.0")
	err = registry.Register(policy2)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	// Should still have only 1 policy
	if registry.Count() != 1 {
		t.Errorf("registry.Count() = %d, want 1", registry.Count())
	}

	// Should have the new version
	retrieved, _ := registry.Get("test-policy")
	if retrieved.Version != "2.0.0" {
		t.Errorf("retrieved policy version = %q, want %q", retrieved.Version, "2.0.0")
	}
}

func TestPolicyRegistry_RegisterMultiple(t *testing.T) {
	registry := NewPolicyRegistry()

	policies := []*ast.Policy{
		createTestPolicy("policy-1", "1.0.0"),
		createTestPolicy("policy-2", "1.0.0"),
		createTestPolicy("policy-3", "1.0.0"),
	}

	err := registry.RegisterMultiple(policies)

	if err != nil {
		t.Fatalf("RegisterMultiple() error = %v, want nil", err)
	}

	if registry.Count() != 3 {
		t.Errorf("registry.Count() = %d, want 3", registry.Count())
	}
}

func TestPolicyRegistry_Unregister(t *testing.T) {
	registry := NewPolicyRegistry()
	policy := createTestPolicy("test-policy", "1.0.0")

	registry.Register(policy)

	err := registry.Unregister("test-policy")

	if err != nil {
		t.Fatalf("Unregister() error = %v, want nil", err)
	}

	if registry.Count() != 0 {
		t.Errorf("registry.Count() = %d, want 0", registry.Count())
	}

	// Verify policy is gone
	_, ok := registry.Get("test-policy")
	if ok {
		t.Error("Get() returned true after Unregister, want false")
	}
}

func TestPolicyRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewPolicyRegistry()

	err := registry.Unregister("nonexistent")

	if err == nil {
		t.Fatal("Unregister(nonexistent) error = nil, want error")
	}

	_, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("Unregister(nonexistent) error type = %T, want *RegistryError", err)
	}
}

func TestPolicyRegistry_Get(t *testing.T) {
	registry := NewPolicyRegistry()
	policy := createTestPolicy("test-policy", "1.0.0")

	registry.Register(policy)

	retrieved, ok := registry.Get("test-policy")

	if !ok {
		t.Error("Get() returned false, want true")
	}

	if retrieved == nil {
		t.Fatal("Get() returned nil policy")
	}

	if retrieved.Name != "test-policy" {
		t.Errorf("retrieved policy name = %q, want %q", retrieved.Name, "test-policy")
	}
}

func TestPolicyRegistry_Get_NotFound(t *testing.T) {
	registry := NewPolicyRegistry()

	_, ok := registry.Get("nonexistent")

	if ok {
		t.Error("Get(nonexistent) returned true, want false")
	}
}

func TestPolicyRegistry_GetAll(t *testing.T) {
	registry := NewPolicyRegistry()

	policies := []*ast.Policy{
		createTestPolicy("policy-1", "1.0.0"),
		createTestPolicy("policy-2", "1.0.0"),
		createTestPolicy("policy-3", "1.0.0"),
	}

	registry.RegisterMultiple(policies)

	allPolicies := registry.GetAll()

	if len(allPolicies) != 3 {
		t.Errorf("GetAll() count = %d, want 3", len(allPolicies))
	}
}

func TestPolicyRegistry_GetAllSorted(t *testing.T) {
	registry := NewPolicyRegistry()

	policies := []*ast.Policy{
		createTestPolicy("policy-c", "1.0.0"),
		createTestPolicy("policy-a", "1.0.0"),
		createTestPolicy("policy-b", "1.0.0"),
	}

	registry.RegisterMultiple(policies)

	sortedPolicies := registry.GetAllSorted()

	if len(sortedPolicies) != 3 {
		t.Errorf("GetAllSorted() count = %d, want 3", len(sortedPolicies))
	}

	// Verify alphabetical order
	expectedOrder := []string{"policy-a", "policy-b", "policy-c"}
	for i, policy := range sortedPolicies {
		if policy.Name != expectedOrder[i] {
			t.Errorf("sortedPolicies[%d].Name = %q, want %q", i, policy.Name, expectedOrder[i])
		}
	}
}

func TestPolicyRegistry_Count(t *testing.T) {
	registry := NewPolicyRegistry()

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}

	registry.Register(createTestPolicy("policy-1", "1.0.0"))

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	registry.Register(createTestPolicy("policy-2", "1.0.0"))

	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}
}

func TestPolicyRegistry_Clear(t *testing.T) {
	registry := NewPolicyRegistry()

	policies := []*ast.Policy{
		createTestPolicy("policy-1", "1.0.0"),
		createTestPolicy("policy-2", "1.0.0"),
	}

	registry.RegisterMultiple(policies)

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", registry.Count())
	}
}

func TestPolicyRegistry_Clone(t *testing.T) {
	registry := NewPolicyRegistry()

	policies := []*ast.Policy{
		createTestPolicy("policy-1", "1.0.0"),
		createTestPolicy("policy-2", "1.0.0"),
	}

	registry.RegisterMultiple(policies)

	clone := registry.Clone()

	if clone == nil {
		t.Fatal("Clone() returned nil")
	}

	if clone.Count() != registry.Count() {
		t.Errorf("clone.Count() = %d, want %d", clone.Count(), registry.Count())
	}

	// Verify cloned policies
	for name := range registry.policies {
		if !clone.HasPolicy(name) {
			t.Errorf("clone missing policy %q", name)
		}
	}

	// Verify modifying clone doesn't affect original
	clone.Register(createTestPolicy("policy-3", "1.0.0"))

	if registry.Count() == clone.Count() {
		t.Error("Modifying clone affected original registry")
	}
}

func TestPolicyRegistry_Replace(t *testing.T) {
	registry := NewPolicyRegistry()

	// Initial policies
	initial := []*ast.Policy{
		createTestPolicy("policy-1", "1.0.0"),
		createTestPolicy("policy-2", "1.0.0"),
	}
	registry.RegisterMultiple(initial)

	// Replace with new policies
	replacement := []*ast.Policy{
		createTestPolicy("policy-3", "1.0.0"),
		createTestPolicy("policy-4", "1.0.0"),
	}

	err := registry.Replace(replacement)

	if err != nil {
		t.Fatalf("Replace() error = %v, want nil", err)
	}

	if registry.Count() != 2 {
		t.Errorf("registry.Count() = %d, want 2", registry.Count())
	}

	// Verify old policies are gone
	if registry.HasPolicy("policy-1") {
		t.Error("registry still has policy-1 after Replace")
	}
	if registry.HasPolicy("policy-2") {
		t.Error("registry still has policy-2 after Replace")
	}

	// Verify new policies are present
	if !registry.HasPolicy("policy-3") {
		t.Error("registry missing policy-3 after Replace")
	}
	if !registry.HasPolicy("policy-4") {
		t.Error("registry missing policy-4 after Replace")
	}
}

func TestPolicyRegistry_GetVersion(t *testing.T) {
	registry := NewPolicyRegistry()

	version1 := registry.GetVersion()

	// Register a policy
	registry.Register(createTestPolicy("policy-1", "1.0.0"))

	version2 := registry.GetVersion()

	if version1 == version2 {
		t.Error("Version did not change after registering policy")
	}
}

func TestPolicyRegistry_GetLoadTime(t *testing.T) {
	registry := NewPolicyRegistry()

	loadTime := registry.GetLoadTime()

	if loadTime.IsZero() {
		t.Error("GetLoadTime() returned zero time")
	}

	// Replace policies (should update load time)
	time.Sleep(10 * time.Millisecond)
	registry.Replace([]*ast.Policy{createTestPolicy("policy-1", "1.0.0")})

	newLoadTime := registry.GetLoadTime()

	if !newLoadTime.After(loadTime) {
		t.Error("LoadTime did not update after Replace")
	}
}

func TestPolicyRegistry_GetMetadata(t *testing.T) {
	registry := NewPolicyRegistry()

	policy := createTestPolicy("test-policy", "1.0.0")
	policy.Author = "Test Author"
	policy.Description = "Test Description"

	registry.Register(policy)

	metadata := registry.GetMetadata()

	if len(metadata) != 1 {
		t.Errorf("GetMetadata() count = %d, want 1", len(metadata))
	}

	meta := metadata[0]
	if meta.Name != "test-policy" {
		t.Errorf("metadata.Name = %q, want %q", meta.Name, "test-policy")
	}

	if meta.Version != "1.0.0" {
		t.Errorf("metadata.Version = %q, want %q", meta.Version, "1.0.0")
	}
}

func TestPolicyRegistry_HasPolicy(t *testing.T) {
	registry := NewPolicyRegistry()

	if registry.HasPolicy("nonexistent") {
		t.Error("HasPolicy(nonexistent) = true, want false")
	}

	registry.Register(createTestPolicy("test-policy", "1.0.0"))

	if !registry.HasPolicy("test-policy") {
		t.Error("HasPolicy(test-policy) = false, want true")
	}
}

func TestPolicyRegistry_GetPolicyNames(t *testing.T) {
	registry := NewPolicyRegistry()

	policies := []*ast.Policy{
		createTestPolicy("policy-c", "1.0.0"),
		createTestPolicy("policy-a", "1.0.0"),
		createTestPolicy("policy-b", "1.0.0"),
	}

	registry.RegisterMultiple(policies)

	names := registry.GetPolicyNames()

	if len(names) != 3 {
		t.Errorf("GetPolicyNames() count = %d, want 3", len(names))
	}

	// Verify alphabetical order
	expectedOrder := []string{"policy-a", "policy-b", "policy-c"}
	for i, name := range names {
		if name != expectedOrder[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expectedOrder[i])
		}
	}
}

func TestPolicyRegistry_GetStats(t *testing.T) {
	registry := NewPolicyRegistry()

	policy1 := createTestPolicy("policy-1", "1.0.0")
	policy1.Rules = []*ast.Rule{
		{Name: "rule-1", Enabled: true},
		{Name: "rule-2", Enabled: true},
	}

	policy2 := createTestPolicy("policy-2", "1.0.0")
	policy2.Rules = []*ast.Rule{
		{Name: "rule-3", Enabled: true},
		{Name: "rule-4", Enabled: false},
	}

	registry.RegisterMultiple([]*ast.Policy{policy1, policy2})

	stats := registry.GetStats()

	if stats.PolicyCount != 2 {
		t.Errorf("stats.PolicyCount = %d, want 2", stats.PolicyCount)
	}

	if stats.TotalRules != 4 {
		t.Errorf("stats.TotalRules = %d, want 4", stats.TotalRules)
	}

	if stats.EnabledRules != 3 {
		t.Errorf("stats.EnabledRules = %d, want 3", stats.EnabledRules)
	}
}

func TestPolicyRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewPolicyRegistry()

	// Load initial policies
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())
	path1 := filepath.Join("testdata", "multi", "policy1.yaml")
	path2 := filepath.Join("testdata", "multi", "policy2.yaml")

	policy1, _ := loader.LoadFromFile(path1)
	policy2, _ := loader.LoadFromFile(path2)

	registry.RegisterMultiple([]*ast.Policy{policy1, policy2})

	// Concurrent reads and writes
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = registry.GetAll()
				_ = registry.Count()
				_ = registry.GetVersion()
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				policy := createTestPolicy("concurrent-policy", "1.0.0")
				_ = registry.Register(policy)
			}
		}(i)
	}

	wg.Wait()

	// Verify registry is still consistent
	if registry.Count() < 2 {
		t.Errorf("registry.Count() = %d, want >= 2", registry.Count())
	}
}

// Helper function to create a test policy
func createTestPolicy(name, version string) *ast.Policy {
	return &ast.Policy{
		Name:       name,
		Version:    version,
		MPLVersion: "1.0",
		Created:    time.Now(),
		Updated:    time.Now(),
		Variables:  make(map[string]*ast.Variable),
		Rules:      []*ast.Rule{},
		SourceFile: "/test/" + name + ".yaml",
	}
}
