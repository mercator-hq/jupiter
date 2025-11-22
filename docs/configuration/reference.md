# Configuration Reference

Complete reference for all Mercator Jupiter configuration options. This document details every configuration section, field, data type, default value, and usage.

## Table of Contents

- [Configuration File Format](#configuration-file-format)
- [Environment Variable Overrides](#environment-variable-overrides)
- [Proxy Configuration](#proxy-configuration)
- [Provider Configuration](#provider-configuration)
- [Policy Configuration](#policy-configuration)
- [Evidence Configuration](#evidence-configuration)
- [Processing Configuration](#processing-configuration)
- [Routing Configuration](#routing-configuration)
- [Limits Configuration](#limits-configuration)
- [Telemetry Configuration](#telemetry-configuration)
- [Security Configuration](#security-configuration)

## Configuration File Format

Mercator Jupiter uses YAML for configuration. The root configuration structure contains these sections:

```yaml
proxy: { ... }       # HTTP proxy server settings
providers: { ... }   # LLM provider configurations
policy: { ... }      # Policy loading settings
evidence: { ... }    # Evidence storage settings
processing: { ... }  # Request/response processing
routing: { ... }     # Routing engine settings
limits: { ... }      # Budget and rate limiting
telemetry: { ... }   # Logging, metrics, tracing
security: { ... }    # TLS, mTLS, authentication
```

## Environment Variable Overrides

Any configuration value can be overridden using environment variables with the format:

```
MERCATOR_<SECTION>_<FIELD>=<value>
```

**Examples:**

```bash
MERCATOR_PROXY_LISTEN_ADDRESS="0.0.0.0:8080"
MERCATOR_PROVIDERS_OPENAI_API_KEY="sk-..."
MERCATOR_TELEMETRY_LOGGING_LEVEL="debug"
```

For nested fields, use underscores:

```bash
MERCATOR_EVIDENCE_SQLITE_PATH="/var/lib/mercator/evidence.db"
MERCATOR_SECURITY_TLS_ENABLED="true"
```

---

## Proxy Configuration

HTTP proxy server settings.

### Section: `proxy`

```yaml
proxy:
  listen_address: "127.0.0.1:8080"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  shutdown_timeout: "30s"
  max_header_bytes: 1048576
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Authorization", "Content-Type", "X-Request-ID"]
    exposed_headers: ["X-Request-ID"]
    max_age: 3600
    allow_credentials: false
```

### Fields

#### `listen_address`

- **Type**: `string`
- **Default**: `"127.0.0.1:8080"`
- **Format**: `"host:port"`
- **Description**: Address and port for the proxy server to listen on
- **Examples**:
  - `"127.0.0.1:8080"` - Localhost only
  - `"0.0.0.0:8080"` - All interfaces
  - `":8080"` - All interfaces (short form)

#### `read_timeout`

- **Type**: `duration`
- **Default**: `"30s"`
- **Description**: Maximum duration for reading entire request including body
- **Valid values**: Any positive duration (e.g., `"10s"`, `"1m"`, `"5m"`)

#### `write_timeout`

- **Type**: `duration`
- **Default**: `"30s"`
- **Description**: Maximum duration before timing out response writes
- **Valid values**: Any positive duration

#### `idle_timeout`

- **Type**: `duration`
- **Default**: `"120s"`
- **Description**: Maximum time to wait for next request with keep-alives enabled
- **Valid values**: Any positive duration

#### `shutdown_timeout`

- **Type**: `duration`
- **Default**: `"30s"`
- **Description**: Maximum duration to wait for graceful shutdown
- **Valid values**: Any positive duration
- **Note**: If requests are still in-flight after this timeout, server forces shutdown

#### `max_header_bytes`

- **Type**: `int`
- **Default**: `1048576` (1MB)
- **Description**: Maximum bytes for request headers (does not limit body size)
- **Valid values**: Positive integer

### CORS Configuration

Cross-Origin Resource Sharing settings.

#### `cors.enabled`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable or disable CORS

#### `cors.allowed_origins`

- **Type**: `[]string`
- **Default**: `["*"]`
- **Description**: List of allowed origins for CORS requests
- **Examples**:
  - `["*"]` - Allow all (development only)
  - `["https://app.example.com"]` - Specific origin
  - `["https://app.example.com", "https://admin.example.com"]` - Multiple origins

#### `cors.allowed_methods`

- **Type**: `[]string`
- **Default**: `["GET", "POST", "PUT", "DELETE", "OPTIONS"]`
- **Description**: Allowed HTTP methods

#### `cors.allowed_headers`

- **Type**: `[]string`
- **Default**: `["Authorization", "Content-Type", "X-Request-ID", "X-User-ID"]`
- **Description**: Allowed request headers

#### `cors.exposed_headers`

- **Type**: `[]string`
- **Default**: `["X-Request-ID"]`
- **Description**: Response headers exposed to client

#### `cors.max_age`

- **Type**: `int`
- **Default**: `3600`
- **Description**: Maximum age in seconds for preflight cache

#### `cors.allow_credentials`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Allow cookies and auth headers in CORS requests

---

## Provider Configuration

LLM provider connection settings.

### Section: `providers`

Providers is a map where keys are provider names and values are provider configurations.

```yaml
providers:
  # Provider name (can be any identifier)
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"
    max_retries: 3
    connection_pool:
      max_idle_conns: 100
      max_idle_conns_per_host: 10
      idle_timeout: "90s"
    health_check_interval: "30s"

  anthropic:
    base_url: "https://api.anthropic.com/v1"
    api_key: "${ANTHROPIC_API_KEY}"
    timeout: "60s"
    max_retries: 3
```

### Fields

#### `base_url`

- **Type**: `string`
- **Required**: Yes
- **Description**: Base URL for provider API endpoint
- **Examples**:
  - `"https://api.openai.com/v1"` - OpenAI
  - `"https://api.anthropic.com/v1"` - Anthropic
  - `"http://localhost:11434"` - Ollama

#### `api_key`

- **Type**: `string`
- **Required**: Yes (for most providers)
- **Description**: Authentication key for provider
- **Best practice**: Use environment variables: `"${PROVIDER_API_KEY}"`

#### `timeout`

- **Type**: `duration`
- **Default**: `"60s"`
- **Description**: Maximum duration for requests to this provider
- **Recommendations**:
  - Fast models (GPT-3.5): `"30s"`
  - Slower models (GPT-4, Claude): `"60s"` to `"120s"`

#### `max_retries`

- **Type**: `int`
- **Default**: `3`
- **Description**: Maximum retry attempts for failed requests
- **Valid values**: 0-10

#### `connection_pool` (optional)

HTTP connection pool settings for the provider.

##### `connection_pool.max_idle_conns`

- **Type**: `int`
- **Default**: `100`
- **Description**: Maximum total idle connections

##### `connection_pool.max_idle_conns_per_host`

- **Type**: `int`
- **Default**: `10`
- **Description**: Maximum idle connections per host

##### `connection_pool.idle_timeout`

- **Type**: `duration`
- **Default**: `"90s"`
- **Description**: How long idle connections are kept

#### `health_check_interval`

- **Type**: `duration`
- **Default**: `"30s"`
- **Description**: Interval between provider health checks
- **Note**: Set to `"0s"` to disable health checks

---

## Policy Configuration

Policy loading and management settings.

### Section: `policy`

```yaml
policy:
  # File mode
  mode: "file"
  file_path: "policies.yaml"
  watch: true

  # OR Git mode
  # mode: "git"
  # git:
  #   repository: "https://github.com/org/policies.git"
  #   branch: "main"
  #   path: "policies/"
  #   auth:
  #     type: "token"
  #     token: "${GITHUB_TOKEN}"
  #   poll:
  #     enabled: true
  #     interval: "60s"

  validation:
    enabled: true
    strict: false
```

### Fields

#### `mode`

- **Type**: `string`
- **Default**: `"file"`
- **Valid values**: `"file"`, `"git"`
- **Description**: Policy loading mode

#### File Mode Fields

##### `file_path`

- **Type**: `string`
- **Default**: `"./policies.yaml"`
- **Description**: Path to policy file (when mode is "file")

##### `watch`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Auto-reload policies when file changes

#### Git Mode Fields

##### `git.repository`

- **Type**: `string`
- **Required**: Yes (when mode is "git")
- **Description**: Git repository URL (HTTPS or SSH)
- **Examples**:
  - `"https://github.com/company/policies.git"`
  - `"git@github.com:company/policies.git"`

##### `git.branch`

- **Type**: `string`
- **Default**: `"main"`
- **Description**: Git branch to track
- **Supports**: Environment variable expansion (e.g., `"${ENVIRONMENT}"`)

##### `git.path`

- **Type**: `string`
- **Default**: `""` (root)
- **Description**: Path within repository to policy files
- **Examples**: `"policies/"`, `"config/policies/"`

##### `git.auth.type`

- **Type**: `string`
- **Default**: `"none"`
- **Valid values**: `"none"`, `"token"`, `"ssh"`
- **Description**: Git authentication type

##### `git.auth.token`

- **Type**: `string`
- **Required**: When auth type is "token"
- **Description**: Personal access token for HTTPS
- **Best practice**: Use environment variable: `"${GITHUB_TOKEN}"`

##### `git.auth.ssh_key_path`

- **Type**: `string`
- **Required**: When auth type is "ssh"
- **Description**: Path to SSH private key
- **Example**: `"/home/user/.ssh/id_rsa"`

##### `git.auth.ssh_key_passphrase`

- **Type**: `string`
- **Optional**: For encrypted SSH keys
- **Description**: SSH key passphrase

##### `git.poll.enabled`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable polling for changes

##### `git.poll.interval`

- **Type**: `duration`
- **Default**: `"30s"`
- **Description**: Polling interval
- **Recommendations**:
  - Production: `"60s"` to `"300s"`
  - Development: `"10s"` to `"30s"`

##### `git.poll.timeout`

- **Type**: `duration`
- **Default**: `"10s"`
- **Description**: Timeout for Git operations

##### `git.clone.depth`

- **Type**: `int`
- **Default**: `1`
- **Description**: Clone depth (0 = full history)
- **Recommendation**: Use `1` for faster cloning

##### `git.clone.local_path`

- **Type**: `string`
- **Default**: System temp directory
- **Description**: Local path for cloned repository

##### `git.clone.clean_on_start`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Remove local repo before cloning on startup

#### Validation Fields

##### `validation.enabled`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable policy validation

##### `validation.strict`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Treat warnings as errors

---

## Evidence Configuration

Evidence generation and storage settings.

### Section: `evidence`

```yaml
evidence:
  enabled: true
  backend: "sqlite"

  sqlite:
    path: "evidence.db"
    max_open_conns: 10
    max_idle_conns: 5
    wal_mode: true
    busy_timeout: "5s"

  recorder:
    async_buffer: 1000
    write_timeout: "5s"
    hash_request: true
    hash_response: true
    redact_api_keys: true
    max_field_length: 500

  retention:
    days: 90
    prune_schedule: "0 3 * * *"
    archive_before_delete: false
    archive_path: "data/archives/"
    max_records: 0

  signing_key_path: "/path/to/signing-key.pem"
```

### Fields

#### `enabled`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable evidence generation

#### `backend`

- **Type**: `string`
- **Default**: `"sqlite"`
- **Valid values**: `"sqlite"`, `"postgres"`, `"s3"`
- **Description**: Storage backend for evidence records

### SQLite Backend

#### `sqlite.path`

- **Type**: `string`
- **Default**: `"data/evidence.db"`
- **Description**: Path to SQLite database file

#### `sqlite.max_open_conns`

- **Type**: `int`
- **Default**: `10`
- **Description**: Maximum open database connections

#### `sqlite.max_idle_conns`

- **Type**: `int`
- **Default**: `5`
- **Description**: Maximum idle database connections

#### `sqlite.wal_mode`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable Write-Ahead Logging for better concurrency

#### `sqlite.busy_timeout`

- **Type**: `duration`
- **Default**: `"5s"`
- **Description**: Wait duration when database is locked

### Recorder Configuration

#### `recorder.async_buffer`

- **Type**: `int`
- **Default**: `1000`
- **Description**: Size of async write channel buffer

#### `recorder.write_timeout`

- **Type**: `duration`
- **Default**: `"5s"`
- **Description**: Timeout for writing evidence to storage

#### `recorder.hash_request`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Hash request bodies

#### `recorder.hash_response`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Hash response bodies

#### `recorder.redact_api_keys`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Redact API keys from logs

#### `recorder.max_field_length`

- **Type**: `int`
- **Default**: `500`
- **Description**: Maximum length for text fields before truncation

### Retention Configuration

#### `retention.days`

- **Type**: `int`
- **Default**: `90`
- **Description**: Number of days to retain evidence (0 = forever)
- **Compliance recommendations**:
  - GDPR: 90 days
  - HIPAA: 2555 days (7 years)
  - SOC2: 365 days (1 year)

#### `retention.prune_schedule`

- **Type**: `string` (cron expression)
- **Default**: `"0 3 * * *"` (daily at 3 AM)
- **Description**: Cron schedule for pruning old records

#### `retention.archive_before_delete`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Archive evidence before deletion

#### `retention.archive_path`

- **Type**: `string`
- **Default**: `"data/archives/"`
- **Description**: Directory for archived evidence

#### `retention.max_records`

- **Type**: `int64`
- **Default**: `0` (unlimited)
- **Description**: Maximum number of records to keep

### Signing

#### `signing_key_path`

- **Type**: `string`
- **Optional**: Yes
- **Description**: Path to Ed25519 private key for signing evidence
- **Note**: If not specified, evidence is not cryptographically signed
- **Generate with**: `mercator keys generate --key-id mykey --output ./keys`

---

## Telemetry Configuration

Logging, metrics, and tracing settings.

### Section: `telemetry`

```yaml
telemetry:
  logging:
    level: "info"
    format: "json"
    add_source: false
    redact_secrets: true

  metrics:
    enabled: true
    prometheus_path: "/metrics"
    namespace: "mercator"
    subsystem: "jupiter"

  tracing:
    enabled: false
    endpoint: "localhost:4317"
    sampling_rate: 1.0
    service_name: "mercator-jupiter"
    service_version: "0.1.0"
    environment: "production"
```

### Logging Fields

#### `logging.level`

- **Type**: `string`
- **Default**: `"info"`
- **Valid values**: `"debug"`, `"info"`, `"warn"`, `"error"`
- **Description**: Log level verbosity

#### `logging.format`

- **Type**: `string`
- **Default**: `"json"`
- **Valid values**: `"json"`, `"text"`
- **Description**: Log output format
- **Recommendations**:
  - Production: `"json"` (for log aggregation)
  - Development: `"text"` (for readability)

#### `logging.add_source`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Include source file and line number in logs

#### `logging.redact_secrets`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Redact sensitive data from logs

### Metrics Fields

#### `metrics.enabled`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable Prometheus metrics

#### `metrics.prometheus_path`

- **Type**: `string`
- **Default**: `"/metrics"`
- **Description**: HTTP path for metrics endpoint

#### `metrics.namespace`

- **Type**: `string`
- **Default**: `"mercator"`
- **Description**: Prometheus namespace for metrics

#### `metrics.subsystem`

- **Type**: `string`
- **Default**: `"jupiter"`
- **Description**: Prometheus subsystem for metrics

### Tracing Fields

#### `tracing.enabled`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable OpenTelemetry distributed tracing

#### `tracing.endpoint`

- **Type**: `string`
- **Required**: When tracing is enabled
- **Description**: OpenTelemetry collector endpoint
- **Example**: `"localhost:4317"`, `"otel-collector:4317"`

#### `tracing.sampling_rate`

- **Type**: `float`
- **Default**: `1.0`
- **Valid values**: `0.0` to `1.0`
- **Description**: Percentage of traces to sample
- **Recommendations**:
  - Development: `1.0` (100%)
  - Production: `0.1` (10%) to reduce overhead

#### `tracing.service_name`

- **Type**: `string`
- **Default**: `"mercator-jupiter"`
- **Description**: Service name in traces

#### `tracing.service_version`

- **Type**: `string`
- **Default**: `"0.1.0"`
- **Description**: Service version in traces

#### `tracing.environment`

- **Type**: `string`
- **Default**: `"production"`
- **Description**: Environment name in traces

---

## Security Configuration

TLS, mTLS, and authentication settings.

### Section: `security`

```yaml
security:
  tls:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    min_version: "1.3"

  mtls:
    enabled: false
    ca_file: "/path/to/ca.pem"
    client_auth: "require"

  api_keys:
    - key: "${API_KEY_PRODUCTION}"
      name: "production"
      metadata:
        team: "platform"
        environment: "production"
```

### TLS Fields

#### `tls.enabled`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable TLS (HTTPS)

#### `tls.cert_file`

- **Type**: `string`
- **Required**: When TLS is enabled
- **Description**: Path to TLS certificate file (PEM format)

#### `tls.key_file`

- **Type**: `string`
- **Required**: When TLS is enabled
- **Description**: Path to TLS private key file (PEM format)

#### `tls.min_version`

- **Type**: `string`
- **Default**: `"1.3"`
- **Valid values**: `"1.2"`, `"1.3"`
- **Description**: Minimum TLS version
- **Recommendation**: Use `"1.3"` for production

### mTLS Fields

#### `mtls.enabled`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable mutual TLS (client authentication)
- **Note**: Requires TLS to be enabled

#### `mtls.ca_file`

- **Type**: `string`
- **Required**: When mTLS is enabled
- **Description**: Path to CA certificate for verifying client certificates

#### `mtls.client_auth`

- **Type**: `string`
- **Default**: `"require"`
- **Valid values**: `"require"`, `"request"`, `"verify"`
- **Description**: Client certificate verification mode

### API Keys

#### `api_keys`

- **Type**: `[]object`
- **Description**: List of API keys for authentication

Each API key object contains:

##### `key`

- **Type**: `string`
- **Required**: Yes
- **Description**: API key value
- **Best practice**: Use environment variable

##### `name`

- **Type**: `string`
- **Required**: Yes
- **Description**: Friendly name for this API key

##### `metadata`

- **Type**: `map[string]string`
- **Optional**: Yes
- **Description**: Key-value pairs for key metadata
- **Examples**: `team`, `environment`, `purpose`

---

## Processing Configuration

Request/response processing settings.

### Section: `processing`

```yaml
processing:
  enrich_requests: true
  estimate_costs: true
  analyze_content: true
```

### Fields

#### `enrich_requests`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enrich requests with metadata

#### `estimate_costs`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Estimate cost for requests

#### `analyze_content`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Analyze request/response content

---

## Routing Configuration

Routing engine settings.

### Section: `routing`

```yaml
routing:
  strategy: "round-robin"
  sticky_routing: false
  sticky_key: "user_id"
  health_check_interval: "30s"
  failover:
    enabled: true
    max_retries: 2
    retry_delay: "1s"
```

### Fields

#### `strategy`

- **Type**: `string`
- **Default**: `"round-robin"`
- **Valid values**: `"round-robin"`, `"least-latency"`, `"least-cost"`, `"weighted"`
- **Description**: Provider selection strategy

#### `sticky_routing`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Route same user/key to same provider

#### `sticky_key`

- **Type**: `string`
- **Default**: `"user_id"`
- **Description**: Metadata key for sticky routing

#### `health_check_interval`

- **Type**: `duration`
- **Default**: `"30s"`
- **Description**: Provider health check interval

#### `failover.enabled`

- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable automatic failover

#### `failover.max_retries`

- **Type**: `int`
- **Default**: `2`
- **Description**: Maximum failover attempts

#### `failover.retry_delay`

- **Type**: `duration`
- **Default**: `"1s"`
- **Description**: Delay between retries

---

## Limits Configuration

Budget and rate limiting settings.

### Section: `limits`

```yaml
limits:
  budgets:
    enabled: true
    default_daily_limit: 100.0
    enforcement: "hard"

  rate_limiting:
    enabled: true
    default_rpm: 60
    default_tpm: 100000
    window_size: "1m"
```

### Budget Fields

#### `budgets.enabled`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable budget tracking

#### `budgets.default_daily_limit`

- **Type**: `float`
- **Default**: `100.0`
- **Description**: Default daily spending limit per user (USD)

#### `budgets.enforcement`

- **Type**: `string`
- **Default**: `"hard"`
- **Valid values**: `"hard"`, `"soft"`
- **Description**: Budget enforcement mode
  - `"hard"`: Block requests when exceeded
  - `"soft"`: Log warnings but allow

### Rate Limiting Fields

#### `rate_limiting.enabled`

- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable rate limiting

#### `rate_limiting.default_rpm`

- **Type**: `int`
- **Default**: `60`
- **Description**: Default requests per minute per user

#### `rate_limiting.default_tpm`

- **Type**: `int`
- **Default**: `100000`
- **Description**: Default tokens per minute per user

#### `rate_limiting.window_size`

- **Type**: `duration`
- **Default**: `"1m"`
- **Description**: Rate limit window size

---

## See Also

- [Configuration Basics](../getting-started/configuration-basics.md) - Configuration fundamentals
- [Security Guide](../SECURITY.md) - TLS/mTLS setup
- [Observability Guide](../observability-guide.md) - Metrics and tracing
- [Provider Setup](../providers/openai.md) - Provider-specific configuration

## Examples

Complete configuration examples:

- [examples/configs/minimal.yaml](../../examples/configs/minimal.yaml) - Minimal config
- [examples/configs/development.yaml](../../examples/configs/development.yaml) - Development config
- [examples/configs/production.yaml](../../examples/configs/production.yaml) - Production config
- [examples/basic-config.yaml](../../examples/basic-config.yaml) - Basic config with comments
