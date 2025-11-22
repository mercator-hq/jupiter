package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
)

// BenchmarkPolicyManager_LoadPolicies_SingleFile benchmarks loading a single policy file
func BenchmarkPolicyManager_LoadPolicies_SingleFile(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "benchmark-policy"
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
	if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: policyFile,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
		if err != nil {
			b.Fatal(err)
		}

		if err := mgr.LoadPolicies(); err != nil {
			b.Fatal(err)
		}

		mgr.Close()
	}
}

// BenchmarkPolicyManager_LoadPolicies_MultiFile benchmarks loading multiple policy files
func BenchmarkPolicyManager_LoadPolicies_MultiFile(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()

	// Create 10 policy files
	for i := 0; i < 10; i++ {
		policyFile := filepath.Join(tmpDir, "policy"+string(rune('0'+i))+".yaml")
		content := `
mpl_version: "1.0"
name: "benchmark-policy-` + string(rune('0'+i)) + `"
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
		if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	cfg := &config.PolicyConfig{
		Mode:     "file",
		FilePath: tmpDir,
		Validation: config.PolicyValidationConfig{
			Enabled: true,
		},
	}

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mgr, err := NewPolicyManager(cfg, parser.NewParser(), validator.NewValidator(), nil)
		if err != nil {
			b.Fatal(err)
		}

		if err := mgr.LoadPolicies(); err != nil {
			b.Fatal(err)
		}

		mgr.Close()
	}
}

// BenchmarkPolicyManager_ReloadPolicies benchmarks policy reloading
func BenchmarkPolicyManager_ReloadPolicies(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "benchmark-policy"
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
	if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
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
		b.Fatal(err)
	}
	defer mgr.Close()

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		b.Fatal(err)
	}

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := mgr.ReloadPolicies(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPolicyManager_GetPolicy benchmarks retrieving a single policy
func BenchmarkPolicyManager_GetPolicy(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "benchmark-policy"
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
	if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
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
		b.Fatal(err)
	}
	defer mgr.Close()

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		b.Fatal(err)
	}

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mgr.GetPolicy("benchmark-policy")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPolicyManager_GetAllPolicies benchmarks retrieving all policies
func BenchmarkPolicyManager_GetAllPolicies(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()

	// Create 10 policy files
	for i := 0; i < 10; i++ {
		policyFile := filepath.Join(tmpDir, "policy"+string(rune('0'+i))+".yaml")
		content := `
mpl_version: "1.0"
name: "benchmark-policy-` + string(rune('0'+i)) + `"
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
		if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
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
		b.Fatal(err)
	}
	defer mgr.Close()

	// Initial load
	if err := mgr.LoadPolicies(); err != nil {
		b.Fatal(err)
	}

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = mgr.GetAllPolicies()
	}
}

// BenchmarkPolicyRegistry_Get benchmarks registry get operations
func BenchmarkPolicyRegistry_Get(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "benchmark-policy"
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
	if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())
	policy, err := loader.LoadFromFile(policyFile)
	if err != nil {
		b.Fatal(err)
	}

	registry := NewPolicyRegistry()
	if err := registry.Register(policy); err != nil {
		b.Fatal(err)
	}

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = registry.Get("benchmark-policy")
	}
}

// BenchmarkPolicyLoader_LoadFromFile benchmarks file loading
func BenchmarkPolicyLoader_LoadFromFile(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "benchmark-policy"
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
	if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := loader.LoadFromFile(policyFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPolicyManager_LoadPoliciesForEngine benchmarks engine integration
func BenchmarkPolicyManager_LoadPoliciesForEngine(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "benchmark-policy"
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
	if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
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
		b.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := mgr.LoadPoliciesForEngine(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
