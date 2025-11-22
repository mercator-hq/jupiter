package providers

import "time"

// Message represents a single message in a conversation.
// It is provider-agnostic and will be transformed to provider-specific formats.
type Message struct {
	// Role identifies the message sender (system, user, assistant, tool)
	Role string `json:"role"`

	// Content is the message text content
	Content string `json:"content"`

	// Name is an optional name for the message sender (used for multi-user conversations)
	Name string `json:"name,omitempty"`

	// ToolCalls contains function/tool calls made by the assistant (for assistant role)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID is used when role is "tool" to reference which tool call this responds to
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function/tool call request from the model.
type ToolCall struct {
	// ID is a unique identifier for this tool call
	ID string `json:"id"`

	// Type is the type of tool call (currently always "function")
	Type string `json:"type"`

	// Function contains the function name and arguments
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a specific function invocation.
type FunctionCall struct {
	// Name is the function name to call
	Name string `json:"name"`

	// Arguments is a JSON string containing the function arguments
	Arguments string `json:"arguments"`
}

// Tool represents a tool/function definition that the model can call.
type Tool struct {
	// Type is the type of tool (currently always "function")
	Type string `json:"type"`

	// Function contains the function definition
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a callable function.
type FunctionDefinition struct {
	// Name is the function name
	Name string `json:"name"`

	// Description explains what the function does
	Description string `json:"description,omitempty"`

	// Parameters is a JSON Schema object describing the function parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used (prompt + completion)
	TotalTokens int `json:"total_tokens"`
}

// CompletionRequest represents a provider-agnostic completion request.
// It is transformed to provider-specific formats by each adapter.
type CompletionRequest struct {
	// Model is the model identifier (e.g., "gpt-4", "claude-3-opus-20240229")
	Model string `json:"model"`

	// Messages is the conversation history
	Messages []Message `json:"messages"`

	// Temperature controls randomness (0.0 to 2.0, typically 0.0 to 1.0)
	Temperature float64 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// TopP controls nucleus sampling (0.0 to 1.0)
	TopP float64 `json:"top_p,omitempty"`

	// Stream indicates whether to stream the response
	Stream bool `json:"stream,omitempty"`

	// Tools is a list of tools the model can call
	Tools []Tool `json:"tools,omitempty"`

	// ToolChoice controls which tools can be called
	// Can be "none", "auto", or {"type": "function", "function": {"name": "my_function"}}
	ToolChoice interface{} `json:"tool_choice,omitempty"`

	// Stop sequences that will halt generation
	Stop []string `json:"stop,omitempty"`

	// PresencePenalty reduces repetition (-2.0 to 2.0)
	PresencePenalty float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty reduces repetition based on frequency (-2.0 to 2.0)
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`

	// User is an optional user identifier for abuse monitoring
	User string `json:"user,omitempty"`

	// Metadata contains additional request context (user ID, API key, etc.)
	// This is not sent to the provider, but used internally
	Metadata map[string]string `json:"-"`
}

// CompletionResponse represents a provider-agnostic completion response.
// It is normalized from provider-specific response formats.
type CompletionResponse struct {
	// ID is the unique response identifier
	ID string `json:"id"`

	// Model is the model that generated the response
	Model string `json:"model"`

	// Content is the generated text content
	Content string `json:"content"`

	// FinishReason indicates why generation stopped
	// (stop, length, tool_calls, content_filter)
	FinishReason string `json:"finish_reason"`

	// Usage contains token consumption information
	Usage TokenUsage `json:"usage"`

	// ToolCalls contains any tool/function calls made by the model
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Created is the Unix timestamp when the response was created
	Created int64 `json:"created"`

	// Metadata contains additional response context
	Metadata map[string]string `json:"metadata,omitempty"`
}

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	// ID is the response identifier (same across all chunks)
	ID string `json:"id"`

	// Model is the model generating the response
	Model string `json:"model"`

	// Delta is the incremental content in this chunk
	Delta string `json:"delta"`

	// FinishReason is set in the final chunk to indicate why generation stopped
	FinishReason string `json:"finish_reason,omitempty"`

	// ToolCalls contains incremental tool call information
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Usage is included in the final chunk (if supported by provider)
	Usage *TokenUsage `json:"usage,omitempty"`

	// Error is set if an error occurred during streaming
	Error error `json:"-"`

	// Created is the Unix timestamp when the chunk was created
	Created int64 `json:"created"`
}

// ProviderHealth tracks the health status of a provider.
type ProviderHealth struct {
	// IsHealthy indicates whether the provider is currently healthy
	IsHealthy bool

	// LastCheck is the timestamp of the last health check
	LastCheck time.Time

	// LastError is the most recent error encountered (nil if healthy)
	LastError error

	// ConsecutiveFailures counts sequential health check failures
	ConsecutiveFailures int

	// LastSuccessfulRequest is the timestamp of the last successful request
	LastSuccessfulRequest time.Time

	// TotalRequests is the total number of requests sent to this provider
	TotalRequests int64

	// FailedRequests is the total number of failed requests
	FailedRequests int64
}

// ProviderConfig contains configuration for a single provider instance.
// This is a subset of config.ProviderConfig with only the fields needed by adapters.
type ProviderConfig struct {
	// Name is the provider identifier (e.g., "openai", "anthropic")
	Name string

	// Type is the provider type (openai, anthropic, generic)
	Type string

	// BaseURL is the API endpoint base URL
	BaseURL string

	// APIKey is the authentication key
	APIKey string

	// Timeout is the request timeout duration
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// HealthCheckInterval is how often to run health checks
	HealthCheckInterval time.Duration

	// MaxIdleConns is the maximum number of idle connections in the pool
	MaxIdleConns int

	// MaxIdleConnsPerHost is the maximum idle connections per host
	MaxIdleConnsPerHost int

	// IdleConnTimeout is how long an idle connection remains in the pool
	IdleConnTimeout time.Duration
}

// Message role constants
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Finish reason constants
const (
	FinishReasonStop          = "stop"
	FinishReasonLength        = "length"
	FinishReasonToolCalls     = "tool_calls"
	FinishReasonContentFilter = "content_filter"
)

// Tool type constants
const (
	ToolTypeFunction = "function"
)
