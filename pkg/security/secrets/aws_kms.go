package secrets

import (
	"context"
	"fmt"
)

// AWSKMSProvider provides secrets using AWS Key Management Service.
//
// This is a stub implementation for Phase 2. AWS KMS integration will
// support decrypting secrets stored in S3 or other AWS services using
// KMS encryption keys.
//
// Phase 2 will add:
// - IAM role-based authentication
// - Envelope encryption (data keys + encrypted secrets)
// - Regional endpoint support
// - Automatic retry with exponential backoff
type AWSKMSProvider struct {
	Enabled bool   // Enable AWS KMS provider (false for MVP)
	Region  string // AWS region
	KeyID   string // KMS key ID or ARN
}

// NewAWSKMSProvider creates a new AWS KMS secret provider stub.
//
// This provider is disabled by default and will return errors when used.
// Phase 2 will implement full AWS KMS integration.
func NewAWSKMSProvider(region, keyID string, enabled bool) *AWSKMSProvider {
	return &AWSKMSProvider{
		Enabled: enabled,
		Region:  region,
		KeyID:   keyID,
	}
}

// GetSecret retrieves a secret from AWS KMS.
//
// This is a stub implementation that returns an error.
// Phase 2 will implement KMS decryption.
func (p *AWSKMSProvider) GetSecret(ctx context.Context, name string) (string, error) {
	if !p.Enabled {
		return "", fmt.Errorf("AWS KMS provider not enabled (Phase 2 feature)")
	}

	// Phase 2: Implement KMS decryption
	// - Fetch encrypted secret from S3/SSM/Secrets Manager
	// - Decrypt using KMS key
	// - Return plaintext value
	return "", fmt.Errorf("AWS KMS provider not implemented (Phase 2)")
}

// ListSecrets returns all secrets available from AWS KMS.
//
// This is a stub implementation that returns an error.
// Phase 2 will implement listing secrets from AWS Secrets Manager or SSM Parameter Store.
func (p *AWSKMSProvider) ListSecrets(ctx context.Context) ([]string, error) {
	if !p.Enabled {
		return nil, fmt.Errorf("AWS KMS provider not enabled (Phase 2 feature)")
	}

	// Phase 2: Implement listing secrets
	// - List secrets from AWS Secrets Manager
	// - Or list parameters from SSM Parameter Store
	return nil, fmt.Errorf("AWS KMS provider not implemented (Phase 2)")
}

// Provider returns the provider name.
func (p *AWSKMSProvider) Provider() string {
	return "aws_kms"
}

// Supports indicates if this provider supports the given secret name.
//
// This stub implementation always returns false.
// Phase 2 will implement secret name matching based on AWS resource naming.
func (p *AWSKMSProvider) Supports(name string) bool {
	// Not supported in MVP
	return false
}
