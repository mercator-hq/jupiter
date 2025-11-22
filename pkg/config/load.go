package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from a YAML file at the specified path.
// It applies default values, validates the configuration, and returns any errors.
// The configuration is not modified by environment variables; use LoadConfigWithEnvOverrides
// for that functionality.
func LoadConfig(path string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", path, err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file %q: %w", path, err)
	}

	// Apply defaults
	ApplyDefaults(&cfg)

	// Validate
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadConfigWithEnvOverrides loads configuration from a YAML file and applies
// environment variable overrides. Environment variables follow the naming
// convention MERCATOR_SECTION_FIELD (e.g., MERCATOR_PROXY_LISTEN_ADDRESS).
// Environment variables always take precedence over file-based configuration.
//
// The loading sequence is:
// 1. Load YAML from file
// 2. Apply default values
// 3. Apply environment variable overrides
// 4. Validate final configuration
func LoadConfigWithEnvOverrides(path string) (*Config, error) {
	// First load from file (this already applies defaults)
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Re-validate after overrides
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed after environment overrides: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration.
// Environment variables use the format MERCATOR_SECTION_FIELD.
func applyEnvOverrides(cfg *Config) {
	// Proxy overrides
	if val := os.Getenv("MERCATOR_PROXY_LISTEN_ADDRESS"); val != "" {
		cfg.Proxy.ListenAddress = val
	}
	if val := os.Getenv("MERCATOR_PROXY_READ_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Proxy.ReadTimeout = d
		}
	}
	if val := os.Getenv("MERCATOR_PROXY_WRITE_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Proxy.WriteTimeout = d
		}
	}
	if val := os.Getenv("MERCATOR_PROXY_IDLE_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Proxy.IdleTimeout = d
		}
	}
	if val := os.Getenv("MERCATOR_PROXY_MAX_HEADER_BYTES"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Proxy.MaxHeaderBytes = i
		}
	}

	// Provider overrides - we need to handle dynamic provider names
	// For now, we'll support common providers: openai, anthropic
	applyProviderEnvOverrides(cfg, "openai")
	applyProviderEnvOverrides(cfg, "anthropic")
	// Add more providers as needed

	// Policy overrides
	if val := os.Getenv("MERCATOR_POLICY_MODE"); val != "" {
		cfg.Policy.Mode = val
	}
	if val := os.Getenv("MERCATOR_POLICY_FILE_PATH"); val != "" {
		cfg.Policy.FilePath = val
	}
	if val := os.Getenv("MERCATOR_POLICY_GIT_REPO"); val != "" {
		cfg.Policy.GitRepo = val
	}
	if val := os.Getenv("MERCATOR_POLICY_GIT_BRANCH"); val != "" {
		cfg.Policy.GitBranch = val
	}
	if val := os.Getenv("MERCATOR_POLICY_GIT_PATH"); val != "" {
		cfg.Policy.GitPath = val
	}
	if val := os.Getenv("MERCATOR_POLICY_WATCH"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Policy.Watch = b
		}
	}
	if val := os.Getenv("MERCATOR_POLICY_VALIDATION_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Policy.Validation.Enabled = b
		}
	}
	if val := os.Getenv("MERCATOR_POLICY_VALIDATION_STRICT"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Policy.Validation.Strict = b
		}
	}

	// Evidence overrides
	if val := os.Getenv("MERCATOR_EVIDENCE_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Evidence.Enabled = b
		}
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_BACKEND"); val != "" {
		cfg.Evidence.Backend = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_SQLITE_PATH"); val != "" {
		cfg.Evidence.SQLite.Path = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_POSTGRES_HOST"); val != "" {
		cfg.Evidence.Postgres.Host = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_POSTGRES_PORT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Evidence.Postgres.Port = i
		}
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_POSTGRES_DATABASE"); val != "" {
		cfg.Evidence.Postgres.Database = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_POSTGRES_USER"); val != "" {
		cfg.Evidence.Postgres.User = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_POSTGRES_PASSWORD"); val != "" {
		cfg.Evidence.Postgres.Password = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_POSTGRES_SSL_MODE"); val != "" {
		cfg.Evidence.Postgres.SSLMode = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_S3_BUCKET"); val != "" {
		cfg.Evidence.S3.Bucket = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_S3_REGION"); val != "" {
		cfg.Evidence.S3.Region = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_S3_PREFIX"); val != "" {
		cfg.Evidence.S3.Prefix = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_S3_ENDPOINT"); val != "" {
		cfg.Evidence.S3.Endpoint = val
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_RETENTION_DAYS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Evidence.Retention.Days = i
		}
	}
	if val := os.Getenv("MERCATOR_EVIDENCE_SIGNING_KEY_PATH"); val != "" {
		cfg.Evidence.SigningKeyPath = val
	}

	// Telemetry overrides
	if val := os.Getenv("MERCATOR_TELEMETRY_LOGGING_LEVEL"); val != "" {
		cfg.Telemetry.Logging.Level = val
	}
	if val := os.Getenv("MERCATOR_TELEMETRY_LOGGING_FORMAT"); val != "" {
		cfg.Telemetry.Logging.Format = val
	}
	if val := os.Getenv("MERCATOR_TELEMETRY_METRICS_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Telemetry.Metrics.Enabled = b
		}
	}
	if val := os.Getenv("MERCATOR_TELEMETRY_METRICS_PATH"); val != "" {
		cfg.Telemetry.Metrics.Path = val
	}
	if val := os.Getenv("MERCATOR_TELEMETRY_TRACING_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Telemetry.Tracing.Enabled = b
		}
	}
	if val := os.Getenv("MERCATOR_TELEMETRY_TRACING_ENDPOINT"); val != "" {
		cfg.Telemetry.Tracing.Endpoint = val
	}
	if val := os.Getenv("MERCATOR_TELEMETRY_TRACING_SAMPLE_RATIO"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Telemetry.Tracing.SampleRatio = f
		}
	}

	// Security overrides
	if val := os.Getenv("MERCATOR_SECURITY_TLS_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Security.TLS.Enabled = b
		}
	}
	if val := os.Getenv("MERCATOR_SECURITY_TLS_CERT_FILE"); val != "" {
		cfg.Security.TLS.CertFile = val
	}
	if val := os.Getenv("MERCATOR_SECURITY_TLS_KEY_FILE"); val != "" {
		cfg.Security.TLS.KeyFile = val
	}
	if val := os.Getenv("MERCATOR_SECURITY_MTLS_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Security.TLS.MTLS.Enabled = b
		}
	}
	if val := os.Getenv("MERCATOR_SECURITY_MTLS_CA_FILE"); val != "" {
		cfg.Security.TLS.MTLS.ClientCAFile = val
	}
}

// applyProviderEnvOverrides applies environment variable overrides for a specific provider.
// Provider environment variables follow the format MERCATOR_PROVIDERS_<NAME>_<FIELD>
// where NAME is the uppercase provider name.
func applyProviderEnvOverrides(cfg *Config, providerName string) {
	// Initialize providers map if nil
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	// Get existing provider config or create new one
	provider, exists := cfg.Providers[providerName]
	if !exists {
		provider = ProviderConfig{}
	}

	// Build environment variable prefix
	prefix := fmt.Sprintf("MERCATOR_PROVIDERS_%s_", strings.ToUpper(providerName))

	// Check for overrides
	modified := false

	if val := os.Getenv(prefix + "BASE_URL"); val != "" {
		provider.BaseURL = val
		modified = true
	}
	if val := os.Getenv(prefix + "API_KEY"); val != "" {
		provider.APIKey = val
		modified = true
	}
	if val := os.Getenv(prefix + "TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			provider.Timeout = d
			modified = true
		}
	}
	if val := os.Getenv(prefix + "MAX_RETRIES"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			provider.MaxRetries = i
			modified = true
		}
	}

	// Only update the map if we found at least one override
	if modified || exists {
		cfg.Providers[providerName] = provider
	}
}
