package secrets

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileProvider_GetSecret(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create test secret file
	secretPath := filepath.Join(tmpDir, "test-secret")
	if err := os.WriteFile(secretPath, []byte("test-value\n"), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	value, err := provider.GetSecret(context.Background(), "test-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Value should have whitespace trimmed
	if value != "test-value" {
		t.Errorf("expected value 'test-value', got '%s'", value)
	}
}

func TestFileProvider_GetSecret_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	_, err = provider.GetSecret(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent secret, got nil")
	}
}

func TestFileProvider_InsecurePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with insecure permissions (0644)
	secretPath := filepath.Join(tmpDir, "insecure-secret")
	if err := os.WriteFile(secretPath, []byte("value"), 0644); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	_, err = provider.GetSecret(context.Background(), "insecure-secret")
	if err == nil {
		t.Error("expected error for insecure permissions, got nil")
	}
}

func TestFileProvider_SecurePermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions os.FileMode
		shouldWork  bool
	}{
		{"0600 permissions", 0600, true},
		{"0400 permissions", 0400, true},
		{"0644 permissions", 0644, false},
		{"0666 permissions", 0666, false},
		{"0700 permissions", 0700, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			secretPath := filepath.Join(tmpDir, "test-secret")
			if err := os.WriteFile(secretPath, []byte("value"), tt.permissions); err != nil {
				t.Fatal(err)
			}

			provider, err := NewFileProvider(tmpDir, false)
			if err != nil {
				t.Fatalf("failed to create provider: %v", err)
			}
			defer provider.Close()

			_, err = provider.GetSecret(context.Background(), "test-secret")

			if tt.shouldWork && err != nil {
				t.Errorf("expected success with permissions %o, got error: %v", tt.permissions, err)
			}

			if !tt.shouldWork && err == nil {
				t.Errorf("expected error with permissions %o, got success", tt.permissions)
			}
		})
	}
}

func TestFileProvider_ListSecrets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test secret files
	secrets := []string{"secret1", "secret2", "secret3"}
	for _, secret := range secrets {
		path := filepath.Join(tmpDir, secret)
		if err := os.WriteFile(path, []byte("value"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Create a subdirectory (should be ignored)
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	listed, err := provider.ListSecrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly 3 secrets (subdirectory should be excluded)
	if len(listed) != 3 {
		t.Errorf("expected 3 secrets, got %d", len(listed))
	}

	// Verify all expected secrets are present
	listedMap := make(map[string]bool)
	for _, secret := range listed {
		listedMap[secret] = true
	}

	for _, secret := range secrets {
		if !listedMap[secret] {
			t.Errorf("expected secret '%s' in list", secret)
		}
	}
}

func TestFileProvider_Caching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create secret file
	secretPath := filepath.Join(tmpDir, "cached-secret")
	if err := os.WriteFile(secretPath, []byte("value1"), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// First read (should cache)
	value1, err := provider.GetSecret(context.Background(), "cached-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify file
	if err := os.WriteFile(secretPath, []byte("value2"), 0600); err != nil {
		t.Fatal(err)
	}

	// Second read (should return cached value)
	value2, err := provider.GetSecret(context.Background(), "cached-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value2 != value1 {
		t.Error("expected cached value to be returned")
	}

	// Refresh cache
	if err := provider.Refresh(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Third read (should return new value)
	value3, err := provider.GetSecret(context.Background(), "cached-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value3 != "value2" {
		t.Errorf("expected refreshed value 'value2', got '%s'", value3)
	}
}

func TestFileProvider_Refresh(t *testing.T) {
	tmpDir := t.TempDir()

	// Create secret file
	secretPath := filepath.Join(tmpDir, "test-secret")
	if err := os.WriteFile(secretPath, []byte("value1"), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Read to populate cache
	_, err = provider.GetSecret(context.Background(), "test-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cache is populated
	provider.mu.RLock()
	cacheSize := len(provider.cache)
	provider.mu.RUnlock()

	if cacheSize != 1 {
		t.Errorf("expected cache size 1, got %d", cacheSize)
	}

	// Refresh
	if err := provider.Refresh(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cache is cleared
	provider.mu.RLock()
	cacheSize = len(provider.cache)
	provider.mu.RUnlock()

	if cacheSize != 0 {
		t.Errorf("expected cache size 0 after refresh, got %d", cacheSize)
	}
}

func TestFileProvider_WatchMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create secret file
	secretPath := filepath.Join(tmpDir, "watched-secret")
	if err := os.WriteFile(secretPath, []byte("value1"), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, true)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Read initial value
	value1, err := provider.GetSecret(context.Background(), "watched-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value1 != "value1" {
		t.Errorf("expected value 'value1', got '%s'", value1)
	}

	// Modify file
	if err := os.WriteFile(secretPath, []byte("value2"), 0600); err != nil {
		t.Fatal(err)
	}

	// Give watcher time to process the event
	time.Sleep(200 * time.Millisecond)

	// Read again (should get new value after cache refresh)
	value2, err := provider.GetSecret(context.Background(), "watched-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value2 != "value2" {
		t.Errorf("expected value 'value2' after file change, got '%s'", value2)
	}
}

func TestFileProvider_Provider(t *testing.T) {
	tmpDir := t.TempDir()

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	if provider.Provider() != "file" {
		t.Errorf("expected provider name 'file', got '%s'", provider.Provider())
	}
}

func TestFileProvider_Supports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create secret file
	secretPath := filepath.Join(tmpDir, "existing-secret")
	if err := os.WriteFile(secretPath, []byte("value"), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	defer provider.Close()

	// Should support existing file
	if !provider.Supports("existing-secret") {
		t.Error("expected Supports to return true for existing file")
	}

	// Should not support nonexistent file
	if provider.Supports("nonexistent-secret") {
		t.Error("expected Supports to return false for nonexistent file")
	}
}

func TestFileProvider_InvalidBasePath(t *testing.T) {
	// Try to create provider with nonexistent directory
	_, err := NewFileProvider("/nonexistent/directory", false)
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}

	// Try to create provider with file instead of directory
	tmpFile := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(tmpFile, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = NewFileProvider(tmpFile, false)
	if err == nil {
		t.Error("expected error for file instead of directory, got nil")
	}
}
