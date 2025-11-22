package conversation

import (
	"strings"
	"sync"

	"mercator-hq/jupiter/pkg/config"
		"mercator-hq/jupiter/pkg/proxy/types"
)

// Analyzer analyzes conversation history for context window management
// and conversation complexity tracking.
type Analyzer struct {
	config *config.ConversationConfig

	// mu protects the analyzer for concurrent access
	mu sync.RWMutex
}

// NewAnalyzer creates a new conversation analyzer with the given configuration.
func NewAnalyzer(cfg *config.ConversationConfig) *Analyzer {
	return &Analyzer{
		config: cfg,
	}
}

// AnalyzeConversation analyzes a conversation history.
// Takes messages, model name, and total token count (from token estimator).
func (a *Analyzer) AnalyzeConversation(messages []types.Message, model string, totalTokens int) (*ConversationContext, error) {
	ctx := &ConversationContext{
		SystemPrompts: make([]string, 0),
	}

	if len(messages) == 0 {
		return ctx, nil
	}

	ctx.MessageCount = len(messages)
	ctx.ContextWindowUsage = totalTokens

	// Extract system prompts and count turns
	userMessages := 0
	assistantMessages := 0

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// Extract system prompt content
			content := extractMessageContent(msg.Content)
			if content != "" {
				ctx.SystemPrompts = append(ctx.SystemPrompts, content)
			}

		case "user":
			userMessages++

		case "assistant":
			assistantMessages++
		}
	}

	// Calculate turn count (a turn is a user message + assistant response)
	// If there are more user messages, some turns are incomplete
	ctx.TurnCount = assistantMessages
	if userMessages > assistantMessages {
		ctx.TurnCount = userMessages
	}

	// Determine if this is a multi-turn conversation
	ctx.HasConversationHistory = ctx.TurnCount > 1 || assistantMessages > 0

	// Calculate average message length (in tokens)
	if ctx.MessageCount > 0 {
		ctx.AverageMessageLength = totalTokens / ctx.MessageCount
	}

	// Get context window limit for this model
	maxContextWindow := a.getContextWindowLimit(model)

	// Calculate context window percentage
	if maxContextWindow > 0 {
		ctx.ContextWindowPercent = float64(totalTokens) / float64(maxContextWindow)
	}

	return ctx, nil
}

// getContextWindowLimit returns the context window limit for a model.
func (a *Analyzer) getContextWindowLimit(model string) int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Try exact model match
	if limit, ok := a.config.MaxContextWindow[model]; ok {
		return limit
	}

	// Try prefix match - find the longest matching prefix
	// This ensures "gpt-4-turbo" is matched before "gpt-4"
	longestMatch := ""
	matchedLimit := 0

	for modelPattern, limit := range a.config.MaxContextWindow {
		if modelPattern == "default" {
			continue
		}
		if strings.HasPrefix(model, modelPattern) && len(modelPattern) > len(longestMatch) {
			longestMatch = modelPattern
			matchedLimit = limit
		}
	}

	if longestMatch != "" {
		return matchedLimit
	}

	// Fall back to default
	if limit, ok := a.config.MaxContextWindow["default"]; ok {
		return limit
	}

	// Ultimate fallback
	return 4096
}

// extractMessageContent extracts text content from a message's Content field.
func extractMessageContent(content interface{}) string {
	if content == nil {
		return ""
	}

	// Handle string content
	if str, ok := content.(string); ok {
		return str
	}

	// Handle array content (multimodal) - extract text parts
	if arr, ok := content.([]interface{}); ok {
		var textParts []string

		for _, part := range arr {
			if partMap, ok := part.(map[string]interface{}); ok {
				if contentType, ok := partMap["type"].(string); ok && contentType == "text" {
					if text, ok := partMap["text"].(string); ok {
						textParts = append(textParts, text)
					}
				}
			}
		}

		return strings.Join(textParts, " ")
	}

	// Unknown content type
	return ""
}
