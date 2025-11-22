// Package processing provides comprehensive request/response processing for LLM governance.
//
// This package extracts detailed metadata, estimates token usage, calculates costs,
// parses conversation history, and provides rich context for policy evaluation.
// It transforms raw HTTP requests and provider responses into structured metadata
// that enables intelligent governance decisions, budget tracking, and evidence generation.
//
// # Architecture
//
// The processing package is organized into specialized sub-packages:
//
//   - tokens: Token estimation and counting using model-specific algorithms
//   - costs: Cost calculation based on provider-specific pricing
//   - content: Content analysis including PII detection, sensitive content, prompt injection
//   - conversation: Conversation history parsing and context window analysis
//
// # Basic Usage
//
// Create a request processor and enrich incoming requests:
//
//	cfg := config.GetConfig()
//	processor := processing.NewRequestProcessor(&cfg.Processing)
//
//	// Enrich request with token estimates, cost, content analysis
//	enriched, err := processor.ProcessRequest(ctx, requestMetadata, chatRequest)
//	if err != nil {
//		log.Error("processing failed", "error", err)
//	}
//
//	// Use enriched metadata for policy evaluation
//	if enriched.RiskScore > 8 {
//		// High-risk request - apply stricter policies
//	}
//
// Process provider responses with actual usage data:
//
//	respProcessor := processing.NewResponseProcessor(&cfg.Processing)
//
//	// Enrich response with actual costs, efficiency metrics
//	enriched, err := respProcessor.ProcessResponse(ctx, responseMetadata, providerResponse)
//	if err != nil {
//		log.Error("response processing failed", "error", err)
//	}
//
//	// Check token efficiency
//	if enriched.TokenEfficiency < 0.1 {
//		log.Warn("low token efficiency", "ratio", enriched.TokenEfficiency)
//	}
//
// # Performance
//
// All processing operations are designed to complete in <10ms per request:
//
//   - Token estimation: <1ms
//   - Cost calculation: <100µs
//   - Content analysis: <5ms
//   - Conversation analysis: <2ms
//
// The package is thread-safe and supports concurrent processing of 1000+ requests/second.
//
// # Integration
//
// The processing package integrates with:
//
//   - Configuration System: Reads pricing and analysis rules
//   - HTTP Proxy: Enriches request/response metadata
//   - Provider Adapters: Uses actual token counts from responses
//   - Policy Engine: Provides enriched metadata for policy evaluation
//   - Evidence Generation: Supplies detailed metadata for audit records
package processing
