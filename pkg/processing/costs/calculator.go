package costs

import (
	"fmt"
	"strings"
	"sync"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/processing/tokens"
	"mercator-hq/jupiter/pkg/providers"
)

// Calculator calculates costs for LLM requests and responses based on token usage.
// It is thread-safe and supports hot-reload of pricing configuration.
type Calculator struct {
	// config contains cost calculation configuration
	config *config.CostsConfig

	// mu protects the calculator for concurrent access
	mu sync.RWMutex
}

// NewCalculator creates a new cost calculator with the given configuration.
func NewCalculator(cfg *config.CostsConfig) *Calculator {
	return &Calculator{
		config: cfg,
	}
}

// CalculateRequestCost calculates the estimated cost for a request based on token estimates.
// Returns a cost estimate with prompt and completion costs.
func (c *Calculator) CalculateRequestCost(estimate *tokens.Estimate, model, provider string) (*CostEstimate, error) {
	if estimate == nil {
		return nil, fmt.Errorf("estimate cannot be nil")
	}

	pricing, err := c.GetModelPricing(model, provider)
	if err != nil {
		// Use default pricing and log warning
		pricing, _ = c.GetModelPricing("default", "default")
	}

	costEst := &CostEstimate{
		Model:       model,
		Provider:    provider,
		PricingTier: "standard",
		Currency:    "USD",
	}

	// Calculate prompt cost
	costEst.PromptCost = calculateTokenCost(estimate.PromptTokens, pricing.PromptCostPer1KTokens)

	// Calculate estimated completion cost
	costEst.CompletionCost = calculateTokenCost(estimate.EstimatedCompletionTokens, pricing.CompletionCostPer1KTokens)

	// Calculate total cost
	costEst.TotalCost = costEst.PromptCost + costEst.CompletionCost

	return costEst, nil
}

// CalculateResponseCost calculates the actual cost for a response based on actual token usage.
// Returns a cost estimate with actual costs from provider usage data.
func (c *Calculator) CalculateResponseCost(usage *TokenUsage, model, provider string) (*CostEstimate, error) {
	if usage == nil {
		return nil, fmt.Errorf("usage cannot be nil")
	}

	pricing, err := c.GetModelPricing(model, provider)
	if err != nil {
		// Use default pricing and log warning
		pricing, _ = c.GetModelPricing("default", "default")
	}

	costEst := &CostEstimate{
		Model:       model,
		Provider:    provider,
		PricingTier: "standard",
		Currency:    "USD",
	}

	// Calculate prompt cost (considering cached tokens if applicable)
	promptTokens := usage.PromptTokens
	if usage.CachedTokens > 0 && pricing.CachedPromptCostPer1KTokens > 0 {
		// Some tokens are cached at a discounted rate
		uncachedTokens := promptTokens - usage.CachedTokens
		if uncachedTokens < 0 {
			uncachedTokens = 0
		}

		costEst.PromptCost = calculateTokenCost(uncachedTokens, pricing.PromptCostPer1KTokens) +
			calculateTokenCost(usage.CachedTokens, pricing.CachedPromptCostPer1KTokens)
	} else {
		costEst.PromptCost = calculateTokenCost(promptTokens, pricing.PromptCostPer1KTokens)
	}

	// Calculate completion cost
	costEst.CompletionCost = calculateTokenCost(usage.CompletionTokens, pricing.CompletionCostPer1KTokens)

	// Calculate total cost
	costEst.TotalCost = costEst.PromptCost + costEst.CompletionCost

	return costEst, nil
}

// CalculateProviderResponseCost calculates cost from a provider's completion response.
// This is a convenience method that extracts usage and calls CalculateResponseCost.
func (c *Calculator) CalculateProviderResponseCost(resp *providers.CompletionResponse, provider string) (*CostEstimate, error) {
	if resp == nil {
		return nil, fmt.Errorf("response cannot be nil")
	}

	usage := &TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	return c.CalculateResponseCost(usage, resp.Model, provider)
}

// GetModelPricing retrieves pricing information for a specific model and provider.
// It first tries exact match, then model prefix match, then default pricing.
func (c *Calculator) GetModelPricing(model, provider string) (*ModelPricing, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact provider and model match
	if providerPricing, ok := c.config.Pricing[provider]; ok {
		if modelConfig, ok := providerPricing[model]; ok {
			return &ModelPricing{
				Model:                       model,
				Provider:                    provider,
				PromptCostPer1KTokens:       modelConfig.Prompt,
				CompletionCostPer1KTokens:   modelConfig.Completion,
				CachedPromptCostPer1KTokens: modelConfig.CachedPrompt,
				Currency:                    "USD",
			}, nil
		}

		// Try model prefix match (e.g., "gpt-4" matches "gpt-4-0613")
		for modelPattern, modelConfig := range providerPricing {
			if strings.HasPrefix(model, modelPattern) {
				return &ModelPricing{
					Model:                       model,
					Provider:                    provider,
					PromptCostPer1KTokens:       modelConfig.Prompt,
					CompletionCostPer1KTokens:   modelConfig.Completion,
					CachedPromptCostPer1KTokens: modelConfig.CachedPrompt,
					Currency:                    "USD",
				}, nil
			}
		}
	}

	// Fall back to default pricing
	if defaultProvider, ok := c.config.Pricing["default"]; ok {
		if defaultModel, ok := defaultProvider["default"]; ok {
			return &ModelPricing{
				Model:                     model,
				Provider:                  provider,
				PromptCostPer1KTokens:     defaultModel.Prompt,
				CompletionCostPer1KTokens: defaultModel.Completion,
				Currency:                  "USD",
			}, nil
		}
	}

	return nil, fmt.Errorf("no pricing found for model %q and provider %q", model, provider)
}

// UpdatePricing updates the pricing configuration (hot-reload support).
// This is thread-safe and can be called while the calculator is in use.
func (c *Calculator) UpdatePricing(newConfig *config.CostsConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = newConfig
}

// ModelPricing contains pricing information for a specific model.
type ModelPricing struct {
	// Model is the model identifier.
	Model string

	// Provider is the provider name.
	Provider string

	// PromptCostPer1KTokens is the cost per 1000 prompt tokens in USD.
	PromptCostPer1KTokens float64

	// CompletionCostPer1KTokens is the cost per 1000 completion tokens in USD.
	CompletionCostPer1KTokens float64

	// CachedPromptCostPer1KTokens is the cost per 1000 cached prompt tokens in USD.
	CachedPromptCostPer1KTokens float64

	// MinimumCost is the minimum cost per request (if applicable).
	MinimumCost float64

	// Currency is the currency code (always "USD" for MVP).
	Currency string
}

// calculateTokenCost calculates the cost for a given number of tokens.
// costPer1K is the cost per 1000 tokens in USD.
func calculateTokenCost(tokens int, costPer1K float64) float64 {
	if tokens <= 0 {
		return 0.0
	}

	return (float64(tokens) / 1000.0) * costPer1K
}
