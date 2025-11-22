package openai

import (
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
)

// OpenAI API request/response types

// OpenAIRequest represents an OpenAI chat completion request.
type OpenAIRequest struct {
	Model            string                 `json:"model"`
	Messages         []OpenAIMessage        `json:"messages"`
	Temperature      float64                `json:"temperature,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	Tools            []OpenAITool           `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	User             string                 `json:"user,omitempty"`
	N                int                    `json:"n,omitempty"`
	ResponseFormat   map[string]interface{} `json:"response_format,omitempty"`
}

// OpenAIMessage represents a message in OpenAI format.
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// OpenAIToolCall represents a tool call in OpenAI format.
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall represents a function call in OpenAI format.
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAITool represents a tool definition in OpenAI format.
type OpenAITool struct {
	Type     string                   `json:"type"`
	Function OpenAIFunctionDefinition `json:"function"`
}

// OpenAIFunctionDefinition represents a function definition in OpenAI format.
type OpenAIFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OpenAIResponse represents an OpenAI chat completion response.
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

// OpenAIChoice represents a completion choice in OpenAI format.
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage in OpenAI format.
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAI streaming response types

// OpenAIStreamResponse represents a chunk in OpenAI's SSE stream.
type OpenAIStreamResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
	Usage   *OpenAIUsage         `json:"usage,omitempty"`
}

// OpenAIStreamChoice represents a choice in a stream chunk.
type OpenAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

// OpenAIStreamDelta represents the incremental content in a stream chunk.
type OpenAIStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// Transformation functions

// transformRequest transforms a provider-agnostic request to OpenAI format.
func transformRequest(req *providers.CompletionRequest) *OpenAIRequest {
	openaiReq := &OpenAIRequest{
		Model:            req.Model,
		Messages:         make([]OpenAIMessage, len(req.Messages)),
		Temperature:      req.Temperature,
		MaxTokens:        req.MaxTokens,
		TopP:             req.TopP,
		Stream:           req.Stream,
		Stop:             req.Stop,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		User:             req.User,
		ToolChoice:       req.ToolChoice,
		N:                1, // Always generate 1 completion
	}

	// Transform messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = OpenAIMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}
	}

	// Transform tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]OpenAITool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = OpenAITool{
				Type: tool.Type,
				Function: OpenAIFunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	return openaiReq
}

// transformResponse transforms an OpenAI response to provider-agnostic format.
func transformResponse(resp *OpenAIResponse) (*providers.CompletionResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Use the first choice (we always request N=1)
	choice := resp.Choices[0]

	result := &providers.CompletionResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      choice.Message.Content,
		FinishReason: normalizeFinishReason(choice.FinishReason),
		Usage: providers.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Created:  resp.Created,
		Metadata: make(map[string]string),
	}

	// Transform tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]providers.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			result.ToolCalls[i] = providers.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: providers.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return result, nil
}

// transformStreamChunk transforms an OpenAI stream chunk to provider-agnostic format.
func transformStreamChunk(chunk *OpenAIStreamResponse) (*providers.StreamChunk, error) {
	if len(chunk.Choices) == 0 {
		return nil, fmt.Errorf("no choices in stream chunk")
	}

	choice := chunk.Choices[0]

	result := &providers.StreamChunk{
		ID:           chunk.ID,
		Model:        chunk.Model,
		Delta:        choice.Delta.Content,
		FinishReason: normalizeFinishReason(choice.FinishReason),
		Created:      chunk.Created,
	}

	// Include usage if present (final chunk)
	if chunk.Usage != nil {
		result.Usage = &providers.TokenUsage{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		}
	}

	// Transform tool calls if present
	if len(choice.Delta.ToolCalls) > 0 {
		result.ToolCalls = make([]providers.ToolCall, len(choice.Delta.ToolCalls))
		for i, tc := range choice.Delta.ToolCalls {
			result.ToolCalls[i] = providers.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: providers.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return result, nil
}

// normalizeFinishReason normalizes OpenAI finish reasons to provider-agnostic values.
func normalizeFinishReason(reason string) string {
	switch reason {
	case "stop":
		return providers.FinishReasonStop
	case "length":
		return providers.FinishReasonLength
	case "tool_calls", "function_call":
		return providers.FinishReasonToolCalls
	case "content_filter":
		return providers.FinishReasonContentFilter
	default:
		return reason
	}
}
