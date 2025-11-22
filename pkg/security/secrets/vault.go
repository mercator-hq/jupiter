package secrets

import (
	"context"
	"fmt"
)

// VaultProvider provides secrets using HashiCorp Vault.
//
// This is a stub implementation for Phase 2. Vault integration will
// support retrieving secrets from Vault's key-value store with support
// for dynamic secrets and secret leasing.
//
// Phase 2 will add:
// - Token-based authentication
// - AppRole authentication
// - Kubernetes authentication
// - Secret leasing and renewal
// - Dynamic secret generation
type VaultProvider struct {
	Enabled bool   // Enable Vault provider (false for MVP)
	Address string // Vault server address (e.g., "https://vault.example.com:8200")
	Token   string // Vault authentication token
	Path    string // Secret path prefix (e.g., "secret/mercator")
}

// NewVaultProvider creates a new HashiCorp Vault secret provider stub.
//
// This provider is disabled by default and will return errors when used.
// Phase 2 will implement full Vault integration.
func NewVaultProvider(address, token, path string, enabled bool) *VaultProvider {
	return &VaultProvider{
		Enabled: enabled,
		Address: address,
		Token:   token,
		Path:    path,
	}
}

// GetSecret retrieves a secret from HashiCorp Vault.
//
// This is a stub implementation that returns an error.
// Phase 2 will implement Vault API integration.
func (p *VaultProvider) GetSecret(ctx context.Context, name string) (string, error) {
	if !p.Enabled {
		return "", fmt.Errorf("Vault provider not enabled (Phase 2 feature)")
	}

	// Phase 2: Implement Vault API integration
	// - Authenticate with Vault
	// - Read secret from configured path
	// - Handle secret leasing
	// - Renew token if needed
	return "", fmt.Errorf("Vault provider not implemented (Phase 2)")
}

// ListSecrets returns all secrets available from HashiCorp Vault.
//
// This is a stub implementation that returns an error.
// Phase 2 will implement listing secrets from Vault.
func (p *VaultProvider) ListSecrets(ctx context.Context) ([]string, error) {
	if !p.Enabled {
		return nil, fmt.Errorf("Vault provider not enabled (Phase 2 feature)")
	}

	// Phase 2: Implement listing secrets
	// - List secrets at configured path
	return nil, fmt.Errorf("Vault provider not implemented (Phase 2)")
}

// Provider returns the provider name.
func (p *VaultProvider) Provider() string {
	return "vault"
}

// Supports indicates if this provider supports the given secret name.
//
// This stub implementation always returns false.
// Phase 2 will implement secret name matching based on Vault path structure.
func (p *VaultProvider) Supports(name string) bool {
	// Not supported in MVP
	return false
}
