package secrets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManager_GetSecret_FromEnv(t *testing.T) {
	// Set up environment variable
	os.Setenv("MERCATOR_SECRET_TEST_KEY", "env-value")
	defer os.Unsetenv("MERCATOR_SECRET_TEST_KEY")

	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	manager := NewManager(
		[]SecretProvider{envProvider},
		CacheConfig{Enabled: false},
	)

	value, err := manager.GetSecret(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "env-value" {
		t.Errorf("expected value 'env-value', got '%s'", value)
	}
}

func TestManager_GetSecret_FromFile(t *testing.T) {
	// Create temporary directory with secret
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "file-secret")
	if err := os.WriteFile(secretPath, []byte("file-value"), 0600); err != nil {
		t.Fatal(err)
	}

	fileProvider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer fileProvider.Close()

	manager := NewManager(
		[]SecretProvider{fileProvider},
		CacheConfig{Enabled: false},
	)

	value, err := manager.GetSecret(context.Background(), "file-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "file-value" {
		t.Errorf("expected value 'file-value', got '%s'", value)
	}
}

func TestManager_GetSecret_ProviderPriority(t *testing.T) {
	// Set up environment variable
	os.Setenv("MERCATOR_SECRET_TEST_KEY", "env-value")
	defer os.Unsetenv("MERCATOR_SECRET_TEST_KEY")

	// Create file with different value
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "test-key")
	if err := os.WriteFile(secretPath, []byte("file-value"), 0600); err != nil {
		t.Fatal(err)
	}

	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	fileProvider, _ := NewFileProvider(tmpDir, false)
	defer fileProvider.Close()

	// Env provider is first, should take priority
	manager := NewManager(
		[]SecretProvider{envProvider, fileProvider},
		CacheConfig{Enabled: false},
	)

	value, err := manager.GetSecret(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "env-value" {
		t.Errorf("expected value from first provider 'env-value', got '%s'", value)
	}

	// Reverse order - file provider first
	manager2 := NewManager(
		[]SecretProvider{fileProvider, envProvider},
		CacheConfig{Enabled: false},
	)

	value2, err := manager2.GetSecret(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value2 != "file-value" {
		t.Errorf("expected value from first provider 'file-value', got '%s'", value2)
	}
}

func TestManager_GetSecret_NotFound(t *testing.T) {
	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	manager := NewManager(
		[]SecretProvider{envProvider},
		CacheConfig{Enabled: false},
	)

	_, err := manager.GetSecret(context.Background(), "nonexistent-key")
	if err == nil {
		t.Error("expected error for nonexistent secret, got nil")
	}
}

func TestManager_GetSecret_Caching(t *testing.T) {
	// Set up environment variable
	os.Setenv("MERCATOR_SECRET_CACHED_KEY", "original-value")
	defer os.Unsetenv("MERCATOR_SECRET_CACHED_KEY")

	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	manager := NewManager(
		[]SecretProvider{envProvider},
		CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			MaxSize: 10,
		},
	)

	// First call - should fetch from provider
	value1, err := manager.GetSecret(context.Background(), "cached-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Change environment variable
	os.Setenv("MERCATOR_SECRET_CACHED_KEY", "new-value")

	// Second call - should return cached value
	value2, err := manager.GetSecret(context.Background(), "cached-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value2 != value1 {
		t.Error("expected cached value to be returned")
	}

	if value2 != "original-value" {
		t.Errorf("expected cached value 'original-value', got '%s'", value2)
	}
}

func TestManager_ResolveReferences(t *testing.T) {
	// Set up test secrets
	os.Setenv("MERCATOR_SECRET_API_KEY", "sk-abc123")
	os.Setenv("MERCATOR_SECRET_USERNAME", "admin")
	defer func() {
		os.Unsetenv("MERCATOR_SECRET_API_KEY")
		os.Unsetenv("MERCATOR_SECRET_USERNAME")
	}()

	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	manager := NewManager(
		[]SecretProvider{envProvider},
		CacheConfig{Enabled: false},
	)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single reference",
			input:    "api_key: ${secret:api-key}",
			expected: "api_key: sk-abc123",
		},
		{
			name:     "multiple references",
			input:    "username: ${secret:username}, api_key: ${secret:api-key}",
			expected: "username: admin, api_key: sk-abc123",
		},
		{
			name:     "no references",
			input:    "plain text without secrets",
			expected: "plain text without secrets",
		},
		{
			name:     "reference in YAML",
			input:    "providers:\n  openai:\n    api_key: ${secret:api-key}",
			expected: "providers:\n  openai:\n    api_key: sk-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := manager.ResolveReferences(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output != tt.expected {
				t.Errorf("expected output '%s', got '%s'", tt.expected, output)
			}
		})
	}
}

func TestManager_ResolveReferences_NotFound(t *testing.T) {
	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	manager := NewManager(
		[]SecretProvider{envProvider},
		CacheConfig{Enabled: false},
	)

	input := "api_key: ${secret:nonexistent-key}"
	_, err := manager.ResolveReferences(context.Background(), input)
	if err == nil {
		t.Error("expected error for nonexistent secret reference, got nil")
	}

	if !strings.Contains(err.Error(), "failed to resolve secret") {
		t.Errorf("expected 'failed to resolve secret' error, got: %v", err)
	}
}

func TestManager_Refresh(t *testing.T) {
	// Create file provider with a secret
	tmpDir := t.TempDir()
	secretPath := filepath.Join(tmpDir, "refresh-test")
	if err := os.WriteFile(secretPath, []byte("value1"), 0600); err != nil {
		t.Fatal(err)
	}

	fileProvider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer fileProvider.Close()

	manager := NewManager(
		[]SecretProvider{fileProvider},
		CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			MaxSize: 10,
		},
	)

	// Get secret (populates cache)
	value1, err := manager.GetSecret(context.Background(), "refresh-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update file
	if err := os.WriteFile(secretPath, []byte("value2"), 0600); err != nil {
		t.Fatal(err)
	}

	// Get secret again (should return cached value)
	value2, err := manager.GetSecret(context.Background(), "refresh-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value2 != value1 {
		t.Error("expected cached value before refresh")
	}

	// Refresh
	if err := manager.Refresh(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get secret after refresh (should return new value)
	value3, err := manager.GetSecret(context.Background(), "refresh-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value3 != "value2" {
		t.Errorf("expected new value 'value2' after refresh, got '%s'", value3)
	}
}

func TestManager_ListSecrets(t *testing.T) {
	// Set up environment variables
	os.Setenv("MERCATOR_SECRET_ENV_SECRET", "value1")
	defer os.Unsetenv("MERCATOR_SECRET_ENV_SECRET")

	// Create file secrets
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file-secret"), []byte("value2"), 0600); err != nil {
		t.Fatal(err)
	}

	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	fileProvider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer fileProvider.Close()

	manager := NewManager(
		[]SecretProvider{envProvider, fileProvider},
		CacheConfig{Enabled: false},
	)

	secrets, err := manager.ListSecrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include secrets from both providers
	secretMap := make(map[string]bool)
	for _, secret := range secrets {
		secretMap[secret] = true
	}

	if !secretMap["env-secret"] {
		t.Error("expected 'env-secret' in list")
	}

	if !secretMap["file-secret"] {
		t.Error("expected 'file-secret' in list")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	// Set up test secrets
	os.Setenv("MERCATOR_SECRET_CONCURRENT", "value")
	defer os.Unsetenv("MERCATOR_SECRET_CONCURRENT")

	envProvider := NewEnvProvider("MERCATOR_SECRET_")
	manager := NewManager(
		[]SecretProvider{envProvider},
		CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			MaxSize: 100,
		},
	)

	// Run concurrent GetSecret calls
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, err := manager.GetSecret(context.Background(), "concurrent")
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestManager_RedactSecretName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long name",
			input:    "openai-api-key",
			expected: "op...ey",
		},
		{
			name:     "short name",
			input:    "key",
			expected: "***",
		},
		{
			name:     "minimum length",
			input:    "abcd",
			expected: "***",
		},
		{
			name:     "exactly 5 chars",
			input:    "abcde",
			expected: "ab...de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSecretName(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
