# OpenAI Provider Setup

Complete guide to configuring and optimizing the OpenAI provider in Mercator Jupiter.

## Table of Contents

- [Basic Configuration](#basic-configuration)
- [API Key Setup](#api-key-setup)
- [Model Configuration](#model-configuration)
- [Connection Settings](#connection-settings)
- [Rate Limits & Quotas](#rate-limits--quotas)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Basic Configuration

### Minimal OpenAI Setup

```yaml
# config.yaml
providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3
```

### Full OpenAI Configuration

```yaml
providers:
  openai:
    # API endpoint
    base_url: "https://api.openai.com/v1"

    # Authentication
    api_key: "${OPENAI_API_KEY}"

    # Timeouts
    timeout: "60s"

    # Retry configuration
    max_retries: 3
    retry_delay: "1s"
    retry_backoff: "exponential"

    # Connection pooling
    connection_pool:
      max_idle_conns: 100
      max_idle_conns_per_host: 10
      idle_timeout: "90s"

    # Health checking
    health_check_interval: "30s"
    health_check_timeout: "5s"
    health_check_failures_threshold: 3
```

---

## API Key Setup

### Obtaining an API Key

1. **Sign up at OpenAI**: https://platform.openai.com/signup
2. **Navigate to API Keys**: https://platform.openai.com/api-keys
3. **Create new secret key**
4. **Copy and secure the key** (shown only once)

### Setting the API Key

#### Option 1: Environment Variable (Recommended)

```bash
# Linux/macOS
export OPENAI_API_KEY="sk-proj-..."

# Windows (Command Prompt)
set OPENAI_API_KEY=sk-proj-...

# Windows (PowerShell)
$env:OPENAI_API_KEY="sk-proj-..."
```

Then reference in config:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
```

#### Option 2: Configuration File

```yaml
providers:
  openai:
    api_key: "sk-proj-..."  # Not recommended - use environment variables
```

⚠️ **Security Warning**: Never commit API keys to version control.

#### Option 3: Secrets Management

For production deployments, use a secrets manager:

```bash
# AWS Secrets Manager
export OPENAI_API_KEY=$(aws secretsmanager get-secret-value \
  --secret-id openai-api-key \
  --query SecretString \
  --output text)

# HashiCorp Vault
export OPENAI_API_KEY=$(vault kv get -field=api_key secret/openai)

# Kubernetes Secret
# Mount secret as environment variable in pod spec
```

### API Key Validation

Test your API key:

```bash
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

---

## Model Configuration

### Available Models

OpenAI provides several model families:

| Model | Context Window | Cost (Input/Output) | Use Case |
|-------|---------------|-------------------|----------|
| `gpt-4-turbo` | 128K tokens | $10/$30 per 1M tokens | Complex tasks, large contexts |
| `gpt-4` | 8K tokens | $30/$60 per 1M tokens | High-quality responses |
| `gpt-3.5-turbo` | 16K tokens | $0.50/$1.50 per 1M tokens | Fast, cost-effective |
| `gpt-3.5-turbo-16k` | 16K tokens | $3/$4 per 1M tokens | Extended context |

### Model Routing Policy

```yaml
# policies.yaml
version: "1.0"

policies:
  - name: "openai-model-routing"
    description: "Route OpenAI models appropriately"
    rules:
      # Route GPT models to OpenAI
      - condition: 'request.model matches "^gpt-"'
        action: "route"
        provider: "openai"

      # Block unsupported models
      - condition: 'request.model not matches "^gpt-"'
        action: "deny"
        reason: "Model {{request.model}} not supported by OpenAI provider"
```

### Model Allowlist

```yaml
policies:
  - name: "openai-model-allowlist"
    rules:
      # Only allow specific GPT models
      - condition: |
          request.model not in [
            "gpt-3.5-turbo",
            "gpt-4",
            "gpt-4-turbo"
          ]
        action: "deny"
        reason: "Model not approved. Allowed: gpt-3.5-turbo, gpt-4, gpt-4-turbo"
```

---

## Connection Settings

### Timeout Configuration

```yaml
providers:
  openai:
    # Request timeout
    timeout: "60s"  # 60 seconds for most requests

    # Model-specific timeouts (if needed via routing)
```

**Recommendations**:
- `gpt-3.5-turbo`: 30-60 seconds
- `gpt-4`: 60-120 seconds (slower)
- Streaming: 120+ seconds

### Retry Configuration

```yaml
providers:
  openai:
    max_retries: 3
    retry_delay: "1s"
    retry_backoff: "exponential"  # 1s, 2s, 4s
```

**Retry behavior**:
- Rate limit errors (429): Retry with backoff
- Server errors (500, 502, 503): Retry with backoff
- Timeout errors: Retry once
- Client errors (400, 401, 403): No retry

### Connection Pooling

```yaml
providers:
  openai:
    connection_pool:
      max_idle_conns: 100           # Total idle connections
      max_idle_conns_per_host: 10   # Per host
      idle_timeout: "90s"           # Keep-alive time
      max_conns_per_host: 50        # Max concurrent per host
```

**Recommendations**:
- High traffic: Increase `max_idle_conns` to 200+
- Multiple regions: Increase `max_idle_conns_per_host`
- Low traffic: Keep defaults

---

## Rate Limits & Quotas

### OpenAI Rate Limits

OpenAI enforces rate limits per organization:

| Tier | RPM | TPM | Batch Queue |
|------|-----|-----|-------------|
| Free | 3 | 40,000 | 200,000 |
| Tier 1 | 500 | 200,000 | 2,000,000 |
| Tier 2 | 5,000 | 2,000,000 | 20,000,000 |
| Tier 3+ | Higher | Higher | Higher |

### Configuring Rate Limits in Jupiter

```yaml
# config.yaml
limits:
  rate_limiting:
    enabled: true
    default_rpm: 450  # Stay under OpenAI's 500 RPM limit
    default_tpm: 180000  # Stay under 200K TPM limit
    window_size: "1m"
```

### Rate Limit Policy

```yaml
# policies.yaml
policies:
  - name: "openai-rate-limits"
    rules:
      # Respect OpenAI RPM limits
      - condition: |
          request.metadata.openai_rpm_current >= 450
        action: "deny"
        reason: "OpenAI rate limit approaching (450 RPM). Please wait before retrying."

      # Respect OpenAI TPM limits
      - condition: |
          request.metadata.openai_tpm_current >= 180000
        action: "deny"
        reason: "OpenAI token rate limit approaching (180K TPM)"
```

### Handling Rate Limit Errors

```yaml
routing:
  failover:
    enabled: true
    on_rate_limit: "retry_with_backoff"
    max_retries: 3
    retry_delay: "5s"
```

---

## Error Handling

### Common OpenAI Errors

| Status Code | Error | Handling |
|-------------|-------|----------|
| 400 | Bad Request | Client error, don't retry |
| 401 | Invalid API Key | Check API key configuration |
| 403 | Forbidden | Check API key permissions |
| 404 | Model Not Found | Check model name |
| 429 | Rate Limit | Retry with backoff |
| 500 | Server Error | Retry automatically |
| 503 | Service Unavailable | Retry automatically |

### Error Handling Policy

```yaml
policies:
  - name: "openai-error-handling"
    rules:
      # Log all provider errors
      - condition: |
          response.error != null
        action: "log"
        log_level: "error"
        message: "OpenAI error: {{response.error.type}} - {{response.error.message}}"

      # Alert on authentication errors
      - condition: |
          response.status_code == 401
        action: "log"
        log_level: "error"
        message: "ALERT: OpenAI API key invalid or expired"

      # Failover on service errors
      - condition: |
          response.status_code >= 500
        action: "route"
        provider: "azure-openai"  # Failover provider
        log_message: "OpenAI server error, failing over to Azure"
```

---

## Best Practices

### 1. Use Streaming for Long Responses

```yaml
# Enable streaming in requests
{
  "model": "gpt-4",
  "messages": [...],
  "stream": true
}
```

Benefits:
- Lower perceived latency
- Better user experience
- Easier to handle timeouts

### 2. Optimize Token Usage

```yaml
policies:
  - name: "token-optimization"
    rules:
      # Limit max_tokens for cost control
      - condition: |
          request.max_tokens > 2000
        action: "modify"
        set:
          max_tokens: 2000
        log_message: "Capped max_tokens at 2000"

      # Warn on large prompts
      - condition: |
          request.estimated_prompt_tokens > 4000
        action: "log"
        log_level: "warn"
        message: "Large prompt: {{request.estimated_prompt_tokens}} tokens"
```

### 3. Monitor Costs

```bash
# Query daily OpenAI spending
mercator evidence query \
  --time-range "today" \
  --provider "openai" \
  --format json | \
  jq '[.[] | .cost] | add'

# Cost by model
mercator evidence query \
  --provider "openai" \
  --format json | \
  jq 'group_by(.request.model) |
      map({model: .[0].request.model, cost: ([.[] | .cost] | add)}) |
      sort_by(.cost) | reverse'
```

### 4. Implement Caching

For repeated requests, implement caching:

```yaml
# Conceptual - requires custom implementation
policies:
  - name: "response-caching"
    rules:
      - condition: |
          cache.has(request.messages_hash)
        action: "modify"
        set:
          use_cached_response: true
```

### 5. Use Organization IDs

For tracking and billing:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    organization_id: "${OPENAI_ORG_ID}"  # Optional
```

### 6. Multiple API Keys for Rate Limit Distribution

```yaml
providers:
  openai-1:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY_1}"

  openai-2:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY_2}"

routing:
  strategy: "round-robin"  # Distribute across keys
```

### 7. Health Monitoring

```yaml
providers:
  openai:
    health_check_interval: "30s"
    health_check_timeout: "5s"
    health_check_failures_threshold: 3
```

```bash
# Check provider health
curl http://localhost:8080/metrics | grep 'provider_health{provider="openai"}'
```

---

## Troubleshooting

### Issue: "Invalid API Key"

**Symptoms**: 401 Unauthorized errors

**Solutions**:
1. Verify API key is set correctly:
   ```bash
   echo $OPENAI_API_KEY
   ```
2. Check API key on OpenAI dashboard
3. Ensure no extra whitespace in key
4. Try using key directly in config temporarily to test

### Issue: "Rate Limit Exceeded"

**Symptoms**: 429 errors, "Rate limit reached"

**Solutions**:
1. Check your OpenAI tier limits
2. Implement rate limiting in Jupiter:
   ```yaml
   limits:
     rate_limiting:
       enabled: true
       default_rpm: 450
   ```
3. Use multiple API keys
4. Upgrade OpenAI tier if needed

### Issue: "Model Not Found"

**Symptoms**: 404 errors

**Solutions**:
1. Check model name spelling
2. Verify model is available in your region
3. Ensure model access is granted to your API key
4. Check OpenAI model availability: https://platform.openai.com/docs/models

### Issue: Slow Responses

**Symptoms**: Requests timing out, high latency

**Solutions**:
1. Increase timeout:
   ```yaml
   providers:
     openai:
       timeout: "120s"
   ```
2. Use faster models (gpt-3.5-turbo instead of gpt-4)
3. Enable streaming
4. Check OpenAI status: https://status.openai.com/

### Issue: High Costs

**Symptoms**: Unexpected bills

**Solutions**:
1. Implement budget limits:
   ```yaml
   limits:
     budgets:
       enabled: true
       default_daily_limit: 100.0
   ```
2. Monitor token usage
3. Use cheaper models when possible
4. Implement token limits per request

### Issue: Connection Errors

**Symptoms**: "Connection refused", timeouts

**Solutions**:
1. Check network connectivity to api.openai.com
2. Verify firewall rules allow HTTPS (443)
3. Check proxy settings if behind corporate proxy
4. Verify DNS resolution

### Debug Mode

Enable debug logging:

```yaml
telemetry:
  logging:
    level: "debug"
```

View OpenAI requests/responses:

```bash
# Watch logs
mercator run --config config.yaml --log-level debug 2>&1 | grep openai
```

---

## Advanced Configuration

### Custom Base URL (Azure OpenAI)

```yaml
providers:
  azure-openai:
    base_url: "https://your-resource.openai.azure.com"
    api_key: "${AZURE_OPENAI_KEY}"
    api_version: "2024-02-01"
```

### Proxy Configuration

If behind a corporate proxy:

```bash
export HTTPS_PROXY="http://proxy.company.com:8080"
export NO_PROXY="localhost,127.0.0.1"
```

Or in code (if needed):

```yaml
providers:
  openai:
    proxy_url: "http://proxy.company.com:8080"
```

### Custom Headers

```yaml
providers:
  openai:
    custom_headers:
      X-Custom-Header: "value"
      User-Agent: "MyApp/1.0"
```

---

## Example Configurations

### Development Configuration

```yaml
providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "30s"
    max_retries: 1  # Fail fast in development
```

### Production Configuration

```yaml
providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3

    connection_pool:
      max_idle_conns: 200
      max_idle_conns_per_host: 20
      idle_timeout: "90s"

    health_check_interval: "30s"
    health_check_timeout: "5s"
```

### High-Availability Configuration

```yaml
providers:
  openai-primary:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY_PRIMARY}"
    timeout: "60s"

  openai-secondary:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY_SECONDARY}"
    timeout: "60s"

routing:
  failover:
    enabled: true
    providers: ["openai-primary", "openai-secondary"]
```

---

## See Also

- [Provider Configuration Reference](../configuration/reference.md#provider-configuration)
- [Routing Guide](../policies/routing.md)
- [Budget & Limits Guide](../policies/budget-limits.md)
- [OpenAI Documentation](https://platform.openai.com/docs)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)

---

## Quick Reference

### Environment Variables

```bash
OPENAI_API_KEY                    # API key (required)
OPENAI_ORG_ID                     # Organization ID (optional)
MERCATOR_PROVIDERS_OPENAI_TIMEOUT # Override timeout
```

### Monitoring Metrics

```bash
# Provider health
provider_health{provider="openai"}

# Request count
provider_requests_total{provider="openai"}

# Error rate
provider_errors_total{provider="openai"}

# Latency
provider_latency_seconds{provider="openai"}
```

### Common Commands

```bash
# Test OpenAI connection
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "test"}]}'

# Check OpenAI usage
mercator evidence query --provider openai --time-range "today"

# Monitor OpenAI costs
mercator evidence query --provider openai --format json | \
  jq '[.[] | .cost] | add'
```
