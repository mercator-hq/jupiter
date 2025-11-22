package tokens

import (
	"mercator-hq/jupiter/pkg/proxy/types"
)

// Estimator estimates token counts for text and messages.
// Implementations may use different algorithms (character-based, BPE, tiktoken, etc.).
type Estimator interface {
	// EstimateText estimates tokens for a single text string.
	EstimateText(text string, model string) (int, error)

	// EstimateMessages estimates tokens for a list of messages.
	// Returns total prompt tokens including overhead.
	EstimateMessages(messages []types.Message, model string) (int, error)

	// EstimateTools estimates tokens for tool/function definitions.
	EstimateTools(tools []types.Tool, model string) (int, error)

	// EstimateRequest estimates all tokens for a complete request.
	// This includes messages, system prompts, tools, and formatting overhead.
	EstimateRequest(req *types.ChatCompletionRequest) (*Estimate, error)
}

// Estimate contains detailed token estimation results.
type Estimate struct {
	// PromptTokens is the estimated number of tokens in the prompt.
	PromptTokens int

	// EstimatedCompletionTokens is the estimated number of completion tokens.
	// This is typically based on MaxTokens if specified, or a default estimate.
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
	// Character-based estimators typically have 0.95 confidence.
	Confidence float64
}
