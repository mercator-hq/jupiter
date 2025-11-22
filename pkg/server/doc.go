// Package server provides the main HTTP proxy server for LLM traffic.
//
// This package ties together all proxy components (handlers, middleware, routing)
// and provides server lifecycle management including start, shutdown, and health checks.
//
// # Architecture
//
// The server package is the top-level orchestrator that:
//   - Sets up HTTP routes and handlers
//   - Chains middleware for cross-cutting concerns
//   - Configures TLS termination
//   - Manages graceful shutdown
//   - Handles OS signals (SIGTERM, SIGINT)
//
// # Basic Usage
//
// Creating and starting a server:
//
//	import (
//	    "context"
//	    "mercator-hq/jupiter/pkg/config"
//	    "mercator-hq/jupiter/pkg/server"
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
//	// Create and start server
//	srv := server.NewServer(&cfg.Proxy, &cfg.Security, manager)
//	if err := srv.Start(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// # Graceful Shutdown
//
// The server handles graceful shutdown automatically when receiving SIGTERM or SIGINT:
//
//	// Server will automatically shutdown on SIGTERM/SIGINT
//	// Or you can trigger shutdown programmatically:
//	if err := srv.Shutdown(context.Background()); err != nil {
//	    log.Error("shutdown error", "error", err)
//	}
//
// The shutdown process:
//  1. Stops accepting new connections
//  2. Waits for active connections to complete (up to shutdown timeout)
//  3. Forces connection closure if timeout exceeded
//  4. Cleans up resources
//
// # Routes
//
// The server exposes the following HTTP endpoints:
//
//   - POST /v1/chat/completions - Chat completion (streaming and non-streaming)
//   - GET /health - Liveness probe (always returns 200)
//   - GET /ready - Readiness probe (checks provider health)
//   - GET /health/providers - Detailed provider health information
//   - WS /v1/chat/completions/ws - WebSocket connection (not implemented in MVP)
//
// # Middleware Chain
//
// Requests pass through the following middleware (innermost to outermost):
//  1. Timeout: Enforces per-request timeout
//  2. CORS: Adds Cross-Origin Resource Sharing headers
//  3. RequestID: Generates unique request ID for tracing
//  4. Logging: Logs request/response details
//  5. Recovery: Recovers from panics and returns 500 error
//
// # TLS Support
//
// The server supports TLS 1.3 with configurable certificates:
//
//	security:
//	  tls:
//	    enabled: true
//	    cert_file: "/path/to/cert.pem"
//	    key_file: "/path/to/key.pem"
//	    min_version: "1.3"
//
// TLS configuration enforces:
//   - TLS 1.3 minimum version
//   - Secure cipher suites only
//   - Server cipher suite preference
//
// # Health Checks
//
// The server provides health check endpoints for Kubernetes/load balancers:
//
//	# Liveness probe (is server running?)
//	GET /health
//
//	# Readiness probe (is server ready to accept traffic?)
//	GET /ready
//
//	# Detailed provider health
//	GET /health/providers
//
// # Thread Safety
//
// All server operations are thread-safe and can be called concurrently from
// multiple goroutines.
package server
