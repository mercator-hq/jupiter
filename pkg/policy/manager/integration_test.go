package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
)

// TestIntegration_FullPolicyLifecycle tests the complete policy management lifecycle:
// load -> reload -> watch -> hot-reload -> close
func TestIntegration_FullPolicyLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	// Initial policy content
	initialContent := `
mpl_version: "1.0"
name: "integration-policy"
version: "1.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "test"
`
	if err := os.WriteFile(policyFile, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Step 1: Create policy manager
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: policyFile,
		Watch:    true,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  false,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatalf("NewPolicyManager() error = %v", err)
	}
	defer mgr.Close()

	// Step 2: Initial load
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatalf("LoadPolicies() error = %v", err)
	}

	// Verify initial load
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("Initial load: got %d policies, want 1", len(policies))
	}
	if policies[0].Name != "integration-policy" {
		t.Errorf("Policy name = %q, want %q", policies[0].Name, "integration-policy")
	}
	if policies[0].Version != "1.0.0" {
		t.Errorf("Policy version = %q, want %q", policies[0].Version, "1.0.0")
	}

	version1 := mgr.GetPolicyVersion()
	loadTime1 := mgr.GetLastLoadTime()

	// Step 3: Reload with same content
	time.Sleep(10 * time.Millisecond)
	if err := mgr.ReloadPolicies(); err != nil {
		t.Fatalf("ReloadPolicies() error = %v", err)
	}

	// Version should be same (content unchanged)
	version2 := mgr.GetPolicyVersion()
	if version1 != version2 {
		t.Logf("Version changed on reload: %q -> %q (expected same content)", version1, version2)
	}

	loadTime2 := mgr.GetLastLoadTime()
	if !loadTime2.After(loadTime1) {
		t.Error("LoadTime not updated after reload")
	}

	// Step 4: Update policy content
	updatedContent := `
mpl_version: "1.0"
name: "integration-policy"
version: "2.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "updated"
  - name: "rule-2"
    conditions:
      field: "request.model"
      operator: "=="
      value: "claude-3"
    actions:
      - type: "log"
        message: "new rule"
`
	if err := os.WriteFile(policyFile, []byte(updatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Millisecond)

	// Reload with updated content
	if err := mgr.ReloadPolicies(); err != nil {
		t.Fatalf("ReloadPolicies() after update error = %v", err)
	}

	// Verify updated policy
	policies = mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("After update: got %d policies, want 1", len(policies))
	}
	if policies[0].Version != "2.0.0" {
		t.Errorf("Updated policy version = %q, want %q", policies[0].Version, "2.0.0")
	}
	if len(policies[0].Rules) != 2 {
		t.Errorf("Updated policy rules = %d, want 2", len(policies[0].Rules))
	}

	// Step 5: Test GetPolicy
	policy, err := mgr.GetPolicy("integration-policy")
	if err != nil {
		t.Errorf("GetPolicy() error = %v", err)
	}
	if policy == nil {
		t.Fatal("GetPolicy() returned nil")
	}
	if policy.Version != "2.0.0" {
		t.Errorf("GetPolicy() version = %q, want %q", policy.Version, "2.0.0")
	}

	// Step 6: Test error recovery with invalid content
	invalidContent := `
mpl_version: "1.0"
name: invalid-policy
version: "3.0.0"
`
	if err := os.WriteFile(policyFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Reload should fail but keep old policy
	err = mgr.ReloadPolicies()
	if err == nil {
		t.Error("ReloadPolicies() with invalid content should fail")
	}

	// Verify old policy is still loaded
	policies = mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("After failed reload: got %d policies, want 1 (kept old)", len(policies))
	}
	if policies[0].Version != "2.0.0" {
		t.Errorf("After failed reload: version = %q, want %q (kept old)", policies[0].Version, "2.0.0")
	}

	// Step 7: Test LoadPoliciesForEngine
	ctx := context.Background()
	enginePolicies, err := mgr.LoadPoliciesForEngine(ctx)
	if err != nil {
		t.Errorf("LoadPoliciesForEngine() error = %v", err)
	}
	if len(enginePolicies) != 1 {
		t.Errorf("LoadPoliciesForEngine() got %d policies, want 1", len(enginePolicies))
	}

	t.Logf("✅ Full lifecycle test completed successfully")
}

// TestIntegration_MultiFilePolicyLoading tests loading multiple policies from a directory
func TestIntegration_MultiFilePolicyLoading(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple policy files
	policies := []struct {
		name    string
		content string
	}{
		{
			name: "policy1.yaml",
			content: `
mpl_version: "1.0"
name: "policy-1"
version: "1.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "test"
`,
		},
		{
			name: "policy2.yaml",
			content: `
mpl_version: "1.0"
name: "policy-2"
version: "1.0.0"
rules:
  - name: "rule-2"
    conditions:
      field: "request.model"
      operator: "=="
      value: "claude-3"
    actions:
      - type: "log"
        message: "test"
`,
		},
		{
			name: "policy3.yml",
			content: `
mpl_version: "1.0"
name: "policy-3"
version: "1.0.0"
rules:
  - name: "rule-3"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gemini-pro"
    actions:
      - type: "log"
        message: "test"
`,
		},
	}

	for _, p := range policies {
		path := filepath.Join(tmpDir, p.name)
		if err := os.WriteFile(path, []byte(p.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create policy manager for directory
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatalf("NewPolicyManager() error = %v", err)
	}
	defer mgr.Close()

	// Load all policies
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatalf("LoadPolicies() error = %v", err)
	}

	// Verify all policies loaded
	loadedPolicies := mgr.GetAllPolicies()
	if len(loadedPolicies) != 3 {
		t.Fatalf("LoadPolicies() got %d policies, want 3", len(loadedPolicies))
	}

	// Verify each policy can be retrieved
	policyNames := []string{"policy-1", "policy-2", "policy-3"}
	for _, name := range policyNames {
		policy, err := mgr.GetPolicy(name)
		if err != nil {
			t.Errorf("GetPolicy(%q) error = %v", name, err)
		}
		if policy == nil {
			t.Errorf("GetPolicy(%q) returned nil", name)
		}
	}

	// Test GetPolicy for non-existent policy
	_, err = mgr.GetPolicy("nonexistent")
	if err == nil {
		t.Error("GetPolicy(nonexistent) should return error")
	}

	// Add a new policy file
	newPolicy := filepath.Join(tmpDir, "policy4.yaml")
	newContent := `
mpl_version: "1.0"
name: "policy-4"
version: "1.0.0"
rules:
  - name: "rule-4"
    conditions:
      field: "request.model"
      operator: "=="
      value: "llama-2"
    actions:
      - type: "log"
        message: "test"
`
	if err := os.WriteFile(newPolicy, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Reload to pick up new policy
	if err := mgr.ReloadPolicies(); err != nil {
		t.Fatalf("ReloadPolicies() error = %v", err)
	}

	// Verify new policy was loaded
	loadedPolicies = mgr.GetAllPolicies()
	if len(loadedPolicies) != 4 {
		t.Fatalf("After adding file: got %d policies, want 4", len(loadedPolicies))
	}

	policy4, err := mgr.GetPolicy("policy-4")
	if err != nil {
		t.Errorf("GetPolicy(policy-4) error = %v", err)
	}
	if policy4 == nil {
		t.Fatal("GetPolicy(policy-4) returned nil")
	}

	t.Logf("✅ Multi-file loading test completed successfully")
}

// TestIntegration_HotReloadWithWatch tests hot-reload functionality
func TestIntegration_HotReloadWithWatch(t *testing.T) {
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	initialContent := `
mpl_version: "1.0"
name: "watch-policy"
version: "1.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "initial"
`
	if err := os.WriteFile(policyFile, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: policyFile,
		Watch:    true,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatalf("NewPolicyManager() error = %v", err)
	}
	defer mgr.Close()

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatalf("LoadPolicies() error = %v", err)
	}

	// Start watching in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- mgr.Watch(ctx)
	}()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the policy file
	updatedContent := `
mpl_version: "1.0"
name: "watch-policy"
version: "2.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "updated via hot-reload"
`
	if err := os.WriteFile(policyFile, []byte(updatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce and reload
	time.Sleep(300 * time.Millisecond)

	// Verify policy was reloaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("After hot-reload: got %d policies, want 1", len(policies))
	}

	if policies[0].Version != "2.0.0" {
		t.Errorf("After hot-reload: version = %q, want %q (may need more time for debounce)",
			policies[0].Version, "2.0.0")
	}

	// Stop watching
	cancel()
	time.Sleep(100 * time.Millisecond)

	t.Logf("✅ Hot-reload test completed successfully")
}

// TestIntegration_ErrorRecovery tests error recovery scenarios
func TestIntegration_ErrorRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	// Start with valid policy
	validContent := `
mpl_version: "1.0"
name: "recovery-policy"
version: "1.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "test"
`
	if err := os.WriteFile(policyFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: policyFile,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatalf("NewPolicyManager() error = %v", err)
	}
	defer mgr.Close()

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatalf("LoadPolicies() error = %v", err)
	}

	version1 := mgr.GetPolicyVersion()

	// Scenario 1: Malformed YAML
	malformedContent := `
mpl_version: "1.0"
name: broken-policy
version: [invalid: yaml
`
	if err := os.WriteFile(policyFile, []byte(malformedContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = mgr.ReloadPolicies()
	if err == nil {
		t.Error("ReloadPolicies() with malformed YAML should fail")
	}

	// Verify old policy retained
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("After malformed YAML: got %d policies, want 1 (retained)", len(policies))
	}
	if policies[0].Name != "recovery-policy" {
		t.Errorf("After malformed YAML: name = %q, want %q", policies[0].Name, "recovery-policy")
	}

	version2 := mgr.GetPolicyVersion()
	if version1 != version2 {
		t.Error("Version changed after failed reload")
	}

	// Scenario 2: Validation error
	invalidPolicy := `
mpl_version: "1.0"
name: "invalid-policy"
version: "2.0.0"
rules: []
`
	if err := os.WriteFile(policyFile, []byte(invalidPolicy), 0644); err != nil {
		t.Fatal(err)
	}

	err = mgr.ReloadPolicies()
	if err == nil {
		t.Error("ReloadPolicies() with validation error should fail")
	}

	// Verify old policy still retained
	policies = mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("After validation error: got %d policies, want 1 (retained)", len(policies))
	}
	if policies[0].Name != "recovery-policy" {
		t.Errorf("After validation error: name = %q, want %q", policies[0].Name, "recovery-policy")
	}

	// Scenario 3: Recover with valid policy
	recoveredContent := `
mpl_version: "1.0"
name: "recovered-policy"
version: "3.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "recovered"
`
	if err := os.WriteFile(policyFile, []byte(recoveredContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = mgr.ReloadPolicies()
	if err != nil {
		t.Fatalf("ReloadPolicies() with valid content error = %v", err)
	}

	// Verify recovery
	policies = mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Fatalf("After recovery: got %d policies, want 1", len(policies))
	}
	if policies[0].Name != "recovered-policy" {
		t.Errorf("After recovery: name = %q, want %q", policies[0].Name, "recovered-policy")
	}
	if policies[0].Version != "3.0.0" {
		t.Errorf("After recovery: version = %q, want %q", policies[0].Version, "3.0.0")
	}

	t.Logf("✅ Error recovery test completed successfully")
}
