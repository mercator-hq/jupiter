package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "0.0.0.0:8080"
  read_timeout: "60s"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key-123"
    timeout: "30s"
    max_retries: 5

policy:
  mode: "file"
  file_path: "./policies.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "./test-evidence.db"

telemetry:
  logging:
    level: "debug"
    format: "text"
  metrics:
    enabled: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Proxy.ListenAddress != "0.0.0.0:8080" {
		t.Errorf("expected listen address %q, got %q", "0.0.0.0:8080", cfg.Proxy.ListenAddress)
	}
	if cfg.Proxy.ReadTimeout != 60*time.Second {
		t.Errorf("expected read timeout %v, got %v", 60*time.Second, cfg.Proxy.ReadTimeout)
	}

	openai, exists := cfg.Providers["openai"]
	if !exists {
		t.Fatal("expected openai provider")
	}
	if openai.APIKey != "test-key-123" {
		t.Errorf("expected API key %q, got %q", "test-key-123", openai.APIKey)
	}
	if openai.Timeout != 30*time.Second {
		t.Errorf("expected timeout %v, got %v", 30*time.Second, openai.Timeout)
	}

	if cfg.Telemetry.Logging.Level != "debug" {
		t.Errorf("expected logging level %q, got %q", "debug", cfg.Telemetry.Logging.Level)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	// Check if error contains file not found message
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("expected file not found error, got: %v", err)
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	malformedContent := `
proxy:
  listen_address: "0.0.0.0:8080"
  invalid yaml here: [
`

	if err := os.WriteFile(configPath, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestLoadConfig_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config with validation errors (no providers, invalid logging level)
	invalidContent := `
proxy:
  listen_address: "0.0.0.0:8080"

providers: {}

telemetry:
  logging:
    level: "invalid"
    format: "json"
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// Check if the error chain contains a ValidationError
	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("expected ValidationError in error chain, got %T: %v", err, err)
	}
}

func TestLoadConfigWithEnvOverrides_BasicOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "file-key"

telemetry:
  logging:
    level: "info"
    format: "json"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set environment variables
	os.Setenv("MERCATOR_PROXY_LISTEN_ADDRESS", "0.0.0.0:9090")
	os.Setenv("MERCATOR_PROVIDERS_OPENAI_API_KEY", "env-key-override")
	os.Setenv("MERCATOR_TELEMETRY_LOGGING_LEVEL", "debug")
	defer func() {
		os.Unsetenv("MERCATOR_PROXY_LISTEN_ADDRESS")
		os.Unsetenv("MERCATOR_PROVIDERS_OPENAI_API_KEY")
		os.Unsetenv("MERCATOR_TELEMETRY_LOGGING_LEVEL")
	}()

	cfg, err := LoadConfigWithEnvOverrides(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify environment overrides took effect
	if cfg.Proxy.ListenAddress != "0.0.0.0:9090" {
		t.Errorf("expected listen address %q from env, got %q", "0.0.0.0:9090", cfg.Proxy.ListenAddress)
	}

	openai := cfg.Providers["openai"]
	if openai.APIKey != "env-key-override" {
		t.Errorf("expected API key %q from env, got %q", "env-key-override", openai.APIKey)
	}

	if cfg.Telemetry.Logging.Level != "debug" {
		t.Errorf("expected logging level %q from env, got %q", "debug", cfg.Telemetry.Logging.Level)
	}
}

func TestLoadConfigWithEnvOverrides_DurationParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "127.0.0.1:8080"
  read_timeout: "30s"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("MERCATOR_PROXY_READ_TIMEOUT", "120s")
	os.Setenv("MERCATOR_PROVIDERS_OPENAI_TIMEOUT", "45s")
	defer func() {
		os.Unsetenv("MERCATOR_PROXY_READ_TIMEOUT")
		os.Unsetenv("MERCATOR_PROVIDERS_OPENAI_TIMEOUT")
	}()

	cfg, err := LoadConfigWithEnvOverrides(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Proxy.ReadTimeout != 120*time.Second {
		t.Errorf("expected read timeout %v, got %v", 120*time.Second, cfg.Proxy.ReadTimeout)
	}

	if cfg.Providers["openai"].Timeout != 45*time.Second {
		t.Errorf("expected provider timeout %v, got %v", 45*time.Second, cfg.Providers["openai"].Timeout)
	}
}

func TestLoadConfigWithEnvOverrides_IntegerParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
    max_retries: 3

evidence:
  enabled: true
  backend: "sqlite"
  retention_days: 90
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("MERCATOR_PROXY_MAX_HEADER_BYTES", "2097152")
	os.Setenv("MERCATOR_PROVIDERS_OPENAI_MAX_RETRIES", "5")
	os.Setenv("MERCATOR_EVIDENCE_RETENTION_DAYS", "30")
	defer func() {
		os.Unsetenv("MERCATOR_PROXY_MAX_HEADER_BYTES")
		os.Unsetenv("MERCATOR_PROVIDERS_OPENAI_MAX_RETRIES")
		os.Unsetenv("MERCATOR_EVIDENCE_RETENTION_DAYS")
	}()

	cfg, err := LoadConfigWithEnvOverrides(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Proxy.MaxHeaderBytes != 2097152 {
		t.Errorf("expected max header bytes %d, got %d", 2097152, cfg.Proxy.MaxHeaderBytes)
	}

	if cfg.Providers["openai"].MaxRetries != 5 {
		t.Errorf("expected max retries %d, got %d", 5, cfg.Providers["openai"].MaxRetries)
	}

	if cfg.Evidence.Retention.Days != 30 {
		t.Errorf("expected retention days %d, got %d", 30, cfg.Evidence.Retention.Days)
	}
}

func TestLoadConfigWithEnvOverrides_BooleanParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"

policy:
  mode: "file"
  file_path: "./policies.yaml"
  watch: false

evidence:
  enabled: false
  backend: "sqlite"

telemetry:
  metrics:
    enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("MERCATOR_POLICY_WATCH", "true")
	os.Setenv("MERCATOR_EVIDENCE_ENABLED", "true")
	os.Setenv("MERCATOR_TELEMETRY_METRICS_ENABLED", "true")
	defer func() {
		os.Unsetenv("MERCATOR_POLICY_WATCH")
		os.Unsetenv("MERCATOR_EVIDENCE_ENABLED")
		os.Unsetenv("MERCATOR_TELEMETRY_METRICS_ENABLED")
	}()

	cfg, err := LoadConfigWithEnvOverrides(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.Policy.Watch {
		t.Error("expected policy watch to be true from env")
	}

	if !cfg.Evidence.Enabled {
		t.Error("expected evidence enabled to be true from env")
	}

	if !cfg.Telemetry.Metrics.Enabled {
		t.Error("expected metrics enabled to be true from env")
	}
}

func TestLoadConfigWithEnvOverrides_InvalidEnvValues(t *testing.T) {
	tmpDir := t.TempDir()
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
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set invalid environment variables (they should be ignored or cause validation to fail)
	os.Setenv("MERCATOR_PROXY_MAX_HEADER_BYTES", "not-a-number")
	os.Setenv("MERCATOR_TELEMETRY_LOGGING_LEVEL", "invalid-level")
	defer func() {
		os.Unsetenv("MERCATOR_PROXY_MAX_HEADER_BYTES")
		os.Unsetenv("MERCATOR_TELEMETRY_LOGGING_LEVEL")
	}()

	_, err := LoadConfigWithEnvOverrides(configPath)
	// Should fail validation due to invalid logging level
	if err == nil {
		t.Error("expected validation error for invalid env values")
	}
}

func TestLoadConfigWithEnvOverrides_NewProvider(t *testing.T) {
	tmpDir := t.TempDir()
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
		t.Fatalf("failed to write config file: %v", err)
	}

	// Try to add a new provider via env vars
	os.Setenv("MERCATOR_PROVIDERS_ANTHROPIC_BASE_URL", "https://api.anthropic.com/v1")
	os.Setenv("MERCATOR_PROVIDERS_ANTHROPIC_API_KEY", "anthropic-key")
	defer func() {
		os.Unsetenv("MERCATOR_PROVIDERS_ANTHROPIC_BASE_URL")
		os.Unsetenv("MERCATOR_PROVIDERS_ANTHROPIC_API_KEY")
	}()

	cfg, err := LoadConfigWithEnvOverrides(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify anthropic provider was added
	anthropic, exists := cfg.Providers["anthropic"]
	if !exists {
		t.Error("expected anthropic provider to be added from env vars")
	} else {
		if anthropic.BaseURL != "https://api.anthropic.com/v1" {
			t.Errorf("expected base URL %q, got %q", "https://api.anthropic.com/v1", anthropic.BaseURL)
		}
		if anthropic.APIKey != "anthropic-key" {
			t.Errorf("expected API key %q, got %q", "anthropic-key", anthropic.APIKey)
		}
	}
}
