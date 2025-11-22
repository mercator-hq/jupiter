// Package recorder provides evidence recording functionality for the Mercator
// Jupiter LLM proxy. It creates evidence records from enriched requests, policy
// decisions, and enriched responses.
//
// # Recording Flow
//
// Evidence is recorded asynchronously to avoid blocking proxy request handling:
//
//  1. Proxy receives request and enriches it with metadata
//  2. Policy engine evaluates the request
//  3. Evidence recorder creates partial evidence record (request + policy decision)
//  4. Proxy forwards request to provider
//  5. Provider returns response
//  6. Evidence recorder updates evidence record with response data
//  7. Evidence record written to storage backend asynchronously
//
// # Basic Usage
//
//	// Create evidence recorder
//	recorder := recorder.NewRecorder(storage, &recorder.Config{
//	    Enabled: true,
//	    AsyncBuffer: 1000,
//	    WriteTimeout: 5 * time.Second,
//	    HashRequest: true,
//	    HashResponse: true,
//	    RedactAPIKeys: true,
//	})
//	defer recorder.Close()
//
//	// Record request evidence (async)
//	recorder.RecordRequest(ctx, enrichedReq, policyDecision)
//
//	// Record response evidence (async)
//	recorder.RecordResponse(ctx, enrichedResp)
//
// # Async Recording
//
// The recorder uses a buffered channel and background goroutine to record
// evidence asynchronously:
//
//   - RecordRequest() creates evidence record and enqueues to channel (non-blocking)
//   - RecordResponse() updates evidence record and enqueues to channel (non-blocking)
//   - Background goroutine drains channel and writes to storage
//   - Graceful shutdown drains channel before exit (zero data loss)
//
// # Hashing
//
// Request and response bodies are hashed using SHA-256:
//
//   - Hash only first 1MB of large bodies (prevents memory exhaustion)
//   - Hashes are hex-encoded for storage
//   - Hashing can be disabled via configuration
//
// # API Key Redaction
//
// API keys are redacted before storage to prevent leakage:
//
//   - Hash API keys with SHA-256 (cannot be reversed)
//   - Optionally truncate to show only first/last 4 characters
//   - Redaction can be disabled via configuration
//
// # Field Truncation
//
// Long text fields are truncated to prevent unbounded memory growth:
//
//   - System prompts truncated to 500 characters
//   - User prompts truncated to 500 characters
//   - Response content truncated to 500 characters
//   - Full content can be reconstructed from request/response hashes
//
// # Thread Safety
//
// The recorder is thread-safe and can be used concurrently:
//
//   - RecordRequest() and RecordResponse() are thread-safe
//   - Internal channel is protected by mutex
//   - Background goroutine is the only writer to storage
package recorder
