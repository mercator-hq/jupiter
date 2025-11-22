package processing

import (
	"time"

	"mercator-hq/jupiter/pkg/processing/content"
	"mercator-hq/jupiter/pkg/processing/conversation"
	"mercator-hq/jupiter/pkg/processing/costs"
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// Re-export content analysis types for convenience
type (
	ContentAnalysis  = content.ContentAnalysis
	PIIDetection     = content.PIIDetection
	PIILocation      = content.PIILocation
	SensitiveContent = content.SensitiveContent
	PromptInjection  = content.PromptInjection
	Sentiment        = content.Sentiment
)

// Re-export conversation types for convenience
type ConversationContext = conversation.ConversationContext

// Re-export cost types for convenience
type (
	CostEstimate = costs.CostEstimate
	TokenUsage   = costs.TokenUsage
)

// EnrichedRequest contains the original request plus all computed metadata.
// This structure is used by the Policy Engine for evaluation and Evidence
// Generation for audit records.
type EnrichedRequest struct {
	// RequestID is the unique identifier for this request.
	RequestID string

	// OriginalRequest is the chat completion request from the client.
	OriginalRequest *types.ChatCompletionRequest

	// TokenEstimate contains estimated token counts for the request.
	TokenEstimate *TokenEstimate

	// CostEstimate contains estimated costs for the request.
	CostEstimate *CostEstimate

	// ContentAnalysis contains content safety and PII detection results.
	ContentAnalysis *ContentAnalysis

	// ConversationContext contains conversation history analysis.
	ConversationContext *ConversationContext

	// ModelFamily identifies the model family (GPT-4, Claude 3, Llama, etc.).
	ModelFamily string

	// PricingTier identifies the provider pricing tier (standard, enterprise, custom).
	PricingTier string

	// EstimatedLatency is the predicted latency based on token count and model.
	EstimatedLatency time.Duration

	// ComplexityScore rates request complexity from 1-10 based on tokens, tools, etc.
	ComplexityScore int

	// RiskScore rates request risk from 1-10 based on content analysis.
	RiskScore int

	// ProcessingDuration is the time taken to enrich this request.
	ProcessingDuration time.Duration
}

// EnrichedResponse contains the original response plus all computed metadata.
// This structure is used for logging, metrics, and evidence generation.
type EnrichedResponse struct {
	// RequestID is the unique identifier for the request.
	RequestID string

	// OriginalResponse is the provider's completion response.
	OriginalResponse *providers.CompletionResponse

	// TokenUsage contains actual token counts from the provider.
	TokenUsage *TokenUsage

	// CostEstimate contains actual costs based on usage.
	CostEstimate *CostEstimate

	// ContentAnalysis contains analysis of the response content.
	ContentAnalysis *ContentAnalysis

	// FinishReasonAnalysis explains why the model stopped generating.
	FinishReasonAnalysis *FinishReasonAnalysis

	// TokenEfficiency is the ratio of output tokens to input tokens.
	TokenEfficiency float64

	// LatencyBreakdown contains timing information for the response.
	LatencyBreakdown *LatencyBreakdown

	// QualityMetrics contains response quality scores.
	QualityMetrics *QualityMetrics

	// ProcessingDuration is the time taken to enrich this response.
	ProcessingDuration time.Duration
}

// TokenEstimate contains estimated token counts for a request.
// Estimates are made before sending to the provider using tiktoken-style algorithms.
type TokenEstimate struct {
	// PromptTokens is the estimated number of tokens in the prompt.
	PromptTokens int

	// EstimatedCompletionTokens is the estimated number of completion tokens.
	EstimatedCompletionTokens int

	// TotalTokens is the total estimated tokens (prompt + completion).
	TotalTokens int

	// SystemPromptTokens is the token count for system prompts.
	SystemPromptTokens int

	// MessageTokens is the token count for user/assistant messages.
	MessageTokens int

	// ToolTokens is the token count for tool/function definitions.
	ToolTokens int

	// OverheadTokens are additional tokens for formatting and special tokens.
	OverheadTokens int

	// Model is the model used for estimation.
	Model string

	// Confidence is the estimation confidence from 0.0 (low) to 1.0 (high).
	Confidence float64
}

// FinishReasonAnalysis explains why the model stopped generating tokens.
// Provides actionable insights based on the finish reason.
type FinishReasonAnalysis struct {
	// FinishReason is the raw finish reason from the provider.
	FinishReason string

	// IsExpected indicates if this finish reason is expected/normal.
	IsExpected bool

	// RequiresAction indicates if user action is needed.
	RequiresAction bool

	// Description explains what the finish reason means.
	Description string

	// Severity indicates the severity level (info, warning, error).
	Severity string
}

// LatencyBreakdown provides timing information for the response.
// Helps identify performance bottlenecks.
type LatencyBreakdown struct {
	// NetworkLatency is the time spent on network communication.
	NetworkLatency time.Duration

	// ProviderProcessing is the time spent by the provider processing the request.
	ProviderProcessing time.Duration

	// StreamingOverhead is the additional time for streaming responses.
	StreamingOverhead time.Duration

	// TotalLatency is the total time from request to response.
	TotalLatency time.Duration
}

// QualityMetrics contains response quality scores.
// Used for monitoring and alerting on response quality issues.
type QualityMetrics struct {
	// Coherence measures response coherence from 0.0 to 1.0.
	Coherence float64

	// Relevance measures response relevance from 0.0 to 1.0.
	Relevance float64

	// Safety measures content safety from 0.0 to 1.0.
	Safety float64

	// OverallScore is the combined quality score from 0.0 to 1.0.
	OverallScore float64
}

// RequestProcessor processes incoming requests and enriches them with metadata.
// This is the main interface for request processing.
type RequestProcessor interface {
	// ProcessRequest enriches a request with all available metadata.
	ProcessRequest(requestMeta *proxy.RequestMetadata, req *types.ChatCompletionRequest) (*EnrichedRequest, error)
}

// ResponseProcessor processes provider responses and enriches them with metadata.
// This is the main interface for response processing.
type ResponseProcessor interface {
	// ProcessResponse enriches a response with all available metadata.
	ProcessResponse(requestID string, responseMeta *proxy.ResponseMetadata, resp *providers.CompletionResponse) (*EnrichedResponse, error)
}
