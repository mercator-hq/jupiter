package providerfactory

import (
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/providers"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.ProviderCount() != 0 {
		t.Errorf("expected 0 providers, got %d", manager.ProviderCount())
	}
}

func TestManager_AddProvider(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config := providers.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	err := manager.AddProvider(config)
	if err != nil {
		t.Fatalf("AddProvider() failed: %v", err)
	}

	if manager.ProviderCount() != 1 {
		t.Errorf("expected 1 provider, got %d", manager.ProviderCount())
	}
}

func TestManager_GetProvider(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config := providers.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	err := manager.AddProvider(config)
	if err != nil {
		t.Fatalf("AddProvider() failed: %v", err)
	}

	provider, err := manager.GetProvider("test-openai")
	if err != nil {
		t.Fatalf("GetProvider() failed: %v", err)
	}

	if provider.GetName() != "test-openai" {
		t.Errorf("expected provider name test-openai, got %s", provider.GetName())
	}
}

func TestManager_GetProvider_NotFound(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	_, err := manager.GetProvider("non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent provider, got nil")
	}
}

func TestManager_RemoveProvider(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config := providers.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	err := manager.AddProvider(config)
	if err != nil {
		t.Fatalf("AddProvider() failed: %v", err)
	}

	if manager.ProviderCount() != 1 {
		t.Errorf("expected 1 provider before removal, got %d", manager.ProviderCount())
	}

	err = manager.RemoveProvider("test-openai")
	if err != nil {
		t.Fatalf("RemoveProvider() failed: %v", err)
	}

	if manager.ProviderCount() != 0 {
		t.Errorf("expected 0 providers after removal, got %d", manager.ProviderCount())
	}
}

func TestManager_GetProviders(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	configs := []providers.ProviderConfig{
		{
			Name:    "openai",
			Type:    "openai",
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "test-key",
		},
		{
			Name:    "anthropic",
			Type:    "anthropic",
			BaseURL: "https://api.anthropic.com",
			APIKey:  "test-key",
		},
	}

	for _, config := range configs {
		err := manager.AddProvider(config)
		if err != nil {
			t.Fatalf("AddProvider() failed: %v", err)
		}
	}

	providers := manager.GetProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}

	if _, ok := providers["openai"]; !ok {
		t.Error("expected openai provider in map")
	}

	if _, ok := providers["anthropic"]; !ok {
		t.Error("expected anthropic provider in map")
	}
}

func TestManager_GetProviderNames(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	configs := []providers.ProviderConfig{
		{
			Name:    "openai",
			Type:    "openai",
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "test-key",
		},
		{
			Name:    "anthropic",
			Type:    "anthropic",
			BaseURL: "https://api.anthropic.com",
			APIKey:  "test-key",
		},
	}

	for _, config := range configs {
		err := manager.AddProvider(config)
		if err != nil {
			t.Fatalf("AddProvider() failed: %v", err)
		}
	}

	names := manager.GetProviderNames()
	if len(names) != 2 {
		t.Errorf("expected 2 provider names, got %d", len(names))
	}
}

func TestManager_LoadFromConfig(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	configs := []providers.ProviderConfig{
		{
			Name:    "openai",
			Type:    "openai",
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "test-key",
		},
		{
			Name:    "anthropic",
			Type:    "anthropic",
			BaseURL: "https://api.anthropic.com",
			APIKey:  "test-key",
		},
	}

	err := manager.LoadFromConfig(configs)
	if err != nil {
		t.Fatalf("LoadFromConfig() failed: %v", err)
	}

	if manager.ProviderCount() != 2 {
		t.Errorf("expected 2 providers, got %d", manager.ProviderCount())
	}
}

func TestManager_GetHealthSummary(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config := providers.ProviderConfig{
		Name:    "test-openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
	}

	err := manager.AddProvider(config)
	if err != nil {
		t.Fatalf("AddProvider() failed: %v", err)
	}

	summary := manager.GetHealthSummary()
	if summary.Total != 1 {
		t.Errorf("expected total 1, got %d", summary.Total)
	}

	if summary.Healthy+summary.Unhealthy != summary.Total {
		t.Errorf("healthy (%d) + unhealthy (%d) should equal total (%d)",
			summary.Healthy, summary.Unhealthy, summary.Total)
	}

	if len(summary.Details) != 1 {
		t.Errorf("expected 1 detail entry, got %d", len(summary.Details))
	}
}

func TestManager_ReplaceProvider(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	config1 := providers.ProviderConfig{
		Name:    "openai",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "old-key",
	}

	err := manager.AddProvider(config1)
	if err != nil {
		t.Fatalf("AddProvider() failed: %v", err)
	}

	// Replace with new config
	config2 := providers.ProviderConfig{
		Name:    "openai", // Same name
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "new-key",
	}

	err = manager.AddProvider(config2)
	if err != nil {
		t.Fatalf("AddProvider() (replace) failed: %v", err)
	}

	// Should still have only 1 provider
	if manager.ProviderCount() != 1 {
		t.Errorf("expected 1 provider after replacement, got %d", manager.ProviderCount())
	}
}
