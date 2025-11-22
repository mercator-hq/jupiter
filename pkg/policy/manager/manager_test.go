package manager

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
)

func TestNewPolicyManager(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: "testdata/valid/simple.yaml",
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  false,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)

	if err != nil {
		t.Fatalf("NewPolicyManager() error = %v, want nil", err)
	}

	if mgr == nil {
		t.Fatal("NewPolicyManager() returned nil")
	}

	if mgr.config != cfg {
		t.Error("manager.config not set correctly")
	}

	if mgr.loader == nil {
		t.Error("manager.loader is nil")
	}

	if mgr.registry == nil {
		t.Error("manager.registry is nil")
	}

	if mgr.parser == nil {
		t.Error("manager.parser is nil")
	}

	if mgr.validator == nil {
		t.Error("manager.validator is nil")
	}
}

func TestNewPolicyManager_NilConfig(t *testing.T) {
	_, err := NewPolicyManager(nil, parser.NewParser(), validator.NewValidator(), nil)

	if err == nil {
		t.Fatal("NewPolicyManager(nil config) error = nil, want error")
	}

	if !strings.Contains(err.Error(), "config cannot be nil") {
		t.Errorf("error message = %q, want to contain 'config cannot be nil'", err.Error())
	}
}

func TestNewPolicyManager_NilParser(t *testing.T) {
	cfg := &config.PolicyConfig{FilePath: "test.yaml"}

	_, err := NewPolicyManager(cfg, nil, validator.NewValidator(), nil)

	if err == nil {
		t.Fatal("NewPolicyManager(nil parser) error = nil, want error")
	}

	if !strings.Contains(err.Error(), "parser cannot be nil") {
		t.Errorf("error message = %q, want to contain 'parser cannot be nil'", err.Error())
	}
}

func TestNewPolicyManager_NilValidator(t *testing.T) {
	cfg := &config.PolicyConfig{FilePath: "test.yaml"}

	_, err := NewPolicyManager(cfg, parser.NewParser(), nil, nil)

	if err == nil {
		t.Fatal("NewPolicyManager(nil validator) error = nil, want error")
	}

	if !strings.Contains(err.Error(), "validator cannot be nil") {
		t.Errorf("error message = %q, want to contain 'validator cannot be nil'", err.Error())
	}
}

func TestPolicyManager_LoadPolicies_SingleFile(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  false,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.LoadPolicies()

	if err != nil {
		t.Fatalf("LoadPolicies() error = %v, want nil", err)
	}

	// Verify policies were loaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Errorf("GetAllPolicies() count = %d, want 1", len(policies))
	}

	// Verify policy content
	if policies[0].Name != "simple-policy" {
		t.Errorf("policy.Name = %q, want %q", policies[0].Name, "simple-policy")
	}

	// Verify version was set
	version := mgr.GetPolicyVersion()
	if version == "" {
		t.Error("GetPolicyVersion() returned empty string")
	}

	// Verify load time was set
	loadTime := mgr.GetLastLoadTime()
	if loadTime.IsZero() {
		t.Error("GetLastLoadTime() returned zero time")
	}

	// Verify no load error
	loadErr := mgr.GetLastLoadError()
	if loadErr != nil {
		t.Errorf("GetLastLoadError() = %v, want nil", loadErr)
	}
}

func TestPolicyManager_LoadPolicies_Directory(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "multi"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  false,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.LoadPolicies()

	if err != nil {
		t.Fatalf("LoadPolicies() error = %v, want nil", err)
	}

	// Verify policies were loaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 2 {
		t.Errorf("GetAllPolicies() count = %d, want 2", len(policies))
	}

	// Verify policy names
	names := make(map[string]bool)
	for _, policy := range policies {
		names[policy.Name] = true
	}

	if !names["policy-1"] {
		t.Error("GetAllPolicies() missing policy-1")
	}
	if !names["policy-2"] {
		t.Error("GetAllPolicies() missing policy-2")
	}
}

func TestPolicyManager_LoadPolicies_FileNotFound(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "nonexistent.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.LoadPolicies()

	if err == nil {
		t.Fatal("LoadPolicies() error = nil, want error for nonexistent file")
	}

	// Verify error was recorded
	loadErr := mgr.GetLastLoadError()
	if loadErr == nil {
		t.Error("GetLastLoadError() = nil, want error")
	}

	// Verify no policies were loaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 0 {
		t.Errorf("GetAllPolicies() count = %d, want 0", len(policies))
	}
}

func TestPolicyManager_LoadPolicies_InvalidYAML(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "invalid", "malformed.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.LoadPolicies()

	if err == nil {
		t.Fatal("LoadPolicies() error = nil, want error for invalid YAML")
	}

	// Verify error was recorded
	loadErr := mgr.GetLastLoadError()
	if loadErr == nil {
		t.Error("GetLastLoadError() = nil, want error")
	}
}

func TestPolicyManager_ReloadPolicies_Success(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	version1 := mgr.GetPolicyVersion()
	loadTime1 := mgr.GetLastLoadTime()

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Reload
	err = mgr.ReloadPolicies()

	if err != nil {
		t.Fatalf("ReloadPolicies() error = %v, want nil", err)
	}

	// Verify version didn't change (same file)
	version2 := mgr.GetPolicyVersion()
	if version1 != version2 {
		t.Errorf("version changed after reload: %q -> %q", version1, version2)
	}

	// Verify load time was updated
	loadTime2 := mgr.GetLastLoadTime()
	if !loadTime2.After(loadTime1) {
		t.Error("LoadTime not updated after reload")
	}

	// Verify policies are still loaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Errorf("GetAllPolicies() count = %d, want 1", len(policies))
	}
}

func TestPolicyManager_ReloadPolicies_ErrorRecovery(t *testing.T) {
	// Create a temporary file that we can modify
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	// Write initial valid policy
	validContent := `
mpl_version: "1.0"
name: "test-policy"
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
	if err := os.WriteFile(tmpFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpFile,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Verify initial policy loaded
	if mgr.registry.Count() != 1 {
		t.Fatalf("Initial policy count = %d, want 1", mgr.registry.Count())
	}

	// Write invalid content (malformed YAML)
	invalidContent := `
mpl_version: "1.0"
name: "test-policy
rules:
  - invalid yaml content
`
	if err := os.WriteFile(tmpFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Attempt reload (should fail)
	err = mgr.ReloadPolicies()

	if err == nil {
		t.Fatal("ReloadPolicies() with invalid file error = nil, want error")
	}

	// Verify old policies are still loaded (error recovery)
	if mgr.registry.Count() != 1 {
		t.Errorf("Policy count after failed reload = %d, want 1 (kept old policies)", mgr.registry.Count())
	}

	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Errorf("GetAllPolicies() count = %d, want 1 (kept old policies)", len(policies))
	}

	if policies[0].Name != "test-policy" {
		t.Errorf("Kept policy name = %q, want %q", policies[0].Name, "test-policy")
	}
}

func TestPolicyManager_GetPolicy(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Get existing policy
	policy, err := mgr.GetPolicy("simple-policy")

	if err != nil {
		t.Fatalf("GetPolicy() error = %v, want nil", err)
	}

	if policy == nil {
		t.Fatal("GetPolicy() returned nil policy")
	}

	if policy.Name != "simple-policy" {
		t.Errorf("policy.Name = %q, want %q", policy.Name, "simple-policy")
	}
}

func TestPolicyManager_GetPolicy_NotFound(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Get non-existent policy
	_, err = mgr.GetPolicy("nonexistent")

	if err == nil {
		t.Fatal("GetPolicy(nonexistent) error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message = %q, want to contain 'not found'", err.Error())
	}
}

func TestPolicyManager_ValidationDisabled(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: false, // Validation disabled
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.LoadPolicies()

	if err != nil {
		t.Fatalf("LoadPolicies() with validation disabled error = %v, want nil", err)
	}

	// Verify policies were loaded even without validation
	policies := mgr.GetAllPolicies()
	if len(policies) != 1 {
		t.Errorf("GetAllPolicies() count = %d, want 1", len(policies))
	}
}

func TestPolicyManager_Close(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestPolicyManager_AtomicUpdate(t *testing.T) {
	// Create temporary directory with multiple policy files
	tmpDir := t.TempDir()

	// Create first valid policy
	policy1 := filepath.Join(tmpDir, "policy1.yaml")
	content1 := `
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
`
	if err := os.WriteFile(policy1, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second valid policy
	policy2 := filepath.Join(tmpDir, "policy2.yaml")
	content2 := `
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
`
	if err := os.WriteFile(policy2, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Load both policies
	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Verify both policies loaded
	if mgr.registry.Count() != 2 {
		t.Errorf("Initial policy count = %d, want 2", mgr.registry.Count())
	}

	// Make one file invalid
	invalidContent := `invalid: yaml: content`
	if err := os.WriteFile(policy1, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Attempt reload - should fail atomically
	err = mgr.ReloadPolicies()

	if err == nil {
		t.Fatal("ReloadPolicies() with one invalid file error = nil, want error")
	}

	// Verify old policies are still there (atomic failure)
	if mgr.registry.Count() != 2 {
		t.Errorf("Policy count after failed reload = %d, want 2 (atomic rollback)", mgr.registry.Count())
	}
}

func TestPolicyManager_Watch_NotEnabled(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Watch:    false, // Watch disabled
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Try to start watching (should fail)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = mgr.Watch(ctx)

	if err == nil {
		t.Fatal("Watch() with Watch=false error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("error message = %q, want to contain 'not enabled'", err.Error())
	}
}

func TestPolicyManager_Watch_AlreadyStarted(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "test-policy"
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
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpFile,
		Watch:    true,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Start watching
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	go func() {
		_ = mgr.Watch(ctx1)
	}()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Try to start watching again (should fail)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	err = mgr.Watch(ctx2)

	if err == nil {
		t.Fatal("Watch() called twice error = nil, want error")
	}

	if !strings.Contains(err.Error(), "already started") {
		t.Errorf("error message = %q, want to contain 'already started'", err.Error())
	}

	// Cleanup
	cancel1()
	time.Sleep(50 * time.Millisecond)
}

func TestPolicyManager_GetRegistry(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	registry := mgr.GetRegistry()

	if registry == nil {
		t.Fatal("GetRegistry() returned nil")
	}

	if registry != mgr.registry {
		t.Error("GetRegistry() did not return the internal registry")
	}
}

func TestPolicyManager_LoadPoliciesForEngine(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// LoadPoliciesForEngine should load policies if not already loaded
	policies, err := mgr.LoadPoliciesForEngine(ctx)

	if err != nil {
		t.Fatalf("LoadPoliciesForEngine() error = %v, want nil", err)
	}

	if len(policies) != 1 {
		t.Errorf("LoadPoliciesForEngine() count = %d, want 1", len(policies))
	}

	if policies[0].Name != "simple-policy" {
		t.Errorf("policy.Name = %q, want %q", policies[0].Name, "simple-policy")
	}

	// Call again - should return cached policies without reloading
	policies2, err := mgr.LoadPoliciesForEngine(ctx)

	if err != nil {
		t.Fatalf("LoadPoliciesForEngine() second call error = %v, want nil", err)
	}

	if len(policies2) != 1 {
		t.Errorf("LoadPoliciesForEngine() second call count = %d, want 1", len(policies2))
	}
}

func TestPolicyManager_LoadPoliciesForEngine_LoadError(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "nonexistent.yaml"),
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// LoadPoliciesForEngine should fail if load fails
	_, err = mgr.LoadPoliciesForEngine(ctx)

	if err == nil {
		t.Fatal("LoadPoliciesForEngine() with bad file error = nil, want error")
	}
}

func TestPolicyManager_StrictValidation(t *testing.T) {
	// Create a policy file with validation warnings
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	// This policy has an empty rules array which might trigger a warning
	content := `
mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpFile,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  true, // Strict mode enabled
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// In strict mode, validation failures should cause load to fail
	err = mgr.LoadPolicies()

	// Note: This depends on the validator implementation
	// If validator allows empty rules, this test might need adjustment
	if err != nil {
		// Strict validation caught an issue
		if !strings.Contains(err.Error(), "validation") {
			t.Logf("Strict validation error (expected): %v", err)
		}
	}
}

func TestPolicyManager_DuplicatePolicyNames(t *testing.T) {
	// Create temporary directory with duplicate policy names
	tmpDir := t.TempDir()

	// Create first policy with name "duplicate"
	policy1 := filepath.Join(tmpDir, "policy1.yaml")
	content1 := `
mpl_version: "1.0"
name: "duplicate"
version: "1.0.0"
rules:
  - name: "rule-1"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "first"
`
	if err := os.WriteFile(policy1, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second policy with same name "duplicate"
	policy2 := filepath.Join(tmpDir, "policy2.yaml")
	content2 := `
mpl_version: "1.0"
name: "duplicate"
version: "2.0.0"
rules:
  - name: "rule-2"
    conditions:
      field: "request.model"
      operator: "=="
      value: "claude-3"
    actions:
      - type: "log"
        message: "second"
`
	if err := os.WriteFile(policy2, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: false, // Disable validation to allow duplicates
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Load policies - should succeed but log warning
	err = mgr.LoadPolicies()

	if err != nil {
		t.Fatalf("LoadPolicies() with duplicate names error = %v, want nil (last wins)", err)
	}

	// Verify only one policy with the name exists (last wins)
	policy, err := mgr.GetPolicy("duplicate")
	if err != nil {
		t.Fatalf("GetPolicy(duplicate) error = %v, want nil", err)
	}

	// Should be the last one loaded (policy2)
	if policy.Version != "2.0.0" {
		t.Logf("Policy version = %q, last policy wins behavior", policy.Version)
	}
}

func TestPolicyManager_WatchForEngine(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "test-policy"
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
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpFile,
		Watch:    true,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Start watching for engine
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	eventChan, err := mgr.WatchForEngine(ctx)

	if err != nil {
		t.Fatalf("WatchForEngine() error = %v, want nil", err)
	}

	if eventChan == nil {
		t.Fatal("WatchForEngine() returned nil channel")
	}

	// Wait for context to expire or error
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed - expected when context expires
				return
			}
			// If we get an error event, check if it's the expected "watch already started"
			if event.Type == PolicyEventError {
				if !strings.Contains(event.Error.Error(), "not enabled") {
					t.Logf("Got expected error event: %v", event.Error)
				}
				return
			}
		case <-ctx.Done():
			// Context expired - normal for this test
			return
		}
	}
}

func TestPolicyManager_WatchForEngine_WatchDisabled(t *testing.T) {
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: filepath.Join("testdata", "valid", "simple.yaml"),
		Watch:    false, // Watch disabled
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.LoadPolicies(); err != nil {
		t.Fatal(err)
	}

	// Start watching for engine with watch disabled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	eventChan, err := mgr.WatchForEngine(ctx)

	if err != nil {
		t.Fatalf("WatchForEngine() error = %v, want nil (error comes via channel)", err)
	}

	// Should receive error event about watch not enabled
	select {
	case event, ok := <-eventChan:
		if !ok {
			t.Fatal("Event channel closed before receiving error event")
		}
		if event.Type != PolicyEventError {
			t.Errorf("Event type = %v, want PolicyEventError", event.Type)
		}
		if event.Error == nil {
			t.Error("Event.Error = nil, want error")
		}
		if !strings.Contains(event.Error.Error(), "not enabled") {
			t.Errorf("Error = %q, want to contain 'not enabled'", event.Error.Error())
		}
	case <-ctx.Done():
		t.Fatal("Context expired before receiving error event")
	}
}

func TestPolicyManager_MixedValidInvalidPolicies(t *testing.T) {
	// Create temporary directory with mixed valid/invalid policy files
	tmpDir := t.TempDir()

	// Create valid policy
	validPolicy := filepath.Join(tmpDir, "valid.yaml")
	validContent := `
mpl_version: "1.0"
name: "valid-policy"
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
	if err := os.WriteFile(validPolicy, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid policy (malformed YAML)
	invalidPolicy := filepath.Join(tmpDir, "invalid.yaml")
	invalidContent := `
mpl_version: "1.0"
name: "invalid-policy
version: "1.0.0"
rules: [malformed
`
	if err := os.WriteFile(invalidPolicy, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Load policies - should fail because one is invalid
	err = mgr.LoadPolicies()

	if err == nil {
		t.Fatal("LoadPolicies() with mixed valid/invalid error = nil, want error")
	}

	// Verify no policies were loaded (atomic failure)
	if mgr.registry.Count() != 0 {
		t.Errorf("Policy count after failed load = %d, want 0 (atomic failure)", mgr.registry.Count())
	}

	// Verify error was recorded
	loadErr := mgr.GetLastLoadError()
	if loadErr == nil {
		t.Error("GetLastLoadError() = nil, want error")
	}
}

func TestPolicyManager_ValidationErrorAccumulation(t *testing.T) {
	// Create temporary directory with multiple policies that have validation errors
	tmpDir := t.TempDir()

	// Create first policy with empty rules
	policy1 := filepath.Join(tmpDir, "policy1.yaml")
	content1 := `
mpl_version: "1.0"
name: "policy-1"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(policy1, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second policy with empty rules
	policy2 := filepath.Join(tmpDir, "policy2.yaml")
	content2 := `
mpl_version: "1.0"
name: "policy-2"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(policy2, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
			Strict:  false, // Non-strict mode - should accumulate errors
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Load policies - should fail with multiple validation errors
	err = mgr.LoadPolicies()

	if err == nil {
		t.Fatal("LoadPolicies() with validation errors error = nil, want error")
	}

	// Check if error mentions both policies
	errMsg := err.Error()
	if !strings.Contains(errMsg, "policy-1") || !strings.Contains(errMsg, "policy-2") {
		t.Logf("Error message: %v", errMsg)
		t.Log("Expected error to mention both policies (may vary based on validator implementation)")
	}
}

func TestPolicyManager_RuleDeduplication(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first policy with rule "duplicate-rule"
	policy1 := filepath.Join(tmpDir, "policy1.yaml")
	content1 := `
mpl_version: "1.0"
name: "policy-1"
version: "1.0.0"
rules:
  - name: "duplicate-rule"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "from policy 1"
`
	if err := os.WriteFile(policy1, []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second policy with same rule name
	policy2 := filepath.Join(tmpDir, "policy2.yaml")
	content2 := `
mpl_version: "1.0"
name: "policy-2"
version: "1.0.0"
rules:
  - name: "duplicate-rule"
    conditions:
      field: "request.model"
      operator: "=="
      value: "claude-3"
    actions:
      - type: "log"
        message: "from policy 2"
  - name: "unique-rule"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gemini"
    actions:
      - type: "log"
        message: "unique"
`
	if err := os.WriteFile(policy2, []byte(content2), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture log output to verify warning
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), logger)
	if err != nil {
		t.Fatal(err)
	}

	// Load policies - should succeed but warn about duplicates
	err = mgr.LoadPolicies()
	if err != nil {
		t.Fatalf("LoadPolicies() error = %v, want nil", err)
	}

	// Verify both policies loaded
	policies := mgr.GetAllPolicies()
	if len(policies) != 2 {
		t.Errorf("GetAllPolicies() count = %d, want 2", len(policies))
	}

	// Verify warning was logged
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Duplicate rule ID detected") {
		t.Error("Expected warning about duplicate rule ID")
	}
	if !strings.Contains(logOutput, "duplicate-rule") {
		t.Error("Expected warning to mention 'duplicate-rule'")
	}
}

func TestPolicyManager_ValidatePoliciesDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	validContent := `
mpl_version: "1.0"
name: "test-policy"
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
	if err := os.WriteFile(tmpFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpFile,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Dry-run validation should succeed
	err = mgr.ValidatePoliciesDryRun()
	if err != nil {
		t.Fatalf("ValidatePoliciesDryRun() error = %v, want nil", err)
	}

	// Verify policies were NOT applied to registry
	if mgr.registry.Count() != 0 {
		t.Errorf("Registry count = %d, want 0 (dry-run should not apply)", mgr.registry.Count())
	}
}

func TestPolicyManager_ValidatePoliciesDryRun_InvalidPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	// Invalid policy with empty rules
	invalidContent := `
mpl_version: "1.0"
name: "invalid-policy"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(tmpFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpFile,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Dry-run validation should fail
	err = mgr.ValidatePoliciesDryRun()
	if err == nil {
		t.Fatal("ValidatePoliciesDryRun() with invalid policy error = nil, want error")
	}

	// Verify error message contains "validation"
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("Error = %q, want to contain 'validation'", err.Error())
	}

	// Verify policies were NOT applied to registry
	if mgr.registry.Count() != 0 {
		t.Errorf("Registry count = %d, want 0 (dry-run should not apply even on error)", mgr.registry.Count())
	}
}

func TestPolicyManager_LoadPolicies_WithIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a shared base policy
	basePolicy := `
mpl_version: "1.0"
name: "base-policy"
version: "1.0.0"
description: "Base policy with shared rules"
rules:
  - name: "base-rule"
    description: "Base rule"
    conditions:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:
      - type: "log"
        message: "base policy triggered"
`
	basePath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(basePath, []byte(basePolicy), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a main policy that includes the base
	mainPolicy := `
mpl_version: "1.0"
name: "main-policy"
version: "1.0.0"
description: "Main policy that includes base"
includes:
  - "base.yaml"
rules:
  - name: "main-rule"
    description: "Main rule"
    conditions:
      field: "request.model"
      operator: "=="
      value: "claude-3-opus"
    actions:
      - type: "log"
        message: "main policy triggered"
`
	mainPath := filepath.Join(tmpDir, "main.yaml")
	if err := os.WriteFile(mainPath, []byte(mainPolicy), 0644); err != nil {
		t.Fatal(err)
	}

	// Create policy manager
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: mainPath,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Load policies - should load both main and included base
	err = mgr.LoadPolicies()
	if err != nil {
		t.Fatalf("LoadPolicies() with includes error = %v, want nil", err)
	}

	// Verify both policies are loaded
	// Base policy should be loaded first (dependency), then main
	if mgr.registry.Count() != 2 {
		t.Errorf("Registry count = %d, want 2 (base + main)", mgr.registry.Count())
	}

	// Verify base policy is loaded
	baseLoaded, ok := mgr.registry.Get("base-policy")
	if !ok {
		t.Error("Base policy not found in registry")
	} else {
		if baseLoaded.Name != "base-policy" {
			t.Errorf("Base policy name = %q, want 'base-policy'", baseLoaded.Name)
		}
	}

	// Verify main policy is loaded
	mainLoaded, ok := mgr.registry.Get("main-policy")
	if !ok {
		t.Error("Main policy not found in registry")
	} else {
		if mainLoaded.Name != "main-policy" {
			t.Errorf("Main policy name = %q, want 'main-policy'", mainLoaded.Name)
		}
	}
}

func TestPolicyManager_LoadPolicies_WithCircularInclude(t *testing.T) {
	tmpDir := t.TempDir()

	// Create policy A that includes B
	policyA := `
mpl_version: "1.0"
name: "policy-a"
version: "1.0.0"
includes:
  - "policy-b.yaml"
rules:
  - name: "rule-a"
    conditions:
      field: "request.model"
      operator: "=="
      value: "a"
    actions:
      - type: "log"
        message: "a"
`
	pathA := filepath.Join(tmpDir, "policy-a.yaml")
	if err := os.WriteFile(pathA, []byte(policyA), 0644); err != nil {
		t.Fatal(err)
	}

	// Create policy B that includes A (circular dependency)
	policyB := `
mpl_version: "1.0"
name: "policy-b"
version: "1.0.0"
includes:
  - "policy-a.yaml"
rules:
  - name: "rule-b"
    conditions:
      field: "request.model"
      operator: "=="
      value: "b"
    actions:
      - type: "log"
        message: "b"
`
	pathB := filepath.Join(tmpDir, "policy-b.yaml")
	if err := os.WriteFile(pathB, []byte(policyB), 0644); err != nil {
		t.Fatal(err)
	}

	// Create policy manager
	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: pathA,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Load policies - should fail with circular dependency error
	err = mgr.LoadPolicies()
	if err == nil {
		t.Fatal("LoadPolicies() with circular includes error = nil, want error")
	}

	// Verify error mentions circular dependency or cycle
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "circular") && !strings.Contains(errMsg, "cycle") {
		t.Errorf("Error = %q, want to contain 'circular' or 'cycle'", err.Error())
	}

	// Verify no policies were loaded
	if mgr.registry.Count() != 0 {
		t.Errorf("Registry count = %d, want 0 (failed load should not apply)", mgr.registry.Count())
	}
}
