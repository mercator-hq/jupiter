package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// EnvProvider loads secrets from environment variables.
//
// Secret names are converted to uppercase environment variable names
// with hyphens replaced by underscores. An optional prefix can be
// configured to namespace secrets.
//
// Example:
//   - Secret name: "openai-api-key"
//   - Env var name: "MERCATOR_SECRET_OPENAI_API_KEY" (with prefix "MERCATOR_SECRET_")
type EnvProvider struct {
	Prefix string // Optional prefix for environment variables
}

// NewEnvProvider creates a new environment variable secret provider.
//
// The prefix is prepended to all environment variable names.
// For example, with prefix "MERCATOR_SECRET_", the secret "openai-api-key"
// will be read from the environment variable "MERCATOR_SECRET_OPENAI_API_KEY".
func NewEnvProvider(prefix string) *EnvProvider {
	return &EnvProvider{
		Prefix: prefix,
	}
}

// GetSecret retrieves a secret from an environment variable.
//
// The secret name is converted to an environment variable name by:
// 1. Converting to uppercase
// 2. Replacing hyphens with underscores
// 3. Prepending the configured prefix
//
// For example: "openai-api-key" -> "MERCATOR_SECRET_OPENAI_API_KEY"
func (p *EnvProvider) GetSecret(ctx context.Context, name string) (string, error) {
	envVar := p.secretNameToEnvVar(name)

	value := os.Getenv(envVar)
	if value == "" {
		return "", fmt.Errorf("secret not found in environment: %s (env var: %s)", name, envVar)
	}

	return value, nil
}

// ListSecrets returns all secret names from environment variables with the configured prefix.
//
// This scans all environment variables and returns those that match the prefix.
// The returned names are converted back to secret format (lowercase with hyphens).
func (p *EnvProvider) ListSecrets(ctx context.Context) ([]string, error) {
	var secrets []string

	for _, env := range os.Environ() {
		// Check if env var starts with prefix
		if !strings.HasPrefix(env, p.Prefix) {
			continue
		}

		// Split into name and value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// Convert env var name back to secret name
		envVarName := parts[0]
		secretName := p.envVarToSecretName(envVarName)

		secrets = append(secrets, secretName)
	}

	return secrets, nil
}

// Provider returns the provider name.
func (p *EnvProvider) Provider() string {
	return "env"
}

// Supports indicates if this provider supports the given secret name.
//
// The environment provider always returns true because any secret
// can potentially be provided via an environment variable.
func (p *EnvProvider) Supports(name string) bool {
	// Environment provider always attempts to resolve
	// This allows it to be used as a fallback
	return true
}

// secretNameToEnvVar converts a secret name to an environment variable name.
//
// Example: "openai-api-key" -> "MERCATOR_SECRET_OPENAI_API_KEY"
func (p *EnvProvider) secretNameToEnvVar(name string) string {
	// Convert to uppercase and replace hyphens with underscores
	envVar := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

	// Prepend prefix
	return p.Prefix + envVar
}

// envVarToSecretName converts an environment variable name back to a secret name.
//
// Example: "MERCATOR_SECRET_OPENAI_API_KEY" -> "openai-api-key"
func (p *EnvProvider) envVarToSecretName(envVar string) string {
	// Remove prefix
	name := strings.TrimPrefix(envVar, p.Prefix)

	// Convert to lowercase and replace underscores with hyphens
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}
