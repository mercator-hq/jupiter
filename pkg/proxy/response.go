package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// FormatChatCompletionResponse converts a provider response to OpenAI chat completion format.
// It generates a unique response ID, sets the object type, and includes usage statistics.
//
// Example usage:
//
//	providerResp, err := provider.SendCompletion(ctx, req)
//	if err != nil {
//	    return err
//	}
//	openaiResp := FormatChatCompletionResponse(providerResp, "gpt-4")
func FormatChatCompletionResponse(resp *providers.CompletionResponse, requestedModel string) *types.ChatCompletionResponse {
	// Generate unique response ID (format: chatcmpl-<id>)
	responseID := fmt.Sprintf("chatcmpl-%s", resp.ID)

	// Create OpenAI-formatted response
	return &types.ChatCompletionResponse{
		ID:      responseID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestedModel,
		Choices: []types.Choice{
			{
				Index: 0,
				Message: types.Message{
					Role:      "assistant",
					Content:   resp.Content,
					ToolCalls: convertToolCalls(resp.ToolCalls),
				},
				FinishReason: resp.FinishReason,
			},
		},
		Usage: types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

// FormatStreamChunk converts a provider stream chunk to OpenAI chat completion chunk format.
// This is used for Server-Sent Events (SSE) streaming responses.
//
// Example usage:
//
//	for chunk := range chunks {
//	    openaiChunk := FormatStreamChunk(chunk, "gpt-4", responseID)
//	    // Write to SSE stream
//	}
func FormatStreamChunk(chunk *providers.StreamChunk, requestedModel string, responseID string) *types.ChatCompletionStreamChunk {
	// Ensure response ID has correct format
	if responseID == "" {
		responseID = fmt.Sprintf("chatcmpl-%s", chunk.ID)
	}

	streamChunk := &types.ChatCompletionStreamChunk{
		ID:      responseID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   requestedModel,
		Choices: []types.StreamChoice{
			{
				Index: 0,
				Delta: types.Delta{
					Content:   chunk.Delta,
					ToolCalls: convertToolCalls(chunk.ToolCalls),
				},
			},
		},
	}

	// Include finish_reason only in final chunk
	if chunk.FinishReason != "" {
		finishReason := chunk.FinishReason
		streamChunk.Choices[0].FinishReason = &finishReason
	}

	return streamChunk
}

// convertToolCalls converts provider tool calls to OpenAI format.
func convertToolCalls(toolCalls []providers.ToolCall) []types.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]types.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = types.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: types.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// WriteJSONResponse writes a JSON response to the HTTP response writer.
// It sets the appropriate content-type header and handles marshaling errors.
//
// If marshaling fails, it writes a 500 error response.
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON response: %w", err)
	}

	return nil
}

// WriteErrorResponse writes an OpenAI-compatible error response.
// It extracts the appropriate HTTP status code from the error type.
func WriteErrorResponse(w http.ResponseWriter, errResp *types.ErrorResponse) error {
	statusCode := errResp.Error.HTTPStatusCode()
	return WriteJSONResponse(w, statusCode, errResp)
}

// WriteSSEChunk writes a single chunk in Server-Sent Events format.
// Each chunk is formatted as:
//
//	data: {"id":"chatcmpl-123","object":"chat.completion.chunk",...}
//
// Followed by two newlines (\n\n).
func WriteSSEChunk(w http.ResponseWriter, chunk *types.ChatCompletionStreamChunk) error {
	// Marshal chunk to JSON
	data, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE chunk: %w", err)
	}

	// Write SSE formatted chunk: "data: <json>\n\n"
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return fmt.Errorf("failed to write SSE chunk: %w", err)
	}

	// Flush immediately for real-time streaming
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// WriteSSEDone writes the final "[DONE]" marker for SSE streams.
// This signals to the client that the stream has completed.
func WriteSSEDone(w http.ResponseWriter) error {
	if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
		return fmt.Errorf("failed to write SSE done marker: %w", err)
	}

	// Flush the done marker
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// WriteSSEError writes an error in SSE format.
// This allows errors to be sent mid-stream if something goes wrong.
func WriteSSEError(w http.ResponseWriter, errResp *types.ErrorResponse) error {
	// Create an error chunk
	errorData := map[string]interface{}{
		"error": errResp.Error,
	}

	// Marshal error to JSON
	data, err := json.Marshal(errorData)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE error: %w", err)
	}

	// Write SSE formatted error
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return fmt.Errorf("failed to write SSE error: %w", err)
	}

	// Flush the error
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// SetSSEHeaders sets the appropriate headers for Server-Sent Events streaming.
func SetSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
}
