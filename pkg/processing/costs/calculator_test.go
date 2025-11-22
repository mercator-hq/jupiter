package costs

import (
	"testing"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/processing/tokens"
	"mercator-hq/jupiter/pkg/providers"
)

func TestCalculator_CalculateRequestCost(t *testing.T) {
	cfg := &config.CostsConfig{
		Pricing: map[string]map[string]config.ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:     0.03,
					Completion: 0.06,
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
			},
			"default": {
				"default": {
					Prompt:     0.001,
					Completion: 0.002,
				},
			},
		},
	}

	calculator := NewCalculator(cfg)

	tests := []struct {
		name        string
		estimate    *tokens.Estimate
		model       string
		provider    string
		expectedMin float64
		expectedMax float64
		expectError bool
	}{
		{
			name:        "nil estimate",
			estimate:    nil,
			model:       "gpt-4",
			provider:    "openai",
			expectError: true,
		},
		{
			name: "gpt-4 request",
			estimate: &tokens.Estimate{
				PromptTokens:              100,
				EstimatedCompletionTokens: 100,
				TotalTokens:               200,
			},
			model:       "gpt-4",
			provider:    "openai",
			expectedMin: 0.008, // (100/1000 * 0.03) + (100/1000 * 0.06) = 0.003 + 0.006 = 0.009
			expectedMax: 0.010,
		},
		{
			name: "gpt-3.5-turbo request",
			estimate: &tokens.Estimate{
				PromptTokens:              1000,
				EstimatedCompletionTokens: 500,
				TotalTokens:               1500,
			},
			model:       "gpt-3.5-turbo",
			provider:    "openai",
			expectedMin: 0.0012, // (1000/1000 * 0.0005) + (500/1000 * 0.0015) = 0.0005 + 0.00075 = 0.00125
			expectedMax: 0.0014,
		},
		{
			name: "claude-3-opus request",
			estimate: &tokens.Estimate{
				PromptTokens:              200,
				EstimatedCompletionTokens: 100,
				TotalTokens:               300,
			},
			model:       "claude-3-opus",
			provider:    "anthropic",
			expectedMin: 0.0104, // (200/1000 * 0.015) + (100/1000 * 0.075) = 0.003 + 0.0075 = 0.0105
			expectedMax: 0.0106,
		},
		{
			name: "unknown model uses default",
			estimate: &tokens.Estimate{
				PromptTokens:              1000,
				EstimatedCompletionTokens: 1000,
				TotalTokens:               2000,
			},
			model:       "unknown-model",
			provider:    "unknown-provider",
			expectedMin: 0.0029, // (1000/1000 * 0.001) + (1000/1000 * 0.002) = 0.001 + 0.002 = 0.003
			expectedMax: 0.0031,
		},
		{
			name: "model prefix match",
			estimate: &tokens.Estimate{
				PromptTokens:              100,
				EstimatedCompletionTokens: 100,
				TotalTokens:               200,
			},
			model:       "gpt-4-turbo",
			provider:    "openai",
			expectedMin: 0.008,
			expectedMax: 0.010,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := calculator.CalculateRequestCost(tt.estimate, tt.model, tt.provider)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cost.TotalCost < tt.expectedMin || cost.TotalCost > tt.expectedMax {
				t.Errorf("expected total cost between $%.4f and $%.4f, got $%.4f",
					tt.expectedMin, tt.expectedMax, cost.TotalCost)
			}

			// Verify that TotalCost = PromptCost + CompletionCost
			expectedTotal := cost.PromptCost + cost.CompletionCost
			diff := cost.TotalCost - expectedTotal
			if diff > 0.0001 || diff < -0.0001 {
				t.Errorf("total cost mismatch: TotalCost=$%.6f, but PromptCost=$%.6f + CompletionCost=$%.6f = $%.6f",
					cost.TotalCost, cost.PromptCost, cost.CompletionCost, expectedTotal)
			}

			// Verify currency is set
			if cost.Currency != "USD" {
				t.Errorf("expected currency USD, got %s", cost.Currency)
			}

			// Verify model and provider are set
			if cost.Model == "" {
				t.Errorf("model not set in cost estimate")
			}
			if cost.Provider == "" {
				t.Errorf("provider not set in cost estimate")
			}
		})
	}
}

func TestCalculator_CalculateResponseCost(t *testing.T) {
	cfg := &config.CostsConfig{
		Pricing: map[string]map[string]config.ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:       0.03,
					Completion:   0.06,
					CachedPrompt: 0.015, // 50% discount for cached
				},
			},
			"default": {
				"default": {
					Prompt:     0.001,
					Completion: 0.002,
				},
			},
		},
	}

	calculator := NewCalculator(cfg)

	tests := []struct {
		name        string
		usage       *TokenUsage
		model       string
		provider    string
		expectedMin float64
		expectedMax float64
		expectError bool
	}{
		{
			name:        "nil usage",
			usage:       nil,
			model:       "gpt-4",
			provider:    "openai",
			expectError: true,
		},
		{
			name: "simple response",
			usage: &TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
			model:       "gpt-4",
			provider:    "openai",
			expectedMin: 0.006, // (100/1000 * 0.03) + (50/1000 * 0.06) = 0.003 + 0.003 = 0.006
			expectedMax: 0.007,
		},
		{
			name: "response with cached tokens",
			usage: &TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				CachedTokens:     50, // Half the prompt is cached
				TotalTokens:      150,
			},
			model:       "gpt-4",
			provider:    "openai",
			expectedMin: 0.0045, // (50/1000 * 0.03) + (50/1000 * 0.015) + (50/1000 * 0.06) = 0.0015 + 0.00075 + 0.003 = 0.00525
			expectedMax: 0.0055,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := calculator.CalculateResponseCost(tt.usage, tt.model, tt.provider)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cost.TotalCost < tt.expectedMin || cost.TotalCost > tt.expectedMax {
				t.Errorf("expected total cost between $%.4f and $%.4f, got $%.4f",
					tt.expectedMin, tt.expectedMax, cost.TotalCost)
			}
		})
	}
}

func TestCalculator_CalculateProviderResponseCost(t *testing.T) {
	cfg := &config.CostsConfig{
		Pricing: map[string]map[string]config.ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:     0.03,
					Completion: 0.06,
				},
			},
		},
	}

	calculator := NewCalculator(cfg)

	tests := []struct {
		name        string
		response    *providers.CompletionResponse
		provider    string
		expectedMin float64
		expectedMax float64
		expectError bool
	}{
		{
			name:        "nil response",
			response:    nil,
			provider:    "openai",
			expectError: true,
		},
		{
			name: "valid response",
			response: &providers.CompletionResponse{
				Model: "gpt-4",
				Usage: providers.TokenUsage{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
				},
			},
			provider:    "openai",
			expectedMin: 0.006,
			expectedMax: 0.007,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := calculator.CalculateProviderResponseCost(tt.response, tt.provider)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cost.TotalCost < tt.expectedMin || cost.TotalCost > tt.expectedMax {
				t.Errorf("expected total cost between $%.4f and $%.4f, got $%.4f",
					tt.expectedMin, tt.expectedMax, cost.TotalCost)
			}
		})
	}
}

func TestCalculator_GetModelPricing(t *testing.T) {
	cfg := &config.CostsConfig{
		Pricing: map[string]map[string]config.ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:     0.03,
					Completion: 0.06,
				},
			},
			"default": {
				"default": {
					Prompt:     0.001,
					Completion: 0.002,
				},
			},
		},
	}

	calculator := NewCalculator(cfg)

	tests := []struct {
		name           string
		model          string
		provider       string
		expectError    bool
		expectedPrompt float64
	}{
		{
			name:           "exact match",
			model:          "gpt-4",
			provider:       "openai",
			expectError:    false,
			expectedPrompt: 0.03,
		},
		{
			name:           "prefix match",
			model:          "gpt-4-turbo",
			provider:       "openai",
			expectError:    false,
			expectedPrompt: 0.03,
		},
		{
			name:           "unknown uses default",
			model:          "unknown",
			provider:       "unknown",
			expectError:    false,
			expectedPrompt: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := calculator.GetModelPricing(tt.model, tt.provider)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if pricing.PromptCostPer1KTokens != tt.expectedPrompt {
				t.Errorf("expected prompt cost $%.4f, got $%.4f",
					tt.expectedPrompt, pricing.PromptCostPer1KTokens)
			}
		})
	}
}

func TestCalculator_UpdatePricing(t *testing.T) {
	cfg := &config.CostsConfig{
		Pricing: map[string]map[string]config.ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:     0.03,
					Completion: 0.06,
				},
			},
		},
	}

	calculator := NewCalculator(cfg)

	// Get initial pricing
	pricing, err := calculator.GetModelPricing("gpt-4", "openai")
	if err != nil {
		t.Fatalf("failed to get initial pricing: %v", err)
	}
	if pricing.PromptCostPer1KTokens != 0.03 {
		t.Errorf("expected initial prompt cost $0.03, got $%.4f", pricing.PromptCostPer1KTokens)
	}

	// Update pricing
	newCfg := &config.CostsConfig{
		Pricing: map[string]map[string]config.ModelPricingConfig{
			"openai": {
				"gpt-4": {
					Prompt:     0.02, // Updated price
					Completion: 0.04,
				},
			},
		},
	}

	calculator.UpdatePricing(newCfg)

	// Get updated pricing
	pricing, err = calculator.GetModelPricing("gpt-4", "openai")
	if err != nil {
		t.Fatalf("failed to get updated pricing: %v", err)
	}
	if pricing.PromptCostPer1KTokens != 0.02 {
		t.Errorf("expected updated prompt cost $0.02, got $%.4f", pricing.PromptCostPer1KTokens)
	}
}

func TestCalculateTokenCost(t *testing.T) {
	tests := []struct {
		name         string
		tokens       int
		costPer1K    float64
		expectedCost float64
	}{
		{
			name:         "zero tokens",
			tokens:       0,
			costPer1K:    0.03,
			expectedCost: 0.0,
		},
		{
			name:         "1000 tokens",
			tokens:       1000,
			costPer1K:    0.03,
			expectedCost: 0.03,
		},
		{
			name:         "500 tokens",
			tokens:       500,
			costPer1K:    0.06,
			expectedCost: 0.03,
		},
		{
			name:         "100 tokens",
			tokens:       100,
			costPer1K:    0.03,
			expectedCost: 0.003,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := calculateTokenCost(tt.tokens, tt.costPer1K)
			diff := cost - tt.expectedCost
			if diff > 0.0001 || diff < -0.0001 {
				t.Errorf("expected cost $%.6f, got $%.6f", tt.expectedCost, cost)
			}
		})
	}
}
