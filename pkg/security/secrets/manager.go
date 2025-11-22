package secrets

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

var (
	// secretRefRegex matches ${secret:name} patterns in configuration
	secretRefRegex = regexp.MustCompile(`\$\{secret:([^}]+)\}`)
)

// Manager orchestrates multiple secret providers with priority-based fallback.
//
// The manager tries each provider in order until one successfully returns
// a value. Secrets are cached to reduce backend calls.
type Manager struct {
	providers []SecretProvider
	cache     *Cache
}

// NewManager creates a new secret manager with the given providers and cache config.
//
// Providers are tried in the order they are provided. The first provider
// that supports a secret and successfully returns a value wins.
func NewManager(providers []SecretProvider, cacheConfig CacheConfig) *Manager {
	return &Manager{
		providers: providers,
		cache:     NewCache(cacheConfig),
	}
}

// GetSecret retrieves a secret from the first provider that supports it.
//
// The manager checks the cache first, then tries each provider in order.
// If a provider successfully returns a value, it is cached and returned.
//
// Returns an error if no provider supports the secret or all providers fail.
func (m *Manager) GetSecret(ctx context.Context, name string) (string, error) {
	// Check cache first
	if value, ok := m.cache.Get(name); ok {
		slog.Debug("secret cache hit", "name", redactSecretName(name))
		return value, nil
	}

	slog.Debug("secret cache miss", "name", redactSecretName(name))

	// Try each provider
	var lastErr error
	for _, provider := range m.providers {
		if !provider.Supports(name) {
			continue
		}

		slog.Debug("trying secret provider",
			"provider", provider.Provider(),
			"name", redactSecretName(name),
		)

		value, err := provider.GetSecret(ctx, name)
		if err != nil {
			lastErr = err
			slog.Debug("provider failed to get secret",
				"provider", provider.Provider(),
				"name", redactSecretName(name),
				"error", err,
			)
			continue
		}

		// Cache the value
		m.cache.Set(name, value)

		slog.Debug("secret retrieved",
			"provider", provider.Provider(),
			"name", redactSecretName(name),
		)

		return value, nil
	}

	if lastErr != nil {
		return "", fmt.Errorf("failed to get secret %q: %w", name, lastErr)
	}

	return "", fmt.Errorf("secret not found: %q (no provider supports this secret)", name)
}

// ResolveReferences replaces ${secret:name} patterns with actual secret values.
//
// This is used to resolve secret references in configuration files.
// For example: "api_key: ${secret:openai-api-key}" becomes "api_key: sk-abc123"
//
// If a secret cannot be retrieved, the original reference is kept in the output.
func (m *Manager) ResolveReferences(ctx context.Context, input string) (string, error) {
	var errors []string

	output := secretRefRegex.ReplaceAllStringFunc(input, func(match string) string {
		// Extract secret name
		matches := secretRefRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			errors = append(errors, fmt.Sprintf("invalid secret reference: %s", match))
			return match
		}

		name := matches[1]
		value, err := m.GetSecret(ctx, name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to resolve secret %q: %v", name, err))
			return match // Keep original reference on error
		}

		return value
	})

	if len(errors) > 0 {
		return output, fmt.Errorf("failed to resolve secret references: %s", strings.Join(errors, "; "))
	}

	return output, nil
}

// Refresh reloads all refreshable providers and clears the cache.
//
// This is typically called when secrets need to be rotated or when
// file-based providers detect changes.
func (m *Manager) Refresh(ctx context.Context) error {
	slog.Info("refreshing secrets from all providers")

	var errors []string
	for _, provider := range m.providers {
		refreshable, ok := provider.(RefreshableProvider)
		if !ok {
			continue
		}

		slog.Debug("refreshing provider", "provider", provider.Provider())

		if err := refreshable.Refresh(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", provider.Provider(), err))
			slog.Error("failed to refresh provider",
				"provider", provider.Provider(),
				"error", err,
			)
		}
	}

	// Clear cache to force re-fetch
	m.cache.Clear()
	slog.Debug("secret cache cleared")

	if len(errors) > 0 {
		return fmt.Errorf("failed to refresh some providers: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ListSecrets returns all secret names from all providers.
//
// This is useful for debugging and administration. Secret values are
// never included for security reasons.
func (m *Manager) ListSecrets(ctx context.Context) ([]string, error) {
	secretMap := make(map[string]bool)

	for _, provider := range m.providers {
		secrets, err := provider.ListSecrets(ctx)
		if err != nil {
			slog.Warn("failed to list secrets from provider",
				"provider", provider.Provider(),
				"error", err,
			)
			continue
		}

		for _, secret := range secrets {
			secretMap[secret] = true
		}
	}

	// Convert map to slice
	secrets := make([]string, 0, len(secretMap))
	for secret := range secretMap {
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// redactSecretName returns a redacted version of the secret name for logging.
//
// This prevents leaking sensitive information in logs while still being
// useful for debugging.
func redactSecretName(name string) string {
	if len(name) <= 4 {
		return "***"
	}
	// Show first 2 and last 2 characters
	return name[:2] + "..." + name[len(name)-2:]
}
