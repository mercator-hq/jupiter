# Anthropic (Claude) Provider Setup

Complete guide to configuring and optimizing the Anthropic provider for Claude models in Mercator Jupiter.

## Table of Contents

- [Basic Configuration](#basic-configuration)
- [API Key Setup](#api-key-setup)
- [Model Configuration](#model-configuration)
- [Connection Settings](#connection-settings)
- [Rate Limits & Quotas](#rate-limits--quotas)
- [Claude-Specific Features](#claude-specific-features)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Basic Configuration

### Minimal Anthropic Setup

```yaml
# config.yaml
providers:
  anthropic:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "60s"
    max_retries: 3
```

### Full Anthropic Configuration

```yaml
providers:
  anthropic:
    # API endpoint
    base_url: "https://api.anthropic.com"

    # Authentication
    api_key: "${ANTHROPIC_API_KEY}"

    # API version
    api_version: "2023-06-01"

    # Timeouts
    timeout: "90s"  # Claude can be slower than GPT

    # Retry configuration
    max_retries: 3
    retry_delay: "2s"
    retry_backoff: "exponential"

    # Connection pooling
    connection_pool:
      max_idle_conns: 100
      max_idle_conns_per_host: 10
      idle_timeout: "90s"

    # Health checking
    health_check_interval: "30s"
    health_check_timeout: "5s"
```

---

## API Key Setup

### Obtaining an API Key

1. **Sign up at Anthropic**: https://console.anthropic.com/
2. **Navigate to API Keys**: https://console.anthropic.com/account/keys
3. **Create new key**
4. **Copy and secure the key** (shown only once)

### Setting the API Key

#### Option 1: Environment Variable (Recommended)

```bash
# Linux/macOS
export ANTHROPIC_API_KEY="sk-ant-..."

# Windows (Command Prompt)
set ANTHROPIC_API_KEY=sk-ant-...

# Windows (PowerShell)
$env:ANTHROPIC_API_KEY="sk-ant-..."
```

Then reference in config:

```yaml
providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
```

#### Option 2: Secrets Management

```bash
# AWS Secrets Manager
export ANTHROPIC_API_KEY=$(aws secretsmanager get-secret-value \
  --secret-id anthropic-api-key \
  --query SecretString \
  --output text)

# HashiCorp Vault
export ANTHROPIC_API_KEY=$(vault kv get -field=api_key secret/anthropic)
```

### API Key Validation

Test your API key:

```bash
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

---

## Model Configuration

### Available Claude Models

| Model | Context Window | Cost (Input/Output) | Use Case |
|-------|---------------|-------------------|----------|
| `claude-3-opus-20240229` | 200K tokens | $15/$75 per 1M tokens | Most capable, complex tasks |
| `claude-3-sonnet-20240229` | 200K tokens | $3/$15 per 1M tokens | Balanced performance/cost |
| `claude-3-haiku-20240307` | 200K tokens | $0.25/$1.25 per 1M tokens | Fast, cost-effective |
| `claude-2.1` | 200K tokens | $8/$24 per 1M tokens | Previous generation |
| `claude-2.0` | 100K tokens | $8/$24 per 1M tokens | Legacy support |

### Model Routing Policy

```yaml
# policies.yaml
version: "1.0"

policies:
  - name: "anthropic-model-routing"
    description: "Route Claude models to Anthropic"
    rules:
      # Route Claude models to Anthropic
      - condition: 'request.model matches "^claude-"'
        action: "route"
        provider: "anthropic"

      # Block unsupported models
      - condition: 'request.model not matches "^claude-"'
        action: "deny"
        reason: "Model {{request.model}} not supported by Anthropic provider"
```

### Model Selection Policy

```yaml
policies:
  - name: "claude-model-selection"
    rules:
      # Use Opus for complex tasks
      - condition: |
          request.metadata.complexity == "high" or
          request.estimated_prompt_tokens > 50000
        action: "modify"
        set:
          model: "claude-3-opus-20240229"

      # Use Sonnet for balanced tasks
      - condition: |
          request.metadata.complexity == "medium"
        action: "modify"
        set:
          model: "claude-3-sonnet-20240229"

      # Use Haiku for simple/fast tasks
      - condition: |
          request.metadata.complexity == "low" or
          request.metadata.priority == "speed"
        action: "modify"
        set:
          model: "claude-3-haiku-20240307"
```

---

## Connection Settings

### Timeout Configuration

```yaml
providers:
  anthropic:
    # Request timeout
    timeout: "90s"  # Claude can take longer than GPT
```

**Recommendations**:
- `claude-3-haiku`: 30-60 seconds
- `claude-3-sonnet`: 60-90 seconds
- `claude-3-opus`: 90-120 seconds (complex reasoning)
- Large prompts (>100K tokens): 120-180 seconds

### Retry Configuration

```yaml
providers:
  anthropic:
    max_retries: 3
    retry_delay: "2s"
    retry_backoff: "exponential"
```

**Note**: Anthropic has lower rate limits than OpenAI, so be conservative with retries.

### Connection Pooling

```yaml
providers:
  anthropic:
    connection_pool:
      max_idle_conns: 100
      max_idle_conns_per_host: 10
      idle_timeout: "90s"
```

---

## Rate Limits & Quotas

### Anthropic Rate Limits

Anthropic enforces rate limits per organization:

| Tier | RPM | TPM (Input) | TPM (Output) |
|------|-----|-------------|--------------|
| Free | 5 | 25,000 | 25,000 |
| Build Tier 1 | 50 | 100,000 | 100,000 |
| Build Tier 2 | 1,000 | 400,000 | 400,000 |
| Build Tier 3+ | 2,000+ | 2,000,000+ | 2,000,000+ |

### Configuring Rate Limits in Jupiter

```yaml
# config.yaml
limits:
  rate_limiting:
    enabled: true
    default_rpm: 45  # Stay under your tier limit
    default_tpm: 90000  # Input + output tokens
    window_size: "1m"
```

### Rate Limit Policy

```yaml
policies:
  - name: "anthropic-rate-limits"
    rules:
      # Respect Anthropic RPM limits
      - condition: |
          request.metadata.anthropic_rpm_current >= 45
        action: "deny"
        reason: "Anthropic rate limit approaching (45 RPM)"

      # Respect TPM limits
      - condition: |
          request.metadata.anthropic_tpm_current >= 90000
        action: "deny"
        reason: "Anthropic token rate limit approaching"

      # Warn at 80% of limit
      - condition: |
          request.metadata.anthropic_rpm_current >= 36
        action: "log"
        log_level: "warn"
        message: "Approaching Anthropic rate limit: {{request.metadata.anthropic_rpm_current}}/50 RPM"
```

---

## Claude-Specific Features

### System Prompts

Claude supports system prompts (similar to OpenAI):

```json
{
  "model": "claude-3-opus-20240229",
  "max_tokens": 1024,
  "system": "You are a helpful assistant that speaks like a pirate.",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ]
}
```

### Extended Context Windows

Claude 3 models support 200K token context windows:

```yaml
policies:
  - name: "claude-large-context"
    rules:
      # Allow large contexts with Claude
      - condition: |
          request.model matches "^claude-3" and
          request.estimated_total_tokens <= 200000
        action: "allow"

      # Warn on very large contexts
      - condition: |
          request.estimated_total_tokens > 150000
        action: "log"
        log_level: "warn"
        message: "Very large context: {{request.estimated_total_tokens}} tokens"
```

### Vision Support (Claude 3)

Claude 3 supports image inputs:

```json
{
  "model": "claude-3-opus-20240229",
  "max_tokens": 1024,
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "image",
          "source": {
            "type": "base64",
            "media_type": "image/jpeg",
            "data": "base64_encoded_image..."
          }
        },
        {
          "type": "text",
          "text": "What's in this image?"
        }
      ]
    }
  ]
}
```

Policy for vision requests:

```yaml
policies:
  - name: "claude-vision"
    rules:
      # Ensure vision model is used
      - condition: |
          request.has_images == true and
          request.model not matches "^claude-3"
        action: "deny"
        reason: "Image inputs require Claude 3 models"

      # Higher cost for vision
      - condition: |
          request.has_images == true
        action: "log"
        log_level: "info"
        message: "Vision request - costs will be higher"
```

---

## Best Practices

### 1. Use Appropriate Model for Task

```yaml
policies:
  - name: "task-based-model-selection"
    rules:
      # Simple queries -> Haiku
      - condition: |
          request.estimated_prompt_tokens < 1000 and
          request.metadata.task_type == "simple"
        action: "modify"
        set:
          model: "claude-3-haiku-20240307"

      # Complex reasoning -> Opus
      - condition: |
          request.metadata.task_type in ["reasoning", "analysis", "research"]
        action: "modify"
        set:
          model: "claude-3-opus-20240229"

      # Everything else -> Sonnet (balanced)
      - condition: "true"
        action: "modify"
        set:
          model: "claude-3-sonnet-20240229"
```

### 2. Optimize Token Usage

```yaml
policies:
  - name: "claude-token-optimization"
    rules:
      # Cap max_tokens for cost control
      - condition: |
          request.max_tokens > 4000
        action: "modify"
        set:
          max_tokens: 4000

      # Use Haiku for token-heavy responses
      - condition: |
          request.max_tokens > 2000
        action: "modify"
        set:
          model: "claude-3-haiku-20240307"
        log_message: "Large response expected, using Haiku for cost optimization"
```

### 3. Leverage Extended Context

```yaml
# Take advantage of 200K context window
policies:
  - name: "large-context-routing"
    rules:
      # Route large contexts to Claude
      - condition: |
          request.estimated_prompt_tokens > 32000
        action: "route"
        provider: "anthropic"
        log_message: "Large context - routing to Claude (200K window)"
```

### 4. Monitor Costs

```bash
# Query daily Anthropic spending
mercator evidence query \
  --time-range "today" \
  --provider "anthropic" \
  --format json | \
  jq '[.[] | .cost] | add'

# Cost by model
mercator evidence query \
  --provider "anthropic" \
  --format json | \
  jq 'group_by(.request.model) |
      map({model: .[0].request.model, cost: ([.[] | .cost] | add)}) |
      sort_by(.cost) | reverse'
```

### 5. Handle Streaming

Claude supports streaming responses:

```json
{
  "model": "claude-3-opus-20240229",
  "max_tokens": 1024,
  "messages": [...],
  "stream": true
}
```

Policy for streaming:

```yaml
policies:
  - name: "streaming-preference"
    rules:
      # Enable streaming for long responses
      - condition: |
          request.max_tokens > 1000 and
          request.stream != true
        action: "modify"
        set:
          stream: true
        log_message: "Enabling streaming for large response"
```

### 6. Failover to GPT

```yaml
policies:
  - name: "claude-failover"
    rules:
      # Failover to GPT if Claude unavailable
      - condition: |
          request.model matches "^claude-" and
          provider.anthropic.health_status != "healthy"
        action: "modify"
        set:
          model: "gpt-4"
          provider: "openai"
        log_level: "warn"
        log_message: "Claude unavailable, failing over to GPT-4"
```

---

## Troubleshooting

### Issue: "Invalid API Key"

**Symptoms**: 401 authentication errors

**Solutions**:
1. Verify API key:
   ```bash
   echo $ANTHROPIC_API_KEY
   ```
2. Check key format (starts with `sk-ant-`)
3. Verify key on Anthropic console
4. Ensure no extra whitespace

### Issue: "Rate Limit Exceeded"

**Symptoms**: 429 errors

**Solutions**:
1. Check your Anthropic tier limits
2. Implement rate limiting:
   ```yaml
   limits:
     rate_limiting:
       enabled: true
       default_rpm: 45
   ```
3. Upgrade Anthropic tier if needed
4. Use multiple API keys with round-robin

### Issue: "Context Length Exceeded"

**Symptoms**: 400 errors about token limit

**Solutions**:
1. Check total tokens (prompt + max_tokens):
   ```yaml
   policies:
     - condition: |
         request.estimated_total_tokens > 200000
       action: "deny"
       reason: "Exceeds Claude 200K context window"
   ```
2. Reduce prompt length
3. Lower max_tokens
4. Summarize long conversations

### Issue: Slow Responses

**Symptoms**: Timeouts, high latency

**Solutions**:
1. Increase timeout:
   ```yaml
   providers:
     anthropic:
       timeout: "120s"
   ```
2. Use faster models (Haiku instead of Opus)
3. Enable streaming
4. Check Anthropic status: https://status.anthropic.com/

### Issue: High Costs

**Symptoms**: Unexpected bills

**Solutions**:
1. Use Haiku for simple tasks
2. Implement budget limits
3. Monitor token usage
4. Set max_tokens limits

### Debug Mode

Enable debug logging:

```yaml
telemetry:
  logging:
    level: "debug"
```

View Anthropic requests:

```bash
mercator run --log-level debug 2>&1 | grep anthropic
```

---

## Example Configurations

### Development Configuration

```yaml
providers:
  anthropic:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    api_version: "2023-06-01"
    timeout: "60s"
    max_retries: 1
```

### Production Configuration

```yaml
providers:
  anthropic:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    api_version: "2023-06-01"
    timeout: "90s"
    max_retries: 3

    connection_pool:
      max_idle_conns: 100
      max_idle_conns_per_host: 10
      idle_timeout: "90s"

    health_check_interval: "30s"
```

### Multi-Model Configuration

```yaml
providers:
  anthropic-opus:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "120s"

  anthropic-sonnet:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "90s"

  anthropic-haiku:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "60s"
```

---

## See Also

- [Provider Configuration Reference](../configuration/reference.md#provider-configuration)
- [Routing Guide](../policies/routing.md)
- [Budget & Limits Guide](../policies/budget-limits.md)
- [Anthropic Documentation](https://docs.anthropic.com/)
- [Anthropic API Reference](https://docs.anthropic.com/claude/reference)

---

## Quick Reference

### Environment Variables

```bash
ANTHROPIC_API_KEY                    # API key (required)
ANTHROPIC_API_VERSION                # API version
MERCATOR_PROVIDERS_ANTHROPIC_TIMEOUT # Override timeout
```

### Monitoring Metrics

```bash
# Provider health
provider_health{provider="anthropic"}

# Request count
provider_requests_total{provider="anthropic"}

# Error rate
provider_errors_total{provider="anthropic"}

# Latency
provider_latency_seconds{provider="anthropic"}
```

### Common Commands

```bash
# Test Anthropic connection
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-3-haiku-20240307", "messages": [{"role": "user", "content": "test"}]}'

# Check Anthropic usage
mercator evidence query --provider anthropic --time-range "today"

# Monitor costs
mercator evidence query --provider anthropic --format json | \
  jq '[.[] | .cost] | add'
```

### Model Comparison

| Feature | Opus | Sonnet | Haiku |
|---------|------|--------|-------|
| Intelligence | ★★★★★ | ★★★★☆ | ★★★☆☆ |
| Speed | ★★☆☆☆ | ★★★★☆ | ★★★★★ |
| Cost | $$$$$ | $$$ | $ |
| Context | 200K | 200K | 200K |
| Vision | ✓ | ✓ | ✓ |
| Best for | Complex reasoning | Balanced tasks | Fast, simple queries |
