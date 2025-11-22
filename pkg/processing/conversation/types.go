package conversation

// ConversationContext contains conversation history analysis.
// Analyzes multi-turn conversations for context window usage and complexity.
type ConversationContext struct {
	// TurnCount is the number of user/assistant message pairs.
	TurnCount int

	// MessageCount is the total number of messages.
	MessageCount int

	// SystemPrompts contains extracted system prompts.
	SystemPrompts []string

	// ContextWindowUsage is the total tokens used in the context window.
	ContextWindowUsage int

	// ContextWindowPercent is the percentage of context window used.
	ContextWindowPercent float64

	// HasConversationHistory indicates if this is a multi-turn conversation.
	HasConversationHistory bool

	// AverageMessageLength is the average message length in tokens.
	AverageMessageLength int
}
