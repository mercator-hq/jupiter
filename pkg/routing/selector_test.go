package routing

import (
	"testing"

	"mercator-hq/jupiter/pkg/providers"
	mockrouting "mercator-hq/jupiter/internal/routing"
)

func TestNewProviderSelector(t *testing.T) {
	tests := []struct {
		name         string
		providers    map[string]providers.Provider
		modelMapping map[string][]string
		wantNil      bool
	}{
		{
			name: "with providers and mapping",
			providers: map[string]providers.Provider{
				"openai": mockrouting.NewMockProvider("openai"),
			},
			modelMapping: map[string][]string{
				"gpt-4": {"openai"},
			},
			wantNil: false,
		},
		{
			name:         "with nil providers",
			providers:    nil,
			modelMapping: map[string][]string{},
			wantNil:      false,
		},
		{
			name:         "with nil model mapping",
			providers:    map[string]providers.Provider{},
			modelMapping: nil,
			wantNil:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewProviderSelector(tt.providers, tt.modelMapping)
			if (selector == nil) != tt.wantNil {
				t.Errorf("NewProviderSelector() = %v, wantNil %v", selector, tt.wantNil)
			}
			if selector != nil {
				if selector.providers == nil {
					t.Error("selector.providers should not be nil")
				}
				if selector.modelMapping == nil {
					t.Error("selector.modelMapping should not be nil")
				}
			}
		})
	}
}

func TestProviderSelector_FilterByHealth(t *testing.T) {
	tests := []struct {
		name      string
		providers []providers.Provider
		want      int
	}{
		{
			name: "all healthy",
			providers: []providers.Provider{
				mockrouting.NewMockProvider("openai"),
				mockrouting.NewMockProvider("anthropic"),
			},
			want: 2,
		},
		{
			name: "some unhealthy",
			providers: []providers.Provider{
				mockrouting.NewMockProvider("openai"),
				func() providers.Provider {
					p := mockrouting.NewMockProvider("anthropic")
					p.SetHealthy(false)
					return p
				}(),
			},
			want: 1,
		},
		{
			name: "all unhealthy",
			providers: []providers.Provider{
				func() providers.Provider {
					p := mockrouting.NewMockProvider("openai")
					p.SetHealthy(false)
					return p
				}(),
				func() providers.Provider {
					p := mockrouting.NewMockProvider("anthropic")
					p.SetHealthy(false)
					return p
				}(),
			},
			want: 0,
		},
		{
			name:      "empty list",
			providers: []providers.Provider{},
			want:      0,
		},
		{
			name:      "nil list",
			providers: nil,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewProviderSelector(nil, nil)
			result := selector.FilterByHealth(tt.providers)
			if len(result) != tt.want {
				t.Errorf("FilterByHealth() returned %d providers, want %d", len(result), tt.want)
			}

			// Verify all returned providers are healthy
			for _, p := range result {
				if !p.IsHealthy() {
					t.Errorf("FilterByHealth() returned unhealthy provider %s", p.GetName())
				}
			}
		})
	}
}

func TestProviderSelector_FilterByModel(t *testing.T) {
	openai := mockrouting.NewMockProvider("openai")
	anthropic := mockrouting.NewMockProvider("anthropic")
	ollama := mockrouting.NewMockProvider("ollama")

	modelMapping := map[string][]string{
		"gpt-4":          {"openai"},
		"gpt-3.5-turbo":  {"openai"},
		"claude-3-opus":  {"anthropic"},
		"claude-3-sonnet": {"anthropic"},
		"llama-3":        {"ollama"},
	}

	tests := []struct {
		name      string
		providers []providers.Provider
		model     string
		want      []string
	}{
		{
			name:      "gpt-4 to openai",
			providers: []providers.Provider{openai, anthropic, ollama},
			model:     "gpt-4",
			want:      []string{"openai"},
		},
		{
			name:      "claude-3-opus to anthropic",
			providers: []providers.Provider{openai, anthropic, ollama},
			model:     "claude-3-opus",
			want:      []string{"anthropic"},
		},
		{
			name:      "unmapped model returns all",
			providers: []providers.Provider{openai, anthropic, ollama},
			model:     "unknown-model",
			want:      []string{"openai", "anthropic", "ollama"},
		},
		{
			name:      "empty model returns all",
			providers: []providers.Provider{openai, anthropic},
			model:     "",
			want:      []string{"openai", "anthropic"},
		},
		{
			name:      "empty provider list",
			providers: []providers.Provider{},
			model:     "gpt-4",
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewProviderSelector(nil, modelMapping)
			result := selector.FilterByModel(tt.providers, tt.model)

			if len(result) != len(tt.want) {
				t.Errorf("FilterByModel() returned %d providers, want %d", len(result), len(tt.want))
				return
			}

			// Convert result to names for comparison
			resultNames := make(map[string]bool)
			for _, p := range result {
				resultNames[p.GetName()] = true
			}

			// Check all expected providers are present
			for _, name := range tt.want {
				if !resultNames[name] {
					t.Errorf("FilterByModel() missing provider %s", name)
				}
			}
		})
	}
}

func TestProviderSelector_GetAvailableProviders(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]providers.Provider
		wantCount int
	}{
		{
			name: "multiple providers",
			providers: map[string]providers.Provider{
				"openai":    mockrouting.NewMockProvider("openai"),
				"anthropic": mockrouting.NewMockProvider("anthropic"),
				"ollama":    mockrouting.NewMockProvider("ollama"),
			},
			wantCount: 3,
		},
		{
			name:      "no providers",
			providers: map[string]providers.Provider{},
			wantCount: 0,
		},
		{
			name:      "nil providers",
			providers: nil,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewProviderSelector(tt.providers, nil)
			result := selector.GetAvailableProviders()

			if tt.wantCount == 0 && result != nil && len(result) != 0 {
				t.Errorf("GetAvailableProviders() = %v, want empty or nil", result)
			}

			if len(result) != tt.wantCount {
				t.Errorf("GetAvailableProviders() returned %d providers, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestProviderSelector_GetProvider(t *testing.T) {
	openai := mockrouting.NewMockProvider("openai")
	anthropic := mockrouting.NewMockProvider("anthropic")

	providers := map[string]providers.Provider{
		"openai":    openai,
		"anthropic": anthropic,
	}

	selector := NewProviderSelector(providers, nil)

	tests := []struct {
		name         string
		providerName string
		wantNil      bool
	}{
		{
			name:         "existing provider",
			providerName: "openai",
			wantNil:      false,
		},
		{
			name:         "non-existing provider",
			providerName: "nonexistent",
			wantNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selector.GetProvider(tt.providerName)
			if (result == nil) != tt.wantNil {
				t.Errorf("GetProvider(%s) = %v, wantNil %v", tt.providerName, result, tt.wantNil)
			}
			if result != nil && result.GetName() != tt.providerName {
				t.Errorf("GetProvider(%s) returned provider with name %s", tt.providerName, result.GetName())
			}
		})
	}
}

func TestProviderSelector_GetProviderNames(t *testing.T) {
	providers := map[string]providers.Provider{
		"openai":    mockrouting.NewMockProvider("openai"),
		"anthropic": mockrouting.NewMockProvider("anthropic"),
		"ollama":    mockrouting.NewMockProvider("ollama"),
	}

	selector := NewProviderSelector(providers, nil)
	names := selector.GetProviderNames()

	if len(names) != 3 {
		t.Errorf("GetProviderNames() returned %d names, want 3", len(names))
	}

	// Check all expected names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	expectedNames := []string{"openai", "anthropic", "ollama"}
	for _, expected := range expectedNames {
		if !nameMap[expected] {
			t.Errorf("GetProviderNames() missing provider %s", expected)
		}
	}
}

func TestProviderSelector_GetSupportedModels(t *testing.T) {
	modelMapping := map[string][]string{
		"gpt-4":         {"openai"},
		"claude-3-opus": {"anthropic"},
		"llama-3":       {"ollama"},
	}

	selector := NewProviderSelector(nil, modelMapping)
	models := selector.GetSupportedModels()

	if len(models) != 3 {
		t.Errorf("GetSupportedModels() returned %d models, want 3", len(models))
	}

	// Check all expected models are present
	modelMap := make(map[string]bool)
	for _, model := range models {
		modelMap[model] = true
	}

	expectedModels := []string{"gpt-4", "claude-3-opus", "llama-3"}
	for _, expected := range expectedModels {
		if !modelMap[expected] {
			t.Errorf("GetSupportedModels() missing model %s", expected)
		}
	}
}

func TestProviderSelector_UpdateProviders(t *testing.T) {
	selector := NewProviderSelector(nil, nil)

	// Initial state should be empty
	if len(selector.GetAvailableProviders()) != 0 {
		t.Error("Initial providers should be empty")
	}

	// Update with new providers
	newProviders := map[string]providers.Provider{
		"openai": mockrouting.NewMockProvider("openai"),
	}
	selector.UpdateProviders(newProviders)

	if len(selector.GetAvailableProviders()) != 1 {
		t.Errorf("After update, got %d providers, want 1", len(selector.GetAvailableProviders()))
	}

	// Update with nil should set empty map
	selector.UpdateProviders(nil)
	if len(selector.GetAvailableProviders()) != 0 {
		t.Error("After nil update, providers should be empty")
	}
}

func TestProviderSelector_UpdateModelMapping(t *testing.T) {
	selector := NewProviderSelector(nil, nil)

	// Initial state should be empty
	if len(selector.GetSupportedModels()) != 0 {
		t.Error("Initial model mapping should be empty")
	}

	// Update with new mapping
	newMapping := map[string][]string{
		"gpt-4": {"openai"},
	}
	selector.UpdateModelMapping(newMapping)

	if len(selector.GetSupportedModels()) != 1 {
		t.Errorf("After update, got %d models, want 1", len(selector.GetSupportedModels()))
	}

	// Update with nil should set empty map
	selector.UpdateModelMapping(nil)
	if len(selector.GetSupportedModels()) != 0 {
		t.Error("After nil update, model mapping should be empty")
	}
}
