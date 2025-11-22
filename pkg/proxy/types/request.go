package types

// ChatCompletionRequest represents an OpenAI-compatible chat completion request.
// This matches the OpenAI Chat Completions API format exactly to ensure
// compatibility with existing OpenAI SDKs and tools.
type ChatCompletionRequest struct {
	// Model is the ID of the model to use (e.g., "gpt-4", "claude-3-opus").
	Model string `json:"model"`

	// Messages is the conversation history as a list of messages.
	Messages []Message `json:"messages"`

	// Temperature controls randomness in the response (0.0 to 2.0).
	// Higher values make output more random. Optional, defaults to 1.0.
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to generate.
	// Optional, defaults to provider-specific limits.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// TopP controls nucleus sampling (0.0 to 1.0).
	// Alternative to temperature. Optional, defaults to 1.0.
	TopP *float64 `json:"top_p,omitempty"`

	// N is the number of completions to generate for each prompt.
	// Optional, defaults to 1. Most providers only support 1.
	N *int `json:"n,omitempty"`

	// Stream enables server-sent events (SSE) streaming.
	// Optional, defaults to false.
	Stream bool `json:"stream,omitempty"`

	// Stop is a list of sequences where the API will stop generating tokens.
	// Optional, maximum 4 sequences.
	Stop []string `json:"stop,omitempty"`

	// PresencePenalty penalizes new tokens based on presence in text so far (-2.0 to 2.0).
	// Optional, defaults to 0.0.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty penalizes new tokens based on frequency in text so far (-2.0 to 2.0).
	// Optional, defaults to 0.0.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// User is a unique identifier for the end-user making the request.
	// Used for abuse detection and tracking. Optional.
	User string `json:"user,omitempty"`

	// Tools is a list of tools/functions the model can call.
	// Optional, only supported by function-calling models.
	Tools []Tool `json:"tools,omitempty"`

	// ToolChoice controls which tool the model should use.
	// Can be "none", "auto", or {"type": "function", "function": {"name": "my_function"}}.
	// Optional, defaults to "auto" when tools are present.
	ToolChoice interface{} `json:"tool_choice,omitempty"`

	// ResponseFormat specifies the format of the response.
	// Optional, can be {"type": "json_object"} for JSON mode.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// Seed enables deterministic sampling (OpenAI beta feature).
	// Optional, not supported by all providers.
	Seed *int `json:"seed,omitempty"`
}

// Message represents a single message in a conversation.
type Message struct {
	// Role is the author of the message ("system", "user", "assistant", or "tool").
	Role string `json:"role"`

	// Content is the text content of the message.
	// Can be a string or an array of content parts (for multimodal models).
	Content interface{} `json:"content"`

	// Name is the name of the author (optional, for user/assistant messages).
	Name string `json:"name,omitempty"`

	// ToolCalls is a list of tool calls made by the assistant (optional).
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID is the ID of the tool call this message is responding to (for tool role).
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// Tool represents a function/tool that the model can call.
type Tool struct {
	// Type is always "function" for function calling.
	Type string `json:"type"`

	// Function describes the function to call.
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition describes a function that can be called by the model.
type FunctionDefinition struct {
	// Name is the name of the function to call.
	Name string `json:"name"`

	// Description explains what the function does.
	Description string `json:"description,omitempty"`

	// Parameters is a JSON Schema object describing the function parameters.
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolCall represents a function call made by the model.
type ToolCall struct {
	// ID is a unique identifier for the tool call.
	ID string `json:"id"`

	// Type is always "function" for function calling.
	Type string `json:"type"`

	// Function contains the function name and arguments.
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function name and arguments.
type FunctionCall struct {
	// Name is the name of the function to call.
	Name string `json:"name"`

	// Arguments is a JSON string containing the function arguments.
	Arguments string `json:"arguments"`
}

// ResponseFormat specifies the format of the model's output.
type ResponseFormat struct {
	// Type is the format type ("text" or "json_object").
	Type string `json:"type"`
}

// Validate validates the chat completion request.
// It checks that required fields are present and values are within acceptable ranges.
func (r *ChatCompletionRequest) Validate() error {
	if r.Model == "" {
		return &ValidationError{
			Field:   "model",
			Message: "model is required",
		}
	}

	if len(r.Messages) == 0 {
		return &ValidationError{
			Field:   "messages",
			Message: "messages must contain at least one message",
		}
	}

	// Validate temperature range
	if r.Temperature != nil && (*r.Temperature < 0.0 || *r.Temperature > 2.0) {
		return &ValidationError{
			Field:   "temperature",
			Message: "temperature must be between 0.0 and 2.0",
		}
	}

	// Validate top_p range
	if r.TopP != nil && (*r.TopP < 0.0 || *r.TopP > 1.0) {
		return &ValidationError{
			Field:   "top_p",
			Message: "top_p must be between 0.0 and 1.0",
		}
	}

	// Validate max_tokens
	if r.MaxTokens != nil && *r.MaxTokens < 1 {
		return &ValidationError{
			Field:   "max_tokens",
			Message: "max_tokens must be greater than 0",
		}
	}

	// Validate n
	if r.N != nil && *r.N < 1 {
		return &ValidationError{
			Field:   "n",
			Message: "n must be greater than 0",
		}
	}

	// Validate stop sequences
	if len(r.Stop) > 4 {
		return &ValidationError{
			Field:   "stop",
			Message: "stop sequences must not exceed 4",
		}
	}

	// Validate presence_penalty range
	if r.PresencePenalty != nil && (*r.PresencePenalty < -2.0 || *r.PresencePenalty > 2.0) {
		return &ValidationError{
			Field:   "presence_penalty",
			Message: "presence_penalty must be between -2.0 and 2.0",
		}
	}

	// Validate frequency_penalty range
	if r.FrequencyPenalty != nil && (*r.FrequencyPenalty < -2.0 || *r.FrequencyPenalty > 2.0) {
		return &ValidationError{
			Field:   "frequency_penalty",
			Message: "frequency_penalty must be between -2.0 and 2.0",
		}
	}

	// Validate messages have required fields
	for i, msg := range r.Messages {
		if msg.Role == "" {
			return &ValidationError{
				Field:   "messages[" + string(rune(i)) + "].role",
				Message: "message role is required",
			}
		}

		if msg.Content == nil && len(msg.ToolCalls) == 0 {
			return &ValidationError{
				Field:   "messages[" + string(rune(i)) + "].content",
				Message: "message content is required when no tool_calls present",
			}
		}
	}

	return nil
}

// ValidationError represents a request validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Message
}
