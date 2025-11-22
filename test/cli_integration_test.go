//go:build integration

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestServerStartStop tests the server start and graceful shutdown
func TestServerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create test config
	configFile := filepath.Join(tmpDir, "config.yaml")
	createTestConfig(t, configFile, `
proxy:
  listen_address: "127.0.0.1:18080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
    timeout: 30s

policy:
  mode: "file"
  file_path: "test-policy.yaml"

evidence:
  enabled: false

telemetry:
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: false
  tracing:
    enabled: false
`)

	// Create minimal policy file
	policyFile := filepath.Join(tmpDir, "test-policy.yaml")
	createTestPolicy(t, policyFile)

	// Build mercator binary if not exists
	binaryPath := buildMercatorBinary(t)

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "run", "--config", configFile)
	cmd.Dir = tmpDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait for server to be ready
	if !waitForHealthy("http://127.0.0.1:18080/health", 10*time.Second) {
		t.Fatalf("server failed to start\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	// Verify health endpoint
	resp, err := http.Get("http://127.0.0.1:18080/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Test graceful shutdown
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Errorf("failed to send SIGINT: %v", err)
	}

	// Wait for shutdown
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Expected - server should shut down cleanly
		// Exit code 130 is SIGINT (Ctrl+C)
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 130 {
				t.Logf("shutdown output - Stdout: %s\nStderr: %s", stdout.String(), stderr.String())
				t.Errorf("unexpected shutdown error: %v (exit code: %d)", err, exitErr.ExitCode())
			}
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not shut down within 5 seconds")
	}
}

// TestPolicyValidationPipeline tests the policy linting and testing workflow
func TestPolicyValidationPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create test policy
	policyFile := filepath.Join(tmpDir, "test.yaml")
	createTestPolicy(t, policyFile)

	// Create test cases
	testFile := filepath.Join(tmpDir, "tests.yaml")
	createTestCases(t, testFile)

	binaryPath := buildMercatorBinary(t)

	// Step 1: Lint policy
	t.Log("Step 1: Linting policy...")
	lintCmd := exec.Command(binaryPath, "lint", "--file", policyFile)
	output, err := lintCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lint failed: %v\nOutput: %s", err, output)
	}

	// Verify lint output contains success message
	if !bytes.Contains(output, []byte("valid")) {
		t.Errorf("expected 'valid' in lint output, got: %s", output)
	}

	// Step 2: Run tests
	t.Log("Step 2: Running policy tests...")
	testCmd := exec.Command(binaryPath, "test",
		"--policy", policyFile,
		"--tests", testFile)

	output, err = testCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test failed: %v\nOutput: %s", err, output)
	}

	// Verify output contains success
	if !bytes.Contains(output, []byte("passed")) {
		t.Errorf("expected 'passed' in output, got: %s", output)
	}

	// Step 3: Test JSON output
	t.Log("Step 3: Testing JSON output...")
	testJSONCmd := exec.Command(binaryPath, "test",
		"--policy", policyFile,
		"--tests", testFile,
		"--format", "json")

	jsonOutput, err := testJSONCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test with JSON output failed: %v\nOutput: %s", err, jsonOutput)
	}

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal(jsonOutput, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, jsonOutput)
	}

	// Verify JSON structure
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("JSON output missing 'summary' field")
	}

	if summary["passed"] == nil || summary["total"] == nil {
		t.Fatalf("JSON summary missing required fields: %+v", summary)
	}
}

// TestEvidenceQueryPipeline tests evidence generation and querying
func TestEvidenceQueryPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "evidence.db")

	// Create config with evidence enabled
	configFile := filepath.Join(tmpDir, "config.yaml")
	createTestConfig(t, configFile, fmt.Sprintf(`
proxy:
  listen_address: "127.0.0.1:18081"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
    timeout: 30s

policy:
  mode: "file"
  file_path: "test-policy.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "%s"

telemetry:
  logging:
    level: "warn"
    format: "json"
  metrics:
    enabled: false
  tracing:
    enabled: false
`, dbPath))

	// Create minimal policy
	policyFile := filepath.Join(tmpDir, "test-policy.yaml")
	createTestPolicy(t, policyFile)

	binaryPath := buildMercatorBinary(t)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "run", "--config", configFile)
	cmd.Dir = tmpDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer cmd.Process.Kill()

	if !waitForHealthy("http://127.0.0.1:18081/health", 10*time.Second) {
		t.Fatalf("server failed to start\nStdout: %s\nStderr: %s", stdout.String(), stderr.String())
	}

	// Send test request to generate evidence
	t.Log("Sending test request to generate evidence...")
	sendTestRequest(t, "http://127.0.0.1:18081")

	// Wait for evidence to be written
	time.Sleep(1 * time.Second)

	// Query evidence
	t.Log("Querying evidence records...")
	queryCmd := exec.Command(binaryPath, "evidence", "query",
		"--config", configFile,
		"--limit", "10",
		"--format", "json")

	output, err := queryCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("evidence query failed: %v\nOutput: %s", err, output)
	}

	// Parse JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify we got records
	records, ok := result["records"].([]interface{})
	if !ok {
		t.Fatalf("JSON output missing 'records' field: %+v", result)
	}

	if len(records) == 0 {
		t.Error("expected evidence records, got none")
	}

	t.Logf("Successfully queried %d evidence records", len(records))
}

// TestKeyGenerationAndUsage tests cryptographic key generation
func TestKeyGenerationAndUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	keysDir := filepath.Join(tmpDir, "keys")

	binaryPath := buildMercatorBinary(t)

	// Generate keypair
	t.Log("Generating keypair...")
	cmd := exec.Command(binaryPath, "keys", "generate",
		"--key-id", "test-key",
		"--output", keysDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("key generation failed: %v\nOutput: %s", err, output)
	}

	t.Logf("Key generation output: %s", output)

	// Verify files exist
	publicKey := filepath.Join(keysDir, "test-key_public.pem")
	privateKey := filepath.Join(keysDir, "test-key_private.pem")

	if _, err := os.Stat(publicKey); err != nil {
		t.Errorf("public key not created: %v", err)
	}
	if _, err := os.Stat(privateKey); err != nil {
		t.Errorf("private key not created: %v", err)
	}

	// Verify private key permissions (should be 0600)
	info, err := os.Stat(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	expectedPerm := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("private key permissions = %o, want %o", info.Mode().Perm(), expectedPerm)
	}

	// Verify public key is readable
	pubKeyData, err := os.ReadFile(publicKey)
	if err != nil {
		t.Fatalf("failed to read public key: %v", err)
	}

	if len(pubKeyData) == 0 {
		t.Error("public key file is empty")
	}

	// Verify key files have PEM header
	if !bytes.Contains(pubKeyData, []byte("BEGIN PUBLIC KEY")) {
		t.Error("public key missing PEM header")
	}

	privKeyData, err := os.ReadFile(privateKey)
	if err != nil {
		t.Fatalf("failed to read private key: %v", err)
	}

	if !bytes.Contains(privKeyData, []byte("BEGIN PRIVATE KEY")) {
		t.Error("private key missing PEM header")
	}

	t.Log("Key generation successful, files verified")
}

// TestCommandVersionOutput tests the version command
func TestCommandVersionOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	binaryPath := buildMercatorBinary(t)

	// Test version command
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v\nOutput: %s", err, output)
	}

	// Verify output contains version info
	outputStr := string(output)
	if !bytes.Contains(output, []byte("Mercator")) {
		t.Errorf("version output should contain 'Mercator', got: %s", outputStr)
	}
}

// TestDryRunValidation tests config validation with --dry-run
func TestDryRunValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()

	// Test with valid config
	t.Run("valid config", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "valid-config.yaml")
		createTestConfig(t, configFile, `
proxy:
  listen_address: "127.0.0.1:18082"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"

policy:
  mode: "file"
  file_path: "policy.yaml"

evidence:
  enabled: false
`)

		policyFile := filepath.Join(tmpDir, "policy.yaml")
		createTestPolicy(t, policyFile)

		binaryPath := buildMercatorBinary(t)
		cmd := exec.Command(binaryPath, "run", "--config", configFile, "--dry-run")
		cmd.Dir = tmpDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("dry-run should succeed with valid config: %v\nOutput: %s", err, output)
		}
	})

	// Test with invalid config (missing required fields)
	t.Run("invalid config", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "invalid-config.yaml")
		createTestConfig(t, configFile, `
proxy:
  listen_address: "127.0.0.1:18083"
# Missing providers section - should fail validation
`)

		binaryPath := buildMercatorBinary(t)
		cmd := exec.Command(binaryPath, "run", "--config", configFile, "--dry-run")

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("dry-run should fail with invalid config\nOutput: %s", output)
		}
	})
}

// Helper functions

// buildMercatorBinary builds the mercator binary for testing
func buildMercatorBinary(t *testing.T) string {
	t.Helper()

	// Check if binary already exists in bin/
	binaryPath := "../bin/mercator"
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Build the binary
	t.Log("Building mercator binary...")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/mercator")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build mercator: %v\nOutput: %s", err, output)
	}

	return binaryPath
}

// waitForHealthy waits for a health endpoint to return 200
func waitForHealthy(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// createTestConfig creates a test configuration file
func createTestConfig(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
}

// createTestPolicy creates a minimal test policy file
func createTestPolicy(t *testing.T, path string) {
	t.Helper()

	policy := `version: "1.0"
policies:
  - name: "test-policy"
    description: "Test policy for integration tests"
    rules:
      - condition: "true"
        action: "allow"
`

	if err := os.WriteFile(path, []byte(policy), 0644); err != nil {
		t.Fatalf("failed to create policy file: %v", err)
	}
}

// createTestCases creates a test cases file for policy testing
func createTestCases(t *testing.T, path string) {
	t.Helper()

	tests := `tests:
  - name: "Allow all requests"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Hello"
    metadata:
      user_id: "test-user"
    expect:
      action: "allow"
`

	if err := os.WriteFile(path, []byte(tests), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
}

// sendTestRequest sends a test request to the proxy to generate evidence
func sendTestRequest(t *testing.T, baseURL string) {
	t.Helper()

	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	url := baseURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-key")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// It's okay if the request fails (provider not reachable)
		// We just need to generate an evidence record
		t.Logf("test request failed (expected): %v", err)
		return
	}
	defer resp.Body.Close()

	t.Logf("test request completed with status: %d", resp.StatusCode)
}
