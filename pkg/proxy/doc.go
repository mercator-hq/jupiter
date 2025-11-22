// Package proxy provides a production-ready HTTP/HTTPS proxy server for LLM traffic.
//
// The proxy server is the network-facing gateway for all LLM requests, handling TLS
// termination, request routing, response streaming, and metadata extraction. It accepts
// OpenAI-compatible API requests from clients and forwards them to configured LLM providers
// via the Provider Adapters.
//
// # Architecture
//
// The proxy server follows a middleware-based architecture with clean separation of concerns:
//
//   - Server: Main HTTP server with lifecycle management
//   - Handlers: Request processing (chat completions, health checks, WebSocket)
//   - Middleware: Cross-cutting concerns (logging, CORS, request ID, recovery, timeouts)
//   - Types: OpenAI-compatible request/response data structures
//
// # Features
//
//   - OpenAI-compatible API endpoint (/v1/chat/completions)
//   - TLS 1.3 support with configurable certificates
//   - Server-Sent Events (SSE) streaming for real-time responses
//   - WebSocket support for compatible providers
//   - Request/response metadata extraction
//   - Health check endpoints (/health, /ready)
//   - Graceful shutdown with connection draining (<30s)
//   - Request ID generation and propagation
//   - <5ms request overhead (excluding provider latency)
//   - 1000+ concurrent connections support
//
// # Basic Usage
//
// Creating and starting a proxy server:
//
//	import (
//	    "context"
//	    "mercator-hq/jupiter/pkg/config"
//	    "mercator-hq/jupiter/pkg/proxy"
//	    "mercator-hq/jupiter/pkg/providerfactory"
//	)
//
//	// Load configuration
//	cfg := config.GetConfig()
//
//	// Create provider manager
//	manager := providerfactory.NewManager()
//	if err := manager.LoadFromConfig(cfg.Providers); err != nil {
//	    log.Fatal(err)
//	}
//	defer manager.Close()
//
//	// Create and start proxy server
//	server := proxy.NewServer(&cfg.Proxy, manager)
//	if err := server.Start(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Shutdown(context.Background())
//
// # Request Flow
//
// The request flow through the proxy server:
//
//  1. Client sends OpenAI-compatible request to /v1/chat/completions
//  2. Middleware chain processes request (requestID → logging → CORS → timeout)
//  3. Handler parses and validates request body
//  4. Request forwarded to Provider Adapter
//  5. Provider response converted to OpenAI format
//  6. Middleware chain processes response
//  7. Response sent to client (streaming or buffered)
//
// # Streaming Support
//
// The proxy supports Server-Sent Events (SSE) streaming:
//
//	req := &types.ChatCompletionRequest{
//	    Model: "gpt-4",
//	    Messages: []types.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	    Stream: true,
//	}
//
//	// Client will receive SSE chunks:
//	// data: {"id":"...","choices":[{"delta":{"content":"Hello"}}]}
//	// data: {"id":"...","choices":[{"delta":{"content":" there"}}]}
//	// data: [DONE]
//
// # Health Checks
//
// The proxy exposes health check endpoints for load balancers:
//
//   - GET /health - Always returns 200 OK (liveness probe)
//   - GET /ready - Returns 200 if providers are healthy, 503 otherwise (readiness probe)
//
// # Configuration
//
// The proxy server reads configuration from the Configuration System:
//
//	proxy:
//	  listen_address: "127.0.0.1:8080"
//	  read_timeout: "60s"
//	  write_timeout: "60s"
//	  max_connections: 1000
//	  shutdown_timeout: "30s"
//	  cors:
//	    enabled: true
//	    allowed_origins: ["*"]
//
//	security:
//	  tls:
//	    enabled: true
//	    cert_file: "/path/to/cert.pem"
//	    key_file: "/path/to/key.pem"
//
// # Error Handling
//
// All errors follow OpenAI error response format:
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
// # Performance
//
// The proxy server is optimized for low latency and high throughput:
//
//   - <5ms proxy overhead (parsing + forwarding + formatting)
//   - <10ms first chunk latency for streaming
//   - 1000+ concurrent connections without degradation
//   - <10MB memory per connection
//   - <30s graceful shutdown with connection draining
//
// # Security
//
// Security features include:
//
//   - TLS 1.3 minimum version enforcement
//   - Request size limits to prevent DoS
//   - Connection limits to prevent exhaustion
//   - Timeout enforcement to prevent hung connections
//   - CORS validation for web client access
//   - Error sanitization (no secret leakage)
//   - Panic recovery with graceful error responses
//
// # Thread Safety
//
// All proxy server operations are thread-safe and can be called concurrently from
// multiple goroutines. The server uses sync primitives for state management and
// connection tracking.
package proxy
