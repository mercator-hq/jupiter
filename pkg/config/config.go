package config

import "time"

// Config is the root configuration structure for Mercator Jupiter.
// It contains all configuration sections for the proxy server, providers,
// policy engine, evidence storage, telemetry, and security settings.
type Config struct {
	// Proxy contains HTTP proxy server configuration including listen address,
	// timeouts, and connection limits.
	Proxy ProxyConfig `yaml:"proxy"`

	// Providers contains configuration for all LLM provider integrations.
	// Keys are provider names (e.g., "openai", "anthropic").
	Providers map[string]ProviderConfig `yaml:"providers"`

	// Policy contains configuration for the policy engine including policy
	// source location, validation settings, and watch mode.
	Policy PolicyConfig `yaml:"policy"`

	// Evidence contains configuration for evidence generation and storage
	// including backend selection, retention, and signing settings.
	Evidence EvidenceConfig `yaml:"evidence"`

	// Processing contains configuration for request/response processing including
	// token estimation, cost calculation, and content analysis.
	Processing ProcessingConfig `yaml:"processing"`

	// Routing contains configuration for the routing engine including
	// strategy selection, sticky routing, and fallback settings.
	Routing RoutingConfig `yaml:"routing"`

	// Limits contains configuration for budget tracking and rate limiting.
	Limits LimitsConfig `yaml:"limits"`

	// Telemetry contains configuration for observability including logging,
	// metrics, and distributed tracing.
	Telemetry TelemetryConfig `yaml:"telemetry"`

	// Security contains security-related configuration including TLS settings,
	// mutual TLS, and authentication.
	Security SecurityConfig `yaml:"security"`
}

// ProxyConfig contains configuration for the HTTP proxy server.
type ProxyConfig struct {
	// ListenAddress is the address and port for the proxy to listen on.
	// Format: "host:port" (e.g., "127.0.0.1:8080", "0.0.0.0:8080").
	// Default: "127.0.0.1:8080"
	ListenAddress string `yaml:"listen_address"`

	// ReadTimeout is the maximum duration for reading the entire request,
	// including the body. A zero or negative value means no timeout.
	// Default: 30s
	ReadTimeout time.Duration `yaml:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the
	// response. A zero or negative value means no timeout.
	// Default: 30s
	WriteTimeout time.Duration `yaml:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alives are enabled. If IdleTimeout is zero, ReadTimeout is used.
	// Default: 120s
	IdleTimeout time.Duration `yaml:"idle_timeout"`

	// ShutdownTimeout is the maximum duration to wait for graceful shutdown.
	// If requests are still in-flight after this timeout, the server will
	// force shutdown.
	// Default: 30s
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

	// MaxHeaderBytes controls the maximum number of bytes the server will
	// read parsing the request header's keys and values, including the
	// request line. It does not limit the size of the request body.
	// Default: 1048576 (1MB)
	MaxHeaderBytes int `yaml:"max_header_bytes"`

	// CORS contains Cross-Origin Resource Sharing configuration.
	CORS CORSConfig `yaml:"cors"`
}

// CORSConfig contains CORS (Cross-Origin Resource Sharing) configuration.
type CORSConfig struct {
	// Enabled controls whether CORS is enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// AllowedOrigins is a list of allowed origins for CORS requests.
	// Use ["*"] to allow all origins (not recommended for production).
	// Default: ["*"]
	AllowedOrigins []string `yaml:"allowed_origins"`

	// AllowedMethods is a list of allowed HTTP methods for CORS requests.
	// Default: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
	AllowedMethods []string `yaml:"allowed_methods"`

	// AllowedHeaders is a list of allowed HTTP headers for CORS requests.
	// Default: ["Authorization", "Content-Type", "X-Request-ID", "X-User-ID"]
	AllowedHeaders []string `yaml:"allowed_headers"`

	// ExposedHeaders is a list of headers that are exposed to the client.
	// Default: ["X-Request-ID"]
	ExposedHeaders []string `yaml:"exposed_headers"`

	// MaxAge is the maximum age (in seconds) for preflight request cache.
	// Default: 3600 (1 hour)
	MaxAge int `yaml:"max_age"`

	// AllowCredentials controls whether credentials (cookies, auth headers)
	// are allowed in CORS requests.
	// Default: false
	AllowCredentials bool `yaml:"allow_credentials"`
}

// ProviderConfig contains configuration for a single LLM provider.
type ProviderConfig struct {
	// BaseURL is the base URL for the provider's API endpoint.
	// Example: "https://api.openai.com/v1"
	BaseURL string `yaml:"base_url"`

	// APIKey is the authentication key for the provider.
	// This should typically be loaded from an environment variable.
	// Required for most providers.
	APIKey string `yaml:"api_key"`

	// Timeout is the maximum duration for requests to this provider.
	// Default: 60s
	Timeout time.Duration `yaml:"timeout"`

	// MaxRetries is the maximum number of retry attempts for failed requests.
	// Default: 3
	MaxRetries int `yaml:"max_retries"`
}

// PolicyConfig contains configuration for the policy engine.
type PolicyConfig struct {
	// Mode specifies how policies are loaded.
	// Options: "file" (local file), "git" (Git repository)
	// Default: "file"
	Mode string `yaml:"mode"`

	// FilePath is the path to the policy file when Mode is "file".
	// Default: "./policies.yaml"
	FilePath string `yaml:"file_path"`

	// GitRepo is the Git repository URL when Mode is "git".
	// Only used when Mode is "git".
	// DEPRECATED: Use Git.Repository instead.
	GitRepo string `yaml:"git_repo"`

	// GitBranch is the Git branch to use when Mode is "git".
	// Default: "main"
	// DEPRECATED: Use Git.Branch instead.
	GitBranch string `yaml:"git_branch"`

	// GitPath is the path within the Git repository to the policy file.
	// Default: "policies.yaml"
	// DEPRECATED: Use Git.Path instead.
	GitPath string `yaml:"git_path"`

	// Git contains comprehensive Git repository configuration.
	// Used when Mode is "git".
	Git GitPolicyConfig `yaml:"git"`

	// Watch enables automatic reloading when policy files change.
	// Default: false
	Watch bool `yaml:"watch"`

	// Validation contains policy validation settings.
	Validation PolicyValidationConfig `yaml:"validation"`
}

// GitPolicyConfig configures Git-based policy loading.
type GitPolicyConfig struct {
	// Enabled determines if Git mode is active.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// Repository URL (HTTPS or SSH).
	// Example: "https://github.com/company/policies.git"
	// Example: "git@github.com:company/policies.git"
	Repository string `yaml:"repository"`

	// Branch to track (supports environment variable expansion).
	// Example: "main", "dev", "${ENVIRONMENT}"
	// Default: "main"
	Branch string `yaml:"branch"`

	// Path within repository to policy files.
	// Example: "policies/", "config/policies/"
	// Default: "" (root directory)
	Path string `yaml:"path"`

	// Auth configures Git authentication.
	Auth GitAuthConfig `yaml:"auth"`

	// Poll configures change detection.
	Poll GitPollConfig `yaml:"poll"`

	// Clone configures repository cloning.
	Clone GitCloneConfig `yaml:"clone"`
}

// GitAuthConfig configures Git authentication.
type GitAuthConfig struct {
	// Type: "token", "ssh", "none"
	// - "token": HTTPS with personal access token
	// - "ssh": SSH with public key
	// - "none": public repositories
	// Default: "none"
	Type string `yaml:"type"`

	// Token for HTTPS authentication (supports env vars).
	// Example: "${GITHUB_TOKEN}"
	// Required when Type is "token".
	Token string `yaml:"token"`

	// SSHKeyPath for SSH authentication.
	// Example: "/home/user/.ssh/id_rsa"
	// Required when Type is "ssh".
	SSHKeyPath string `yaml:"ssh_key_path"`

	// SSHKeyPassphrase for encrypted SSH keys (supports env vars).
	// Example: "${SSH_PASSPHRASE}"
	// Optional, leave empty if key is not encrypted.
	SSHKeyPassphrase string `yaml:"ssh_key_passphrase"`
}

// GitPollConfig configures change detection.
type GitPollConfig struct {
	// Enabled determines if polling is active.
	// When false, policies are loaded once at startup.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Interval between polls (e.g., "30s", "1m", "5m").
	// Lower values = faster change detection but more load.
	// Default: 30s
	Interval time.Duration `yaml:"interval"`

	// Timeout for Git operations.
	// Default: 10s
	Timeout time.Duration `yaml:"timeout"`
}

// GitCloneConfig configures repository cloning.
type GitCloneConfig struct {
	// Depth for shallow clones (0 = full clone).
	// Shallow clones are faster but don't include full history.
	// Set to 1 for fastest cloning of large repositories.
	// Default: 1
	Depth int `yaml:"depth"`

	// LocalPath where repository is cloned.
	// Example: "/var/lib/mercator/policies"
	// Default: system temp directory
	LocalPath string `yaml:"local_path"`

	// CleanOnStart removes local repo before cloning.
	// Useful for ensuring clean state on restart.
	// Default: false
	CleanOnStart bool `yaml:"clean_on_start"`
}

// PolicyValidationConfig contains configuration for policy validation.
type PolicyValidationConfig struct {
	// Enabled controls whether policy validation is performed.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Strict controls whether validation warnings are treated as errors.
	// When false, warnings are logged but don't prevent policy loading.
	// Default: false
	Strict bool `yaml:"strict"`
}

// EvidenceConfig contains configuration for evidence generation and storage.
type EvidenceConfig struct {
	// Enabled controls whether evidence generation is active.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Backend specifies the storage backend for evidence records.
	// Options: "sqlite", "postgres", "s3"
	// Default: "sqlite"
	Backend string `yaml:"backend"`

	// SQLite contains SQLite-specific configuration.
	SQLite SQLiteConfig `yaml:"sqlite"`

	// Postgres contains PostgreSQL-specific configuration.
	Postgres PostgresConfig `yaml:"postgres"`

	// S3 contains S3-specific configuration.
	S3 S3Config `yaml:"s3"`

	// Recorder contains evidence recorder configuration.
	Recorder RecorderConfig `yaml:"recorder"`

	// Retention contains retention policy configuration.
	Retention RetentionConfig `yaml:"retention"`

	// Query contains query configuration.
	Query QueryConfig `yaml:"query"`

	// Export contains export configuration.
	Export ExportConfig `yaml:"export"`

	// SigningKeyPath is the path to the private key used for signing
	// evidence records. If not specified, evidence is not signed.
	SigningKeyPath string `yaml:"signing_key_path"`
}

// SQLiteConfig contains SQLite-specific configuration.
type SQLiteConfig struct {
	// Path is the file path for the SQLite database.
	// Default: "data/evidence.db"
	Path string `yaml:"path"`

	// MaxOpenConns is the maximum number of open database connections.
	// Default: 10
	MaxOpenConns int `yaml:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle database connections.
	// Default: 5
	MaxIdleConns int `yaml:"max_idle_conns"`

	// WALMode enables Write-Ahead Logging mode for better concurrency.
	// Default: true
	WALMode bool `yaml:"wal_mode"`

	// BusyTimeout is the duration to wait when the database is locked.
	// Default: 5s
	BusyTimeout time.Duration `yaml:"busy_timeout"`
}

// RecorderConfig contains evidence recorder configuration.
type RecorderConfig struct {
	// AsyncBuffer is the size of the async write channel buffer.
	// Default: 1000
	AsyncBuffer int `yaml:"async_buffer"`

	// WriteTimeout is the timeout for writing evidence to storage.
	// Default: 5s
	WriteTimeout time.Duration `yaml:"write_timeout"`

	// HashRequest enables hashing of request bodies.
	// Default: true
	HashRequest bool `yaml:"hash_request"`

	// HashResponse enables hashing of response bodies.
	// Default: true
	HashResponse bool `yaml:"hash_response"`

	// RedactAPIKeys enables API key redaction.
	// Default: true
	RedactAPIKeys bool `yaml:"redact_api_keys"`

	// MaxFieldLength is the maximum length for text fields before truncation.
	// Default: 500
	MaxFieldLength int `yaml:"max_field_length"`
}

// RetentionConfig contains retention policy configuration.
type RetentionConfig struct {
	// Days is the number of days to retain evidence records.
	// Records older than this are eligible for deletion.
	// 0 means keep evidence forever (no pruning).
	// Default: 90
	Days int `yaml:"days"`

	// PruneSchedule is a cron expression for scheduling pruning.
	// Default: "0 3 * * *" (daily at 3 AM)
	PruneSchedule string `yaml:"prune_schedule"`

	// ArchiveBeforeDelete enables archiving evidence before deletion.
	// Default: false
	ArchiveBeforeDelete bool `yaml:"archive_before_delete"`

	// ArchivePath is the directory to store archived evidence.
	// Default: "data/archives/"
	ArchivePath string `yaml:"archive_path"`

	// MaxRecords is the maximum number of records to keep.
	// 0 means unlimited.
	// Default: 0
	MaxRecords int64 `yaml:"max_records"`
}

// QueryConfig contains query configuration.
type QueryConfig struct {
	// DefaultLimit is the default number of records to return if not specified.
	// Default: 100
	DefaultLimit int `yaml:"default_limit"`

	// MaxLimit is the maximum number of records that can be returned in a single query.
	// Default: 10000
	MaxLimit int `yaml:"max_limit"`

	// Timeout is the query execution timeout.
	// Default: 30s
	Timeout time.Duration `yaml:"timeout"`
}

// ExportConfig contains export configuration.
type ExportConfig struct {
	// JSONPretty enables pretty-printing for JSON exports.
	// Default: true
	JSONPretty bool `yaml:"json_pretty"`

	// CSVIncludeHeader includes a header row in CSV exports.
	// Default: true
	CSVIncludeHeader bool `yaml:"csv_include_header"`

	// MaxExportSize is the maximum number of records per export.
	// Default: 1000000 (1 million)
	MaxExportSize int `yaml:"max_export_size"`
}

// PostgresConfig contains PostgreSQL-specific configuration.
type PostgresConfig struct {
	// Host is the PostgreSQL server hostname.
	Host string `yaml:"host"`

	// Port is the PostgreSQL server port.
	// Default: 5432
	Port int `yaml:"port"`

	// Database is the name of the database to use.
	Database string `yaml:"database"`

	// User is the PostgreSQL user for authentication.
	User string `yaml:"user"`

	// Password is the PostgreSQL password for authentication.
	// This should typically be loaded from an environment variable.
	Password string `yaml:"password"`

	// SSLMode controls SSL/TLS connection mode.
	// Options: "disable", "require", "verify-ca", "verify-full"
	// Default: "require"
	SSLMode string `yaml:"ssl_mode"`
}

// S3Config contains S3-specific configuration.
type S3Config struct {
	// Bucket is the S3 bucket name for evidence storage.
	Bucket string `yaml:"bucket"`

	// Region is the AWS region for the S3 bucket.
	Region string `yaml:"region"`

	// Prefix is an optional key prefix for all evidence objects.
	Prefix string `yaml:"prefix"`

	// Endpoint is an optional custom S3 endpoint (for S3-compatible services).
	Endpoint string `yaml:"endpoint"`
}

// TelemetryConfig contains configuration for observability.
type TelemetryConfig struct {
	// Logging contains logging configuration.
	Logging LoggingConfig `yaml:"logging"`

	// Metrics contains metrics collection configuration.
	Metrics MetricsConfig `yaml:"metrics"`

	// Tracing contains distributed tracing configuration.
	Tracing TracingConfig `yaml:"tracing"`

	// Health contains health check configuration.
	Health HealthConfig `yaml:"health"`
}

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	// Level is the minimum log level to emit.
	// Options: "debug", "info", "warn", "error"
	// Default: "info"
	Level string `yaml:"level"`

	// Format controls the log output format.
	// Options: "json", "text", "console"
	// Default: "json"
	Format string `yaml:"format"`

	// AddSource includes file and line number in log entries.
	// Default: false
	AddSource bool `yaml:"add_source"`

	// RedactPII enables automatic PII redaction in logs.
	// Redacts API keys, emails, SSN, IP addresses, etc.
	// Default: true
	RedactPII bool `yaml:"redact_pii"`

	// BufferSize is the size of the async log buffer.
	// Logs are written asynchronously to avoid blocking.
	// Default: 10000
	BufferSize int `yaml:"buffer_size"`

	// RedactPatterns contains custom PII redaction patterns.
	// Each pattern has a name, regex, and replacement string.
	RedactPatterns []RedactPattern `yaml:"redact_patterns"`
}

// RedactPattern defines a custom PII redaction pattern.
type RedactPattern struct {
	// Name is a descriptive name for the pattern.
	Name string `yaml:"name"`

	// Pattern is the regular expression to match.
	Pattern string `yaml:"pattern"`

	// Replacement is the string to replace matches with.
	Replacement string `yaml:"replacement"`
}

// MetricsConfig contains metrics collection configuration.
type MetricsConfig struct {
	// Enabled controls whether metrics collection is active.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Path is the HTTP path for the Prometheus metrics endpoint.
	// Default: "/metrics"
	Path string `yaml:"path"`

	// Port is an optional separate port for metrics (0 = use proxy port).
	// Default: 0
	Port int `yaml:"port"`

	// Namespace is the metric name prefix.
	// Default: "mercator"
	Namespace string `yaml:"namespace"`

	// Subsystem is the metric subsystem name.
	// Default: "jupiter"
	Subsystem string `yaml:"subsystem"`

	// RequestDurationBuckets defines histogram buckets for request duration (seconds).
	// Default: [0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0]
	RequestDurationBuckets []float64 `yaml:"request_duration_buckets"`

	// TokenCountBuckets defines histogram buckets for token counts.
	// Default: [100, 500, 1000, 5000, 10000, 50000, 100000]
	TokenCountBuckets []float64 `yaml:"token_count_buckets"`
}

// TracingConfig contains distributed tracing configuration.
type TracingConfig struct {
	// Enabled controls whether distributed tracing is active.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// Sampler determines the sampling strategy.
	// Options: "always", "never", "ratio"
	// Default: "ratio"
	Sampler string `yaml:"sampler"`

	// SampleRatio is the fraction of traces to sample (0.0 to 1.0).
	// Only used when Sampler is "ratio".
	// Default: 0.1 (10%)
	SampleRatio float64 `yaml:"sample_ratio"`

	// Exporter determines the trace exporter to use.
	// Options: "otlp", "jaeger", "zipkin"
	// Default: "otlp"
	Exporter string `yaml:"exporter"`

	// Endpoint is the trace collector endpoint.
	// Example: "localhost:4317" (OTLP), "localhost:6831" (Jaeger)
	Endpoint string `yaml:"endpoint"`

	// ServiceName is the service name in traces.
	// Default: "mercator-jupiter"
	ServiceName string `yaml:"service_name"`

	// OTLP contains OTLP exporter specific configuration.
	OTLP OTLPConfig `yaml:"otlp"`

	// Jaeger contains Jaeger exporter specific configuration.
	Jaeger JaegerConfig `yaml:"jaeger"`
}

// OTLPConfig contains OTLP exporter configuration.
type OTLPConfig struct {
	// Insecure disables TLS for OTLP connection.
	// Default: true
	Insecure bool `yaml:"insecure"`

	// Timeout is the timeout for OTLP exports.
	// Default: 10s
	Timeout time.Duration `yaml:"timeout"`
}

// JaegerConfig contains Jaeger exporter configuration.
type JaegerConfig struct {
	// AgentHost is the Jaeger agent hostname.
	// Default: "localhost"
	AgentHost string `yaml:"agent_host"`

	// AgentPort is the Jaeger agent port.
	// Default: 6831
	AgentPort int `yaml:"agent_port"`
}

// HealthConfig contains health check endpoint configuration.
type HealthConfig struct {
	// Enabled controls whether health check endpoints are enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// LivenessPath is the path for the liveness probe endpoint.
	// Default: "/health"
	LivenessPath string `yaml:"liveness_path"`

	// ReadinessPath is the path for the readiness probe endpoint.
	// Default: "/ready"
	ReadinessPath string `yaml:"readiness_path"`

	// VersionPath is the path for the version information endpoint.
	// Default: "/version"
	VersionPath string `yaml:"version_path"`

	// CheckTimeout is the timeout for individual component health checks.
	// Default: 5s
	CheckTimeout time.Duration `yaml:"check_timeout"`

	// MinHealthyProviders is the minimum number of healthy providers required
	// for the system to be considered ready.
	// Default: 1
	MinHealthyProviders int `yaml:"min_healthy_providers"`
}

// ProcessingConfig contains configuration for request/response processing.
type ProcessingConfig struct {
	// Tokens contains token estimation configuration.
	Tokens TokensConfig `yaml:"tokens"`

	// Costs contains cost calculation configuration.
	Costs CostsConfig `yaml:"costs"`

	// Content contains content analysis configuration.
	Content ContentConfig `yaml:"content"`

	// Conversation contains conversation analysis configuration.
	Conversation ConversationConfig `yaml:"conversation"`
}

// TokensConfig contains token estimation configuration.
type TokensConfig struct {
	// Estimator is the token estimator type (simple, tiktoken).
	// Default: "simple"
	Estimator string `yaml:"estimator"`

	// CacheSize is the cache size for tokenizers.
	// Default: 100
	CacheSize int `yaml:"cache_size"`

	// Models contains model-specific characters-per-token ratios.
	Models map[string]float64 `yaml:"models"`
}

// CostsConfig contains cost calculation configuration.
type CostsConfig struct {
	// Pricing contains model pricing configurations by provider.
	Pricing map[string]map[string]ModelPricingConfig `yaml:"pricing"`
}

// ModelPricingConfig contains pricing for a specific model.
type ModelPricingConfig struct {
	// Prompt is the cost per 1K prompt tokens in USD.
	Prompt float64 `yaml:"prompt"`

	// Completion is the cost per 1K completion tokens in USD.
	Completion float64 `yaml:"completion"`

	// CachedPrompt is the cost per 1K cached prompt tokens in USD (optional).
	CachedPrompt float64 `yaml:"cached_prompt,omitempty"`
}

// ContentConfig contains content analysis configuration.
type ContentConfig struct {
	// PII contains PII detection configuration.
	PII PIIConfig `yaml:"pii"`

	// Sensitive contains sensitive content detection configuration.
	Sensitive SensitiveConfig `yaml:"sensitive"`

	// Injection contains prompt injection detection configuration.
	Injection InjectionConfig `yaml:"injection"`
}

// PIIConfig contains PII detection configuration.
type PIIConfig struct {
	// Enabled controls whether PII detection is active.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Types is a list of PII types to detect (email, phone, ssn, credit_card, etc.).
	Types []string `yaml:"types"`

	// RedactInLogs controls whether PII should be redacted from logs.
	// Default: true
	RedactInLogs bool `yaml:"redact_in_logs"`
}

// SensitiveConfig contains sensitive content detection configuration.
type SensitiveConfig struct {
	// Enabled controls whether sensitive content detection is active.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// SeverityThreshold is the minimum severity to report (low, medium, high).
	// Default: "medium"
	SeverityThreshold string `yaml:"severity_threshold"`

	// Categories is a list of sensitive content categories to detect.
	Categories []string `yaml:"categories"`
}

// InjectionConfig contains prompt injection detection configuration.
type InjectionConfig struct {
	// Enabled controls whether prompt injection detection is active.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// ConfidenceThreshold is the minimum confidence to report (0.0 to 1.0).
	// Default: 0.7
	ConfidenceThreshold float64 `yaml:"confidence_threshold"`

	// Patterns is a list of injection patterns to detect.
	Patterns []string `yaml:"patterns"`
}

// ConversationConfig contains conversation analysis configuration.
type ConversationConfig struct {
	// MaxContextWindow contains context window limits by model.
	MaxContextWindow map[string]int `yaml:"max_context_window"`

	// WarnThreshold is the percentage of context window usage to trigger warnings.
	// Default: 0.8 (80%)
	WarnThreshold float64 `yaml:"warn_threshold"`
}

// SecurityConfig contains security-related configuration.
type SecurityConfig struct {
	// TLS contains TLS configuration for the proxy server.
	TLS TLSConfig `yaml:"tls"`

	// Secrets contains secret management configuration.
	Secrets SecretsConfig `yaml:"secrets"`

	// Authentication contains API key authentication configuration.
	Authentication AuthenticationConfig `yaml:"authentication"`
}

// TLSConfig contains TLS configuration.
type TLSConfig struct {
	// Enabled controls whether TLS is enabled for the proxy server.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// CertFile is the path to the TLS certificate file.
	// Required when Enabled is true.
	CertFile string `yaml:"cert_file"`

	// KeyFile is the path to the TLS private key file.
	// Required when Enabled is true.
	KeyFile string `yaml:"key_file"`

	// MinVersion is the minimum TLS version to accept.
	// Options: "1.2", "1.3"
	// Default: "1.3"
	MinVersion string `yaml:"min_version"`

	// CipherSuites is a list of enabled TLS cipher suites.
	// If empty, Go's default secure cipher suites are used.
	// Example: ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"]
	CipherSuites []string `yaml:"cipher_suites"`

	// ReloadInterval is how often to check for certificate changes.
	// Certificates are automatically reloaded when changed.
	// Format: "5m", "1h", etc.
	// Default: "5m"
	ReloadInterval string `yaml:"cert_reload_interval"`

	// MTLS contains mutual TLS (client certificate) configuration.
	MTLS MTLSConfig `yaml:"mtls"`
}

// MTLSConfig contains mutual TLS configuration.
type MTLSConfig struct {
	// Enabled controls whether mutual TLS is required.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// ClientCAFile is the path to the CA certificate file for verifying client certificates.
	// Required when Enabled is true.
	ClientCAFile string `yaml:"client_ca_file"`

	// ClientAuthType specifies how to handle client certificates.
	// Options: "require", "request", "verify_if_given"
	// - "require": client certificate required, reject if missing
	// - "request": request client certificate, but allow if missing
	// - "verify_if_given": verify client cert if provided, allow if not
	// Default: "require"
	ClientAuthType string `yaml:"client_auth_type"`

	// VerifyClientCert controls whether to verify client certificates against the CA.
	// Default: true
	VerifyClientCert bool `yaml:"verify_client_cert"`

	// IdentitySource specifies how to extract client identity from the certificate.
	// Options: "subject.CN", "subject.OU", "subject.O", "SAN"
	// - "subject.CN": Common Name from Subject
	// - "subject.OU": Organizational Unit from Subject
	// - "subject.O": Organization from Subject
	// - "SAN": First DNS name from Subject Alternative Names
	// Default: "subject.CN"
	IdentitySource string `yaml:"identity_source"`
}

// SecretsConfig contains secret management configuration.
type SecretsConfig struct {
	// Providers is a list of secret providers to use.
	// Providers are tried in order until one successfully returns a value.
	Providers []SecretProviderConfig `yaml:"providers"`

	// Cache contains secret caching configuration.
	Cache SecretsCacheConfig `yaml:"cache"`
}

// SecretProviderConfig contains configuration for a secret provider.
type SecretProviderConfig struct {
	// Type is the provider type.
	// Options: "env", "file", "aws_kms", "gcp_kms", "vault"
	Type string `yaml:"type"`

	// Enabled controls whether this provider is enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// Prefix is the environment variable prefix (for "env" provider).
	// Example: "MERCATOR_SECRET_"
	Prefix string `yaml:"prefix,omitempty"`

	// Path is the base path for file-based secrets (for "file" provider).
	// Example: "/var/secrets"
	Path string `yaml:"path,omitempty"`

	// Watch enables file watching for auto-reload (for "file" provider).
	// Default: true
	Watch bool `yaml:"watch,omitempty"`

	// Region is the AWS region (for "aws_kms" provider).
	Region string `yaml:"region,omitempty"`

	// KeyID is the KMS key ID or ARN (for "aws_kms" provider).
	KeyID string `yaml:"key_id,omitempty"`

	// Project is the GCP project ID (for "gcp_kms" provider).
	Project string `yaml:"project,omitempty"`

	// Location is the KMS location (for "gcp_kms" provider).
	// Example: "global", "us-east1"
	Location string `yaml:"location,omitempty"`

	// KeyRing is the KMS key ring name (for "gcp_kms" provider).
	KeyRing string `yaml:"keyring,omitempty"`

	// Key is the KMS key name (for "gcp_kms" provider).
	Key string `yaml:"key,omitempty"`

	// Address is the Vault server address (for "vault" provider).
	// Example: "https://vault.example.com:8200"
	Address string `yaml:"address,omitempty"`

	// Token is the Vault authentication token (for "vault" provider).
	Token string `yaml:"token,omitempty"`

	// VaultPath is the secret path prefix in Vault (for "vault" provider).
	// Example: "secret/mercator"
	VaultPath string `yaml:"vault_path,omitempty"`
}

// SecretsCacheConfig contains configuration for secret caching.
type SecretsCacheConfig struct {
	// Enabled controls whether secret caching is enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// TTL is the time-to-live for cached secrets.
	// Format: "5m", "1h", etc.
	// Default: "5m"
	TTL string `yaml:"ttl"`

	// MaxSize is the maximum number of secrets to cache.
	// Default: 1000
	MaxSize int `yaml:"max_size"`
}

// AuthenticationConfig contains API key authentication configuration.
type AuthenticationConfig struct {
	// Enabled controls whether API key authentication is enabled.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// Sources defines where to extract API keys from (headers, query params).
	Sources []APIKeySource `yaml:"sources"`

	// Keys is the list of valid API keys.
	Keys []APIKeyConfig `yaml:"keys"`
}

// APIKeySource defines where to extract API keys from in HTTP requests.
type APIKeySource struct {
	// Type is the source type.
	// Options: "header", "query"
	Type string `yaml:"type"`

	// Name is the header name or query parameter name.
	// Examples: "Authorization", "X-API-Key", "api_key"
	Name string `yaml:"name"`

	// Scheme is the authentication scheme for header-based extraction.
	// Example: "Bearer" (for "Authorization: Bearer <token>")
	// Leave empty for raw value extraction.
	Scheme string `yaml:"scheme,omitempty"`
}

// APIKeyConfig contains configuration for a single API key.
type APIKeyConfig struct {
	// Key is the API key value.
	// Should be cryptographically random (min 32 bytes recommended).
	Key string `yaml:"key"`

	// UserID is the user identifier associated with this key.
	UserID string `yaml:"user_id"`

	// TeamID is the team identifier associated with this key.
	TeamID string `yaml:"team_id,omitempty"`

	// Enabled controls whether this key is enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// RateLimit is the rate limit for this key.
	// Format: "1000/hour", "100/minute", etc.
	// Empty means no rate limit.
	RateLimit string `yaml:"rate_limit,omitempty"`
}

// RoutingConfig contains configuration for the routing engine.
type RoutingConfig struct {
	// Strategy is the default routing strategy.
	// Options: "round-robin", "sticky", "manual", "health-based"
	// Default: "round-robin"
	Strategy string `yaml:"strategy"`

	// Sticky contains sticky routing configuration.
	Sticky StickyConfig `yaml:"sticky"`

	// ModelMapping maps model names to capable provider names.
	// Example: "gpt-4" -> ["openai"]
	ModelMapping map[string][]string `yaml:"model_mapping"`

	// ProviderWeights contains weights for weighted round-robin.
	// Higher weight = more traffic. Default weight: 1
	ProviderWeights map[string]int `yaml:"provider_weights"`

	// Fallback contains fallback configuration.
	Fallback FallbackConfig `yaml:"fallback"`

	// HealthBased contains health-based routing configuration.
	HealthBased HealthBasedConfig `yaml:"health_based"`
}

// StickyConfig contains sticky routing configuration.
type StickyConfig struct {
	// TTL is the time-to-live for sticky routing entries.
	// 0 means no expiry.
	// Default: 1h
	TTL time.Duration `yaml:"ttl"`

	// MaxEntries is the maximum number of sticky routing entries.
	// When exceeded, LRU eviction is used.
	// Default: 10000
	MaxEntries int `yaml:"max_entries"`

	// KeyType specifies which field to use for sticky routing.
	// Options: "user", "api_key", "session", "composite"
	// Default: "user"
	KeyType string `yaml:"key_type"`
}

// FallbackConfig contains fallback routing configuration.
type FallbackConfig struct {
	// Enabled controls whether fallback to next provider is enabled.
	// Default: true
	Enabled bool `yaml:"enabled"`

	// MaxAttempts is the maximum number of fallback attempts.
	// 0 means try all available providers.
	// Default: 2
	MaxAttempts int `yaml:"max_attempts"`

	// DefaultProvider is the default provider when all strategies fail.
	// Optional. If empty, routing will fail.
	DefaultProvider string `yaml:"default_provider"`
}

// HealthBasedConfig contains health-based routing configuration.
type HealthBasedConfig struct {
	// RequireHealthy controls whether only healthy providers are used.
	// Default: true
	RequireHealthy bool `yaml:"require_healthy"`

	// RetryAfter is the duration to wait before retrying an unhealthy provider.
	// This is informational only; health status is controlled by health checker.
	// Default: 5m
	RetryAfter time.Duration `yaml:"retry_after"`
}

// LimitsConfig contains configuration for budget tracking and rate limiting.
type LimitsConfig struct {
	// Budgets contains budget limits for different dimensions.
	Budgets BudgetsConfig `yaml:"budgets"`

	// RateLimits contains rate limits for different dimensions.
	RateLimits RateLimitsConfig `yaml:"rate_limits"`

	// Enforcement configures enforcement actions when limits are exceeded.
	Enforcement EnforcementConfig `yaml:"enforcement"`

	// Storage configures the limits storage backend.
	Storage LimitsStorageConfig `yaml:"storage"`
}

// BudgetsConfig contains budget tracking configuration.
type BudgetsConfig struct {
	// Enabled controls whether budget tracking is enabled.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// AlertThreshold is the percentage (0.0-1.0) at which to trigger alerts.
	// For example, 0.8 means alert when 80% of budget is used.
	// Default: 0.8
	AlertThreshold float64 `yaml:"alert_threshold"`

	// ByAPIKey contains per-API key budget limits.
	ByAPIKey map[string]BudgetLimits `yaml:"by_api_key"`

	// ByUser contains per-user budget limits.
	ByUser map[string]BudgetLimits `yaml:"by_user"`

	// ByTeam contains per-team budget limits.
	ByTeam map[string]BudgetLimits `yaml:"by_team"`
}

// BudgetLimits contains budget limits for different time windows.
type BudgetLimits struct {
	// Hourly is the budget limit for a rolling 60-minute window (USD).
	// 0 means no hourly limit.
	Hourly float64 `yaml:"hourly"`

	// Daily is the budget limit for a rolling 24-hour window (USD).
	// 0 means no daily limit.
	Daily float64 `yaml:"daily"`

	// Monthly is the budget limit for a rolling 30-day window (USD).
	// 0 means no monthly limit.
	Monthly float64 `yaml:"monthly"`
}

// RateLimitsConfig contains rate limiting configuration.
type RateLimitsConfig struct {
	// Enabled controls whether rate limiting is enabled.
	// Default: false
	Enabled bool `yaml:"enabled"`

	// ByAPIKey contains per-API key rate limits.
	ByAPIKey map[string]RateLimits `yaml:"by_api_key"`

	// ByUser contains per-user rate limits.
	ByUser map[string]RateLimits `yaml:"by_user"`

	// ByTeam contains per-team rate limits.
	ByTeam map[string]RateLimits `yaml:"by_team"`
}

// RateLimits contains rate limits for different metrics.
type RateLimits struct {
	// RequestsPerSecond limits requests per second using token bucket.
	// 0 means no limit.
	RequestsPerSecond int `yaml:"requests_per_second"`

	// RequestsPerMinute limits requests per minute using token bucket.
	// 0 means no limit.
	RequestsPerMinute int `yaml:"requests_per_minute"`

	// RequestsPerHour limits requests per hour using token bucket.
	// 0 means no limit.
	RequestsPerHour int `yaml:"requests_per_hour"`

	// TokensPerMinute limits tokens (prompt+completion) per minute.
	// 0 means no limit.
	TokensPerMinute int `yaml:"tokens_per_minute"`

	// TokensPerHour limits tokens per hour.
	// 0 means no limit.
	TokensPerHour int `yaml:"tokens_per_hour"`

	// MaxConcurrent limits simultaneous requests.
	// 0 means no limit.
	MaxConcurrent int `yaml:"max_concurrent"`
}

// EnforcementConfig configures enforcement actions for limit violations.
type EnforcementConfig struct {
	// Action is the default action to take when limits are exceeded.
	// Options: "block", "queue", "downgrade", "alert"
	// Default: "block"
	Action string `yaml:"action"`

	// QueueDepth is the maximum number of requests to queue (when action=queue).
	// Default: 100
	QueueDepth int `yaml:"queue_depth"`

	// QueueTimeout is how long to wait for queue capacity before giving up.
	// Default: 30s
	QueueTimeout time.Duration `yaml:"queue_timeout"`

	// ModelDowngrades maps expensive models to cheaper alternatives.
	// Used when action=downgrade.
	// Example: "gpt-4" -> "gpt-3.5-turbo"
	ModelDowngrades map[string]string `yaml:"model_downgrades"`
}

// LimitsStorageConfig configures the limits storage backend.
type LimitsStorageConfig struct {
	// Backend specifies the storage backend to use.
	// Options: "memory", "sqlite"
	// Default: "memory"
	Backend string `yaml:"backend"`

	// SQLite contains SQLite-specific configuration.
	SQLite LimitsSQLiteConfig `yaml:"sqlite"`

	// Memory contains memory backend configuration.
	Memory LimitsMemoryConfig `yaml:"memory"`
}

// LimitsSQLiteConfig contains SQLite storage configuration.
type LimitsSQLiteConfig struct {
	// Path is the path to the SQLite database file.
	// Default: "/var/lib/mercator/limits.db"
	Path string `yaml:"path"`

	// SnapshotInterval is how often to checkpoint the WAL.
	// Default: 5m
	SnapshotInterval time.Duration `yaml:"snapshot_interval"`
}

// LimitsMemoryConfig contains memory backend configuration.
type LimitsMemoryConfig struct {
	// MaxEntries is the maximum number of state entries to store.
	// Oldest entries are evicted when this limit is reached (LRU).
	// Default: 100000
	MaxEntries int `yaml:"max_entries"`

	// CleanupInterval is how often to cleanup expired entries.
	// Default: 1m
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}
