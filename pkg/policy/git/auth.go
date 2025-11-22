package git

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"mercator-hq/jupiter/pkg/config"
)

// AuthProvider handles Git authentication.
type AuthProvider interface {
	// GetAuth returns git transport authentication method.
	GetAuth() (transport.AuthMethod, error)

	// Type returns auth type for logging purposes.
	Type() string
}

// TokenAuth implements token-based HTTPS authentication.
// Supports GitHub Personal Access Tokens, GitLab tokens, and Bitbucket tokens.
type TokenAuth struct {
	token string
}

// NewTokenAuth creates a new token-based authentication provider.
// The token parameter should be a valid personal access token or OAuth token
// with repository read permissions.
func NewTokenAuth(token string) *TokenAuth {
	return &TokenAuth{token: token}
}

// GetAuth returns HTTP basic auth with the token as password.
// The username can be anything for token authentication.
func (a *TokenAuth) GetAuth() (transport.AuthMethod, error) {
	if a.token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	return &http.BasicAuth{
		Username: "git", // Can be anything for token auth
		Password: a.token,
	}, nil
}

// Type returns the authentication type.
func (a *TokenAuth) Type() string {
	return "token"
}

// SSHAuth implements SSH key-based authentication.
// Supports public key authentication with optional passphrase.
type SSHAuth struct {
	keyPath    string
	passphrase string
}

// NewSSHAuth creates a new SSH key-based authentication provider.
// The keyPath parameter should be the path to a private SSH key file.
// The passphrase parameter is optional and should be empty if the key is not encrypted.
func NewSSHAuth(keyPath, passphrase string) *SSHAuth {
	return &SSHAuth{
		keyPath:    keyPath,
		passphrase: passphrase,
	}
}

// GetAuth returns SSH public key authentication method.
// The SSH key file must exist and be readable.
func (a *SSHAuth) GetAuth() (transport.AuthMethod, error) {
	if a.keyPath == "" {
		return nil, fmt.Errorf("ssh key path cannot be empty")
	}

	// Check if key file exists
	if _, err := os.Stat(a.keyPath); err != nil {
		return nil, fmt.Errorf("failed to access SSH key file: %w", err)
	}

	// Check file permissions (should be 0600 or more restrictive)
	info, err := os.Stat(a.keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat SSH key file: %w", err)
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		return nil, fmt.Errorf("SSH key file permissions too open (%o), should be 0600", mode)
	}

	// Load SSH key
	auth, err := ssh.NewPublicKeysFromFile("git", a.keyPath, a.passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key: %w", err)
	}

	return auth, nil
}

// Type returns the authentication type.
func (a *SSHAuth) Type() string {
	return "ssh"
}

// NoAuth implements authentication for public repositories.
// This provider returns nil authentication, allowing access to public repositories.
type NoAuth struct{}

// NewNoAuth creates a new no-authentication provider for public repositories.
func NewNoAuth() *NoAuth {
	return &NoAuth{}
}

// GetAuth returns nil authentication for public repositories.
func (a *NoAuth) GetAuth() (transport.AuthMethod, error) {
	return nil, nil
}

// Type returns the authentication type.
func (a *NoAuth) Type() string {
	return "none"
}

// NewAuthProvider creates an appropriate auth provider from configuration.
// Supported types: "token", "ssh", "none".
// Returns an error if the type is unknown or required fields are missing.
func NewAuthProvider(cfg *config.GitAuthConfig) (AuthProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("auth config cannot be nil")
	}

	switch cfg.Type {
	case "token":
		if cfg.Token == "" {
			return nil, fmt.Errorf("token auth requires non-empty token")
		}
		return NewTokenAuth(cfg.Token), nil

	case "ssh":
		if cfg.SSHKeyPath == "" {
			return nil, fmt.Errorf("ssh auth requires ssh_key_path")
		}
		return NewSSHAuth(cfg.SSHKeyPath, cfg.SSHKeyPassphrase), nil

	case "none", "":
		return NewNoAuth(), nil

	default:
		return nil, fmt.Errorf("unknown auth type: %s", cfg.Type)
	}
}
