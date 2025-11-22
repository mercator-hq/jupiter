// Package types defines OpenAI-compatible request and response types for the proxy server.
//
// This package contains all data transfer objects (DTOs) used for HTTP request/response
// handling. The types are designed to match the OpenAI Chat Completions API format,
// ensuring compatibility with existing OpenAI SDKs and tools.
//
// # Core Types
//
// Request types:
//   - ChatCompletionRequest: Main request body for /v1/chat/completions
//   - Message: Individual message in conversation history
//   - Tool: Function/tool definition for function calling
//   - ToolChoice: Controls which tool the model should use
//
// Response types:
//   - ChatCompletionResponse: Non-streaming response format
//   - ChatCompletionStreamChunk: Streaming response chunk (SSE)
//   - Choice: Individual completion choice
//   - Delta: Incremental content in streaming responses
//   - Usage: Token usage statistics
//
// Error types:
//   - ErrorResponse: OpenAI-compatible error response
//   - ErrorDetail: Error details with type, message, param, code
//
// # OpenAI Compatibility
//
// All types match the OpenAI API specification exactly, allowing clients to use
// standard OpenAI SDKs without modification:
//
//	# Python OpenAI SDK
//	from openai import OpenAI
//	client = OpenAI(base_url="http://localhost:8080/v1")
//	response = client.chat.completions.create(
//	    model="gpt-4",
//	    messages=[{"role": "user", "content": "Hello!"}]
//	)
//
//	// Node.js OpenAI SDK
//	import OpenAI from 'openai';
//	const client = new OpenAI({ baseURL: 'http://localhost:8080/v1' });
//	const response = await client.chat.completions.create({
//	    model: 'gpt-4',
//	    messages: [{ role: 'user', content: 'Hello!' }]
//	});
//
// # JSON Serialization
//
// All types use standard encoding/json for serialization with appropriate struct tags.
// Field names follow OpenAI's snake_case convention for JSON compatibility.
//
// # Validation
//
// Request types include validation logic to ensure required fields are present and
// values are within acceptable ranges. Validation errors are returned in OpenAI
// error format.
package types
