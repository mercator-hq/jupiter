package costs

// CostEstimate contains cost calculations in USD.
// Costs are calculated based on provider-specific pricing and token usage.
type CostEstimate struct {
	// PromptCost is the cost for prompt tokens in USD.
	PromptCost float64

	// CompletionCost is the cost for completion tokens in USD.
	CompletionCost float64

	// TotalCost is the total cost in USD.
	TotalCost float64

	// Model is the model used for pricing.
	Model string

	// Provider is the provider name (openai, anthropic, etc.).
	Provider string

	// PricingTier identifies the pricing tier used.
	PricingTier string

	// Currency is the currency code (always "USD" for MVP).
	Currency string
}

// TokenUsage contains actual token counts from the provider response.
// This is extracted from the provider's usage statistics.
type TokenUsage struct {
	// PromptTokens is the actual number of tokens in the prompt.
	PromptTokens int

	// CompletionTokens is the actual number of tokens in the completion.
	CompletionTokens int

	// TotalTokens is the total number of tokens used.
	TotalTokens int

	// CachedTokens is the number of cached tokens (if provider supports caching).
	CachedTokens int
}
