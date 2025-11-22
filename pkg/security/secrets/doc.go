/*
Package secrets provides a pluggable framework for loading secrets from multiple sources.

# Overview

The secrets package allows Mercator to securely load credentials (API keys, certificates,
passwords) from various backends including environment variables, files, AWS KMS, GCP KMS,
and HashiCorp Vault. Secrets are cached in memory with TTL to reduce backend calls.

# Secret Providers

The package supports multiple secret providers that can be chained together with priority-based
fallback. Each provider implements the SecretProvider interface:

  - Environment Variable Provider: Load secrets from environment variables
  - File-Based Provider: Load secrets from individual files (Kubernetes-style)
  - AWS KMS Provider: Decrypt secrets using AWS KMS (Phase 2)
  - GCP KMS Provider: Decrypt secrets using GCP KMS (Phase 2)
  - HashiCorp Vault Provider: Load secrets from Vault (Phase 2)

# Basic Usage

Create a secret manager with multiple providers:

	import (
		"context"
		"time"
		"mercator-hq/jupiter/pkg/security/secrets"
	)

	// Create providers
	envProvider := secrets.NewEnvProvider("MERCATOR_SECRET_")
	fileProvider, _ := secrets.NewFileProvider("/var/secrets", true)

	// Create manager with cache config
	cacheConfig := secrets.CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 1000,
	}

	manager := secrets.NewManager(
		[]secrets.SecretProvider{envProvider, fileProvider},
		cacheConfig,
	)

	// Get a secret
	apiKey, err := manager.GetSecret(context.Background(), "openai-api-key")
	if err != nil {
		log.Fatal(err)
	}

# Secret References

The manager can resolve secret references in configuration strings using the ${secret:name} syntax:

	configValue := "api_key: ${secret:openai-api-key}"
	resolved, err := manager.ResolveReferences(context.Background(), configValue)
	// resolved = "api_key: sk-abc123..."

# Environment Variable Provider

The environment variable provider loads secrets from environment variables with an optional prefix:

	provider := secrets.NewEnvProvider("MERCATOR_SECRET_")

	// Secret name "openai-api-key" maps to env var "MERCATOR_SECRET_OPENAI_API_KEY"
	value, err := provider.GetSecret(ctx, "openai-api-key")

Environment variable naming:
  - Secret name: "openai-api-key"
  - Env var name: "MERCATOR_SECRET_OPENAI_API_KEY"
  - Conversion: uppercase, replace hyphens with underscores, add prefix

# File-Based Provider

The file-based provider loads secrets from individual files in a directory:

	provider, err := secrets.NewFileProvider("/var/secrets", true)
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Close()

	// Secret name "openai-api-key" reads from "/var/secrets/openai-api-key"
	value, err := provider.GetSecret(ctx, "openai-api-key")

File-based features:
  - File permissions validation (0600 or 0400 only)
  - Optional file watching for auto-reload
  - Kubernetes-style secret mounting support
  - Automatic cache invalidation on file changes

# Secret Caching

Secrets are cached in memory to reduce backend calls:

	cacheConfig := secrets.CacheConfig{
		Enabled: true,        // Enable caching
		TTL:     5 * time.Minute,  // Cache for 5 minutes
		MaxSize: 1000,        // Maximum 1000 secrets
	}

Cache features:
  - LRU eviction when MaxSize is reached
  - TTL-based expiration
  - Automatic invalidation on provider refresh
  - Thread-safe access

# Provider Priority

When multiple providers are configured, they are tried in order:

	manager := secrets.NewManager(
		[]secrets.SecretProvider{
			envProvider,    // Try environment variables first
			fileProvider,   // Then try files
			kmsProvider,    // Finally try KMS
		},
		cacheConfig,
	)

The first provider that supports the secret and successfully returns a value wins.

# Secret Rotation

Providers that implement RefreshableProvider can reload secrets without restart:

	// Refresh all providers and clear cache
	err := manager.Refresh(context.Background())
	if err != nil {
		log.Error("failed to refresh secrets", "error", err)
	}

File-based providers automatically refresh when files change if watching is enabled.

# Security Considerations

Secret values are protected:
  - Never logged (secret names are redacted in logs)
  - Never included in error messages
  - File permissions validated (0600 or 0400 only)
  - Cached with TTL to minimize exposure window
  - Cleared from cache on refresh

# Configuration Example

YAML configuration for secret management:

	security:
	  secrets:
	    providers:
	      # Environment variables (always enabled)
	      - type: "env"
	        prefix: "MERCATOR_SECRET_"

	      # File-based secrets (Kubernetes-style)
	      - type: "file"
	        path: "/var/secrets"
	        watch: true

	      # AWS KMS (Phase 2)
	      - type: "aws_kms"
	        enabled: false
	        region: "us-west-2"
	        key_id: "arn:aws:kms:..."

	    cache:
	      enabled: true
	      ttl: "5m"
	      max_size: 1000

# Error Handling

Errors are returned for:
  - Secret not found in any provider
  - File permission errors (too permissive)
  - Provider-specific errors (network, authentication, etc.)

Example error handling:

	value, err := manager.GetSecret(ctx, "my-secret")
	if err != nil {
		log.Error("failed to get secret",
			"name", "my-secret",
			"error", err,
		)
		return err
	}

# Thread Safety

All components are thread-safe:
  - Cache uses sync.RWMutex for concurrent access
  - Manager supports concurrent GetSecret calls
  - Providers implement their own synchronization as needed

# Best Practices

1. Use environment variables for development
2. Use file-based secrets for Kubernetes
3. Use KMS/Vault for production (Phase 2)
4. Enable caching to reduce backend load
5. Set appropriate TTL based on rotation frequency
6. Use file watching for zero-downtime rotation
7. Never commit secrets to version control
8. Validate file permissions on startup

# Phase 2 Features

The following features are planned for Phase 2:

  - AWS KMS integration with IAM authentication
  - GCP KMS integration with service accounts
  - HashiCorp Vault integration with token auth
  - Certificate revocation list (CRL) support
  - OCSP stapling for certificate validation
  - Secret versioning and rotation tracking
  - Audit logging for secret access

# Example: Complete Setup

	package main

	import (
		"context"
		"log"
		"time"

		"mercator-hq/jupiter/pkg/security/secrets"
	)

	func main() {
		// Create providers
		envProvider := secrets.NewEnvProvider("MERCATOR_SECRET_")
		fileProvider, err := secrets.NewFileProvider("/var/secrets", true)
		if err != nil {
			log.Fatal(err)
		}
		defer fileProvider.Close()

		// Create manager
		manager := secrets.NewManager(
			[]secrets.SecretProvider{envProvider, fileProvider},
			secrets.CacheConfig{
				Enabled: true,
				TTL:     5 * time.Minute,
				MaxSize: 1000,
			},
		)

		// Get secrets
		ctx := context.Background()

		openaiKey, err := manager.GetSecret(ctx, "openai-api-key")
		if err != nil {
			log.Fatal(err)
		}

		anthropicKey, err := manager.GetSecret(ctx, "anthropic-api-key")
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Loaded %d secrets", 2)

		// Resolve references in config
		configValue := `
		providers:
		  openai:
		    api_key: ${secret:openai-api-key}
		  anthropic:
		    api_key: ${secret:anthropic-api-key}
		`

		resolved, err := manager.ResolveReferences(ctx, configValue)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Resolved config:\n%s", resolved)
	}
*/
package secrets
