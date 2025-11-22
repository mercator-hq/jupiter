package anthropic

import (
	"encoding/json"
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
)

// Anthropic API request/response types

// AnthropicRequest represents an Anthropic messages request.
type AnthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	System        string             `json:"system,omitempty"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   float64            `json:"temperature,omitempty"`
	TopP          float64            `json:"top_p,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	Tools         []AnthropicTool    `json:"tools,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format.
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or []ContentBlock
}

// ContentBlock represents a content block in Anthropic format.
type ContentBlock struct {
	Type string `json:"type"` // "text" or "tool_use" or "tool_result"
	Text string `json:"text,omitempty"`

	// For tool_use blocks
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	// For tool_result blocks
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// AnthropicTool represents a tool definition in Anthropic format.
type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AnthropicResponse represents an Anthropic messages response.
type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage `json:"usage"`
}

// AnthropicUsage represents token usage in Anthropic format.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Anthropic streaming response types

// AnthropicStreamEvent represents an event in Anthropic's SSE stream.
type AnthropicStreamEvent struct {
	Type string `json:"type"`

	// For message_start event
	Message *AnthropicResponse `json:"message,omitempty"`

	// For content_block_start event
	Index        int           `json:"index,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`

	// For content_block_delta event
	Delta *ContentBlockDelta `json:"delta,omitempty"`

	// For message_delta event
	Delta2 *MessageDelta   `json:"delta,omitempty"`
	Usage  *AnthropicUsage `json:"usage,omitempty"`
}

// ContentBlockDelta represents incremental content in Anthropic format.
type ContentBlockDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MessageDelta represents message-level deltas.
type MessageDelta struct {
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// Transformation functions

// transformRequest transforms a provider-agnostic request to Anthropic format.
func transformRequest(req *providers.CompletionRequest) (*AnthropicRequest, error) {
	anthropicReq := &AnthropicRequest{
		Model:         req.Model,
		Messages:      make([]AnthropicMessage, 0, len(req.Messages)),
		MaxTokens:     req.MaxTokens,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		Stream:        req.Stream,
		StopSequences: req.Stop,
	}

	// Set default max_tokens if not provided (required by Anthropic)
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	// Extract system message (Anthropic requires it as a separate field)
	var systemMessage string
	for _, msg := range req.Messages {
		if msg.Role == providers.RoleSystem {
			systemMessage = msg.Content
		} else {
			// Add non-system messages
			anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	anthropicReq.System = systemMessage

	// Transform tools
	if len(req.Tools) > 0 {
		anthropicReq.Tools = make([]AnthropicTool, len(req.Tools))
		for i, tool := range req.Tools {
			anthropicReq.Tools[i] = AnthropicTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: tool.Function.Parameters,
			}
		}
	}

	// Validate: Anthropic requires alternating user/assistant messages
	if err := validateMessageSequence(anthropicReq.Messages); err != nil {
		return nil, err
	}

	return anthropicReq, nil
}

// validateMessageSequence validates that messages alternate between user and assistant.
func validateMessageSequence(messages []AnthropicMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// First message must be from user
	if messages[0].Role != providers.RoleUser {
		return &providers.ValidationError{
			Field:   "messages",
			Message: "first message must be from user (Anthropic requirement)",
		}
	}

	// Check alternation
	for i := 1; i < len(messages); i++ {
		prev := messages[i-1].Role
		curr := messages[i].Role

		// Messages must alternate
		if prev == curr {
			return &providers.ValidationError{
				Field:   "messages",
				Message: fmt.Sprintf("messages must alternate between user and assistant (Anthropic requirement), found consecutive %s messages at index %d", curr, i),
			}
		}
	}

	return nil
}

// transformResponse transforms an Anthropic response to provider-agnostic format.
func transformResponse(resp *AnthropicResponse) (*providers.CompletionResponse, error) {
	// Extract text content from content blocks
	var content string
	var toolCalls []providers.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content += block.Text

		case "tool_use":
			// Convert tool use to tool call
			// For Anthropic, input is a map, we need to convert to JSON string
			argsJSON, err := jsonMarshalString(block.Input)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool input: %w", err)
			}

			toolCalls = append(toolCalls, providers.ToolCall{
				ID:   block.ID,
				Type: providers.ToolTypeFunction,
				Function: providers.FunctionCall{
					Name:      block.Name,
					Arguments: argsJSON,
				},
			})
		}
	}

	result := &providers.CompletionResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      content,
		FinishReason: normalizeStopReason(resp.StopReason),
		Usage: providers.TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		ToolCalls: toolCalls,
		Metadata:  make(map[string]string),
	}

	return result, nil
}

// transformStreamChunk transforms an Anthropic stream event to provider-agnostic format.
func transformStreamChunk(event *AnthropicStreamEvent, state *streamState) (*providers.StreamChunk, error) {
	switch event.Type {
	case "message_start":
		// Initialize stream state
		if event.Message != nil {
			state.id = event.Message.ID
			state.model = event.Message.Model
		}
		return nil, nil // Don't emit chunk for message_start

	case "content_block_start":
		// Start of a new content block
		return nil, nil // Don't emit chunk yet

	case "content_block_delta":
		// Incremental content
		if event.Delta != nil && event.Delta.Text != "" {
			return &providers.StreamChunk{
				ID:    state.id,
				Model: state.model,
				Delta: event.Delta.Text,
			}, nil
		}
		return nil, nil

	case "content_block_stop":
		// End of content block
		return nil, nil // Don't emit chunk

	case "message_delta":
		// Message-level delta (includes stop_reason)
		chunk := &providers.StreamChunk{
			ID:    state.id,
			Model: state.model,
			Delta: "",
		}
		if event.Delta2 != nil {
			chunk.FinishReason = normalizeStopReason(event.Delta2.StopReason)
		}
		if event.Usage != nil {
			chunk.Usage = &providers.TokenUsage{
				PromptTokens:     event.Usage.InputTokens,
				CompletionTokens: event.Usage.OutputTokens,
				TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
			}
		}
		return chunk, nil

	case "message_stop":
		// End of stream
		return nil, nil

	case "ping":
		// Keep-alive ping
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown stream event type: %s", event.Type)
	}
}

// streamState tracks state across stream events.
type streamState struct {
	id    string
	model string
}

// normalizeStopReason normalizes Anthropic stop reasons to provider-agnostic values.
func normalizeStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return providers.FinishReasonStop
	case "max_tokens":
		return providers.FinishReasonLength
	case "tool_use":
		return providers.FinishReasonToolCalls
	case "stop_sequence":
		return providers.FinishReasonStop
	default:
		return reason
	}
}

// Helper function to marshal map to JSON string
func jsonMarshalString(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
