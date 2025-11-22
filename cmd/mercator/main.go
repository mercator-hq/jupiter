// Mercator Jupiter is a GitOps-native LLM governance runtime and policy engine.
//
// It acts as an HTTP proxy for LLM API requests, providing:
//   - Policy-based request governance and routing
//   - Multi-provider LLM routing (OpenAI, Anthropic, etc.)
//   - Cryptographic evidence generation for audit trails
//   - Cost tracking and budget enforcement
//   - Content analysis and PII detection
//
// Usage:
//
//	# Start server with default configuration
//	mercator run
//
//	# Start with custom configuration file
//	mercator run --config /path/to/config.yaml
//
//	# Show version information
//	mercator version
//
//	# Validate policy files
//	mercator lint --file policies.yaml
//
//	# Run policy tests
//	mercator test --policy policies.yaml --tests policy_tests.yaml
//
//	# Query evidence database
//	mercator evidence query --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"
//
// For complete documentation, see: https://github.com/mercator-hq/jupiter
package main

func main() {
	Execute()
}
