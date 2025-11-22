package config

import (
	"testing"
	"time"
)

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name  string
		input Config
		check func(*testing.T, *Config)
	}{
		{
			name:  "empty config gets all defaults",
			input: Config{Providers: make(map[string]ProviderConfig)},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Proxy.ListenAddress != DefaultListenAddress {
					t.Errorf("expected listen address %q, got %q", DefaultListenAddress, cfg.Proxy.ListenAddress)
				}
				if cfg.Proxy.ReadTimeout != DefaultReadTimeout {
					t.Errorf("expected read timeout %v, got %v", DefaultReadTimeout, cfg.Proxy.ReadTimeout)
				}
				if cfg.Proxy.WriteTimeout != DefaultWriteTimeout {
					t.Errorf("expected write timeout %v, got %v", DefaultWriteTimeout, cfg.Proxy.WriteTimeout)
				}
				if cfg.Proxy.IdleTimeout != DefaultIdleTimeout {
					t.Errorf("expected idle timeout %v, got %v", DefaultIdleTimeout, cfg.Proxy.IdleTimeout)
				}
				if cfg.Proxy.MaxHeaderBytes != DefaultMaxHeaderBytes {
					t.Errorf("expected max header bytes %d, got %d", DefaultMaxHeaderBytes, cfg.Proxy.MaxHeaderBytes)
				}
				if cfg.Policy.Mode != DefaultPolicyMode {
					t.Errorf("expected policy mode %q, got %q", DefaultPolicyMode, cfg.Policy.Mode)
				}
				if cfg.Policy.FilePath != DefaultPolicyFilePath {
					t.Errorf("expected policy file path %q, got %q", DefaultPolicyFilePath, cfg.Policy.FilePath)
				}
				if cfg.Evidence.Backend != DefaultEvidenceBackend {
					t.Errorf("expected evidence backend %q, got %q", DefaultEvidenceBackend, cfg.Evidence.Backend)
				}
				if cfg.Evidence.SQLite.Path != DefaultEvidenceSQLitePath {
					t.Errorf("expected SQLite path %q, got %q", DefaultEvidenceSQLitePath, cfg.Evidence.SQLite.Path)
				}
				if cfg.Evidence.Retention.Days != DefaultEvidenceRetentionDays {
					t.Errorf("expected retention days %d, got %d", DefaultEvidenceRetentionDays, cfg.Evidence.Retention.Days)
				}
				if cfg.Telemetry.Logging.Level != DefaultLoggingLevel {
					t.Errorf("expected logging level %q, got %q", DefaultLoggingLevel, cfg.Telemetry.Logging.Level)
				}
				if cfg.Telemetry.Logging.Format != DefaultLoggingFormat {
					t.Errorf("expected logging format %q, got %q", DefaultLoggingFormat, cfg.Telemetry.Logging.Format)
				}
				if cfg.Telemetry.Metrics.Path != DefaultPrometheusPath {
					t.Errorf("expected prometheus path %q, got %q", DefaultPrometheusPath, cfg.Telemetry.Metrics.Path)
				}
			},
		},
		{
			name: "existing values are preserved",
			input: Config{
				Proxy: ProxyConfig{
					ListenAddress:  "192.168.1.1:9090",
					ReadTimeout:    60 * time.Second,
					MaxHeaderBytes: 2097152,
				},
				Providers: map[string]ProviderConfig{
					"openai": {
						BaseURL: "https://custom.openai.com",
						Timeout: 90 * time.Second,
					},
				},
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Proxy.ListenAddress != "192.168.1.1:9090" {
					t.Error("existing listen address was overwritten")
				}
				if cfg.Proxy.ReadTimeout != 60*time.Second {
					t.Error("existing read timeout was overwritten")
				}
				if cfg.Proxy.MaxHeaderBytes != 2097152 {
					t.Error("existing max header bytes was overwritten")
				}
				// Check that unset values got defaults
				if cfg.Proxy.WriteTimeout != DefaultWriteTimeout {
					t.Error("write timeout should get default when not set")
				}
			},
		},
		{
			name: "provider defaults applied",
			input: Config{
				Providers: map[string]ProviderConfig{
					"openai": {
						BaseURL: "https://api.openai.com/v1",
						APIKey:  "test-key",
						// Timeout and MaxRetries not set
					},
				},
			},
			check: func(t *testing.T, cfg *Config) {
				provider := cfg.Providers["openai"]
				if provider.Timeout != DefaultProviderTimeout {
					t.Errorf("expected provider timeout %v, got %v", DefaultProviderTimeout, provider.Timeout)
				}
				if provider.MaxRetries != DefaultProviderMaxRetries {
					t.Errorf("expected provider max retries %d, got %d", DefaultProviderMaxRetries, provider.MaxRetries)
				}
				// Verify existing values preserved
				if provider.BaseURL != "https://api.openai.com/v1" {
					t.Error("existing base URL was overwritten")
				}
				if provider.APIKey != "test-key" {
					t.Error("existing API key was overwritten")
				}
			},
		},
		{
			name: "postgres defaults applied",
			input: Config{
				Providers: make(map[string]ProviderConfig),
				Evidence: EvidenceConfig{
					Backend: "postgres",
					Postgres: PostgresConfig{
						Host:     "localhost",
						Database: "mercator",
						User:     "user",
						// Port and SSLMode not set
					},
				},
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Evidence.Postgres.Port != DefaultPostgresPort {
					t.Errorf("expected postgres port %d, got %d", DefaultPostgresPort, cfg.Evidence.Postgres.Port)
				}
				if cfg.Evidence.Postgres.SSLMode != DefaultPostgresSSLMode {
					t.Errorf("expected SSL mode %q, got %q", DefaultPostgresSSLMode, cfg.Evidence.Postgres.SSLMode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			ApplyDefaults(&cfg)
			tt.check(t, &cfg)
		})
	}
}

func TestApplyDefaults_Idempotent(t *testing.T) {
	cfg := Config{
		Providers: make(map[string]ProviderConfig),
	}

	// Apply defaults twice
	ApplyDefaults(&cfg)
	firstPass := cfg.Proxy.ListenAddress

	ApplyDefaults(&cfg)
	secondPass := cfg.Proxy.ListenAddress

	if firstPass != secondPass {
		t.Error("ApplyDefaults should be idempotent")
	}
}
