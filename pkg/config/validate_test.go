package config

import (
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := MinimalConfig()

	err := Validate(cfg)
	if err != nil {
		t.Errorf("expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		// No proxy config (empty listen address)
		Providers: map[string]ProviderConfig{},
		// No providers (should fail)
		// Empty telemetry logging level
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation to fail")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if len(validationErr.Errors) < 2 {
		t.Errorf("expected multiple errors, got %d", len(validationErr.Errors))
	}

	// Verify error message includes multiple errors
	errMsg := validationErr.Error()
	if !strings.Contains(errMsg, "validation failed with") {
		t.Errorf("error message should mention multiple errors: %s", errMsg)
	}
}

func TestValidate_ProxyConfig(t *testing.T) {
	tests := []struct {
		name       string
		proxy      ProxyConfig
		wantError  bool
		errorField string
	}{
		{
			name: "valid proxy config",
			proxy: ProxyConfig{
				ListenAddress:  "127.0.0.1:8080",
				ReadTimeout:    DefaultReadTimeout,
				WriteTimeout:   DefaultWriteTimeout,
				IdleTimeout:    DefaultIdleTimeout,
				MaxHeaderBytes: DefaultMaxHeaderBytes,
			},
			wantError: false,
		},
		{
			name: "empty listen address",
			proxy: ProxyConfig{
				ListenAddress: "",
			},
			wantError:  true,
			errorField: "proxy.listen_address",
		},
		{
			name: "negative read timeout",
			proxy: ProxyConfig{
				ListenAddress: "127.0.0.1:8080",
				ReadTimeout:   -1,
			},
			wantError:  true,
			errorField: "proxy.read_timeout",
		},
		{
			name: "excessive max header bytes",
			proxy: ProxyConfig{
				ListenAddress:  "127.0.0.1:8080",
				MaxHeaderBytes: 20 * 1024 * 1024, // 20MB
			},
			wantError:  true,
			errorField: "proxy.max_header_bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateProxy(&tt.proxy)
			if tt.wantError && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
			if tt.wantError && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestValidate_Providers(t *testing.T) {
	tests := []struct {
		name       string
		providers  map[string]ProviderConfig
		wantError  bool
		errorField string
	}{
		{
			name: "valid provider",
			providers: map[string]ProviderConfig{
				"openai": {
					BaseURL:    "https://api.openai.com/v1",
					APIKey:     "test-key",
					Timeout:    DefaultProviderTimeout,
					MaxRetries: DefaultProviderMaxRetries,
				},
			},
			wantError: false,
		},
		{
			name:       "no providers",
			providers:  map[string]ProviderConfig{},
			wantError:  true,
			errorField: "providers",
		},
		{
			name: "missing base URL",
			providers: map[string]ProviderConfig{
				"openai": {
					APIKey: "test-key",
				},
			},
			wantError:  true,
			errorField: "providers.openai.base_url",
		},
		{
			name: "invalid URL",
			providers: map[string]ProviderConfig{
				"openai": {
					BaseURL: "not a valid url ://",
					APIKey:  "test-key",
				},
			},
			wantError:  true,
			errorField: "providers.openai.base_url",
		},
		{
			name: "negative timeout",
			providers: map[string]ProviderConfig{
				"openai": {
					BaseURL: "https://api.openai.com/v1",
					Timeout: -1,
				},
			},
			wantError:  true,
			errorField: "providers.openai.timeout",
		},
		{
			name: "excessive retries",
			providers: map[string]ProviderConfig{
				"openai": {
					BaseURL:    "https://api.openai.com/v1",
					MaxRetries: 100,
				},
			},
			wantError:  true,
			errorField: "providers.openai.max_retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateProviders(tt.providers)
			if tt.wantError && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
			if tt.wantError && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestValidate_Policy(t *testing.T) {
	tests := []struct {
		name       string
		policy     PolicyConfig
		wantError  bool
		errorField string
	}{
		{
			name: "valid file mode",
			policy: PolicyConfig{
				Mode:     "file",
				FilePath: "./policies.yaml",
			},
			wantError: false,
		},
		{
			name: "valid git mode",
			policy: PolicyConfig{
				Mode:      "git",
				GitRepo:   "https://github.com/example/policies",
				GitBranch: "main",
				GitPath:   "policies.yaml",
			},
			wantError: false,
		},
		{
			name:       "invalid mode",
			policy:     PolicyConfig{Mode: "invalid"},
			wantError:  true,
			errorField: "policy.mode",
		},
		{
			name: "file mode missing path",
			policy: PolicyConfig{
				Mode:     "file",
				FilePath: "",
			},
			wantError:  true,
			errorField: "policy.file_path",
		},
		{
			name: "git mode missing repo",
			policy: PolicyConfig{
				Mode:      "git",
				GitBranch: "main",
				GitPath:   "policies.yaml",
			},
			wantError:  true,
			errorField: "policy.git_repo",
		},
		{
			name: "git mode missing branch",
			policy: PolicyConfig{
				Mode:    "git",
				GitRepo: "https://github.com/example/policies",
				GitPath: "policies.yaml",
			},
			wantError:  true,
			errorField: "policy.git_branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validatePolicy(&tt.policy)
			if tt.wantError && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
			if tt.wantError && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestValidate_Evidence(t *testing.T) {
	tests := []struct {
		name       string
		evidence   EvidenceConfig
		wantError  bool
		errorField string
	}{
		{
			name: "valid sqlite backend",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "sqlite",
				SQLite:  SQLiteConfig{Path: "./evidence.db"},
			},
			wantError: false,
		},
		{
			name: "valid postgres backend",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "postgres",
				Postgres: PostgresConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "mercator",
					User:     "user",
					SSLMode:  "require",
				},
			},
			wantError: false,
		},
		{
			name: "valid s3 backend",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "s3",
				S3:      S3Config{Bucket: "my-bucket", Region: "us-east-1"},
			},
			wantError: false,
		},
		{
			name: "disabled evidence skips validation",
			evidence: EvidenceConfig{
				Enabled: false,
				// Missing backend - should not fail
			},
			wantError: false,
		},
		{
			name: "invalid backend",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "invalid",
			},
			wantError:  true,
			errorField: "evidence.backend",
		},
		{
			name: "postgres missing host",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "postgres",
				Postgres: PostgresConfig{
					Port:     5432,
					Database: "mercator",
					User:     "user",
				},
			},
			wantError:  true,
			errorField: "evidence.postgres.host",
		},
		{
			name: "postgres invalid port",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "postgres",
				Postgres: PostgresConfig{
					Host:     "localhost",
					Port:     99999,
					Database: "mercator",
					User:     "user",
				},
			},
			wantError:  true,
			errorField: "evidence.postgres.port",
		},
		{
			name: "s3 missing bucket",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "s3",
				S3:      S3Config{Region: "us-east-1"},
			},
			wantError:  true,
			errorField: "evidence.s3.bucket",
		},
		{
			name: "excessive retention days",
			evidence: EvidenceConfig{
				Enabled: true,
				Backend: "sqlite",
				SQLite:  SQLiteConfig{Path: "./evidence.db"},
				Retention: RetentionConfig{
					Days: 5000,
				},
			},
			wantError:  true,
			errorField: "evidence.retention.days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateEvidence(&tt.evidence)
			if tt.wantError && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
			if tt.wantError && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestValidate_Telemetry(t *testing.T) {
	tests := []struct {
		name       string
		telemetry  TelemetryConfig
		wantError  bool
		errorField string
	}{
		{
			name: "valid telemetry config",
			telemetry: TelemetryConfig{
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Metrics: MetricsConfig{Enabled: true, Path: "/metrics"},
				Tracing: TracingConfig{Enabled: false},
			},
			wantError: false,
		},
		{
			name: "invalid logging level",
			telemetry: TelemetryConfig{
				Logging: LoggingConfig{Level: "invalid", Format: "json"},
			},
			wantError:  true,
			errorField: "telemetry.logging.level",
		},
		{
			name: "invalid logging format",
			telemetry: TelemetryConfig{
				Logging: LoggingConfig{Level: "info", Format: "invalid"},
			},
			wantError:  true,
			errorField: "telemetry.logging.format",
		},
		{
			name: "metrics enabled without path",
			telemetry: TelemetryConfig{
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Metrics: MetricsConfig{Enabled: true, Path: ""},
			},
			wantError:  true,
			errorField: "telemetry.metrics.path",
		},
		{
			name: "tracing enabled without endpoint",
			telemetry: TelemetryConfig{
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Tracing: TracingConfig{Enabled: true, Endpoint: ""},
			},
			wantError:  true,
			errorField: "telemetry.tracing.endpoint",
		},
		{
			name: "invalid sampling rate",
			telemetry: TelemetryConfig{
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Tracing: TracingConfig{SampleRatio: 1.5},
			},
			wantError:  true,
			errorField: "telemetry.tracing.sample_ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateTelemetry(&tt.telemetry)
			if tt.wantError && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
			if tt.wantError && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestValidate_Security(t *testing.T) {
	tests := []struct {
		name       string
		security   SecurityConfig
		wantError  bool
		errorField string
	}{
		{
			name: "tls disabled",
			security: SecurityConfig{
				TLS: TLSConfig{Enabled: false},
			},
			wantError: false,
		},
		{
			name: "valid tls config",
			security: SecurityConfig{
				TLS: TLSConfig{
					Enabled:  true,
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
				},
			},
			wantError: false,
		},
		{
			name: "tls enabled without cert",
			security: SecurityConfig{
				TLS: TLSConfig{
					Enabled: true,
					KeyFile: "/path/to/key.pem",
				},
			},
			wantError:  true,
			errorField: "security.tls.cert_file",
		},
		{
			name: "tls enabled without key",
			security: SecurityConfig{
				TLS: TLSConfig{
					Enabled:  true,
					CertFile: "/path/to/cert.pem",
				},
			},
			wantError:  true,
			errorField: "security.tls.key_file",
		},
		{
			name: "mtls enabled without ca",
			security: SecurityConfig{
				TLS: TLSConfig{
					Enabled:  true,
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
					MTLS:     MTLSConfig{Enabled: true},
				},
			},
			wantError:  true,
			errorField: "security.tls.mtls.client_ca_file",
		},
		{
			name: "mtls without tls",
			security: SecurityConfig{
				TLS: TLSConfig{
					Enabled: false,
					MTLS:    MTLSConfig{Enabled: true, ClientCAFile: "/path/to/ca.pem"},
				},
			},
			wantError:  true,
			errorField: "security.tls.mtls.enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateSecurity(&tt.security)
			if tt.wantError && len(errs) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantError && len(errs) > 0 {
				t.Errorf("expected no validation error, got: %v", errs)
			}
			if tt.wantError && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if err.Field == tt.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", tt.errorField, errs)
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		contains string
	}{
		{
			name:     "empty errors",
			err:      ValidationError{Errors: []FieldError{}},
			contains: "configuration validation failed",
		},
		{
			name: "single error",
			err: ValidationError{
				Errors: []FieldError{
					{Field: "proxy.listen_address", Message: "required"},
				},
			},
			contains: "proxy.listen_address",
		},
		{
			name: "multiple errors",
			err: ValidationError{
				Errors: []FieldError{
					{Field: "proxy.listen_address", Message: "required"},
					{Field: "providers", Message: "at least one required"},
				},
			},
			contains: "2 errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			if !strings.Contains(errMsg, tt.contains) {
				t.Errorf("expected error message to contain %q, got: %s", tt.contains, errMsg)
			}
		})
	}
}
