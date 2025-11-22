package processing

import (
	"fmt"
	"strings"
	"time"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/processing/content"
	"mercator-hq/jupiter/pkg/processing/conversation"
	"mercator-hq/jupiter/pkg/processing/costs"
	"mercator-hq/jupiter/pkg/processing/tokens"
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// Processor orchestrates all request/response processing components.
// It is thread-safe and can process multiple requests concurrently.
type Processor struct {
	tokenEstimator       tokens.Estimator
	costCalculator       *costs.Calculator
	contentAnalyzer      *content.Analyzer
	conversationAnalyzer *conversation.Analyzer
}

// NewProcessor creates a new processor with the given configuration.
func NewProcessor(cfg *config.ProcessingConfig) *Processor {
	return &Processor{
		tokenEstimator:       tokens.NewSimpleEstimator(&cfg.Tokens),
		costCalculator:       costs.NewCalculator(&cfg.Costs),
		contentAnalyzer:      content.NewAnalyzer(&cfg.Content),
		conversationAnalyzer: conversation.NewAnalyzer(&cfg.Conversation),
	}
}

// ProcessRequest enriches a request with all available metadata.
// This includes token estimation, cost estimation, content analysis, and conversation analysis.
func (p *Processor) ProcessRequest(requestMeta *proxy.RequestMetadata, req *types.ChatCompletionRequest) (*EnrichedRequest, error) {
	startTime := time.Now()

	enriched := &EnrichedRequest{
		RequestID:       requestMeta.RequestID,
		OriginalRequest: req,
	}

	// Estimate tokens
	tokenEst, err := p.tokenEstimator.EstimateRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate tokens: %w", err)
	}

	enriched.TokenEstimate = &TokenEstimate{
		PromptTokens:              tokenEst.PromptTokens,
		EstimatedCompletionTokens: tokenEst.EstimatedCompletionTokens,
		TotalTokens:               tokenEst.TotalTokens,
		SystemPromptTokens:        tokenEst.SystemPromptTokens,
		MessageTokens:             tokenEst.MessageTokens,
		ToolTokens:                tokenEst.ToolTokens,
		OverheadTokens:            tokenEst.OverheadTokens,
		Model:                     tokenEst.Model,
		Confidence:                tokenEst.Confidence,
	}

	// Estimate cost (use provider from metadata if available)
	provider := inferProvider(req.Model)
	costEst, err := p.costCalculator.CalculateRequestCost(tokenEst, req.Model, provider)
	if err == nil {
		enriched.CostEstimate = costEst
	}

	// Analyze content (combine all message content)
	contentText := combineMessageContent(req.Messages)
	if contentText != "" {
		contentAnalysis, err := p.contentAnalyzer.AnalyzeText(contentText)
		if err == nil {
			enriched.ContentAnalysis = contentAnalysis
		}
	}

	// Analyze conversation
	conversationCtx, err := p.conversationAnalyzer.AnalyzeConversation(req.Messages, req.Model, tokenEst.PromptTokens)
	if err == nil {
		enriched.ConversationContext = conversationCtx
	}

	// Determine model family
	enriched.ModelFamily = inferModelFamily(req.Model)

	// Set pricing tier (default to standard for MVP)
	enriched.PricingTier = "standard"

	// Estimate latency based on token count (rough estimate)
	enriched.EstimatedLatency = estimateLatency(tokenEst.TotalTokens, req.Model)

	// Calculate complexity score (1-10) based on various factors
	enriched.ComplexityScore = calculateComplexityScore(req, tokenEst)

	// Calculate risk score (1-10) based on content analysis
	enriched.RiskScore = calculateRiskScore(enriched.ContentAnalysis)

	enriched.ProcessingDuration = time.Since(startTime)

	return enriched, nil
}

// ProcessResponse enriches a response with all available metadata.
// This includes actual token usage, actual costs, and response quality metrics.
func (p *Processor) ProcessResponse(requestID string, responseMeta *proxy.ResponseMetadata, resp *providers.CompletionResponse) (*EnrichedResponse, error) {
	startTime := time.Now()

	enriched := &EnrichedResponse{
		RequestID:        requestID,
		OriginalResponse: resp,
	}

	// Extract actual token usage
	enriched.TokenUsage = &TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	// Calculate actual cost (use provider from response metadata if available)
	provider := inferProvider(resp.Model)
	costEst, err := p.costCalculator.CalculateResponseCost(enriched.TokenUsage, resp.Model, provider)
	if err == nil {
		enriched.CostEstimate = costEst
	}

	// Analyze response content
	if resp.Content != "" {
		contentAnalysis, err := p.contentAnalyzer.AnalyzeText(resp.Content)
		if err == nil {
			enriched.ContentAnalysis = contentAnalysis
		}
	}

	// Analyze finish reason
	enriched.FinishReasonAnalysis = analyzeFinishReason(resp.FinishReason)

	// Calculate token efficiency (completion tokens / total tokens)
	if enriched.TokenUsage.TotalTokens > 0 {
		enriched.TokenEfficiency = float64(enriched.TokenUsage.CompletionTokens) / float64(enriched.TokenUsage.TotalTokens)
	}

	// Calculate latency breakdown
	enriched.LatencyBreakdown = &LatencyBreakdown{
		TotalLatency:       responseMeta.Latency,
		ProviderProcessing: responseMeta.ProviderLatency,
		NetworkLatency:     responseMeta.Latency - responseMeta.ProviderLatency,
	}

	// Calculate quality metrics (simplified for MVP)
	enriched.QualityMetrics = calculateQualityMetrics(resp, enriched.ContentAnalysis)

	enriched.ProcessingDuration = time.Since(startTime)

	return enriched, nil
}

// inferProvider infers the provider from the model name.
func inferProvider(model string) string {
	modelLower := strings.ToLower(model)

	if strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "openai") {
		return "openai"
	}

	if strings.Contains(modelLower, "claude") || strings.Contains(modelLower, "anthropic") {
		return "anthropic"
	}

	if strings.Contains(modelLower, "gemini") || strings.Contains(modelLower, "palm") {
		return "google"
	}

	return "default"
}

// inferModelFamily determines the model family from the model name.
func inferModelFamily(model string) string {
	modelLower := strings.ToLower(model)

	if strings.Contains(modelLower, "gpt-4") {
		return "GPT-4"
	}

	if strings.Contains(modelLower, "gpt-3.5") {
		return "GPT-3.5"
	}

	if strings.Contains(modelLower, "claude-3-opus") {
		return "Claude 3 Opus"
	}

	if strings.Contains(modelLower, "claude-3-sonnet") {
		return "Claude 3 Sonnet"
	}

	if strings.Contains(modelLower, "claude-3-haiku") {
		return "Claude 3 Haiku"
	}

	if strings.Contains(modelLower, "claude") {
		return "Claude"
	}

	return "Unknown"
}

// combineMessageContent combines all message content into a single string for analysis.
func combineMessageContent(messages []types.Message) string {
	var parts []string

	for _, msg := range messages {
		if content, ok := msg.Content.(string); ok && content != "" {
			parts = append(parts, content)
		}
	}

	return strings.Join(parts, " ")
}

// estimateLatency estimates response latency based on token count and model.
// This is a very rough estimate for planning purposes.
func estimateLatency(tokens int, model string) time.Duration {
	// Base latency (network + overhead)
	baseLatency := 500 * time.Millisecond

	// Per-token latency varies by model
	perTokenLatency := 5 * time.Millisecond // ~200 tokens/second

	if strings.Contains(strings.ToLower(model), "turbo") {
		perTokenLatency = 3 * time.Millisecond // Faster models
	}

	return baseLatency + time.Duration(tokens)*perTokenLatency
}

// calculateComplexityScore calculates a complexity score from 1-10.
func calculateComplexityScore(req *types.ChatCompletionRequest, tokenEst *tokens.Estimate) int {
	score := 1

	// Factor in token count
	if tokenEst.TotalTokens > 10000 {
		score += 3
	} else if tokenEst.TotalTokens > 5000 {
		score += 2
	} else if tokenEst.TotalTokens > 1000 {
		score += 1
	}

	// Factor in number of messages (conversation complexity)
	if len(req.Messages) > 20 {
		score += 3
	} else if len(req.Messages) > 10 {
		score += 2
	} else if len(req.Messages) > 5 {
		score += 1
	}

	// Factor in tools/functions
	if len(req.Tools) > 5 {
		score += 3
	} else if len(req.Tools) > 0 {
		score += 1
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

// calculateRiskScore calculates a risk score from 1-10 based on content analysis.
func calculateRiskScore(analysis *ContentAnalysis) int {
	if analysis == nil {
		return 1
	}

	score := 1

	// Factor in PII detection
	if analysis.PIIDetection != nil && analysis.PIIDetection.HasPII {
		score += 2
		if analysis.PIIDetection.PIICount > 5 {
			score += 2
		}
	}

	// Factor in sensitive content
	if analysis.SensitiveContent != nil && analysis.SensitiveContent.HasSensitiveContent {
		switch analysis.SensitiveContent.Severity {
		case "critical":
			score += 5
		case "high":
			score += 3
		case "medium":
			score += 2
		case "low":
			score += 1
		}
	}

	// Factor in prompt injection
	if analysis.PromptInjection != nil && analysis.PromptInjection.HasPromptInjection {
		score += 3
		if analysis.PromptInjection.Confidence > 0.9 {
			score += 2
		}
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

// analyzeFinishReason analyzes the finish reason and provides actionable insights.
func analyzeFinishReason(reason string) *FinishReasonAnalysis {
	analysis := &FinishReasonAnalysis{
		FinishReason: reason,
	}

	switch reason {
	case "stop", "end_turn":
		analysis.IsExpected = true
		analysis.RequiresAction = false
		analysis.Description = "Completion finished naturally"
		analysis.Severity = "info"

	case "length", "max_tokens":
		analysis.IsExpected = false
		analysis.RequiresAction = true
		analysis.Description = "Completion truncated due to length limit. Consider increasing max_tokens."
		analysis.Severity = "warning"

	case "content_filter":
		analysis.IsExpected = false
		analysis.RequiresAction = true
		analysis.Description = "Content was filtered due to safety policies. Review request content."
		analysis.Severity = "error"

	case "tool_calls", "function_call":
		analysis.IsExpected = true
		analysis.RequiresAction = false
		analysis.Description = "Model requested tool/function call"
		analysis.Severity = "info"

	default:
		analysis.IsExpected = false
		analysis.RequiresAction = false
		analysis.Description = fmt.Sprintf("Unknown finish reason: %s", reason)
		analysis.Severity = "warning"
	}

	return analysis
}

// calculateQualityMetrics calculates response quality metrics (simplified for MVP).
func calculateQualityMetrics(resp *providers.CompletionResponse, contentAnalysis *ContentAnalysis) *QualityMetrics {
	metrics := &QualityMetrics{
		Coherence: 0.8, // Default - would use ML in production
		Relevance: 0.8, // Default - would use ML in production
		Safety:    1.0, // Assume safe unless flagged
	}

	// Adjust safety based on content analysis
	if contentAnalysis != nil {
		if contentAnalysis.SensitiveContent != nil && contentAnalysis.SensitiveContent.HasSensitiveContent {
			switch contentAnalysis.SensitiveContent.Severity {
			case "critical":
				metrics.Safety = 0.2
			case "high":
				metrics.Safety = 0.5
			case "medium":
				metrics.Safety = 0.7
			}
		}

		// Adjust coherence based on sentiment (very rough heuristic)
		if contentAnalysis.Sentiment != nil {
			metrics.Coherence = 0.7 + (contentAnalysis.Sentiment.Confidence * 0.3)
		}
	}

	// Calculate overall score (weighted average)
	metrics.OverallScore = (metrics.Coherence*0.3 + metrics.Relevance*0.3 + metrics.Safety*0.4)

	return metrics
}
