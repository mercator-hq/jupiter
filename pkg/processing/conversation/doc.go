// Package conversation provides conversation history analysis for LLM requests.
//
// This package analyzes multi-turn conversations to extract:
//
//   - Turn count (user/assistant message pairs)
//   - Context window usage and percentage
//   - System prompts
//   - Conversation complexity metrics
//
// # Context Window Management
//
// The analyzer tracks token usage relative to model-specific context windows:
//
//   - GPT-4: 8K tokens
//   - GPT-4 Turbo: 128K tokens
//   - Claude 3: 200K tokens
//
// # Usage
//
// Create an analyzer and analyze conversation history:
//
//	cfg := config.GetConfig()
//	analyzer := conversation.NewAnalyzer(&cfg.Processing.Conversation)
//
//	// Analyze conversation
//	ctx, err := analyzer.AnalyzeConversation(messages, "gpt-4", 1500)
//	if err != nil {
//		log.Error("analysis failed", "error", err)
//	}
//
//	if ctx.ContextWindowPercent > 0.8 {
//		log.Warn("context window usage high",
//			"percent", ctx.ContextWindowPercent,
//			"tokens", ctx.ContextWindowUsage)
//	}
package conversation
