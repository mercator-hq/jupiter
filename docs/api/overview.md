# HTTP API Overview

Mercator Jupiter provides an OpenAI-compatible HTTP API for LLM requests. This allows drop-in replacement of OpenAI client libraries.

## Table of Contents

- [API Compatibility](#api-compatibility)
- [Base URL](#base-url)
- [Authentication](#authentication)
- [Endpoints](#endpoints)
- [Request Format](#request-format)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

---

## API Compatibility

Mercator Jupiter implements the **OpenAI Chat Completions API** format, making it compatible with:

- OpenAI Python SDK
- OpenAI Node.js SDK
- LangChain
- LlamaIndex
- Any OpenAI-compatible client

### Supported Features

✅ Chat completions (non-streaming)
✅ Chat completions (streaming)
✅ Function calling
✅ System messages
✅ Multi-turn conversations
✅ Vision/multimodal inputs (provider-dependent)
✅ Token usage tracking

### Unsupported Features

❌ Embeddings (planned for Phase 2)
❌ Fine-tuning
❌ Image generation
❌ Audio transcription
❌ Moderation endpoint (use policies instead)

---

## Base URL

Default base URL when running locally:

```
http://localhost:8080
```

**Production**:
- With TLS: `https://jupiter.your-domain.com`
- Without TLS: `http://jupiter.your-domain.com:8080`

---

## Authentication

### API Key Authentication

Pass your API key in the `Authorization` header:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-3.5-turbo", "messages": [...]}'
```

### No Authentication (Development)

For development, authentication can be disabled:

```yaml
# config.yaml
security:
  api_keys: []  # Empty = no auth required
```

⚠️ **Never disable authentication in production!**

---

## Endpoints

### Chat Completions

**POST** `/v1/chat/completions`

Create a chat completion (the main endpoint for LLM requests).

**Request Body**:
```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7,
  "max_tokens": 150,
  "stream": false
}
```

**Response** (non-streaming):
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "gpt-3.5-turbo",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 10,
    "total_tokens": 30
  }
}
```

See: [Chat Completions Documentation](chat-completions.md)

### Health Check

**GET** `/health`

Check server health.

**Response**:
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "providers": {
    "openai": "healthy",
    "anthropic": "healthy"
  }
}
```

### Metrics (Prometheus)

**GET** `/metrics`

Prometheus metrics endpoint.

**Response**: Prometheus format
```
# HELP mercator_requests_total Total requests
# TYPE mercator_requests_total counter
mercator_requests_total{provider="openai"} 1234
...
```

See: [Observability Guide](../observability-guide.md)

---

## Request Format

### Required Fields

```json
{
  "model": "gpt-3.5-turbo",  // Required: Model to use
  "messages": [              // Required: Conversation messages
    {"role": "user", "content": "Hello"}
  ]
}
```

### Optional Fields

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [...],

  // Generation parameters
  "temperature": 0.7,        // 0.0 to 2.0
  "max_tokens": 150,         // Maximum completion tokens
  "top_p": 1.0,              // Nucleus sampling
  "frequency_penalty": 0.0,  // -2.0 to 2.0
  "presence_penalty": 0.0,   // -2.0 to 2.0
  "stop": ["\n"],            // Stop sequences

  // Streaming
  "stream": false,           // Enable SSE streaming

  // Function calling
  "functions": [...],        // Function definitions
  "function_call": "auto",   // "auto", "none", or {"name": "function"}

  // Metadata (Mercator-specific)
  "metadata": {
    "user_id": "user-123",
    "team_id": "team-456"
  }
}
```

### Message Format

```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant."
    },
    {
      "role": "user",
      "content": "Hello!"
    },
    {
      "role": "assistant",
      "content": "Hi! How can I help?"
    },
    {
      "role": "user",
      "content": "Tell me a joke."
    }
  ]
}
```

**Valid roles**: `system`, `user`, `assistant`, `function`

### Multimodal Messages

For vision-capable models (Claude 3, GPT-4V):

```json
{
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "What's in this image?"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://example.com/image.jpg"
          }
        }
      ]
    }
  ]
}
```

---

## Response Format

### Non-Streaming Response

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1700000000,
  "model": "gpt-3.5-turbo",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Response text here"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 50,
    "total_tokens": 70
  }
}
```

**Finish Reasons**:
- `stop`: Natural completion
- `length`: Hit max_tokens limit
- `function_call`: Called a function
- `content_filter`: Filtered by policy

### Streaming Response

When `stream: true`, responses use Server-Sent Events (SSE):

```
data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1700000000,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1700000000,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1700000000,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1700000000,"model":"gpt-3.5-turbo","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

See: [Streaming Documentation](streaming.md)

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "message": "Request denied by policy: Budget exceeded",
    "type": "policy_violation",
    "code": "policy_denied"
  }
}
```

### HTTP Status Codes

| Status | Meaning | Retry? |
|--------|---------|--------|
| 200 | Success | - |
| 400 | Bad Request | ❌ No |
| 401 | Unauthorized | ❌ No |
| 403 | Forbidden (Policy) | ❌ No |
| 404 | Model Not Found | ❌ No |
| 429 | Rate Limit | ✅ Yes (with backoff) |
| 500 | Server Error | ✅ Yes |
| 502 | Bad Gateway | ✅ Yes |
| 503 | Service Unavailable | ✅ Yes |

### Error Types

**`invalid_request_error`**: Client error (bad request format)
```json
{
  "error": {
    "message": "Missing required field: messages",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

**`policy_violation`**: Request blocked by policy
```json
{
  "error": {
    "message": "Request denied by policy: Budget exceeded",
    "type": "policy_violation",
    "code": "policy_denied",
    "policy": "budget-enforcement"
  }
}
```

**`rate_limit_error`**: Rate limit exceeded
```json
{
  "error": {
    "message": "Rate limit exceeded: 60 requests per minute",
    "type": "rate_limit_error",
    "code": "rate_limit_exceeded"
  }
}
```

**`provider_error`**: Upstream provider error
```json
{
  "error": {
    "message": "Provider error: Service temporarily unavailable",
    "type": "provider_error",
    "code": "service_unavailable",
    "provider": "openai"
  }
}
```

---

## Rate Limiting

### Rate Limit Headers

Mercator includes rate limit information in response headers:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1700000060
```

### Handling Rate Limits

When rate limited (429 response):

```json
{
  "error": {
    "message": "Rate limit exceeded. Retry after 30 seconds.",
    "type": "rate_limit_error",
    "code": "rate_limit_exceeded",
    "retry_after": 30
  }
}
```

**Retry Logic**:
1. Wait for `retry_after` seconds
2. Implement exponential backoff
3. Maximum 3 retry attempts

**Example (Python)**:
```python
import time
import openai

for attempt in range(3):
    try:
        response = client.chat.completions.create(...)
        break
    except openai.RateLimitError as e:
        if attempt < 2:
            wait_time = 2 ** attempt  # Exponential backoff
            time.sleep(wait_time)
        else:
            raise
```

---

## Using with Client Libraries

### OpenAI Python SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

### OpenAI Node.js SDK

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'your-api-key'
});

const response = await client.chat.completions.create({
  model: 'gpt-3.5-turbo',
  messages: [
    { role: 'user', content: 'Hello!' }
  ]
});

console.log(response.choices[0].message.content);
```

### LangChain

```python
from langchain.chat_models import ChatOpenAI

llm = ChatOpenAI(
    openai_api_base="http://localhost:8080/v1",
    openai_api_key="your-api-key",
    model_name="gpt-3.5-turbo"
)

response = llm.predict("Hello!")
print(response)
```

### cURL

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

---

## Custom Metadata

Mercator allows custom metadata for policy enforcement:

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [...],
  "metadata": {
    "user_id": "user-123",
    "team_id": "team-456",
    "department": "engineering",
    "cost_center": "cc-789",
    "priority": "high"
  }
}
```

This metadata is available in policies:
```yaml
- condition: 'request.metadata.priority == "high"'
  action: "route"
  provider: "openai-premium"
```

---

## See Also

- [Chat Completions API](chat-completions.md) - Detailed API reference
- [Streaming Responses](streaming.md) - SSE streaming guide
- [Authentication](authentication.md) - API key management
- [CLI Reference](../CLI.md) - Command-line tools
- [Policy Language](../mpl/SPECIFICATION.md) - MPL documentation

---

## Quick Reference

### Endpoints

```
POST /v1/chat/completions    # Chat completions
GET  /health                 # Health check
GET  /metrics                # Prometheus metrics
```

### Headers

```
Authorization: Bearer <api-key>    # Required
Content-Type: application/json     # Required
X-Request-ID: <uuid>              # Optional (for tracing)
X-User-ID: <user-id>              # Optional (for policies)
```

### Common Models

```
gpt-3.5-turbo          # OpenAI (fast, cheap)
gpt-4                  # OpenAI (high quality)
gpt-4-turbo            # OpenAI (large context)
claude-3-opus          # Anthropic (highest quality)
claude-3-sonnet        # Anthropic (balanced)
claude-3-haiku         # Anthropic (fast, cheap)
llama2                 # Ollama (local, free)
```

### Example Request

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}],
    "temperature": 0.7,
    "max_tokens": 150
  }'
```
