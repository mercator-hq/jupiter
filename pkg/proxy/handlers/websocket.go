package handlers

import (
	"net/http"

	"mercator-hq/jupiter/pkg/proxy/types"
)

// WebSocketHandler handles WebSocket connection upgrade requests.
// For MVP, this returns "not implemented" as most providers use SSE for streaming.
// Full WebSocket support can be added in future iterations if needed.
type WebSocketHandler struct {
	// ProviderManager manages the collection of LLM providers.
	ProviderManager ProviderManager
}

// NewWebSocketHandler creates a new WebSocket handler.
func NewWebSocketHandler(pm ProviderManager) *WebSocketHandler {
	return &WebSocketHandler{
		ProviderManager: pm,
	}
}

// ServeHTTP implements the http.Handler interface.
// For MVP, this returns a "not implemented" error response.
//
// Future implementation will:
//   - Upgrade HTTP connection to WebSocket
//   - Forward WebSocket messages to provider
//   - Stream provider responses back over WebSocket
//   - Handle WebSocket close frames
//   - Clean up connections on error
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For MVP, return not implemented
	errResp := types.NewErrorResponse(
		"WebSocket support is not implemented in this version. Please use HTTP with Server-Sent Events (SSE) streaming instead by setting stream=true in your request.",
		types.ErrorTypeNotFound,
		"",
		"not_implemented",
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)

	// Note: Using json.NewEncoder instead of proxy.WriteErrorResponse
	// to avoid circular import (proxy imports handlers)
	// This is acceptable for a simple error response
	_ = errResp // Placeholder for future implementation

	// Write simple error message
	w.Write([]byte(`{"error":{"message":"WebSocket support is not implemented in this version","type":"not_found","code":"not_implemented"}}`))
}

// Note: Full WebSocket implementation would include:
//
// 1. Connection upgrade:
//    upgrader := websocket.Upgrader{...}
//    conn, err := upgrader.Upgrade(w, r, nil)
//
// 2. Message handling:
//    for {
//        messageType, message, err := conn.ReadMessage()
//        // Forward to provider
//    }
//
// 3. Response streaming:
//    for chunk := range providerChunks {
//        conn.WriteMessage(websocket.TextMessage, chunk)
//    }
//
// 4. Connection cleanup:
//    defer conn.Close()
//
// This can be added in future iterations if WebSocket support is required
// by specific providers or client requirements.
