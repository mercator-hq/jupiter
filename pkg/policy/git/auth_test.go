package git

import (
	"os"
	"path/filepath"
	"testing"

	"mercator-hq/jupiter/pkg/config"
)

// TestTokenAuth_GetAuth tests token authentication.
func TestTokenAuth_GetAuth(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "ghp_validtoken123",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewTokenAuth(tt.token)

			if auth.Type() != "token" {
				t.Errorf("Type() = %v, want %v", auth.Type(), "token")
			}

			_, err := auth.GetAuth()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTokenAuth_Type tests token auth type.
func TestTokenAuth_Type(t *testing.T) {
	auth := NewTokenAuth("test-token")
	if auth.Type() != "token" {
		t.Errorf("Type() = %v, want %v", auth.Type(), "token")
	}
}

// TestSSHAuth_GetAuth tests SSH key authentication.
func TestSSHAuth_GetAuth(t *testing.T) {
	// Create temporary directory for test keys
	tmpDir := t.TempDir()

	// Create a dummy SSH key file with correct permissions
	validKeyPath := filepath.Join(tmpDir, "valid_key")
	if err := os.WriteFile(validKeyPath, []byte("dummy key content"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create a key file with wrong permissions
	wrongPermsPath := filepath.Join(tmpDir, "wrong_perms_key")
	if err := os.WriteFile(wrongPermsPath, []byte("dummy key content"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		keyPath    string
		passphrase string
		wantErr    bool
	}{
		{
			name:       "empty key path",
			keyPath:    "",
			passphrase: "",
			wantErr:    true,
		},
		{
			name:       "non-existent key file",
			keyPath:    "/nonexistent/key",
			passphrase: "",
			wantErr:    true,
		},
		{
			name:       "wrong permissions",
			keyPath:    wrongPermsPath,
			passphrase: "",
			wantErr:    true,
		},
		{
			name:       "valid key path but invalid key format",
			keyPath:    validKeyPath,
			passphrase: "",
			wantErr:    true, // Will fail because it's not a real SSH key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewSSHAuth(tt.keyPath, tt.passphrase)

			if auth.Type() != "ssh" {
				t.Errorf("Type() = %v, want %v", auth.Type(), "ssh")
			}

			_, err := auth.GetAuth()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSSHAuth_Type tests SSH auth type.
func TestSSHAuth_Type(t *testing.T) {
	auth := NewSSHAuth("/path/to/key", "")
	if auth.Type() != "ssh" {
		t.Errorf("Type() = %v, want %v", auth.Type(), "ssh")
	}
}

// TestSSHAuth_FilePermissions tests SSH key file permission checking.
func TestSSHAuth_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		permissions os.FileMode
		wantErr     bool
	}{
		{
			name:        "correct permissions 0600",
			permissions: 0600,
			wantErr:     true, // Still error because not a real key
		},
		{
			name:        "correct permissions 0400",
			permissions: 0400,
			wantErr:     true, // Still error because not a real key
		},
		{
			name:        "too open 0644",
			permissions: 0644,
			wantErr:     true,
		},
		{
			name:        "too open 0666",
			permissions: 0666,
			wantErr:     true,
		},
		{
			name:        "too open 0777",
			permissions: 0777,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := filepath.Join(tmpDir, "test_key_"+tt.name)
			if err := os.WriteFile(keyPath, []byte("dummy key"), tt.permissions); err != nil {
				t.Fatal(err)
			}

			auth := NewSSHAuth(keyPath, "")
			_, err := auth.GetAuth()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNoAuth_GetAuth tests no authentication.
func TestNoAuth_GetAuth(t *testing.T) {
	auth := NewNoAuth()

	if auth.Type() != "none" {
		t.Errorf("Type() = %v, want %v", auth.Type(), "none")
	}

	method, err := auth.GetAuth()
	if err != nil {
		t.Errorf("GetAuth() error = %v, want nil", err)
	}
	if method != nil {
		t.Errorf("GetAuth() = %v, want nil", method)
	}
}

// TestNoAuth_Type tests no auth type.
func TestNoAuth_Type(t *testing.T) {
	auth := NewNoAuth()
	if auth.Type() != "none" {
		t.Errorf("Type() = %v, want %v", auth.Type(), "none")
	}
}

// TestNewAuthProvider tests auth provider factory.
func TestNewAuthProvider(t *testing.T) {
	tmpDir := t.TempDir()
	validKeyPath := filepath.Join(tmpDir, "valid_key")
	if err := os.WriteFile(validKeyPath, []byte("dummy key"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		cfg      *config.GitAuthConfig
		wantType string
		wantErr  bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			wantType: "",
			wantErr:  true,
		},
		{
			name: "token auth valid",
			cfg: &config.GitAuthConfig{
				Type:  "token",
				Token: "ghp_validtoken",
			},
			wantType: "token",
			wantErr:  false,
		},
		{
			name: "token auth missing token",
			cfg: &config.GitAuthConfig{
				Type:  "token",
				Token: "",
			},
			wantType: "",
			wantErr:  true,
		},
		{
			name: "ssh auth valid",
			cfg: &config.GitAuthConfig{
				Type:       "ssh",
				SSHKeyPath: validKeyPath,
			},
			wantType: "ssh",
			wantErr:  false,
		},
		{
			name: "ssh auth missing key path",
			cfg: &config.GitAuthConfig{
				Type:       "ssh",
				SSHKeyPath: "",
			},
			wantType: "",
			wantErr:  true,
		},
		{
			name: "no auth explicit",
			cfg: &config.GitAuthConfig{
				Type: "none",
			},
			wantType: "none",
			wantErr:  false,
		},
		{
			name:     "no auth implicit",
			cfg:      &config.GitAuthConfig{},
			wantType: "none",
			wantErr:  false,
		},
		{
			name: "unknown auth type",
			cfg: &config.GitAuthConfig{
				Type: "unknown",
			},
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAuthProvider(tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuthProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if provider.Type() != tt.wantType {
					t.Errorf("NewAuthProvider().Type() = %v, want %v", provider.Type(), tt.wantType)
				}
			}
		})
	}
}

// TestAuthProvider_Interface tests that all auth types implement AuthProvider.
func TestAuthProvider_Interface(t *testing.T) {
	var _ AuthProvider = (*TokenAuth)(nil)
	var _ AuthProvider = (*SSHAuth)(nil)
	var _ AuthProvider = (*NoAuth)(nil)
}

// TestTokenAuth_WithEmptyPassword tests token auth with empty password specifically.
func TestTokenAuth_WithEmptyPassword(t *testing.T) {
	auth := &TokenAuth{token: ""}
	_, err := auth.GetAuth()
	if err == nil {
		t.Error("GetAuth() with empty token should return error")
	}
}

// TestSSHAuth_WithPassphrase tests SSH auth with passphrase.
func TestSSHAuth_WithPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "encrypted_key")
	if err := os.WriteFile(keyPath, []byte("encrypted key content"), 0600); err != nil {
		t.Fatal(err)
	}

	auth := NewSSHAuth(keyPath, "my-passphrase")
	if auth.passphrase != "my-passphrase" {
		t.Errorf("passphrase = %v, want %v", auth.passphrase, "my-passphrase")
	}
}

// TestAuthProvider_ErrorMessages tests that error messages are descriptive.
func TestAuthProvider_ErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.GitAuthConfig
		wantErrText string
	}{
		{
			name: "token missing",
			cfg: &config.GitAuthConfig{
				Type:  "token",
				Token: "",
			},
			wantErrText: "token auth requires non-empty token",
		},
		{
			name: "ssh key path missing",
			cfg: &config.GitAuthConfig{
				Type:       "ssh",
				SSHKeyPath: "",
			},
			wantErrText: "ssh auth requires ssh_key_path",
		},
		{
			name: "unknown type",
			cfg: &config.GitAuthConfig{
				Type: "invalid",
			},
			wantErrText: "unknown auth type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAuthProvider(tt.cfg)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			// Check if error message contains expected text (allowing for additional context)
			if len(err.Error()) < len(tt.wantErrText) || err.Error()[:len(tt.wantErrText)] != tt.wantErrText {
				t.Errorf("error message = %v, want to start with %v", err.Error(), tt.wantErrText)
			}
		})
	}
}

// TestAuthProvider_MultipleInstances tests creating multiple auth providers.
func TestAuthProvider_MultipleInstances(t *testing.T) {
	auth1 := NewTokenAuth("token1")
	auth2 := NewTokenAuth("token2")
	auth3 := NewNoAuth()

	if auth1.token == auth2.token {
		t.Error("auth1 and auth2 should have different tokens")
	}

	if auth1.Type() != auth2.Type() {
		t.Error("auth1 and auth2 should have same type")
	}

	if auth1.Type() == auth3.Type() {
		t.Error("auth1 and auth3 should have different types")
	}
}
