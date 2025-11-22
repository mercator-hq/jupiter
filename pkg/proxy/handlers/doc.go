// Package handlers provides HTTP request handlers for the proxy server.
//
// This package implements all HTTP endpoint handlers including chat completions,
// streaming, health checks, and WebSocket connections. Each handler is responsible
// for parsing requests, forwarding to providers, and formatting responses.
//
// # Handler Types
//
// Chat completion handlers:
//   - HandleChatCompletion: Non-streaming chat completions
//   - HandleStreamingCompletion: Server-Sent Events (SSE) streaming
//
// Health check handlers:
//   - HandleHealth: Liveness probe (always returns 200)
//   - HandleReady: Readiness probe (checks provider health)
//
// WebSocket handlers:
//   - HandleWebSocket: WebSocket connection upgrade and proxying
//
// # Request Flow
//
// Each handler follows a consistent pattern:
//
//  1. Parse request body (JSON unmarshaling)
//  2. Validate required fields and parameters
//  3. Extract metadata (request ID, user ID, API key)
//  4. Convert to provider format
//  5. Forward to provider via Provider Adapter
//  6. Convert provider response to OpenAI format
//  7. Write response to client (JSON or SSE)
//  8. Handle errors with OpenAI error format
//
// # Error Handling
//
// All handlers return errors in OpenAI-compatible format:
//
//	{
//	  "error": {
//	    "message": "Invalid request: missing required field 'model'",
//	    "type": "invalid_request_error",
//	    "param": "model",
//	    "code": "missing_field"
//	  }
//	}
//
// # Streaming Format
//
// Streaming handlers use Server-Sent Events (SSE) format:
//
//	data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[...]}
//	data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[...]}
//	data: [DONE]
//
// Each chunk is prefixed with "data: " and ends with "\n\n". The stream is
// terminated with "data: [DONE]\n\n".
//
// # Context Propagation
//
// Handlers extract metadata from requests and store it in context.Context for
// middleware access:
//
//	ctx = context.WithValue(ctx, RequestIDKey, requestID)
//	ctx = context.WithValue(ctx, UserIDKey, userID)
//	ctx = context.WithValue(ctx, ModelKey, model)
//
// # Health Checks
//
// Health check endpoints are designed for Kubernetes liveness/readiness probes:
//
//	# Kubernetes deployment
//	livenessProbe:
//	  httpGet:
//	    path: /health
//	    port: 8080
//	  initialDelaySeconds: 10
//	  periodSeconds: 30
//
//	readinessProbe:
//	  httpGet:
//	    path: /ready
//	    port: 8080
//	  initialDelaySeconds: 5
//	  periodSeconds: 10
//
// # WebSocket Support
//
// WebSocket handler upgrades HTTP connections for providers that require
// bidirectional communication:
//
//	ws://localhost:8080/v1/chat/completions/ws
//
// The handler forwards WebSocket messages to providers and streams responses
// back to clients.
//
// # Performance
//
// Handlers are optimized for low latency:
//
//   - <1ms request parsing (JSON unmarshaling)
//   - <1ms response formatting (JSON marshaling)
//   - <10ms first chunk delivery for streaming
//   - Buffered writes to reduce syscalls
//   - Connection pooling for provider requests
package handlers
