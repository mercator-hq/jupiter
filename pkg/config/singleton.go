package config

import (
	"fmt"
	"sync"
)

var (
	// globalConfig holds the singleton configuration instance.
	globalConfig *Config

	// configMutex protects access to globalConfig.
	configMutex sync.RWMutex

	// initOnce ensures configuration is initialized only once.
	initOnce sync.Once
)

// Initialize loads configuration from the specified path with environment
// variable overrides and stores it as the global singleton configuration.
// This function should be called once at application startup.
// Subsequent calls are ignored (uses sync.Once internally).
//
// Returns an error if configuration loading or validation fails.
func Initialize(path string) error {
	var initErr error

	initOnce.Do(func() {
		cfg, err := LoadConfigWithEnvOverrides(path)
		if err != nil {
			initErr = err
			return
		}

		configMutex.Lock()
		globalConfig = cfg
		configMutex.Unlock()
	})

	return initErr
}

// GetConfig returns the global configuration instance.
// It returns nil if Initialize has not been called successfully.
// This function is thread-safe and can be called concurrently.
//
// For testing, prefer using dependency injection with explicit Config
// instances rather than relying on the global singleton.
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// SetConfig sets the global configuration instance.
// This function is primarily intended for testing and should not be used
// in production code. Use Initialize for normal configuration loading.
//
// This function is thread-safe.
func SetConfig(cfg *Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	globalConfig = cfg
}

// ReloadConfig reloads the configuration from the specified path.
// This is useful for hot-reloading configuration without restarting
// the application. The new configuration replaces the global instance
// only if loading and validation succeed.
//
// Returns an error if reloading fails, in which case the existing
// configuration remains unchanged.
func ReloadConfig(path string) error {
	// Load new configuration
	cfg, err := LoadConfigWithEnvOverrides(path)
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Replace global configuration
	configMutex.Lock()
	globalConfig = cfg
	configMutex.Unlock()

	return nil
}

// MustGetConfig returns the global configuration instance.
// It panics if the configuration has not been initialized.
// This should only be used in code paths where configuration is
// guaranteed to be initialized (e.g., after successful application startup).
//
// For most use cases, prefer GetConfig which returns nil instead of panicking.
func MustGetConfig() *Config {
	cfg := GetConfig()
	if cfg == nil {
		panic("configuration not initialized: call Initialize first")
	}
	return cfg
}
