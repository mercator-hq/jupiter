package types

// ChatCompletionResponse represents an OpenAI-compatible chat completion response.
// This is returned for non-streaming requests.
type ChatCompletionResponse struct {
	// ID is a unique identifier for the chat completion.
	ID string `json:"id"`

	// Object is always "chat.completion".
	Object string `json:"object"`

	// Created is the Unix timestamp (seconds since epoch) of when the completion was created.
	Created int64 `json:"created"`

	// Model is the model used for the completion.
	Model string `json:"model"`

	// Choices is a list of completion choices (typically only one).
	Choices []Choice `json:"choices"`

	// Usage contains token usage statistics.
	Usage Usage `json:"usage"`

	// SystemFingerprint is a unique identifier for the backend configuration.
	// Optional, used for reproducibility.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	// Index is the index of this choice in the list of choices.
	Index int `json:"index"`

	// Message is the generated message.
	Message Message `json:"message"`

	// FinishReason explains why the model stopped generating tokens.
	// Possible values: "stop", "length", "tool_calls", "content_filter", "function_call".
	FinishReason string `json:"finish_reason"`

	// LogProbs contains log probability information (optional).
	LogProbs interface{} `json:"logprobs,omitempty"`
}

// Usage contains token usage statistics.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens (prompt + completion).
	TotalTokens int `json:"total_tokens"`
}

// ChatCompletionStreamChunk represents a chunk in a streaming response.
// This is sent as Server-Sent Events (SSE) when stream=true.
type ChatCompletionStreamChunk struct {
	// ID is a unique identifier for the chat completion.
	ID string `json:"id"`

	// Object is always "chat.completion.chunk".
	Object string `json:"object"`

	// Created is the Unix timestamp (seconds since epoch) of when the chunk was created.
	Created int64 `json:"created"`

	// Model is the model used for the completion.
	Model string `json:"model"`

	// Choices is a list of streaming choices.
	Choices []StreamChoice `json:"choices"`

	// SystemFingerprint is a unique identifier for the backend configuration.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// StreamChoice represents a single choice in a streaming response.
type StreamChoice struct {
	// Index is the index of this choice in the list of choices.
	Index int `json:"index"`

	// Delta contains incremental content.
	Delta Delta `json:"delta"`

	// FinishReason explains why the model stopped generating tokens.
	// Only present in the final chunk.
	FinishReason *string `json:"finish_reason"`

	// LogProbs contains log probability information (optional).
	LogProbs interface{} `json:"logprobs,omitempty"`
}

// Delta contains incremental content in a streaming response.
type Delta struct {
	// Role is the role of the message author (only in first chunk).
	Role string `json:"role,omitempty"`

	// Content is the incremental text content.
	Content string `json:"content,omitempty"`

	// ToolCalls contains incremental tool call information.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
