package config

import (
	"testing"
	"time"
)

func TestNewTestConfig(t *testing.T) {
	cfg := NewTestConfig().Build()

	// Verify defaults are applied
	if cfg.Proxy.ListenAddress != DefaultListenAddress {
		t.Errorf("expected listen address %q, got %q", DefaultListenAddress, cfg.Proxy.ListenAddress)
	}

	if cfg.Proxy.ReadTimeout != DefaultReadTimeout {
		t.Errorf("expected read timeout %v, got %v", DefaultReadTimeout, cfg.Proxy.ReadTimeout)
	}

	if cfg.Policy.Mode != DefaultPolicyMode {
		t.Errorf("expected policy mode %q, got %q", DefaultPolicyMode, cfg.Policy.Mode)
	}

	// Verify test provider is added
	if len(cfg.Providers) == 0 {
		t.Error("expected at least one provider, got none")
	}

	openai, exists := cfg.Providers["openai"]
	if !exists {
		t.Error("expected openai provider, got none")
	}
	if openai.BaseURL == "" {
		t.Error("expected openai base URL to be set")
	}
}

func TestConfigBuilder_WithListenAddress(t *testing.T) {
	cfg := NewTestConfig().
		WithListenAddress("0.0.0.0:9090").
		Build()

	if cfg.Proxy.ListenAddress != "0.0.0.0:9090" {
		t.Errorf("expected listen address %q, got %q", "0.0.0.0:9090", cfg.Proxy.ListenAddress)
	}
}

func TestConfigBuilder_WithProvider(t *testing.T) {
	anthropic := ProviderConfig{
		BaseURL:    "https://api.anthropic.com/v1",
		APIKey:     "test-anthropic-key",
		Timeout:    30 * time.Second,
		MaxRetries: 5,
	}

	cfg := NewTestConfig().
		WithProvider("anthropic", anthropic).
		Build()

	provider, exists := cfg.Providers["anthropic"]
	if !exists {
		t.Fatal("expected anthropic provider, got none")
	}

	if provider.BaseURL != anthropic.BaseURL {
		t.Errorf("expected base URL %q, got %q", anthropic.BaseURL, provider.BaseURL)
	}
	if provider.APIKey != anthropic.APIKey {
		t.Errorf("expected API key %q, got %q", anthropic.APIKey, provider.APIKey)
	}
	if provider.Timeout != anthropic.Timeout {
		t.Errorf("expected timeout %v, got %v", anthropic.Timeout, provider.Timeout)
	}
}

func TestConfigBuilder_WithPolicyGitRepo(t *testing.T) {
	cfg := NewTestConfig().
		WithPolicyGitRepo("https://github.com/example/policies").
		Build()

	if cfg.Policy.Mode != "git" {
		t.Errorf("expected policy mode %q, got %q", "git", cfg.Policy.Mode)
	}
	if cfg.Policy.GitRepo != "https://github.com/example/policies" {
		t.Errorf("expected git repo %q, got %q", "https://github.com/example/policies", cfg.Policy.GitRepo)
	}
	if cfg.Policy.GitBranch == "" {
		t.Error("expected git branch to be set")
	}
	if cfg.Policy.GitPath == "" {
		t.Error("expected git path to be set")
	}
}

func TestConfigBuilder_WithEvidenceBackends(t *testing.T) {
	tests := []struct {
		name    string
		builder func() *ConfigBuilder
		want    string
	}{
		{
			name: "sqlite",
			builder: func() *ConfigBuilder {
				return NewTestConfig().WithSQLitePath("/tmp/evidence.db")
			},
			want: "sqlite",
		},
		{
			name: "postgres",
			builder: func() *ConfigBuilder {
				return NewTestConfig().WithPostgresConfig("localhost", "mercator", "user", "pass", 5432)
			},
			want: "postgres",
		},
		{
			name: "s3",
			builder: func() *ConfigBuilder {
				return NewTestConfig().WithS3Config("my-bucket", "us-east-1")
			},
			want: "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.builder().Build()
			if cfg.Evidence.Backend != tt.want {
				t.Errorf("expected backend %q, got %q", tt.want, cfg.Evidence.Backend)
			}
		})
	}
}

func TestConfigBuilder_WithTLS(t *testing.T) {
	cfg := NewTestConfig().
		WithTLS("/path/to/cert.pem", "/path/to/key.pem").
		Build()

	if !cfg.Security.TLS.Enabled {
		t.Error("expected TLS to be enabled")
	}
	if cfg.Security.TLS.CertFile != "/path/to/cert.pem" {
		t.Errorf("expected cert file %q, got %q", "/path/to/cert.pem", cfg.Security.TLS.CertFile)
	}
	if cfg.Security.TLS.KeyFile != "/path/to/key.pem" {
		t.Errorf("expected key file %q, got %q", "/path/to/key.pem", cfg.Security.TLS.KeyFile)
	}
}

func TestConfigBuilder_WithMTLS(t *testing.T) {
	cfg := NewTestConfig().
		WithMTLS("/path/to/ca.pem").
		Build()

	if !cfg.Security.TLS.MTLS.Enabled {
		t.Error("expected mTLS to be enabled")
	}
	if !cfg.Security.TLS.Enabled {
		t.Error("expected TLS to be enabled when mTLS is enabled")
	}
	if cfg.Security.TLS.MTLS.ClientCAFile != "/path/to/ca.pem" {
		t.Errorf("expected CA file %q, got %q", "/path/to/ca.pem", cfg.Security.TLS.MTLS.ClientCAFile)
	}
}

func TestConfigBuilder_ChainedCalls(t *testing.T) {
	cfg := NewTestConfig().
		WithListenAddress("0.0.0.0:8080").
		WithPolicyFilePath("/etc/mercator/policies.yaml").
		WithLoggingLevel("debug").
		WithMetricsEnabled(true).
		Build()

	if cfg.Proxy.ListenAddress != "0.0.0.0:8080" {
		t.Error("chained WithListenAddress failed")
	}
	if cfg.Policy.FilePath != "/etc/mercator/policies.yaml" {
		t.Error("chained WithPolicyFilePath failed")
	}
	if cfg.Telemetry.Logging.Level != "debug" {
		t.Error("chained WithLoggingLevel failed")
	}
	if !cfg.Telemetry.Metrics.Enabled {
		t.Error("chained WithMetricsEnabled failed")
	}
}

func TestMinimalConfig(t *testing.T) {
	cfg := MinimalConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Verify it's a valid config that would pass validation
	if err := Validate(cfg); err != nil {
		t.Errorf("minimal config should be valid, got error: %v", err)
	}
}
