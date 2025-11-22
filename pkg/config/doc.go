// Package config provides configuration management for Mercator Jupiter.
//
// This package handles loading, validating, and managing configuration from
// YAML files with environment variable overrides. It provides a type-safe
// configuration system with comprehensive validation and sensible defaults.
//
// # Configuration Loading
//
// Configuration can be loaded in two ways:
//
//  1. From a YAML file only:
//     cfg, err := config.LoadConfig("config.yaml")
//
//  2. From a YAML file with environment variable overrides:
//     cfg, err := config.LoadConfigWithEnvOverrides("config.yaml")
//
// # Environment Variable Overrides
//
// Environment variables follow the naming convention MERCATOR_SECTION_FIELD.
// For example:
//
//   - MERCATOR_PROXY_LISTEN_ADDRESS overrides proxy.listen_address
//   - MERCATOR_PROVIDERS_OPENAI_API_KEY overrides providers.openai.api_key
//   - MERCATOR_TELEMETRY_LOGGING_LEVEL overrides telemetry.logging.level
//
// Environment variables always take precedence over file-based configuration.
//
// # Configuration Precedence
//
// Configuration values are applied in the following order (later overrides earlier):
//
//  1. Default values (defined in defaults.go)
//  2. Values from YAML file
//  3. Environment variable overrides
//  4. Validation (fails fast if invalid)
//
// # Singleton Pattern
//
// For application-wide configuration access, use the singleton pattern:
//
//	// At application startup
//	if err := config.Initialize("config.yaml"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Anywhere in the application
//	cfg := config.GetConfig()
//	fmt.Println(cfg.Proxy.ListenAddress)
//
// For testing, prefer dependency injection with explicit Config instances
// rather than the global singleton.
//
// # Validation
//
// All configuration is validated automatically during loading. Validation includes:
//
//   - Required field checks (e.g., provider API keys, base URLs)
//   - Range validation (e.g., ports must be 1-65535)
//   - Format validation (e.g., valid URL format)
//   - Logical validation (e.g., mTLS requires TLS to be enabled)
//
// Validation errors include field paths and helpful messages:
//
//	configuration validation failed with 2 errors:
//	  - providers.openai.api_key: field is required
//	  - security.mtls.enabled: mTLS requires TLS to be enabled
//
// # Example Configuration
//
// Here is a minimal configuration file:
//
//	proxy:
//	  listen_address: "127.0.0.1:8080"
//
//	providers:
//	  openai:
//	    base_url: "https://api.openai.com/v1"
//	    api_key: "${OPENAI_API_KEY}"
//
//	policy:
//	  mode: "file"
//	  file_path: "./policies.yaml"
//
//	evidence:
//	  enabled: true
//	  backend: "sqlite"
//
//	telemetry:
//	  logging:
//	    level: "info"
//	    format: "json"
//
// # Thread Safety
//
// All configuration access is thread-safe. The singleton pattern uses read-write
// locks to allow concurrent reads while protecting against concurrent writes during
// reload operations.
package config
