package config

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkLoadConfig benchmarks loading a typical configuration file.
// Target: <10ms p99 latency
func BenchmarkLoadConfig(b *testing.B) {
	// Create a temporary config file
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "127.0.0.1:8080"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
    timeout: "60s"
    max_retries: 3

  anthropic:
    base_url: "https://api.anthropic.com/v1"
    api_key: "test-key"
    timeout: "60s"
    max_retries: 3

policy:
  mode: "file"
  file_path: "./policies.yaml"
  watch: false
  validation:
    enabled: true
    strict: false

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "./evidence.db"
  retention_days: 90

telemetry:
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: true
    prometheus_path: "/metrics"
  tracing:
    enabled: false
    sampling_rate: 1.0

security:
  tls:
    enabled: false
  mtls:
    enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		b.Fatalf("failed to write config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath)
		if err != nil {
			b.Fatalf("failed to load config: %v", err)
		}
	}
}

// BenchmarkLoadConfigWithEnvOverrides benchmarks loading with environment variable overrides.
func BenchmarkLoadConfigWithEnvOverrides(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"

telemetry:
  logging:
    level: "info"
    format: "json"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		b.Fatalf("failed to write config file: %v", err)
	}

	// Set some environment variables
	os.Setenv("MERCATOR_PROXY_LISTEN_ADDRESS", "0.0.0.0:9090")
	os.Setenv("MERCATOR_PROVIDERS_OPENAI_API_KEY", "env-key")
	defer func() {
		os.Unsetenv("MERCATOR_PROXY_LISTEN_ADDRESS")
		os.Unsetenv("MERCATOR_PROVIDERS_OPENAI_API_KEY")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfigWithEnvOverrides(configPath)
		if err != nil {
			b.Fatalf("failed to load config: %v", err)
		}
	}
}

// BenchmarkValidate benchmarks configuration validation.
// Target: <1ms for full validation
func BenchmarkValidate(b *testing.B) {
	cfg := NewTestConfig().Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Validate(cfg)
		if err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}
}

// BenchmarkApplyDefaults benchmarks applying default values.
func BenchmarkApplyDefaults(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := Config{
			Providers: make(map[string]ProviderConfig),
		}
		ApplyDefaults(&cfg)
	}
}

// BenchmarkGetConfig benchmarks singleton config access.
// Target: <1µs (simple pointer return)
func BenchmarkGetConfig(b *testing.B) {
	// Set up config
	SetConfig(MinimalConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetConfig()
	}
}

// BenchmarkConfigBuilder benchmarks building config programmatically.
func BenchmarkConfigBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewTestConfig().
			WithListenAddress("0.0.0.0:8080").
			WithPolicyFilePath("./policies.yaml").
			WithLoggingLevel("debug").
			Build()
	}
}
