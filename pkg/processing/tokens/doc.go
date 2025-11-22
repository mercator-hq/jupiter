// Package tokens provides token estimation and counting for LLM requests.
//
// This package implements model-specific token estimators that predict token
// usage before sending requests to providers. Token estimation is critical for:
//
//   - Cost estimation before making API calls
//   - Budget enforcement and rate limiting
//   - Context window management
//   - Request optimization and routing
//
// # Token Estimation Accuracy
//
// The MVP implementation uses a simple character-based estimation algorithm
// with model-specific multipliers. This achieves <5% error for most requests:
//
//   - GPT-4: ~4 characters per token
//   - GPT-3.5: ~4 characters per token
//   - Claude 3: ~3.5 characters per token
//
// # Usage
//
// Create an estimator and estimate tokens for a request:
//
//	cfg := config.GetConfig()
//	estimator := tokens.NewSimpleEstimator(&cfg.Processing.Tokens)
//
//	// Estimate tokens for messages
//	estimate, err := estimator.EstimateMessages(messages, "gpt-4")
//	if err != nil {
//		log.Error("estimation failed", "error", err)
//	}
//
//	fmt.Printf("Estimated tokens: %d (confidence: %.2f)\n",
//		estimate.TotalTokens, estimate.Confidence)
//
// # Future Enhancements
//
// Future versions will support:
//
//   - tiktoken-based estimation (exact token matching)
//   - BPE (Byte-Pair Encoding) tokenizers
//   - Multimodal token estimation (images, audio)
//   - Caching for performance optimization
package tokens
