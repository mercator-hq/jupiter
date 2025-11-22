# Quick Start Guide

Get Mercator Jupiter up and running in 15 minutes! This guide will walk you through installing Jupiter, configuring a basic setup, and making your first LLM request through the proxy.

## Prerequisites

Before you begin, ensure you have:

- **Go 1.21 or later** installed ([Download](https://go.dev/doc/install))
- **Git** for cloning the repository
- **OpenAI API key** (or another LLM provider API key)
- **Terminal/Command line** access

## Step 1: Install Mercator Jupiter (2 minutes)

### Option A: Install from Source

```bash
# Clone the repository
git clone https://github.com/mercator-hq/jupiter.git
cd jupiter

# Build the binary
go build -o mercator ./cmd/mercator

# (Optional) Install to PATH
sudo mv mercator /usr/local/bin/
```

### Option B: Install with Go Install

```bash
go install mercator-hq/jupiter/cmd/mercator@latest
```

### Verify Installation

```bash
mercator version
# Output: Mercator Jupiter v0.1.0
```

## Step 2: Create Configuration (3 minutes)

Create a file named `config.yaml`:

```yaml
# Proxy server configuration
proxy:
  listen_address: "127.0.0.1:8080"

# Configure your LLM provider (OpenAI example)
providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3

# Policy configuration
policy:
  mode: "file"
  file_path: "policies.yaml"

# Evidence storage (audit trail)
evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "evidence.db"

# Logging configuration
telemetry:
  logging:
    level: "info"
    format: "json"
```

**Set your API key** as an environment variable:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

## Step 3: Create a Simple Policy (3 minutes)

Create a file named `policies.yaml` with a basic policy:

```yaml
version: "1.0"

policies:
  - name: "basic-logging"
    description: "Log all requests for audit purposes"
    rules:
      - condition: "true"
        action: "log"
        log_level: "info"
        message: "LLM request from user {{request.metadata.user_id}}"

  - name: "model-allowlist"
    description: "Only allow specific models"
    rules:
      - condition: 'request.model not in ["gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"]'
        action: "deny"
        reason: "Model {{request.model}} is not allowed. Allowed models: gpt-3.5-turbo, gpt-4, gpt-4-turbo"
```

**What this policy does:**
- **basic-logging**: Logs every LLM request with user information
- **model-allowlist**: Only allows specific OpenAI models, blocking others

## Step 4: Validate Your Setup (2 minutes)

Before starting the server, validate your configuration and policies:

```bash
# Validate configuration
mercator run --config config.yaml --dry-run
# Output: Configuration is valid ✓

# Validate policy file
mercator lint --file policies.yaml
# Output: Policy validation passed ✓
```

If validation fails, check:
- API key is set correctly (`echo $OPENAI_API_KEY`)
- YAML syntax is correct (no tabs, proper indentation)
- File paths in config.yaml are correct

## Step 5: Start Mercator Jupiter (1 minute)

Start the proxy server:

```bash
mercator run --config config.yaml
```

You should see output like:

```json
{"level":"info","msg":"Mercator Jupiter starting","version":"0.1.0"}
{"level":"info","msg":"Configuration loaded","config_path":"config.yaml"}
{"level":"info","msg":"Policy engine initialized","policies_loaded":2}
{"level":"info","msg":"Provider registered","provider":"openai"}
{"level":"info","msg":"Evidence recorder initialized","backend":"sqlite"}
{"level":"info","msg":"Proxy server started","address":"127.0.0.1:8080"}
{"level":"info","msg":"Health check endpoint available","url":"http://127.0.0.1:8080/health"}
```

The server is now running! Keep this terminal open.

## Step 6: Make Your First Request (2 minutes)

In a new terminal, send a request through Mercator Jupiter:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-openai-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello! What is Mercator Jupiter?"}
    ]
  }'
```

**Expected response:**

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
        "content": "Mercator Jupiter is a GitOps-native LLM governance runtime..."
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

**What just happened:**
1. Your request went through Mercator Jupiter proxy
2. Jupiter evaluated your policies (logged the request, checked model allowlist)
3. Request was forwarded to OpenAI
4. Response was returned to you
5. Evidence record was created in `evidence.db`

## Step 7: Test Policy Enforcement (2 minutes)

Try requesting a blocked model:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-openai-api-key" \
  -d '{
    "model": "gpt-4-vision-preview",
    "messages": [
      {"role": "user", "content": "This should be blocked"}
    ]
  }'
```

**Expected response:**

```json
{
  "error": {
    "message": "Request denied by policy: Model gpt-4-vision-preview is not allowed. Allowed models: gpt-3.5-turbo, gpt-4, gpt-4-turbo",
    "type": "policy_violation",
    "code": "policy_denied"
  }
}
```

✅ **Policy enforcement works!** Jupiter blocked the request before it reached OpenAI.

## Step 8: Query Evidence Records (Optional, 2 minutes)

Check the audit trail of your requests:

```bash
# Query recent evidence records
mercator evidence query --limit 10

# Export to JSON
mercator evidence query --limit 10 --format json --output evidence.json

# View evidence file
cat evidence.json | jq
```

**Example evidence record:**

```json
{
  "id": "uuid-here",
  "timestamp": "2025-11-21T23:30:00Z",
  "request": {
    "model": "gpt-3.5-turbo",
    "messages": [...]
  },
  "response": {
    "choices": [...],
    "usage": {"total_tokens": 70}
  },
  "policy_decision": {
    "action": "allow",
    "matched_policies": ["basic-logging", "model-allowlist"]
  },
  "cost": 0.0014,
  "latency_ms": 850
}
```

## 🎉 Congratulations!

You've successfully:
- ✅ Installed Mercator Jupiter
- ✅ Created a basic configuration
- ✅ Written your first policy
- ✅ Started the proxy server
- ✅ Made LLM requests through Jupiter
- ✅ Enforced policy rules
- ✅ Generated evidence records

## Next Steps

Now that you have Mercator Jupiter running, explore more features:

### 1. **Create Advanced Policies**
   - [First Policy Guide](first-policy.md) - Learn policy syntax
   - [MPL Language Reference](../mpl/SPECIFICATION.md) - Complete policy language
   - [Policy Cookbook](../policies/cookbook.md) - Real-world examples

### 2. **Add More Providers**
   - [OpenAI Setup](../providers/openai.md) - Advanced OpenAI configuration
   - [Anthropic Setup](../providers/anthropic.md) - Add Claude support
   - [Ollama Setup](../providers/ollama.md) - Use local models

### 3. **Configure Advanced Features**
   - [Budget & Rate Limiting](../limits-usage-guide.md) - Control costs
   - [TLS/mTLS](../SECURITY.md) - Secure communications
   - [Observability](../observability-guide.md) - Metrics and tracing

### 4. **Deploy to Production**
   - [Docker Deployment](../deployment/docker.md) - Containerize Jupiter
   - [Kubernetes Deployment](../deployment/kubernetes.md) - Run on K8s
   - [High Availability](../deployment/high-availability.md) - Scale Jupiter

### 5. **Learn CLI Commands**
   - [CLI Reference](../CLI.md) - All available commands
   - [CLI Cookbook](../CLI-COOKBOOK.md) - Practical recipes

## Common Issues

### Issue: "Connection refused"

**Problem**: Can't connect to http://localhost:8080

**Solution**: Ensure the server is running in another terminal. Check for error messages in the server logs.

### Issue: "Policy validation failed"

**Problem**: `mercator lint` reports errors

**Solution**:
- Check YAML syntax (no tabs, proper indentation)
- Ensure `version: "1.0"` is present
- Verify condition syntax matches MPL specification

### Issue: "Provider authentication failed"

**Problem**: OpenAI returns 401 Unauthorized

**Solution**:
- Verify API key: `echo $OPENAI_API_KEY`
- Check API key is valid on OpenAI dashboard
- Ensure key is passed in Authorization header

### Issue: "Evidence database locked"

**Problem**: SQLite database errors

**Solution**: Only one Mercury process can access SQLite at a time. Stop other instances.

## Stopping Mercator Jupiter

To stop the server:

1. Press `Ctrl+C` in the terminal running `mercator run`
2. Server will gracefully shutdown (waiting up to 30s for in-flight requests)

```
{"level":"info","msg":"Shutdown signal received"}
{"level":"info","msg":"Draining in-flight requests","timeout":"30s"}
{"level":"info","msg":"All requests completed"}
{"level":"info","msg":"Mercator Jupiter stopped"}
```

## Clean Up

To remove test files:

```bash
rm evidence.db evidence.db-shm evidence.db-wal
```

## Getting Help

- **Documentation**: [docs/](../)
- **Examples**: [examples/](../../examples/)
- **Issues**: [GitHub Issues](https://github.com/mercator-hq/jupiter/issues)
- **Discussions**: [GitHub Discussions](https://github.com/mercator-hq/jupiter/discussions)

---

**Next**: [Creating Your First Policy](first-policy.md) →
