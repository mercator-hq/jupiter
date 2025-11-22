package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// BenchmarkVersionCommand benchmarks the version command startup time
// Target: < 100ms per iteration
func BenchmarkVersionCommand(b *testing.B) {
	// Build binary once
	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "version")
		if err := cmd.Run(); err != nil {
			b.Fatalf("version command failed: %v", err)
		}
	}
}

// BenchmarkVersionCommandShort benchmarks the version --short command
// Target: < 50ms per iteration
func BenchmarkVersionCommandShort(b *testing.B) {
	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "version", "--short")
		if err := cmd.Run(); err != nil {
			b.Fatalf("version --short command failed: %v", err)
		}
	}
}

// BenchmarkHelpCommand benchmarks the help command
// Target: < 100ms per iteration
func BenchmarkHelpCommand(b *testing.B) {
	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "--help")
		if err := cmd.Run(); err != nil {
			// Help command exits with code 0, so this should not fail
			b.Fatalf("help command failed: %v", err)
		}
	}
}

// BenchmarkLintCommand benchmarks the lint command with a valid policy
// Target: < 500ms per iteration
func BenchmarkLintCommand(b *testing.B) {
	// Setup test files
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "valid-policy.yaml")
	createBenchmarkPolicy(b, policyFile)

	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "lint", "--file", policyFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("lint command failed: %v", err)
		}
	}
}

// BenchmarkLintCommandJSON benchmarks lint with JSON output
// Target: < 600ms per iteration
func BenchmarkLintCommandJSON(b *testing.B) {
	tmpDir := b.TempDir()
	policyFile := filepath.Join(tmpDir, "valid-policy.yaml")
	createBenchmarkPolicy(b, policyFile)

	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "lint", "--file", policyFile, "--format", "json")
		if err := cmd.Run(); err != nil {
			b.Fatalf("lint command with JSON output failed: %v", err)
		}
	}
}

// BenchmarkRunDryRun benchmarks config validation with --dry-run
// Target: < 1s per iteration
func BenchmarkRunDryRun(b *testing.B) {
	tmpDir := b.TempDir()

	configFile := filepath.Join(tmpDir, "config.yaml")
	createBenchmarkConfig(b, configFile)

	policyFile := filepath.Join(tmpDir, "policy.yaml")
	createBenchmarkPolicy(b, policyFile)

	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "run", "--config", configFile, "--dry-run")
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			b.Fatalf("run --dry-run failed: %v", err)
		}
	}
}

// BenchmarkKeysGenerate benchmarks key generation
// Target: < 200ms per iteration
func BenchmarkKeysGenerate(b *testing.B) {
	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		cmd := exec.Command(binaryPath, "keys", "generate",
			"--key-id", "bench-key",
			"--output", tmpDir)

		if err := cmd.Run(); err != nil {
			b.Fatalf("keys generate failed: %v", err)
		}
	}
}

// BenchmarkLintDirectory benchmarks linting a directory with multiple policies
// Target: < 5s per iteration for 100 files
func BenchmarkLintDirectory(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 100 policy files
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		policyFile := filepath.Join(policyDir, "policy-"+string(rune(i))+".yaml")
		createBenchmarkPolicy(b, policyFile)
	}

	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "lint", "--dir", policyDir)
		if err := cmd.Run(); err != nil {
			b.Fatalf("lint directory failed: %v", err)
		}
	}
}

// BenchmarkTestCommand benchmarks running policy tests
// Target: < 1s per iteration for simple tests
func BenchmarkTestCommand(b *testing.B) {
	tmpDir := b.TempDir()

	policyFile := filepath.Join(tmpDir, "policy.yaml")
	createBenchmarkPolicy(b, policyFile)

	testFile := filepath.Join(tmpDir, "tests.yaml")
	createBenchmarkTests(b, testFile)

	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "test",
			"--policy", policyFile,
			"--tests", testFile)

		if err := cmd.Run(); err != nil {
			b.Fatalf("test command failed: %v", err)
		}
	}
}

// BenchmarkEvidenceQuery benchmarks evidence querying
// Note: This requires a pre-populated database
// Target: < 500ms per iteration for queries
func BenchmarkEvidenceQuery(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "evidence.db")

	// Create config with evidence enabled
	configFile := filepath.Join(tmpDir, "config.yaml")
	createBenchmarkConfigWithEvidence(b, configFile, dbPath)

	// Pre-populate database with test records
	populateEvidenceDB(b, dbPath, 1000)

	binaryPath := buildBinary(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath, "evidence", "query",
			"--config", configFile,
			"--limit", "100")

		if err := cmd.Run(); err != nil {
			b.Fatalf("evidence query failed: %v", err)
		}
	}
}

// BenchmarkCompletionGeneration benchmarks shell completion generation
// Target: < 100ms per iteration
func BenchmarkCompletionGeneration(b *testing.B) {
	binaryPath := buildBinary(b)

	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		b.Run(shell, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cmd := exec.Command(binaryPath, "completion", shell)
				if err := cmd.Run(); err != nil {
					b.Fatalf("completion %s failed: %v", shell, err)
				}
			}
		})
	}
}

// Helper functions

var cachedBinaryPath string

// buildBinary builds the mercator binary once and caches the path
func buildBinary(b *testing.B) string {
	b.Helper()

	if cachedBinaryPath != "" {
		return cachedBinaryPath
	}

	// Check if binary exists in ../../bin/
	binaryPath := "../../bin/mercator"
	if _, err := os.Stat(binaryPath); err == nil {
		cachedBinaryPath = binaryPath
		return binaryPath
	}

	// Build new binary
	tmpBinary := filepath.Join(b.TempDir(), "mercator")
	cmd := exec.Command("go", "build", "-o", tmpBinary, ".")
	if err := cmd.Run(); err != nil {
		b.Fatalf("failed to build mercator: %v", err)
	}

	cachedBinaryPath = tmpBinary
	return tmpBinary
}

// createBenchmarkPolicy creates a standard policy file for benchmarking
func createBenchmarkPolicy(b *testing.B, path string) {
	b.Helper()

	policy := `version: "1.0"
policies:
  - name: "benchmark-policy"
    description: "Policy for benchmarking"
    rules:
      - condition: "true"
        action: "allow"
`

	if err := os.WriteFile(path, []byte(policy), 0644); err != nil {
		b.Fatalf("failed to create policy file: %v", err)
	}
}

// createBenchmarkConfig creates a standard config file for benchmarking
func createBenchmarkConfig(b *testing.B, path string) {
	b.Helper()

	config := `proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
    timeout: 30s

policy:
  mode: "file"
  file_path: "policy.yaml"

evidence:
  enabled: false

telemetry:
  logging:
    level: "info"
  metrics:
    enabled: false
  tracing:
    enabled: false
`

	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		b.Fatalf("failed to create config file: %v", err)
	}
}

// createBenchmarkConfigWithEvidence creates config with evidence enabled
func createBenchmarkConfigWithEvidence(b *testing.B, path, dbPath string) {
	b.Helper()

	config := `proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"

policy:
  mode: "file"
  file_path: "policy.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "` + dbPath + `"

telemetry:
  logging:
    level: "warn"
  metrics:
    enabled: false
  tracing:
    enabled: false
`

	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		b.Fatalf("failed to create config file: %v", err)
	}
}

// createBenchmarkTests creates a test file for benchmarking
func createBenchmarkTests(b *testing.B, path string) {
	b.Helper()

	tests := `tests:
  - name: "Test allow"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Hello"
    metadata:
      user_id: "bench-user"
    expect:
      action: "allow"
`

	if err := os.WriteFile(path, []byte(tests), 0644); err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}
}

// populateEvidenceDB populates the evidence database with test records
func populateEvidenceDB(b *testing.B, dbPath string, count int) {
	b.Helper()

	// This is a placeholder - actual implementation would use the evidence package
	// to create test records in the database
	// For now, we'll skip this and the benchmark will work with an empty DB

	// TODO: Implement evidence DB population when evidence package is available
	// db, err := evidence.NewSQLiteStore(dbPath)
	// if err != nil {
	//     b.Fatalf("failed to create evidence store: %v", err)
	// }
	// defer db.Close()
	//
	// for i := 0; i < count; i++ {
	//     record := createTestRecord(i)
	//     if err := db.Store(record); err != nil {
	//         b.Fatalf("failed to store record: %v", err)
	//     }
	// }
}
