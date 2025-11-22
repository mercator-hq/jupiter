# Configuration Basics

This guide explains the fundamentals of configuring Mercator Jupiter. You'll learn about configuration file structure, environment variables, validation, and best practices.

## Configuration File Format

Mercator Jupiter uses **YAML** for configuration. The configuration file defines:

- Proxy server settings
- LLM provider credentials
- Policy loading options
- Evidence storage backend
- Observability settings
- Security options (TLS, mTLS, API keys)

## Basic Configuration Structure

A minimal configuration looks like this:

```yaml
# config.yaml
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"

policy:
  mode: "file"
  file_path: "policies.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "evidence.db"

telemetry:
  logging:
    level: "info"
    format: "json"
```

## Configuration Sections

### 1. Proxy Configuration

Controls the HTTP proxy server:

```yaml
proxy:
  # Listen address (host:port)
  listen_address: "127.0.0.1:8080"

  # Request/response timeouts
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

  # Graceful shutdown timeout
  shutdown_timeout: "30s"

  # Maximum header size (bytes)
  max_header_bytes: 1048576

  # CORS settings
  cors:
    enabled: true
    allowed_origins: ["*"]
```

**Key settings:**
- `listen_address`: Where the proxy listens (use `0.0.0.0:8080` for all interfaces)
- Timeouts: Prevent hung connections
- CORS: For browser-based clients

### 2. Provider Configuration

Defines LLM provider connections:

```yaml
providers:
  # Provider name (can be anything)
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3

  anthropic:
    base_url: "https://api.anthropic.com/v1"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "60s"

  ollama:
    base_url: "http://localhost:11434"
    timeout: "120s"
```

**Key settings:**
- `base_url`: Provider API endpoint
- `api_key`: Authentication (use environment variables!)
- `timeout`: Request timeout
- `max_retries`: Automatic retry attempts

### 3. Policy Configuration

Specifies where policies are loaded from:

```yaml
policy:
  # Loading mode: "file" or "git"
  mode: "file"

  # File mode settings
  file_path: "policies.yaml"
  watch: true  # Auto-reload on changes

  # Git mode settings (alternative to file)
  # mode: "git"
  # git_repo: "https://github.com/your-org/policies.git"
  # git_branch: "main"
  # git_path: "policies.yaml"
  # git_poll_interval: "60s"

  # Validation
  validation:
    enabled: true
    strict: false
```

**Key settings:**
- `mode`: "file" for local files, "git" for GitOps
- `watch`: Auto-reload when policy file changes
- `strict`: Treat warnings as errors

### 4. Evidence Configuration

Configures audit trail storage:

```yaml
evidence:
  enabled: true
  backend: "sqlite"

  # SQLite backend
  sqlite:
    path: "evidence.db"

  # Retention
  retention_days: 90

  # Cryptographic signing (optional)
  signing_key_path: "/path/to/signing-key.pem"
```

**Key settings:**
- `backend`: "sqlite", "postgres", or "s3"
- `retention_days`: Auto-delete old records
- `signing_key_path`: For cryptographic signatures

### 5. Telemetry Configuration

Controls logging, metrics, and tracing:

```yaml
telemetry:
  logging:
    level: "info"  # debug, info, warn, error
    format: "json"  # json or text

  metrics:
    enabled: true
    prometheus_path: "/metrics"

  tracing:
    enabled: false
    endpoint: "localhost:4317"  # OpenTelemetry collector
```

**Key settings:**
- `logging.level`: Verbosity control
- `metrics.enabled`: Prometheus metrics
- `tracing`: Distributed tracing

### 6. Security Configuration

TLS, mTLS, and authentication:

```yaml
security:
  tls:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"

  mtls:
    enabled: false
    ca_file: "/path/to/ca.pem"

  api_keys:
    - key: "sk-your-api-key"
      name: "production"
      metadata:
        team: "platform"
```

**Key settings:**
- `tls`: HTTPS support
- `mtls`: Client certificate authentication
- `api_keys`: Multiple API keys with metadata

## Environment Variable Overrides

Any configuration value can be overridden with environment variables.

### Format

```
MERCATOR_<SECTION>_<FIELD>
```

### Examples

```bash
# Override listen address
export MERCATOR_PROXY_LISTEN_ADDRESS="0.0.0.0:8080"

# Override OpenAI API key
export MERCATOR_PROVIDERS_OPENAI_API_KEY="sk-abc123"

# Override log level
export MERCATOR_TELEMETRY_LOGGING_LEVEL="debug"

# Override evidence retention
export MERCATOR_EVIDENCE_RETENTION_DAYS="30"
```

### Nested Fields

For nested structures, use underscores:

```bash
# proxy.cors.enabled
export MERCATOR_PROXY_CORS_ENABLED="false"

# evidence.sqlite.path
export MERCATOR_EVIDENCE_SQLITE_PATH="/var/lib/mercator/evidence.db"

# providers.openai.max_retries
export MERCATOR_PROVIDERS_OPENAI_MAX_RETRIES="5"
```

### Using in Config Files

Reference environment variables in YAML:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"

evidence:
  postgres:
    password: "${POSTGRES_PASSWORD}"

security:
  tls:
    cert_file: "${TLS_CERT_PATH}"
```

## Configuration Validation

Mercator Jupiter validates configuration on startup and provides detailed error messages.

### Validate Before Starting

Use dry-run mode to validate without starting:

```bash
mercator run --config config.yaml --dry-run
```

**Success output:**

```
✓ Configuration loaded successfully
✓ All required fields present
✓ All values within valid ranges
✓ Provider credentials valid
✓ Policy file accessible
✓ Evidence backend connectable

Configuration is valid ✓
```

**Error output:**

```
✗ Configuration validation failed

Errors:
  - proxy.listen_address: invalid format, expected "host:port"
  - providers.openai.api_key: required field is empty
  - policy.file_path: file does not exist: policies.yaml

Fix these errors and try again.
```

### Common Validation Errors

**Missing Required Fields:**

```
Error: providers.openai.api_key is required
```

**Solution:** Set the API key in config or environment variable.

**Invalid Values:**

```
Error: proxy.read_timeout: duration must be positive
```

**Solution:** Set a positive duration like "30s".

**File Not Found:**

```
Error: policy.file_path: file not found: /path/to/policies.yaml
```

**Solution:** Check the file path is correct and file exists.

## Loading Configuration

### Default Location

By default, Mercator Jupiter looks for `config.yaml` in:

1. Current directory: `./config.yaml`
2. User config: `~/.mercator/config.yaml`
3. System config: `/etc/mercator/config.yaml`

### Specify Config Path

```bash
# Use custom config file
mercator run --config /path/to/my-config.yaml

# Short form
mercator run -c /path/to/my-config.yaml
```

### Override Specific Values

```bash
# Override listen address
mercator run --config config.yaml --listen 0.0.0.0:8080

# Override log level
mercator run --config config.yaml --log-level debug
```

## Configuration Examples

### Minimal Development Config

For local development and testing:

```yaml
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"

policy:
  mode: "file"
  file_path: "policies.yaml"
  watch: true

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "evidence.db"

telemetry:
  logging:
    level: "debug"
    format: "text"
```

### Production Config

For production deployments:

```yaml
proxy:
  listen_address: "0.0.0.0:8080"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  shutdown_timeout: "30s"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3

  anthropic:
    base_url: "https://api.anthropic.com/v1"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "60s"
    max_retries: 3

policy:
  mode: "git"
  git_repo: "${POLICY_GIT_REPO}"
  git_branch: "main"
  git_path: "policies.yaml"
  git_poll_interval: "60s"
  validation:
    enabled: true
    strict: true

evidence:
  enabled: true
  backend: "postgres"
  postgres:
    host: "${POSTGRES_HOST}"
    port: 5432
    database: "mercator"
    user: "mercator"
    password: "${POSTGRES_PASSWORD}"
    ssl_mode: "require"
  retention_days: 90
  signing_key_path: "/etc/mercator/signing-key.pem"

telemetry:
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: true
    prometheus_path: "/metrics"
  tracing:
    enabled: true
    endpoint: "${OTEL_ENDPOINT}"

security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/tls/cert.pem"
    key_file: "/etc/mercator/tls/key.pem"
  mtls:
    enabled: true
    ca_file: "/etc/mercator/tls/ca.pem"
```

## Best Practices

### 1. Use Environment Variables for Secrets

❌ **Don't** hardcode secrets:

```yaml
providers:
  openai:
    api_key: "sk-abc123..."  # NEVER DO THIS
```

✅ **Do** use environment variables:

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
```

### 2. Use Appropriate Log Levels

- **Development**: `debug` - See everything
- **Staging**: `info` - Normal operations
- **Production**: `info` or `warn` - Reduce noise

### 3. Enable TLS in Production

Always use TLS for production deployments:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/tls/cert.pem"
    key_file: "/etc/mercator/tls/key.pem"
```

### 4. Configure Appropriate Timeouts

Set timeouts based on your use case:

```yaml
proxy:
  read_timeout: "30s"   # For typical requests
  write_timeout: "30s"  # For typical responses
  idle_timeout: "120s"  # For keep-alive connections

providers:
  openai:
    timeout: "60s"   # For quick models

  anthropic:
    timeout: "120s"  # For slower models with long contexts
```

### 5. Use Git Mode for Policy Management

For production, use Git for policy management:

```yaml
policy:
  mode: "git"
  git_repo: "https://github.com/your-org/policies.git"
  git_branch: "production"
  watch: true
```

Benefits:
- Version control for policies
- Code review process
- Rollback capability
- Audit trail

### 6. Configure Evidence Retention

Set appropriate retention based on compliance requirements:

```yaml
evidence:
  retention_days: 90  # GDPR: 90 days
  # retention_days: 2555  # HIPAA: 7 years
  # retention_days: 365  # SOC2: 1 year
```

### 7. Monitor with Observability

Enable metrics and tracing:

```yaml
telemetry:
  metrics:
    enabled: true
  tracing:
    enabled: true
    endpoint: "localhost:4317"
```

## Configuration Hot-Reload

Some configuration can be reloaded without restarting:

### Policy Hot-Reload

```yaml
policy:
  watch: true  # Auto-reload on file changes
```

When policy file changes, Jupiter:
1. Detects the change
2. Validates the new policy
3. If valid, loads it atomically
4. Logs the reload

### Manual Reload

Send `SIGHUP` to reload configuration:

```bash
# Find process ID
pgrep mercator

# Send reload signal
kill -HUP <pid>
```

## Troubleshooting

### Issue: "Configuration file not found"

**Problem**: `mercator run` can't find config file

**Solution:**
- Specify path: `mercator run --config /path/to/config.yaml`
- Or place in default location: `./config.yaml`

### Issue: "Invalid YAML syntax"

**Problem**: YAML parsing error

**Solution:**
- Check indentation (use spaces, not tabs)
- Ensure colons have spaces: `key: value` not `key:value`
- Quote special characters in strings
- Validate YAML: `yamllint config.yaml`

### Issue: "Environment variable not expanded"

**Problem**: `${VAR}` not replaced with value

**Solution:**
- Ensure variable is exported: `export VAR=value`
- Check variable name matches: `echo $VAR`
- Use correct format in YAML: `"${VAR}"`

### Issue: "Provider authentication failed"

**Problem**: API key rejected by provider

**Solution:**
- Verify API key is correct
- Check key has proper permissions
- Ensure key is not expired
- Test key directly with provider

## Next Steps

Now that you understand configuration basics:

- **[Configuration Reference](../configuration/reference.md)** - Complete field documentation
- **[Provider Setup](../providers/openai.md)** - Configure specific providers
- **[Security Configuration](../SECURITY.md)** - TLS, mTLS, API keys
- **[Observability](../observability-guide.md)** - Metrics and tracing setup

## Example Configurations

See the `examples/` directory for more configuration examples:

- `examples/configs/minimal.yaml` - Minimal working config
- `examples/configs/production.yaml` - Production-ready config
- `examples/configs/development.yaml` - Development config
- `examples/basic-config.yaml` - Basic features
- `examples/tls-config.yaml` - TLS configuration
- `examples/observability-config.yaml` - Full observability

---

**Previous**: [Creating Your First Policy](first-policy.md) ← | **Next**: [Configuration Reference](../configuration/reference.md) →
