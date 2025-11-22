// Package costs provides cost calculation for LLM requests and responses.
//
// This package implements provider-specific cost calculation based on token usage
// and model pricing. It supports:
//
//   - Per-token pricing (input/output tokens)
//   - Provider-specific pricing tiers
//   - Cached token pricing (where applicable)
//   - Hot-reload of pricing configuration
//
// # Pricing Model
//
// Costs are calculated using provider and model-specific pricing per 1K tokens:
//
//   - Input (prompt) tokens: Typically lower cost
//   - Output (completion) tokens: Typically 2-3x input cost
//   - Cached tokens: Discounted rate (where supported)
//
// # Usage
//
// Calculate cost for a request estimate:
//
//	cfg := config.GetConfig()
//	calculator := costs.NewCalculator(&cfg.Processing.Costs)
//
//	// Calculate cost from token estimate
//	cost, err := calculator.CalculateRequestCost(estimate, "gpt-4", "openai")
//	if err != nil {
//		log.Error("cost calculation failed", "error", err)
//	}
//
//	fmt.Printf("Estimated cost: $%.4f\n", cost.TotalCost)
//
// Calculate cost for an actual response:
//
//	// Calculate cost from actual usage
//	cost, err := calculator.CalculateResponseCost(usage, "gpt-4", "openai")
//	if err != nil {
//		log.Warn("cost calculation failed", "error", err)
//	}
//
//	fmt.Printf("Actual cost: $%.4f (prompt: $%.4f, completion: $%.4f)\n",
//		cost.TotalCost, cost.PromptCost, cost.CompletionCost)
//
// # Pricing Updates
//
// Pricing can be updated dynamically by reloading configuration. The calculator
// uses read-write locks for thread-safe access to pricing data.
package costs
