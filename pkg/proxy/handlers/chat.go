package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy"
	"mercator-hq/jupiter/pkg/proxy/middleware"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// convertToProviderRequest converts an OpenAI request to provider format.
// Handles chat messages including tool calls and multimodal content.
func convertToProviderRequest(req *types.ChatCompletionRequest) *providers.CompletionRequest {
	providerReq := &providers.CompletionRequest{
		Model:    req.Model,
		Messages: make([]providers.Message, 0, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Convert messages
	for _, msg := range req.Messages {
		providerMsg := providers.Message{
			Role:       msg.Role,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		// Convert content based on type
		providerMsg.Content = convertMessageContent(msg.Content)

		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			providerMsg.ToolCalls = convertToolCalls(msg.ToolCalls)
		}

		providerReq.Messages = append(providerReq.Messages, providerMsg)
	}

	// Copy optional parameters
	if req.Temperature != nil {
		providerReq.Temperature = *req.Temperature
	}

	if req.MaxTokens != nil {
		providerReq.MaxTokens = *req.MaxTokens
	}

	if req.TopP != nil {
		providerReq.TopP = *req.TopP
	}

	if len(req.Stop) > 0 {
		providerReq.Stop = req.Stop
	}

	if req.PresencePenalty != nil {
		providerReq.PresencePenalty = *req.PresencePenalty
	}

	if req.FrequencyPenalty != nil {
		providerReq.FrequencyPenalty = *req.FrequencyPenalty
	}

	if req.User != "" {
		providerReq.User = req.User
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		providerReq.Tools = convertTools(req.Tools)
	}

	// Copy tool choice if present
	if req.ToolChoice != nil {
		providerReq.ToolChoice = req.ToolChoice
	}

	return providerReq
}

// convertMessageContent converts message content from interface{} to string.
// Handles both simple string content and multimodal content arrays.
func convertMessageContent(content interface{}) string {
	if content == nil {
		return ""
	}

	// Handle string content (most common case)
	if str, ok := content.(string); ok {
		return str
	}

	// Handle array of content parts (multimodal content)
	if arr, ok := content.([]interface{}); ok {
		return convertMultimodalContent(arr)
	}

	// Fallback: convert to string using fmt
	return fmt.Sprintf("%v", content)
}

// convertMultimodalContent extracts text from multimodal content array.
// For now, this extracts text parts and ignores image/other media.
// Future enhancement: support image URLs for vision models.
func convertMultimodalContent(parts []interface{}) string {
	var textParts []string

	for _, part := range parts {
		// Each part should be a map with "type" and type-specific fields
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		partType, ok := partMap["type"].(string)
		if !ok {
			continue
		}

		switch partType {
		case "text":
			// Extract text content
			if text, ok := partMap["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case "image_url":
			// For now, skip image content
			// Future: extract URL and pass to vision-capable models
			continue
		default:
			// Unknown content type, skip
			continue
		}
	}

	// Join all text parts with space
	if len(textParts) == 0 {
		return ""
	}

	// Use strings.Join for proper concatenation
	var result string
	for i, part := range textParts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	return result
}

// convertToolCalls converts tool calls from proxy format to provider format.
func convertToolCalls(toolCalls []types.ToolCall) []providers.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	providerCalls := make([]providers.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		providerCalls[i] = providers.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: providers.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	return providerCalls
}

// convertTools converts tool definitions from proxy format to provider format.
func convertTools(tools []types.Tool) []providers.Tool {
	if len(tools) == 0 {
		return nil
	}

	providerTools := make([]providers.Tool, len(tools))
	for i, tool := range tools {
		providerTools[i] = providers.Tool{
			Type: tool.Type,
			Function: providers.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	return providerTools
}

// selectProviderFromManager selects an appropriate provider for the request.
// Uses model-based routing to select the best provider for the requested model.
func selectProviderFromManager(pm ProviderManager, req *types.ChatCompletionRequest) (providers.Provider, error) {
	// Get all healthy providers
	healthy := pm.GetHealthyProviders()

	if len(healthy) == 0 {
		return nil, &providers.ProviderError{
			Message:    "No healthy providers available",
			StatusCode: 503,
			Provider:   "none",
		}
	}

	// Try model-based routing first
	provider := selectProviderByModel(healthy, req.Model)
	if provider != nil {
		return provider, nil
	}

	// Fall back to first healthy provider
	for _, p := range healthy {
		return p, nil
	}

	// Should never reach here
	return nil, &providers.ProviderError{
		Message:    "Provider selection failed",
		StatusCode: 500,
		Provider:   "none",
	}
}

// selectProviderByModel selects a provider based on the model name.
// It matches model prefixes to provider names (e.g., "gpt-" -> "openai").
func selectProviderByModel(healthyProviders map[string]providers.Provider, model string) providers.Provider {
	// Model prefix to provider name mappings
	modelPrefixes := map[string]string{
		"gpt-":         "openai",
		"gpt4":         "openai",
		"text-davinci": "openai",
		"claude-":      "anthropic",
		"anthropic.":   "anthropic",
		"gemini-":      "google",
		"command":      "cohere",
		"mistral-":     "mistral",
		"llama-":       "meta",
		"mixtral":      "mistral",
	}

	// Check each prefix
	for prefix, providerName := range modelPrefixes {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			// Found matching prefix, try to get provider
			if provider, exists := healthyProviders[providerName]; exists {
				return provider
			}
		}
	}

	// No model-specific match found
	return nil
}

// handleChatRequest handles a chat completion request (non-streaming).
func handleChatRequest(w http.ResponseWriter, r *http.Request, pm ProviderManager) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	startTime := time.Now()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		errResp := types.NewInvalidRequestError(
			fmt.Sprintf("Method %s not allowed. Use POST instead.", r.Method),
			"method",
			"method_not_allowed",
		)
		if err := proxy.WriteErrorResponse(w, errResp); err != nil {
			slog.ErrorContext(ctx, "failed to write error response", "error", err)
		}
		return
	}

	// Parse request body
	chatReq, err := proxy.ParseChatCompletionRequest(r)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse request",
			"request_id", requestID,
			"error", err,
		)

		errResp := proxy.HandleError(err)
		if err := proxy.WriteErrorResponse(w, errResp); err != nil {
			slog.ErrorContext(ctx, "failed to write error response", "error", err)
		}
		return
	}

	// Handle streaming requests separately
	if chatReq.Stream {
		handleStreamRequest(w, r, pm, chatReq)
		return
	}

	// Log request
	slog.InfoContext(ctx, "processing chat completion request",
		"request_id", requestID,
		"model", chatReq.Model,
		"messages", len(chatReq.Messages),
	)

	// Select provider
	provider, err := selectProviderFromManager(pm, chatReq)
	if err != nil {
		slog.ErrorContext(ctx, "failed to select provider",
			"request_id", requestID,
			"model", chatReq.Model,
			"error", err,
		)

		errResp := proxy.HandleError(err)
		if err := proxy.WriteErrorResponse(w, errResp); err != nil {
			slog.ErrorContext(ctx, "failed to write error response", "error", err)
		}
		return
	}

	// Convert to provider format
	providerReq := convertToProviderRequest(chatReq)

	// Forward request to provider
	providerStartTime := time.Now()
	providerResp, err := provider.SendCompletion(ctx, providerReq)
	providerLatency := time.Since(providerStartTime)

	if err != nil {
		slog.ErrorContext(ctx, "provider request failed",
			"request_id", requestID,
			"provider", provider.GetName(),
			"model", chatReq.Model,
			"error", err,
			"provider_latency_ms", providerLatency.Milliseconds(),
		)

		errResp := proxy.HandleError(err)
		if err := proxy.WriteErrorResponse(w, errResp); err != nil {
			slog.ErrorContext(ctx, "failed to write error response", "error", err)
		}
		return
	}

	// Convert provider response to OpenAI format
	openaiResp := proxy.FormatChatCompletionResponse(providerResp, chatReq.Model)

	// Log successful completion
	totalLatency := time.Since(startTime)
	slog.InfoContext(ctx, "chat completion successful",
		"request_id", requestID,
		"provider", provider.GetName(),
		"model", chatReq.Model,
		"finish_reason", providerResp.FinishReason,
		"prompt_tokens", providerResp.Usage.PromptTokens,
		"completion_tokens", providerResp.Usage.CompletionTokens,
		"total_tokens", providerResp.Usage.TotalTokens,
		"provider_latency_ms", providerLatency.Milliseconds(),
		"total_latency_ms", totalLatency.Milliseconds(),
	)

	// Write response
	if err := proxy.WriteJSONResponse(w, http.StatusOK, openaiResp); err != nil {
		slog.ErrorContext(ctx, "failed to write response",
			"request_id", requestID,
			"error", err,
		)
	}
}

// handleStreamRequest handles a streaming chat completion request.
func handleStreamRequest(w http.ResponseWriter, r *http.Request, pm ProviderManager, chatReq *types.ChatCompletionRequest) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	startTime := time.Now()

	// Log request
	slog.InfoContext(ctx, "processing streaming chat completion request",
		"request_id", requestID,
		"model", chatReq.Model,
		"messages", len(chatReq.Messages),
	)

	// Select provider
	provider, err := selectProviderFromManager(pm, chatReq)
	if err != nil {
		slog.ErrorContext(ctx, "failed to select provider",
			"request_id", requestID,
			"model", chatReq.Model,
			"error", err,
		)

		errResp := proxy.HandleError(err)
		if err := proxy.WriteErrorResponse(w, errResp); err != nil {
			slog.ErrorContext(ctx, "failed to write error response", "error", err)
		}
		return
	}

	// Convert to provider format
	providerReq := convertToProviderRequest(chatReq)

	// Set SSE headers
	proxy.SetSSEHeaders(w)

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Forward streaming request to provider
	providerStartTime := time.Now()
	chunks, err := provider.StreamCompletion(ctx, providerReq)
	if err != nil {
		slog.ErrorContext(ctx, "provider streaming request failed",
			"request_id", requestID,
			"provider", provider.GetName(),
			"model", chatReq.Model,
			"error", err,
		)

		errResp := proxy.HandleError(err)
		if err := proxy.WriteSSEError(w, errResp); err != nil {
			slog.ErrorContext(ctx, "failed to write SSE error", "error", err)
		}
		return
	}

	// Generate response ID for all chunks
	responseID := fmt.Sprintf("chatcmpl-%s", requestID)

	// Stream chunks to client
	chunkCount := 0
	var firstChunkTime time.Time
	totalTokens := 0

	for chunk := range chunks {
		// Record first chunk timing
		if chunkCount == 0 {
			firstChunkTime = time.Now()
		}

		// Check for errors in chunk
		if chunk.Error != nil {
			slog.ErrorContext(ctx, "error in stream chunk",
				"request_id", requestID,
				"provider", provider.GetName(),
				"chunk_count", chunkCount,
				"error", chunk.Error,
			)

			errResp := proxy.HandleError(chunk.Error)
			if err := proxy.WriteSSEError(w, errResp); err != nil {
				slog.ErrorContext(ctx, "failed to write SSE error", "error", err)
			}
			break
		}

		// Convert chunk to OpenAI format
		openaiChunk := proxy.FormatStreamChunk(chunk, chatReq.Model, responseID)

		// Write SSE chunk
		if err := proxy.WriteSSEChunk(w, openaiChunk); err != nil {
			slog.ErrorContext(ctx, "failed to write SSE chunk",
				"request_id", requestID,
				"chunk_count", chunkCount,
				"error", err,
			)
			break
		}

		chunkCount++

		// Track tokens if present in chunk
		if chunk.Usage != nil {
			totalTokens = chunk.Usage.TotalTokens
		}

		// Check if client disconnected
		select {
		case <-ctx.Done():
			slog.WarnContext(ctx, "client disconnected during streaming",
				"request_id", requestID,
				"provider", provider.GetName(),
				"chunks_sent", chunkCount,
			)
			return
		default:
			// Continue streaming
		}
	}

	// Write [DONE] marker
	if err := proxy.WriteSSEDone(w); err != nil {
		slog.ErrorContext(ctx, "failed to write SSE done marker",
			"request_id", requestID,
			"error", err,
		)
	}

	// Log successful completion
	totalLatency := time.Since(startTime)
	providerLatency := time.Since(providerStartTime)
	var firstChunkLatency time.Duration
	if !firstChunkTime.IsZero() {
		firstChunkLatency = firstChunkTime.Sub(providerStartTime)
	}

	slog.InfoContext(ctx, "streaming chat completion successful",
		"request_id", requestID,
		"provider", provider.GetName(),
		"model", chatReq.Model,
		"chunks_sent", chunkCount,
		"total_tokens", totalTokens,
		"provider_latency_ms", providerLatency.Milliseconds(),
		"first_chunk_latency_ms", firstChunkLatency.Milliseconds(),
		"total_latency_ms", totalLatency.Milliseconds(),
	)
}

// ChatHandler wraps the chat request handling for use by the server.
type ChatHandler struct {
	ProviderManager ProviderManager
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(pm ProviderManager) *ChatHandler {
	return &ChatHandler{ProviderManager: pm}
}

// ServeHTTP implements http.Handler.
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleChatRequest(w, r, h.ProviderManager)
}

// StreamHandler wraps the streaming request handling for use by the server.
type StreamHandler struct {
	ProviderManager ProviderManager
}

// NewStreamHandler creates a new streaming handler.
func NewStreamHandler(pm ProviderManager) *StreamHandler {
	return &StreamHandler{ProviderManager: pm}
}

// ServeHTTP implements http.Handler.
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For MVP, streaming is handled by ChatHandler
	// This is kept for future separation if needed
	handleChatRequest(w, r, h.ProviderManager)
}
