package config

import "time"

// Default values for configuration fields.
const (
	// Proxy defaults
	DefaultListenAddress   = "127.0.0.1:8080"
	DefaultReadTimeout     = 30 * time.Second
	DefaultWriteTimeout    = 30 * time.Second
	DefaultIdleTimeout     = 120 * time.Second
	DefaultShutdownTimeout = 30 * time.Second
	DefaultMaxHeaderBytes  = 1048576 // 1MB

	// CORS defaults
	DefaultCORSEnabled          = true
	DefaultCORSMaxAge           = 3600 // 1 hour
	DefaultCORSAllowCredentials = false

	// Provider defaults
	DefaultProviderTimeout    = 60 * time.Second
	DefaultProviderMaxRetries = 3

	// Policy defaults
	DefaultPolicyMode              = "file"
	DefaultPolicyFilePath          = "./policies.yaml"
	DefaultPolicyGitBranch         = "main"
	DefaultPolicyGitPath           = "policies.yaml"
	DefaultPolicyWatch             = false
	DefaultPolicyValidationEnabled = true
	DefaultPolicyValidationStrict  = false

	// Evidence defaults
	DefaultEvidenceEnabled              = true
	DefaultEvidenceBackend              = "sqlite"
	DefaultEvidenceSQLitePath           = "data/evidence.db"
	DefaultEvidenceSQLiteMaxOpenConns   = 10
	DefaultEvidenceSQLiteMaxIdleConns   = 5
	DefaultEvidenceSQLiteWALMode        = true
	DefaultEvidenceSQLiteBusyTimeout    = 5 * time.Second
	DefaultEvidenceRecorderAsyncBuffer  = 1000
	DefaultEvidenceRecorderWriteTimeout = 5 * time.Second
	DefaultEvidenceRecorderHashRequest  = true
	DefaultEvidenceRecorderHashResponse = true
	DefaultEvidenceRecorderRedactKeys   = true
	DefaultEvidenceRecorderMaxFieldLen  = 500
	DefaultEvidenceRetentionDays        = 90
	DefaultEvidenceRetentionSchedule    = "0 3 * * *"
	DefaultEvidenceRetentionArchive     = false
	DefaultEvidenceRetentionArchivePath = "data/archives/"
	DefaultEvidenceRetentionMaxRecords  = int64(0)
	DefaultEvidenceQueryDefaultLimit    = 100
	DefaultEvidenceQueryMaxLimit        = 10000
	DefaultEvidenceQueryTimeout         = 30 * time.Second
	DefaultEvidenceExportJSONPretty     = true
	DefaultEvidenceExportCSVHeader      = true
	DefaultEvidenceExportMaxSize        = 1000000
	DefaultPostgresPort                 = 5432
	DefaultPostgresSSLMode              = "require"

	// Telemetry defaults
	DefaultLoggingLevel        = "info"
	DefaultLoggingFormat       = "json"
	DefaultMetricsEnabled      = true
	DefaultPrometheusPath      = "/metrics"
	DefaultTracingEnabled      = false
	DefaultTracingSamplingRate = 1.0

	// Security defaults
	DefaultTLSEnabled  = false
	DefaultMTLSEnabled = false

	// Processing defaults
	DefaultTokensEstimator              = "simple"
	DefaultTokensCacheSize              = 100
	DefaultTokensCharsPerToken          = 4.0
	DefaultCostsPricing                 = 0.001 // $0.001 per 1K tokens
	DefaultContentPIIEnabled            = true
	DefaultContentPIIRedactInLogs       = true
	DefaultContentSensitiveEnabled      = true
	DefaultContentSensitiveSeverity     = "medium"
	DefaultContentInjectionEnabled      = true
	DefaultContentInjectionConfidence   = 0.7
	DefaultConversationWarnThreshold    = 0.8
	DefaultConversationContextWindow    = 4096
)

// ApplyDefaults applies default values to a Config struct.
// It sets defaults for any fields that have zero values.
// This function is idempotent and safe to call multiple times.
func ApplyDefaults(cfg *Config) {
	// Proxy defaults
	if cfg.Proxy.ListenAddress == "" {
		cfg.Proxy.ListenAddress = DefaultListenAddress
	}
	if cfg.Proxy.ReadTimeout == 0 {
		cfg.Proxy.ReadTimeout = DefaultReadTimeout
	}
	if cfg.Proxy.WriteTimeout == 0 {
		cfg.Proxy.WriteTimeout = DefaultWriteTimeout
	}
	if cfg.Proxy.IdleTimeout == 0 {
		cfg.Proxy.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.Proxy.MaxHeaderBytes == 0 {
		cfg.Proxy.MaxHeaderBytes = DefaultMaxHeaderBytes
	}

	// Provider defaults - applied to each provider
	for name, provider := range cfg.Providers {
		if provider.Timeout == 0 {
			provider.Timeout = DefaultProviderTimeout
		}
		if provider.MaxRetries == 0 {
			provider.MaxRetries = DefaultProviderMaxRetries
		}
		// Update the provider in the map
		cfg.Providers[name] = provider
	}

	// Policy defaults
	if cfg.Policy.Mode == "" {
		cfg.Policy.Mode = DefaultPolicyMode
	}
	if cfg.Policy.FilePath == "" {
		cfg.Policy.FilePath = DefaultPolicyFilePath
	}
	if cfg.Policy.GitBranch == "" {
		cfg.Policy.GitBranch = DefaultPolicyGitBranch
	}
	if cfg.Policy.GitPath == "" {
		cfg.Policy.GitPath = DefaultPolicyGitPath
	}
	// Watch defaults to false (zero value), use explicit default
	if !cfg.Policy.Watch {
		cfg.Policy.Watch = DefaultPolicyWatch
	}
	// Validation defaults - need to check if struct was set at all
	// Since bools have zero value false, we apply defaults unconditionally
	// unless the config explicitly sets them
	applyPolicyValidationDefaults(cfg)

	// Evidence defaults
	if !cfg.Evidence.Enabled {
		// Evidence enabled defaults to true, so we need special handling
		// We'll apply this in validation instead to distinguish between
		// "not set" and "explicitly set to false"
	}
	if cfg.Evidence.Backend == "" {
		cfg.Evidence.Backend = DefaultEvidenceBackend
	}

	// SQLite defaults
	if cfg.Evidence.SQLite.Path == "" {
		cfg.Evidence.SQLite.Path = DefaultEvidenceSQLitePath
	}
	if cfg.Evidence.SQLite.MaxOpenConns == 0 {
		cfg.Evidence.SQLite.MaxOpenConns = DefaultEvidenceSQLiteMaxOpenConns
	}
	if cfg.Evidence.SQLite.MaxIdleConns == 0 {
		cfg.Evidence.SQLite.MaxIdleConns = DefaultEvidenceSQLiteMaxIdleConns
	}
	if !cfg.Evidence.SQLite.WALMode {
		cfg.Evidence.SQLite.WALMode = DefaultEvidenceSQLiteWALMode
	}
	if cfg.Evidence.SQLite.BusyTimeout == 0 {
		cfg.Evidence.SQLite.BusyTimeout = DefaultEvidenceSQLiteBusyTimeout
	}

	// Recorder defaults
	if cfg.Evidence.Recorder.AsyncBuffer == 0 {
		cfg.Evidence.Recorder.AsyncBuffer = DefaultEvidenceRecorderAsyncBuffer
	}
	if cfg.Evidence.Recorder.WriteTimeout == 0 {
		cfg.Evidence.Recorder.WriteTimeout = DefaultEvidenceRecorderWriteTimeout
	}
	if !cfg.Evidence.Recorder.HashRequest {
		cfg.Evidence.Recorder.HashRequest = DefaultEvidenceRecorderHashRequest
	}
	if !cfg.Evidence.Recorder.HashResponse {
		cfg.Evidence.Recorder.HashResponse = DefaultEvidenceRecorderHashResponse
	}
	if !cfg.Evidence.Recorder.RedactAPIKeys {
		cfg.Evidence.Recorder.RedactAPIKeys = DefaultEvidenceRecorderRedactKeys
	}
	if cfg.Evidence.Recorder.MaxFieldLength == 0 {
		cfg.Evidence.Recorder.MaxFieldLength = DefaultEvidenceRecorderMaxFieldLen
	}

	// Retention defaults
	if cfg.Evidence.Retention.Days == 0 {
		cfg.Evidence.Retention.Days = DefaultEvidenceRetentionDays
	}
	if cfg.Evidence.Retention.PruneSchedule == "" {
		cfg.Evidence.Retention.PruneSchedule = DefaultEvidenceRetentionSchedule
	}
	if !cfg.Evidence.Retention.ArchiveBeforeDelete {
		cfg.Evidence.Retention.ArchiveBeforeDelete = DefaultEvidenceRetentionArchive
	}
	if cfg.Evidence.Retention.ArchivePath == "" {
		cfg.Evidence.Retention.ArchivePath = DefaultEvidenceRetentionArchivePath
	}
	if cfg.Evidence.Retention.MaxRecords == 0 {
		cfg.Evidence.Retention.MaxRecords = DefaultEvidenceRetentionMaxRecords
	}

	// Query defaults
	if cfg.Evidence.Query.DefaultLimit == 0 {
		cfg.Evidence.Query.DefaultLimit = DefaultEvidenceQueryDefaultLimit
	}
	if cfg.Evidence.Query.MaxLimit == 0 {
		cfg.Evidence.Query.MaxLimit = DefaultEvidenceQueryMaxLimit
	}
	if cfg.Evidence.Query.Timeout == 0 {
		cfg.Evidence.Query.Timeout = DefaultEvidenceQueryTimeout
	}

	// Export defaults
	if !cfg.Evidence.Export.JSONPretty {
		cfg.Evidence.Export.JSONPretty = DefaultEvidenceExportJSONPretty
	}
	if !cfg.Evidence.Export.CSVIncludeHeader {
		cfg.Evidence.Export.CSVIncludeHeader = DefaultEvidenceExportCSVHeader
	}
	if cfg.Evidence.Export.MaxExportSize == 0 {
		cfg.Evidence.Export.MaxExportSize = DefaultEvidenceExportMaxSize
	}

	// Postgres defaults
	if cfg.Evidence.Postgres.Port == 0 {
		cfg.Evidence.Postgres.Port = DefaultPostgresPort
	}
	if cfg.Evidence.Postgres.SSLMode == "" {
		cfg.Evidence.Postgres.SSLMode = DefaultPostgresSSLMode
	}

	// Telemetry defaults
	if cfg.Telemetry.Logging.Level == "" {
		cfg.Telemetry.Logging.Level = DefaultLoggingLevel
	}
	if cfg.Telemetry.Logging.Format == "" {
		cfg.Telemetry.Logging.Format = DefaultLoggingFormat
	}
	if cfg.Telemetry.Metrics.Path == "" {
		cfg.Telemetry.Metrics.Path = DefaultPrometheusPath
	}
	if cfg.Telemetry.Tracing.SampleRatio == 0 {
		cfg.Telemetry.Tracing.SampleRatio = DefaultTracingSamplingRate
	}

	// Proxy shutdown timeout
	if cfg.Proxy.ShutdownTimeout == 0 {
		cfg.Proxy.ShutdownTimeout = DefaultShutdownTimeout
	}

	// CORS defaults
	applyCORSDefaults(cfg)

	// Processing defaults
	applyProcessingDefaults(cfg)

	// Security defaults are false (zero values), which is correct
}

// applyCORSDefaults applies default values to CORS configuration.
func applyCORSDefaults(cfg *Config) {
	cors := &cfg.Proxy.CORS

	// Set enabled default (true)
	if !cors.Enabled {
		// Check if any CORS fields are set - if so, user wants CORS
		// Otherwise, use default
		hasAnyConfig := len(cors.AllowedOrigins) > 0 ||
			len(cors.AllowedMethods) > 0 ||
			len(cors.AllowedHeaders) > 0 ||
			len(cors.ExposedHeaders) > 0 ||
			cors.MaxAge > 0

		if !hasAnyConfig {
			cors.Enabled = DefaultCORSEnabled
		}
	}

	// Set allowed origins default
	if len(cors.AllowedOrigins) == 0 {
		cors.AllowedOrigins = []string{"*"}
	}

	// Set allowed methods default
	if len(cors.AllowedMethods) == 0 {
		cors.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}

	// Set allowed headers default
	if len(cors.AllowedHeaders) == 0 {
		cors.AllowedHeaders = []string{"Authorization", "Content-Type", "X-Request-ID", "X-User-ID"}
	}

	// Set exposed headers default
	if len(cors.ExposedHeaders) == 0 {
		cors.ExposedHeaders = []string{"X-Request-ID"}
	}

	// Set max age default
	if cors.MaxAge == 0 {
		cors.MaxAge = DefaultCORSMaxAge
	}

	// AllowCredentials defaults to false (zero value), which is correct
}

// applyPolicyValidationDefaults applies defaults to policy validation config.
// This is separated because boolean defaults are tricky with zero values.
func applyPolicyValidationDefaults(cfg *Config) {
	// For validation.enabled, we want default to be true
	// But if someone explicitly sets it to false, we should respect that
	// For now, we'll handle this in validation or use a different approach
	// For simplicity in MVP, we'll just document the defaults
	// and let validation handle required vs optional logic
}

// applyProcessingDefaults applies default values to processing configuration.
func applyProcessingDefaults(cfg *Config) {
	// Tokens defaults
	if cfg.Processing.Tokens.Estimator == "" {
		cfg.Processing.Tokens.Estimator = DefaultTokensEstimator
	}
	if cfg.Processing.Tokens.CacheSize == 0 {
		cfg.Processing.Tokens.CacheSize = DefaultTokensCacheSize
	}
	if cfg.Processing.Tokens.Models == nil {
		cfg.Processing.Tokens.Models = map[string]float64{
			"gpt-4":          4.0,
			"gpt-3.5-turbo":  4.0,
			"claude-3-opus":  3.5,
			"claude-3-sonnet": 3.5,
			"claude-3-haiku": 3.5,
			"default":        DefaultTokensCharsPerToken,
		}
	}

	// Costs defaults
	if cfg.Processing.Costs.Pricing == nil {
		cfg.Processing.Costs.Pricing = map[string]map[string]ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:     0.03,
					Completion: 0.06,
				},
				"gpt-4-turbo": {
					Prompt:     0.01,
					Completion: 0.03,
				},
				"gpt-3.5-turbo": {
					Prompt:     0.0005,
					Completion: 0.0015,
				},
			},
			"anthropic": {
				"claude-3-opus": {
					Prompt:     0.015,
					Completion: 0.075,
				},
				"claude-3-sonnet": {
					Prompt:     0.003,
					Completion: 0.015,
				},
				"claude-3-haiku": {
					Prompt:     0.00025,
					Completion: 0.00125,
				},
			},
			"default": {
				"default": {
					Prompt:     DefaultCostsPricing,
					Completion: DefaultCostsPricing * 2,
				},
			},
		}
	}

	// Content PII defaults
	if len(cfg.Processing.Content.PII.Types) == 0 {
		cfg.Processing.Content.PII.Types = []string{
			"email",
			"phone",
			"ssn",
			"credit_card",
			"ip_address",
		}
	}

	// Content sensitive defaults
	if cfg.Processing.Content.Sensitive.SeverityThreshold == "" {
		cfg.Processing.Content.Sensitive.SeverityThreshold = DefaultContentSensitiveSeverity
	}
	if len(cfg.Processing.Content.Sensitive.Categories) == 0 {
		cfg.Processing.Content.Sensitive.Categories = []string{
			"profanity",
			"violence",
			"hate_speech",
			"adult_content",
		}
	}

	// Content injection defaults
	if cfg.Processing.Content.Injection.ConfidenceThreshold == 0 {
		cfg.Processing.Content.Injection.ConfidenceThreshold = DefaultContentInjectionConfidence
	}
	if len(cfg.Processing.Content.Injection.Patterns) == 0 {
		cfg.Processing.Content.Injection.Patterns = []string{
			"ignore previous instructions",
			"disregard system prompt",
			"you are now",
			"new instructions",
			"forget everything",
		}
	}

	// Conversation defaults
	if cfg.Processing.Conversation.WarnThreshold == 0 {
		cfg.Processing.Conversation.WarnThreshold = DefaultConversationWarnThreshold
	}
	if cfg.Processing.Conversation.MaxContextWindow == nil {
		cfg.Processing.Conversation.MaxContextWindow = map[string]int{
			"gpt-4":          8192,
			"gpt-4-turbo":    128000,
			"gpt-3.5-turbo":  4096,
			"claude-3-opus":  200000,
			"claude-3-sonnet": 200000,
			"claude-3-haiku": 200000,
			"default":        DefaultConversationContextWindow,
		}
	}

	// Limits defaults
	if cfg.Limits.Budgets.AlertThreshold == 0 {
		cfg.Limits.Budgets.AlertThreshold = 0.8 // 80%
	}
	if cfg.Limits.Enforcement.Action == "" {
		cfg.Limits.Enforcement.Action = "block"
	}
	if cfg.Limits.Enforcement.QueueDepth == 0 {
		cfg.Limits.Enforcement.QueueDepth = 100
	}
	if cfg.Limits.Enforcement.QueueTimeout == 0 {
		cfg.Limits.Enforcement.QueueTimeout = 30 * time.Second
	}
	if cfg.Limits.Storage.Backend == "" {
		cfg.Limits.Storage.Backend = "memory"
	}
	if cfg.Limits.Storage.SQLite.Path == "" {
		cfg.Limits.Storage.SQLite.Path = "/var/lib/mercator/limits.db"
	}
	if cfg.Limits.Storage.SQLite.SnapshotInterval == 0 {
		cfg.Limits.Storage.SQLite.SnapshotInterval = 5 * time.Minute
	}
	if cfg.Limits.Storage.Memory.MaxEntries == 0 {
		cfg.Limits.Storage.Memory.MaxEntries = 100000
	}
	if cfg.Limits.Storage.Memory.CleanupInterval == 0 {
		cfg.Limits.Storage.Memory.CleanupInterval = time.Minute
	}
}
