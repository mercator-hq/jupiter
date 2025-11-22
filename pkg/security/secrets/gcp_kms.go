package secrets

import (
	"context"
	"fmt"
)

// GCPKMSProvider provides secrets using Google Cloud Key Management Service.
//
// This is a stub implementation for Phase 2. GCP KMS integration will
// support decrypting secrets stored in Google Cloud Storage or Secret Manager
// using KMS encryption keys.
//
// Phase 2 will add:
// - Service account authentication
// - Envelope encryption (data keys + encrypted secrets)
// - Regional endpoint support
// - Integration with GCP Secret Manager
type GCPKMSProvider struct {
	Enabled  bool   // Enable GCP KMS provider (false for MVP)
	Project  string // GCP project ID
	Location string // KMS location (e.g., "global", "us-east1")
	KeyRing  string // KMS key ring name
	Key      string // KMS key name
}

// NewGCPKMSProvider creates a new GCP KMS secret provider stub.
//
// This provider is disabled by default and will return errors when used.
// Phase 2 will implement full GCP KMS integration.
func NewGCPKMSProvider(project, location, keyRing, key string, enabled bool) *GCPKMSProvider {
	return &GCPKMSProvider{
		Enabled:  enabled,
		Project:  project,
		Location: location,
		KeyRing:  keyRing,
		Key:      key,
	}
}

// GetSecret retrieves a secret from GCP KMS.
//
// This is a stub implementation that returns an error.
// Phase 2 will implement KMS decryption.
func (p *GCPKMSProvider) GetSecret(ctx context.Context, name string) (string, error) {
	if !p.Enabled {
		return "", fmt.Errorf("GCP KMS provider not enabled (Phase 2 feature)")
	}

	// Phase 2: Implement KMS decryption
	// - Fetch encrypted secret from GCS/Secret Manager
	// - Decrypt using KMS key
	// - Return plaintext value
	return "", fmt.Errorf("GCP KMS provider not implemented (Phase 2)")
}

// ListSecrets returns all secrets available from GCP Secret Manager.
//
// This is a stub implementation that returns an error.
// Phase 2 will implement listing secrets from GCP Secret Manager.
func (p *GCPKMSProvider) ListSecrets(ctx context.Context) ([]string, error) {
	if !p.Enabled {
		return nil, fmt.Errorf("GCP KMS provider not enabled (Phase 2 feature)")
	}

	// Phase 2: Implement listing secrets
	// - List secrets from GCP Secret Manager
	return nil, fmt.Errorf("GCP KMS provider not implemented (Phase 2)")
}

// Provider returns the provider name.
func (p *GCPKMSProvider) Provider() string {
	return "gcp_kms"
}

// Supports indicates if this provider supports the given secret name.
//
// This stub implementation always returns false.
// Phase 2 will implement secret name matching based on GCP resource naming.
func (p *GCPKMSProvider) Supports(name string) bool {
	// Not supported in MVP
	return false
}
