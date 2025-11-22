package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestInitialize(t *testing.T) {
	// Reset global state
	globalConfig = nil
	initOnce = *new(sync.Once)

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

	err := Initialize(configPath)
	if err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	cfg := GetConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config after initialization")
	}

	if cfg.Proxy.ListenAddress != "127.0.0.1:8080" {
		t.Errorf("expected listen address %q, got %q", "127.0.0.1:8080", cfg.Proxy.ListenAddress)
	}
}

func TestInitialize_MultipleCallsIgnored(t *testing.T) {
	// Reset global state
	globalConfig = nil
	initOnce = *new(sync.Once)

	tmpDir := t.TempDir()
	configPath1 := filepath.Join(tmpDir, "config1.yaml")
	configPath2 := filepath.Join(tmpDir, "config2.yaml")

	config1Content := `
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "key1"

telemetry:
  logging:
    level: "info"
    format: "json"
`

	config2Content := `
proxy:
  listen_address: "0.0.0.0:9090"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "key2"

telemetry:
  logging:
    level: "debug"
    format: "text"
`

	if err := os.WriteFile(configPath1, []byte(config1Content), 0644); err != nil {
		t.Fatalf("failed to write config1 file: %v", err)
	}
	if err := os.WriteFile(configPath2, []byte(config2Content), 0644); err != nil {
		t.Fatalf("failed to write config2 file: %v", err)
	}

	// First initialization
	err := Initialize(configPath1)
	if err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	firstConfig := GetConfig()

	// Second initialization should be ignored
	Initialize(configPath2)

	secondConfig := GetConfig()

	// Should still have the first config
	if firstConfig.Proxy.ListenAddress != secondConfig.Proxy.ListenAddress {
		t.Error("second Initialize call should be ignored")
	}
	if firstConfig.Providers["openai"].APIKey != secondConfig.Providers["openai"].APIKey {
		t.Error("second Initialize call should be ignored")
	}
}

func TestGetConfig_BeforeInitialize(t *testing.T) {
	// Reset global state
	globalConfig = nil

	cfg := GetConfig()
	if cfg != nil {
		t.Error("expected nil config before initialization")
	}
}

func TestSetConfig(t *testing.T) {
	// Reset global state
	globalConfig = nil

	testCfg := NewTestConfig().
		WithListenAddress("192.168.1.1:7070").
		Build()

	SetConfig(testCfg)

	retrievedCfg := GetConfig()
	if retrievedCfg == nil {
		t.Fatal("expected non-nil config after SetConfig")
	}

	if retrievedCfg.Proxy.ListenAddress != "192.168.1.1:7070" {
		t.Errorf("expected listen address %q, got %q", "192.168.1.1:7070", retrievedCfg.Proxy.ListenAddress)
	}
}

func TestReloadConfig(t *testing.T) {
	// Reset global state
	globalConfig = nil
	initOnce = *new(sync.Once)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := `
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "initial-key"

telemetry:
  logging:
    level: "info"
    format: "json"
`

	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial config file: %v", err)
	}

	// Initialize with initial config
	if err := Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	initialCfg := GetConfig()
	if initialCfg.Providers["openai"].APIKey != "initial-key" {
		t.Error("initial config not loaded correctly")
	}

	// Update the file
	updatedContent := `
proxy:
  listen_address: "0.0.0.0:9090"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "updated-key"

telemetry:
  logging:
    level: "debug"
    format: "text"
`

	if err := os.WriteFile(configPath, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("failed to write updated config file: %v", err)
	}

	// Reload config
	if err := ReloadConfig(configPath); err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	reloadedCfg := GetConfig()
	if reloadedCfg.Proxy.ListenAddress != "0.0.0.0:9090" {
		t.Errorf("expected updated listen address %q, got %q", "0.0.0.0:9090", reloadedCfg.Proxy.ListenAddress)
	}
	if reloadedCfg.Providers["openai"].APIKey != "updated-key" {
		t.Errorf("expected updated API key %q, got %q", "updated-key", reloadedCfg.Providers["openai"].APIKey)
	}
	if reloadedCfg.Telemetry.Logging.Level != "debug" {
		t.Errorf("expected updated logging level %q, got %q", "debug", reloadedCfg.Telemetry.Logging.Level)
	}
}

func TestReloadConfig_ValidationFailure(t *testing.T) {
	// Reset global state
	globalConfig = nil
	initOnce = *new(sync.Once)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validContent := `
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

	if err := os.WriteFile(configPath, []byte(validContent), 0644); err != nil {
		t.Fatalf("failed to write initial config file: %v", err)
	}

	// Initialize with valid config
	if err := Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	originalCfg := GetConfig()

	// Update file with invalid config
	invalidContent := `
proxy:
  listen_address: "127.0.0.1:8080"

providers: {}

telemetry:
  logging:
    level: "invalid"
    format: "json"
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write invalid config file: %v", err)
	}

	// Try to reload - should fail
	err := ReloadConfig(configPath)
	if err == nil {
		t.Fatal("expected error when reloading invalid config")
	}

	// Original config should be preserved
	currentCfg := GetConfig()
	if currentCfg.Proxy.ListenAddress != originalCfg.Proxy.ListenAddress {
		t.Error("original config should be preserved on reload failure")
	}
}

func TestMustGetConfig(t *testing.T) {
	// Reset global state
	globalConfig = nil
	initOnce = *new(sync.Once)

	// Test panic when not initialized
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustGetConfig to panic when not initialized")
		}
	}()

	MustGetConfig()
}

func TestMustGetConfig_AfterInitialize(t *testing.T) {
	// Reset global state
	globalConfig = nil
	initOnce = *new(sync.Once)

	SetConfig(MinimalConfig())

	// Should not panic
	cfg := MustGetConfig()
	if cfg == nil {
		t.Error("expected non-nil config from MustGetConfig")
	}
}
