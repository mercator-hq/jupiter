// Package secrets provides a pluggable framework for loading secrets from multiple sources.
package secrets

import "context"

// SecretProvider retrieves secrets from a backend.
//
// Implementations include environment variables, files, AWS KMS, GCP KMS,
// and HashiCorp Vault. Providers can be chained together with priority-based
// fallback.
type SecretProvider interface {
	// GetSecret retrieves a secret by name.
	// Returns an error if the secret is not found or cannot be retrieved.
	GetSecret(ctx context.Context, name string) (string, error)

	// ListSecrets returns all secret names available from this provider.
	// Values are not included for security reasons.
	ListSecrets(ctx context.Context) ([]string, error)

	// Provider returns the provider name (env, file, aws_kms, gcp_kms, vault).
	Provider() string

	// Supports indicates if this provider supports the given secret name.
	// This is used to determine which provider to use when multiple are configured.
	Supports(name string) bool
}

// RefreshableProvider can reload secrets without restart.
//
// This is implemented by providers that support dynamic secret rotation,
// such as file-based providers that watch for file changes.
type RefreshableProvider interface {
	SecretProvider

	// Refresh reloads all secrets from the backend.
	// This is typically called when files change or on a periodic basis.
	Refresh(ctx context.Context) error
}
