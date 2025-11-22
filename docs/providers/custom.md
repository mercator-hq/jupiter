# Custom Provider Integration

Guide to integrating custom LLM providers with Mercator Jupiter.

## Overview

Mercator Jupiter can work with any OpenAI-compatible API endpoint. This allows you to integrate:

- Custom fine-tuned models
- Self-hosted models (vLLM, Text Generation Inference)
- Alternative cloud providers (Together AI, Replicate, Anyscale)
- Internal model endpoints
- Proxy services (OpenRouter, Portkey)

## Basic Custom Provider Setup

### OpenAI-Compatible Endpoint

If your provider implements the OpenAI API format:

```yaml
# config.yaml
providers:
  my-custom-provider:
    base_url: "https://api.myprovider.com/v1"
    api_key: "${CUSTOM_PROVIDER_API_KEY}"
    timeout: "60s"
    max_retries: 3
```

### Example: Together AI

```yaml
providers:
  together:
    base_url: "https://api.together.xyz/v1"
    api_key: "${TOGETHER_API_KEY}"
    timeout: "60s"
```

### Example: OpenRouter

```yaml
providers:
  openrouter:
    base_url: "https://openrouter.ai/api/v1"
    api_key: "${OPENROUTER_API_KEY}"
    timeout: "60s"
```

### Example: vLLM (Self-Hosted)

```yaml
providers:
  vllm:
    base_url: "http://vllm-server:8000/v1"
    timeout: "120s"
    # No API key for self-hosted
```

## Routing to Custom Providers

```yaml
# policies.yaml
version: "1.0"

policies:
  - name: "custom-provider-routing"
    rules:
      # Route specific models to custom provider
      - condition: 'request.model == "my-custom-model"'
        action: "route"
        provider: "my-custom-provider"

      # Fallback to OpenAI
      - condition: "true"
        action: "route"
        provider: "openai"
```

## Advanced Configuration

### Custom Headers

```yaml
providers:
  my-custom-provider:
    base_url: "https://api.example.com/v1"
    api_key: "${API_KEY}"
    custom_headers:
      X-Custom-Header: "value"
      X-Organization-ID: "${ORG_ID}"
```

### Authentication Variations

#### Bearer Token

```yaml
providers:
  custom:
    base_url: "https://api.example.com"
    api_key: "Bearer ${TOKEN}"
```

#### Basic Auth

```yaml
providers:
  custom:
    base_url: "https://api.example.com"
    api_key: "Basic ${BASE64_CREDENTIALS}"
```

#### API Key in Header

```yaml
providers:
  custom:
    base_url: "https://api.example.com"
    custom_headers:
      X-API-Key: "${API_KEY}"
```

## Testing Custom Providers

### Test Connection

```bash
# Test provider directly
curl -X POST https://api.myprovider.com/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-model",
    "messages": [{"role": "user", "content": "test"}]
  }'

# Test through Mercator
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-model",
    "messages": [{"role": "user", "content": "test"}]
  }'
```

### Validate Configuration

```bash
# Dry-run to validate
mercator run --config config.yaml --dry-run
```

## Common Integration Patterns

### 1. Cost-Effective Alternative

```yaml
# Use cheaper custom provider
providers:
  custom-cheap:
    base_url: "https://cheap-api.com/v1"
    api_key: "${CHEAP_API_KEY}"

# Route when budget low
policies:
  - name: "cost-optimization"
    rules:
      - condition: |
          request.metadata.user_budget_remaining < 10.0
        action: "route"
        provider: "custom-cheap"
```

### 2. Specialized Models

```yaml
# Provider with domain-specific models
providers:
  medical-llm:
    base_url: "https://medical-api.com/v1"
    api_key: "${MEDICAL_API_KEY}"

policies:
  - name: "domain-routing"
    rules:
      - condition: |
          request.metadata.domain == "healthcare"
        action: "route"
        provider: "medical-llm"
```

### 3. Geographic Distribution

```yaml
# Providers in different regions
providers:
  provider-us:
    base_url: "https://us.api.com/v1"
  provider-eu:
    base_url: "https://eu.api.com/v1"
  provider-asia:
    base_url: "https://asia.api.com/v1"

policies:
  - name: "geographic-routing"
    rules:
      - condition: 'request.metadata.user_region == "US"'
        action: "route"
        provider: "provider-us"
```

## Self-Hosted Models

### vLLM

```bash
# Start vLLM server
python -m vllm.entrypoints.openai.api_server \
  --model meta-llama/Llama-2-7b-hf \
  --port 8000
```

```yaml
providers:
  vllm:
    base_url: "http://localhost:8000/v1"
    timeout: "120s"
```

### Text Generation Inference

```bash
# Start TGI
docker run -p 8080:80 \
  -v /data:/data \
  ghcr.io/huggingface/text-generation-inference:latest \
  --model-id meta-llama/Llama-2-7b-hf
```

```yaml
providers:
  tgi:
    base_url: "http://localhost:8080/v1"
    timeout: "120s"
```

## Troubleshooting

### Issue: "Provider Not Responding"

**Solutions**:
1. Verify base_url is correct
2. Check network connectivity
3. Verify API key
4. Check provider status
5. Increase timeout

### Issue: "Invalid Response Format"

**Solutions**:
1. Verify provider uses OpenAI-compatible format
2. Check API version compatibility
3. Review provider documentation
4. Test provider API directly

### Issue: "Authentication Failed"

**Solutions**:
1. Verify API key format
2. Check header requirements
3. Review provider auth documentation
4. Test authentication separately

## See Also

- [OpenAI Provider](openai.md)
- [Anthropic Provider](anthropic.md)
- [Ollama Provider](ollama.md)
- [Routing Guide](../policies/routing.md)
- [Configuration Reference](../configuration/reference.md)

## Examples

For complete configuration examples, see:
- [examples/providers-config.yaml](../../examples/providers-config.yaml)
- [examples/configs/production.yaml](../../examples/configs/production.yaml)
