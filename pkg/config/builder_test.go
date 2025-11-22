package config

import "time"

// ConfigBuilder provides a fluent API for building Config instances in tests.
// It starts with default values and allows selective overrides.
type ConfigBuilder struct {
	cfg Config
}

// NewTestConfig creates a new ConfigBuilder with sensible defaults for testing.
// The resulting configuration is valid and can be used immediately.
func NewTestConfig() *ConfigBuilder {
	cfg := Config{
		Providers: make(map[string]ProviderConfig),
	}
	ApplyDefaults(&cfg)

	// Add a default provider for tests
	cfg.Providers["openai"] = ProviderConfig{
		BaseURL:    "https://api.openai.com/v1",
		APIKey:     "test-key",
		Timeout:    DefaultProviderTimeout,
		MaxRetries: DefaultProviderMaxRetries,
	}

	return &ConfigBuilder{cfg: cfg}
}

// Build returns the built Config instance.
func (b *ConfigBuilder) Build() *Config {
	return &b.cfg
}

// WithListenAddress sets the proxy listen address.
func (b *ConfigBuilder) WithListenAddress(addr string) *ConfigBuilder {
	b.cfg.Proxy.ListenAddress = addr
	return b
}

// WithReadTimeout sets the proxy read timeout.
func (b *ConfigBuilder) WithReadTimeout(d time.Duration) *ConfigBuilder {
	b.cfg.Proxy.ReadTimeout = d
	return b
}

// WithProvider adds or updates a provider configuration.
func (b *ConfigBuilder) WithProvider(name string, provider ProviderConfig) *ConfigBuilder {
	if b.cfg.Providers == nil {
		b.cfg.Providers = make(map[string]ProviderConfig)
	}
	b.cfg.Providers[name] = provider
	return b
}

// WithPolicyMode sets the policy mode.
func (b *ConfigBuilder) WithPolicyMode(mode string) *ConfigBuilder {
	b.cfg.Policy.Mode = mode
	return b
}

// WithPolicyFilePath sets the policy file path.
func (b *ConfigBuilder) WithPolicyFilePath(path string) *ConfigBuilder {
	b.cfg.Policy.FilePath = path
	return b
}

// WithPolicyGitRepo sets the policy git repository.
func (b *ConfigBuilder) WithPolicyGitRepo(repo string) *ConfigBuilder {
	b.cfg.Policy.GitRepo = repo
	b.cfg.Policy.Mode = "git"
	if b.cfg.Policy.GitBranch == "" {
		b.cfg.Policy.GitBranch = "main"
	}
	if b.cfg.Policy.GitPath == "" {
		b.cfg.Policy.GitPath = "policies.yaml"
	}
	return b
}

// WithEvidenceEnabled sets whether evidence is enabled.
func (b *ConfigBuilder) WithEvidenceEnabled(enabled bool) *ConfigBuilder {
	b.cfg.Evidence.Enabled = enabled
	return b
}

// WithEvidenceBackend sets the evidence backend.
func (b *ConfigBuilder) WithEvidenceBackend(backend string) *ConfigBuilder {
	b.cfg.Evidence.Backend = backend
	return b
}

// WithSQLitePath sets the SQLite database path for evidence.
func (b *ConfigBuilder) WithSQLitePath(path string) *ConfigBuilder {
	b.cfg.Evidence.SQLite.Path = path
	b.cfg.Evidence.Backend = "sqlite"
	return b
}

// WithPostgresConfig sets PostgreSQL configuration for evidence.
func (b *ConfigBuilder) WithPostgresConfig(host, database, user, password string, port int) *ConfigBuilder {
	b.cfg.Evidence.Backend = "postgres"
	b.cfg.Evidence.Postgres = PostgresConfig{
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
		SSLMode:  DefaultPostgresSSLMode,
	}
	return b
}

// WithS3Config sets S3 configuration for evidence.
func (b *ConfigBuilder) WithS3Config(bucket, region string) *ConfigBuilder {
	b.cfg.Evidence.Backend = "s3"
	b.cfg.Evidence.S3 = S3Config{
		Bucket: bucket,
		Region: region,
	}
	return b
}

// WithLoggingLevel sets the logging level.
func (b *ConfigBuilder) WithLoggingLevel(level string) *ConfigBuilder {
	b.cfg.Telemetry.Logging.Level = level
	return b
}

// WithLoggingFormat sets the logging format.
func (b *ConfigBuilder) WithLoggingFormat(format string) *ConfigBuilder {
	b.cfg.Telemetry.Logging.Format = format
	return b
}

// WithMetricsEnabled sets whether metrics are enabled.
func (b *ConfigBuilder) WithMetricsEnabled(enabled bool) *ConfigBuilder {
	b.cfg.Telemetry.Metrics.Enabled = enabled
	return b
}

// WithTracingEnabled sets whether tracing is enabled.
func (b *ConfigBuilder) WithTracingEnabled(enabled bool, endpoint string) *ConfigBuilder {
	b.cfg.Telemetry.Tracing.Enabled = enabled
	b.cfg.Telemetry.Tracing.Endpoint = endpoint
	if b.cfg.Telemetry.Tracing.SampleRatio == 0 {
		b.cfg.Telemetry.Tracing.SampleRatio = DefaultTracingSamplingRate
	}
	return b
}

// WithTLS sets TLS configuration.
func (b *ConfigBuilder) WithTLS(certFile, keyFile string) *ConfigBuilder {
	b.cfg.Security.TLS.Enabled = true
	b.cfg.Security.TLS.CertFile = certFile
	b.cfg.Security.TLS.KeyFile = keyFile
	return b
}

// WithMTLS sets mutual TLS configuration.
func (b *ConfigBuilder) WithMTLS(caFile string) *ConfigBuilder {
	b.cfg.Security.TLS.MTLS.Enabled = true
	b.cfg.Security.TLS.MTLS.ClientCAFile = caFile
	// mTLS requires TLS
	if !b.cfg.Security.TLS.Enabled {
		b.cfg.Security.TLS.Enabled = true
	}
	return b
}

// MinimalConfig returns a minimal valid configuration for testing.
// This is useful for tests that don't care about most configuration values.
func MinimalConfig() *Config {
	return NewTestConfig().Build()
}
