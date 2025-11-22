package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// FieldError represents a validation error for a specific configuration field.
type FieldError struct {
	// Field is the dotted path to the configuration field (e.g., "proxy.listen_address").
	Field string

	// Message is a human-readable error message.
	Message string
}

// Error returns the error message for this field error.
func (e FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationError represents one or more validation errors in a configuration.
// It implements the error interface and provides access to all field errors.
type ValidationError struct {
	// Errors contains all validation errors found in the configuration.
	Errors []FieldError
}

// Error returns a formatted string containing all validation errors.
func (e ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "configuration validation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("configuration validation failed: %s", e.Errors[0].Error())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("configuration validation failed with %d errors:\n", len(e.Errors)))
	for _, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// Validate validates the entire configuration and returns a ValidationError
// if any validation rules fail. It returns nil if the configuration is valid.
// All validation errors are collected and returned together.
func Validate(cfg *Config) error {
	var errs []FieldError

	// Validate proxy configuration
	errs = append(errs, validateProxy(&cfg.Proxy)...)

	// Validate providers configuration
	errs = append(errs, validateProviders(cfg.Providers)...)

	// Validate policy configuration
	errs = append(errs, validatePolicy(&cfg.Policy)...)

	// Validate evidence configuration
	errs = append(errs, validateEvidence(&cfg.Evidence)...)

	// Validate limits configuration
	errs = append(errs, validateLimits(&cfg.Limits)...)

	// Validate telemetry configuration
	errs = append(errs, validateTelemetry(&cfg.Telemetry)...)

	// Validate security configuration
	errs = append(errs, validateSecurity(&cfg.Security)...)

	if len(errs) > 0 {
		return ValidationError{Errors: errs}
	}

	return nil
}

// validateProxy validates proxy configuration.
func validateProxy(cfg *ProxyConfig) []FieldError {
	var errs []FieldError

	// Validate listen address is not empty
	if cfg.ListenAddress == "" {
		errs = append(errs, FieldError{
			Field:   "proxy.listen_address",
			Message: "listen address is required",
		})
	}

	// Validate timeouts are positive
	if cfg.ReadTimeout < 0 {
		errs = append(errs, FieldError{
			Field:   "proxy.read_timeout",
			Message: "read timeout must be positive",
		})
	}
	if cfg.WriteTimeout < 0 {
		errs = append(errs, FieldError{
			Field:   "proxy.write_timeout",
			Message: "write timeout must be positive",
		})
	}
	if cfg.IdleTimeout < 0 {
		errs = append(errs, FieldError{
			Field:   "proxy.idle_timeout",
			Message: "idle timeout must be positive",
		})
	}

	// Validate max header bytes is reasonable
	if cfg.MaxHeaderBytes < 0 {
		errs = append(errs, FieldError{
			Field:   "proxy.max_header_bytes",
			Message: "max header bytes must be non-negative",
		})
	}
	if cfg.MaxHeaderBytes > 10*1024*1024 { // 10MB is excessive
		errs = append(errs, FieldError{
			Field:   "proxy.max_header_bytes",
			Message: "max header bytes exceeds reasonable limit (10MB)",
		})
	}

	return errs
}

// validateProviders validates provider configurations.
func validateProviders(providers map[string]ProviderConfig) []FieldError {
	var errs []FieldError

	if len(providers) == 0 {
		errs = append(errs, FieldError{
			Field:   "providers",
			Message: "at least one provider must be configured",
		})
		return errs
	}

	for name, provider := range providers {
		prefix := fmt.Sprintf("providers.%s", name)

		// Validate base URL
		if provider.BaseURL == "" {
			errs = append(errs, FieldError{
				Field:   prefix + ".base_url",
				Message: "base URL is required",
			})
		} else {
			// Validate URL format
			if _, err := url.Parse(provider.BaseURL); err != nil {
				errs = append(errs, FieldError{
					Field:   prefix + ".base_url",
					Message: fmt.Sprintf("invalid URL format: %v", err),
				})
			}
		}

		// Validate API key is present (it can be empty if loaded from env var)
		// We'll allow empty API keys here and let runtime fail if needed
		// This allows for configurations where the key is injected later

		// Validate timeout
		if provider.Timeout < 0 {
			errs = append(errs, FieldError{
				Field:   prefix + ".timeout",
				Message: "timeout must be positive",
			})
		}

		// Validate max retries
		if provider.MaxRetries < 0 {
			errs = append(errs, FieldError{
				Field:   prefix + ".max_retries",
				Message: "max retries must be non-negative",
			})
		}
		if provider.MaxRetries > 10 {
			errs = append(errs, FieldError{
				Field:   prefix + ".max_retries",
				Message: "max retries exceeds reasonable limit (10)",
			})
		}
	}

	return errs
}

// validatePolicy validates policy configuration.
func validatePolicy(cfg *PolicyConfig) []FieldError {
	var errs []FieldError

	// Validate mode
	validModes := map[string]bool{"file": true, "git": true}
	if cfg.Mode == "" {
		errs = append(errs, FieldError{
			Field:   "policy.mode",
			Message: "mode is required",
		})
	} else if !validModes[cfg.Mode] {
		errs = append(errs, FieldError{
			Field:   "policy.mode",
			Message: fmt.Sprintf("invalid mode %q: must be 'file' or 'git'", cfg.Mode),
		})
	}

	// Validate file path when in file mode
	if cfg.Mode == "file" && cfg.FilePath == "" {
		errs = append(errs, FieldError{
			Field:   "policy.file_path",
			Message: "file path is required when mode is 'file'",
		})
	}

	// Validate git configuration when in git mode
	if cfg.Mode == "git" {
		if cfg.GitRepo == "" {
			errs = append(errs, FieldError{
				Field:   "policy.git_repo",
				Message: "git repository is required when mode is 'git'",
			})
		}
		if cfg.GitBranch == "" {
			errs = append(errs, FieldError{
				Field:   "policy.git_branch",
				Message: "git branch is required when mode is 'git'",
			})
		}
		if cfg.GitPath == "" {
			errs = append(errs, FieldError{
				Field:   "policy.git_path",
				Message: "git path is required when mode is 'git'",
			})
		}
	}

	return errs
}

// validateEvidence validates evidence configuration.
func validateEvidence(cfg *EvidenceConfig) []FieldError {
	var errs []FieldError

	// If evidence is disabled, skip validation
	if !cfg.Enabled {
		return errs
	}

	// Validate backend
	validBackends := map[string]bool{"sqlite": true, "postgres": true, "s3": true}
	if cfg.Backend == "" {
		errs = append(errs, FieldError{
			Field:   "evidence.backend",
			Message: "backend is required when evidence is enabled",
		})
	} else if !validBackends[cfg.Backend] {
		errs = append(errs, FieldError{
			Field:   "evidence.backend",
			Message: fmt.Sprintf("invalid backend %q: must be 'sqlite', 'postgres', or 's3'", cfg.Backend),
		})
	}

	// Validate backend-specific configuration
	switch cfg.Backend {
	case "sqlite":
		if cfg.SQLite.Path == "" {
			errs = append(errs, FieldError{
				Field:   "evidence.sqlite.path",
				Message: "SQLite path is required when backend is 'sqlite'",
			})
		}
	case "postgres":
		if cfg.Postgres.Host == "" {
			errs = append(errs, FieldError{
				Field:   "evidence.postgres.host",
				Message: "PostgreSQL host is required when backend is 'postgres'",
			})
		}
		if cfg.Postgres.Port < 1 || cfg.Postgres.Port > 65535 {
			errs = append(errs, FieldError{
				Field:   "evidence.postgres.port",
				Message: "PostgreSQL port must be between 1 and 65535",
			})
		}
		if cfg.Postgres.Database == "" {
			errs = append(errs, FieldError{
				Field:   "evidence.postgres.database",
				Message: "PostgreSQL database is required when backend is 'postgres'",
			})
		}
		if cfg.Postgres.User == "" {
			errs = append(errs, FieldError{
				Field:   "evidence.postgres.user",
				Message: "PostgreSQL user is required when backend is 'postgres'",
			})
		}
		// Password can be empty if using other auth methods
		validSSLModes := map[string]bool{"disable": true, "require": true, "verify-ca": true, "verify-full": true}
		if !validSSLModes[cfg.Postgres.SSLMode] {
			errs = append(errs, FieldError{
				Field:   "evidence.postgres.ssl_mode",
				Message: fmt.Sprintf("invalid SSL mode %q: must be 'disable', 'require', 'verify-ca', or 'verify-full'", cfg.Postgres.SSLMode),
			})
		}
	case "s3":
		if cfg.S3.Bucket == "" {
			errs = append(errs, FieldError{
				Field:   "evidence.s3.bucket",
				Message: "S3 bucket is required when backend is 's3'",
			})
		}
		if cfg.S3.Region == "" {
			errs = append(errs, FieldError{
				Field:   "evidence.s3.region",
				Message: "S3 region is required when backend is 's3'",
			})
		}
	}

	// Validate retention days
	if cfg.Retention.Days < 0 {
		errs = append(errs, FieldError{
			Field:   "evidence.retention.days",
			Message: "retention days must be non-negative",
		})
	}
	if cfg.Retention.Days > 3650 { // 10 years is excessive
		errs = append(errs, FieldError{
			Field:   "evidence.retention.days",
			Message: "retention days exceeds reasonable limit (3650 days / 10 years)",
		})
	}

	return errs
}

// validateTelemetry validates telemetry configuration.
func validateTelemetry(cfg *TelemetryConfig) []FieldError {
	var errs []FieldError

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if cfg.Logging.Level == "" {
		errs = append(errs, FieldError{
			Field:   "telemetry.logging.level",
			Message: "logging level is required",
		})
	} else if !validLevels[cfg.Logging.Level] {
		errs = append(errs, FieldError{
			Field:   "telemetry.logging.level",
			Message: fmt.Sprintf("invalid logging level %q: must be 'debug', 'info', 'warn', or 'error'", cfg.Logging.Level),
		})
	}

	// Validate logging format
	validFormats := map[string]bool{"json": true, "text": true}
	if cfg.Logging.Format == "" {
		errs = append(errs, FieldError{
			Field:   "telemetry.logging.format",
			Message: "logging format is required",
		})
	} else if !validFormats[cfg.Logging.Format] {
		errs = append(errs, FieldError{
			Field:   "telemetry.logging.format",
			Message: fmt.Sprintf("invalid logging format %q: must be 'json' or 'text'", cfg.Logging.Format),
		})
	}

	// Validate metrics prometheus path
	if cfg.Metrics.Enabled && cfg.Metrics.Path == "" {
		errs = append(errs, FieldError{
			Field:   "telemetry.metrics.path",
			Message: "metrics path is required when metrics are enabled",
		})
	}

	// Validate tracing configuration
	if cfg.Tracing.Enabled && cfg.Tracing.Endpoint == "" {
		errs = append(errs, FieldError{
			Field:   "telemetry.tracing.endpoint",
			Message: "tracing endpoint is required when tracing is enabled",
		})
	}
	if cfg.Tracing.SampleRatio < 0 || cfg.Tracing.SampleRatio > 1.0 {
		errs = append(errs, FieldError{
			Field:   "telemetry.tracing.sample_ratio",
			Message: "sample ratio must be between 0.0 and 1.0",
		})
	}

	// Validate health check configuration
	if cfg.Health.Enabled {
		// Validate paths are non-empty
		if cfg.Health.LivenessPath == "" {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.liveness_path",
				Message: "liveness path is required when health checks are enabled",
			})
		}
		if cfg.Health.ReadinessPath == "" {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.readiness_path",
				Message: "readiness path is required when health checks are enabled",
			})
		}
		if cfg.Health.VersionPath == "" {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.version_path",
				Message: "version path is required when health checks are enabled",
			})
		}

		// Validate paths start with /
		if cfg.Health.LivenessPath != "" && cfg.Health.LivenessPath[0] != '/' {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.liveness_path",
				Message: "liveness path must start with /",
			})
		}
		if cfg.Health.ReadinessPath != "" && cfg.Health.ReadinessPath[0] != '/' {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.readiness_path",
				Message: "readiness path must start with /",
			})
		}
		if cfg.Health.VersionPath != "" && cfg.Health.VersionPath[0] != '/' {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.version_path",
				Message: "version path must start with /",
			})
		}

		// Validate check timeout is reasonable
		if cfg.Health.CheckTimeout < 0 {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.check_timeout",
				Message: "check timeout must be positive",
			})
		}
		if cfg.Health.CheckTimeout > 60*time.Second {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.check_timeout",
				Message: "check timeout exceeds reasonable limit (60s)",
			})
		}

		// Validate min healthy providers
		if cfg.Health.MinHealthyProviders < 0 {
			errs = append(errs, FieldError{
				Field:   "telemetry.health.min_healthy_providers",
				Message: "min healthy providers must be non-negative",
			})
		}
	}

	return errs
}

// validateLimits validates limits configuration.
func validateLimits(cfg *LimitsConfig) []FieldError {
	var errs []FieldError

	// Validate budgets configuration
	if cfg.Budgets.Enabled {
		// Validate alert threshold
		if cfg.Budgets.AlertThreshold < 0.0 || cfg.Budgets.AlertThreshold > 1.0 {
			errs = append(errs, FieldError{
				Field:   "limits.budgets.alert_threshold",
				Message: "alert threshold must be between 0.0 and 1.0",
			})
		}

		// Validate per-API key budgets
		for apiKey, limits := range cfg.Budgets.ByAPIKey {
			prefix := fmt.Sprintf("limits.budgets.by_api_key.%s", apiKey)
			errs = append(errs, validateBudgetLimits(prefix, &limits)...)
		}

		// Validate per-user budgets
		for user, limits := range cfg.Budgets.ByUser {
			prefix := fmt.Sprintf("limits.budgets.by_user.%s", user)
			errs = append(errs, validateBudgetLimits(prefix, &limits)...)
		}

		// Validate per-team budgets
		for team, limits := range cfg.Budgets.ByTeam {
			prefix := fmt.Sprintf("limits.budgets.by_team.%s", team)
			errs = append(errs, validateBudgetLimits(prefix, &limits)...)
		}
	}

	// Validate rate limits configuration
	if cfg.RateLimits.Enabled {
		// Validate per-API key rate limits
		for apiKey, limits := range cfg.RateLimits.ByAPIKey {
			prefix := fmt.Sprintf("limits.rate_limits.by_api_key.%s", apiKey)
			errs = append(errs, validateRateLimits(prefix, &limits)...)
		}

		// Validate per-user rate limits
		for user, limits := range cfg.RateLimits.ByUser {
			prefix := fmt.Sprintf("limits.rate_limits.by_user.%s", user)
			errs = append(errs, validateRateLimits(prefix, &limits)...)
		}

		// Validate per-team rate limits
		for team, limits := range cfg.RateLimits.ByTeam {
			prefix := fmt.Sprintf("limits.rate_limits.by_team.%s", team)
			errs = append(errs, validateRateLimits(prefix, &limits)...)
		}
	}

	// Validate enforcement configuration
	errs = append(errs, validateEnforcement(&cfg.Enforcement)...)

	// Validate storage configuration
	errs = append(errs, validateLimitsStorage(&cfg.Storage)...)

	return errs
}

// validateBudgetLimits validates budget limit values.
func validateBudgetLimits(prefix string, limits *BudgetLimits) []FieldError {
	var errs []FieldError

	if limits.Hourly < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".hourly",
			Message: "hourly budget must be non-negative",
		})
	}
	if limits.Daily < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".daily",
			Message: "daily budget must be non-negative",
		})
	}
	if limits.Monthly < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".monthly",
			Message: "monthly budget must be non-negative",
		})
	}

	// Validate that smaller windows don't exceed larger windows
	if limits.Hourly > 0 && limits.Daily > 0 && limits.Hourly > limits.Daily {
		errs = append(errs, FieldError{
			Field:   prefix + ".hourly",
			Message: "hourly budget cannot exceed daily budget",
		})
	}
	if limits.Daily > 0 && limits.Monthly > 0 && limits.Daily > limits.Monthly {
		errs = append(errs, FieldError{
			Field:   prefix + ".daily",
			Message: "daily budget cannot exceed monthly budget",
		})
	}

	return errs
}

// validateRateLimits validates rate limit values.
func validateRateLimits(prefix string, limits *RateLimits) []FieldError {
	var errs []FieldError

	if limits.RequestsPerSecond < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".requests_per_second",
			Message: "requests per second must be non-negative",
		})
	}
	if limits.RequestsPerMinute < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".requests_per_minute",
			Message: "requests per minute must be non-negative",
		})
	}
	if limits.RequestsPerHour < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".requests_per_hour",
			Message: "requests per hour must be non-negative",
		})
	}
	if limits.TokensPerMinute < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".tokens_per_minute",
			Message: "tokens per minute must be non-negative",
		})
	}
	if limits.TokensPerHour < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".tokens_per_hour",
			Message: "tokens per hour must be non-negative",
		})
	}
	if limits.MaxConcurrent < 0 {
		errs = append(errs, FieldError{
			Field:   prefix + ".max_concurrent",
			Message: "max concurrent must be non-negative",
		})
	}

	// Validate reasonable limits
	if limits.RequestsPerSecond > 100000 {
		errs = append(errs, FieldError{
			Field:   prefix + ".requests_per_second",
			Message: "requests per second exceeds reasonable limit (100,000)",
		})
	}
	if limits.TokensPerMinute > 10000000 { // 10M tokens/min
		errs = append(errs, FieldError{
			Field:   prefix + ".tokens_per_minute",
			Message: "tokens per minute exceeds reasonable limit (10,000,000)",
		})
	}
	if limits.MaxConcurrent > 10000 {
		errs = append(errs, FieldError{
			Field:   prefix + ".max_concurrent",
			Message: "max concurrent exceeds reasonable limit (10,000)",
		})
	}

	return errs
}

// validateEnforcement validates enforcement configuration.
func validateEnforcement(cfg *EnforcementConfig) []FieldError {
	var errs []FieldError

	// Validate action
	validActions := map[string]bool{"block": true, "queue": true, "downgrade": true, "alert": true}
	if cfg.Action != "" && !validActions[cfg.Action] {
		errs = append(errs, FieldError{
			Field:   "limits.enforcement.action",
			Message: fmt.Sprintf("invalid action %q: must be 'block', 'queue', 'downgrade', or 'alert'", cfg.Action),
		})
	}

	// Validate queue configuration
	if cfg.QueueDepth < 0 {
		errs = append(errs, FieldError{
			Field:   "limits.enforcement.queue_depth",
			Message: "queue depth must be non-negative",
		})
	}
	if cfg.QueueDepth > 100000 {
		errs = append(errs, FieldError{
			Field:   "limits.enforcement.queue_depth",
			Message: "queue depth exceeds reasonable limit (100,000)",
		})
	}
	if cfg.QueueTimeout < 0 {
		errs = append(errs, FieldError{
			Field:   "limits.enforcement.queue_timeout",
			Message: "queue timeout must be positive",
		})
	}

	// Validate model downgrades (ensure no circular references)
	if len(cfg.ModelDowngrades) > 0 {
		visited := make(map[string]bool)
		for model := range cfg.ModelDowngrades {
			if err := checkCircularDowngrade(model, cfg.ModelDowngrades, visited); err != nil {
				errs = append(errs, FieldError{
					Field:   "limits.enforcement.model_downgrades",
					Message: err.Error(),
				})
				break // Only report one circular reference error
			}
		}
	}

	return errs
}

// checkCircularDowngrade checks for circular references in model downgrades.
func checkCircularDowngrade(model string, downgrades map[string]string, visited map[string]bool) error {
	if visited[model] {
		return fmt.Errorf("circular downgrade detected for model %q", model)
	}

	visited[model] = true
	if next, ok := downgrades[model]; ok {
		if err := checkCircularDowngrade(next, downgrades, visited); err != nil {
			return err
		}
	}
	delete(visited, model)

	return nil
}

// validateLimitsStorage validates limits storage configuration.
func validateLimitsStorage(cfg *LimitsStorageConfig) []FieldError {
	var errs []FieldError

	// Validate backend
	validBackends := map[string]bool{"memory": true, "sqlite": true}
	if cfg.Backend == "" {
		errs = append(errs, FieldError{
			Field:   "limits.storage.backend",
			Message: "backend is required",
		})
	} else if !validBackends[cfg.Backend] {
		errs = append(errs, FieldError{
			Field:   "limits.storage.backend",
			Message: fmt.Sprintf("invalid backend %q: must be 'memory' or 'sqlite'", cfg.Backend),
		})
	}

	// Validate backend-specific configuration
	switch cfg.Backend {
	case "sqlite":
		if cfg.SQLite.Path == "" {
			errs = append(errs, FieldError{
				Field:   "limits.storage.sqlite.path",
				Message: "SQLite path is required when backend is 'sqlite'",
			})
		}
		if cfg.SQLite.SnapshotInterval < 0 {
			errs = append(errs, FieldError{
				Field:   "limits.storage.sqlite.snapshot_interval",
				Message: "snapshot interval must be positive",
			})
		}
	case "memory":
		if cfg.Memory.MaxEntries < 0 {
			errs = append(errs, FieldError{
				Field:   "limits.storage.memory.max_entries",
				Message: "max entries must be non-negative",
			})
		}
		if cfg.Memory.CleanupInterval < 0 {
			errs = append(errs, FieldError{
				Field:   "limits.storage.memory.cleanup_interval",
				Message: "cleanup interval must be positive",
			})
		}
	}

	return errs
}

// validateSecurity validates security configuration.
func validateSecurity(cfg *SecurityConfig) []FieldError {
	var errs []FieldError

	// Validate TLS configuration
	if cfg.TLS.Enabled {
		if cfg.TLS.CertFile == "" {
			errs = append(errs, FieldError{
				Field:   "security.tls.cert_file",
				Message: "TLS certificate file is required when TLS is enabled",
			})
		}
		if cfg.TLS.KeyFile == "" {
			errs = append(errs, FieldError{
				Field:   "security.tls.key_file",
				Message: "TLS key file is required when TLS is enabled",
			})
		}
	}

	// Validate mTLS configuration
	if cfg.TLS.MTLS.Enabled {
		if cfg.TLS.MTLS.ClientCAFile == "" {
			errs = append(errs, FieldError{
				Field:   "security.tls.mtls.client_ca_file",
				Message: "mTLS client CA file is required when mTLS is enabled",
			})
		}
		// mTLS requires TLS to be enabled
		if !cfg.TLS.Enabled {
			errs = append(errs, FieldError{
				Field:   "security.tls.mtls.enabled",
				Message: "mTLS requires TLS to be enabled (security.tls.enabled must be true)",
			})
		}
	}

	return errs
}
