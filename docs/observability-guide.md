# Mercator Jupiter - Observability Guide

## Table of Contents

1. [Overview](#overview)
2. [Structured Logging](#structured-logging)
3. [Prometheus Metrics](#prometheus-metrics)
4. [Distributed Tracing](#distributed-tracing)
5. [Health Check Endpoints](#health-check-endpoints)
6. [Integration Examples](#integration-examples)
7. [Troubleshooting](#troubleshooting)

---

## Overview

Mercator Jupiter provides comprehensive observability through four pillars:

| Component | Purpose | Target Audience |
|-----------|---------|-----------------|
| **Structured Logging** | Request flow, errors, debugging | Developers, Operators |
| **Prometheus Metrics** | System health, performance, costs | SRE, Operations |
| **Distributed Tracing** | Request latency breakdown | Performance Engineers |
| **Health Checks** | Service status, K8s integration | Orchestration Systems |

### Performance

All observability components are designed for minimal overhead:

- **Logging**: <1µs when disabled, <10µs when enabled
- **Metrics**: <50µs per metric update
- **Tracing**: <100µs per span
- **Health Checks**: <11µs for full check cycle

---

## Structured Logging

### Configuration

```yaml
telemetry:
  logging:
    level: info                # debug, info, warn, error
    format: json               # json, text, console
    add_source: false          # Include file:line
    redact_pii: true           # Automatic PII redaction
    buffer_size: 10000         # Async buffer size
```

### Log Levels

| Level | When to Use | Example |
|-------|-------------|---------|
| `debug` | Development, troubleshooting | Function entry/exit, variable values |
| `info` | Normal operations | Request processed, provider selected |
| `warn` | Unexpected but recoverable | Rate limit approaching, retry attempt |
| `error` | Errors requiring attention | Request failed, provider unavailable |

**Production Recommendation**: Use `info` level. Switch to `debug` temporarily for troubleshooting.

### Log Formats

#### JSON Format (Production)

```json
{
  "timestamp": "2025-11-20T10:30:00.000Z",
  "level": "info",
  "msg": "Request processed successfully",
  "request_id": "req-abc123",
  "api_key": "sk-a***",
  "user": "user@example.com",
  "provider": "openai",
  "model": "gpt-4",
  "duration_ms": 1234,
  "tokens": 1500,
  "cost": 0.05,
  "status": "success"
}
```

**Advantages**:
- Machine-readable
- Easy to parse and aggregate
- Works with log aggregation systems (Loki, Elasticsearch)

#### Text Format (Development)

```
2025-11-20T10:30:00Z INFO Request processed successfully request_id=req-abc123 api_key=sk-a*** provider=openai model=gpt-4 duration_ms=1234 tokens=1500 cost=0.05
```

**Advantages**:
- Human-readable
- Good for local development
- Easier to scan visually

### PII Redaction

Automatic redaction protects sensitive information:

| Data Type | Example Input | Redacted Output |
|-----------|---------------|-----------------|
| API Keys | `sk-abc123xyz789` | `sk-a***` |
| Emails | `user@example.com` | `u***@example.com` |
| SSN | `123-45-6789` | `***-**-****` |
| IP Addresses | `192.168.1.100` | `192.*.*.*` |
| Credit Cards | `4111-1111-1111-1111` | `****-****-****-****` |
| Bearer Tokens | `Bearer eyJhbGc...` | `Bearer ***` |

#### Custom Redaction Patterns

```yaml
telemetry:
  logging:
    redact_pii: true
    redact_patterns:
      - name: internal_token
        pattern: "tok_[a-zA-Z0-9]{32}"
        replacement: "tok_***"
      - name: account_number
        pattern: "ACC[0-9]{8}"
        replacement: "ACC********"
```

### Context-Aware Logging

Logs automatically include context from the request:

```go
// Context is propagated automatically
logger.InfoContext(ctx, "Policy evaluated",
    "rule_id", "cost-limit",
    "action", "allow",
    "duration_ms", 2.1,
)

// Output includes request_id, api_key, user from context
```

### Viewing Logs

#### Local Development

```bash
# View logs with jq (JSON)
tail -f mercator.log | jq .

# Filter by level
tail -f mercator.log | jq 'select(.level == "error")'

# Filter by request_id
tail -f mercator.log | jq 'select(.request_id == "req-123")'

# View only messages and timestamps
tail -f mercator.log | jq -r '"\(.timestamp) \(.level) \(.msg)"'
```

#### Production (with Loki)

```logql
# All logs from Mercator Jupiter
{job="mercator-jupiter"}

# Error logs only
{job="mercator-jupiter"} |= "ERROR"

# Logs for specific request
{job="mercator-jupiter"} | json | request_id="req-123"

# High-cost requests
{job="mercator-jupiter"} | json | cost > 1.0

# Requests by provider
{job="mercator-jupiter"} | json | provider="openai"
```

---

## Prometheus Metrics

### Configuration

```yaml
telemetry:
  metrics:
    enabled: true
    path: /metrics
    namespace: mercator
    subsystem: jupiter
```

### Available Metrics

#### Request Metrics

```promql
# Total requests by provider, model, status
mercator_jupiter_requests_total{provider="openai", model="gpt-4", status="success"}

# Request duration histogram
mercator_jupiter_request_duration_seconds{provider="openai", model="gpt-4"}

# Token counts
mercator_jupiter_request_tokens_total{provider="openai", model="gpt-4", type="prompt"}
mercator_jupiter_request_tokens_total{provider="openai", model="gpt-4", type="completion"}

# Request/response size
mercator_jupiter_request_size_bytes{provider="openai", model="gpt-4"}
```

#### Provider Metrics

```promql
# Provider health (1=healthy, 0=unhealthy)
mercator_jupiter_provider_health{provider="openai"}

# Provider latency
mercator_jupiter_provider_latency_seconds{provider="openai", model="gpt-4"}

# Provider errors
mercator_jupiter_provider_errors_total{provider="openai", error_type="rate_limit"}

# Provider request count
mercator_jupiter_provider_requests_total{provider="openai", model="gpt-4"}
```

#### Policy Metrics

```promql
# Policy evaluations
mercator_jupiter_policy_evaluations_total{rule_id="cost-limit", action="block"}

# Policy evaluation duration
mercator_jupiter_policy_evaluation_duration_seconds{rule_id="cost-limit"}

# Policy hits/misses
mercator_jupiter_policy_hits_total{rule_id="cost-limit"}
mercator_jupiter_policy_misses_total{rule_id="cost-limit"}
```

#### Cost Metrics

```promql
# Total cost in USD
mercator_jupiter_cost_total{provider="openai", model="gpt-4"}

# Cost per request
mercator_jupiter_cost_per_request{provider="openai", model="gpt-4"}

# Average cost per token
mercator_jupiter_cost_per_token{provider="openai", model="gpt-4"}
```

#### Cache Metrics

```promql
# Cache hits/misses
mercator_jupiter_cache_hits_total{cache="policy"}
mercator_jupiter_cache_misses_total{cache="policy"}

# Cache size
mercator_jupiter_cache_entries{cache="policy"}

# Cache evictions
mercator_jupiter_cache_evictions_total{cache="policy"}
```

### Common Queries

See [metrics-queries.md](metrics-queries.md) for a comprehensive list of PromQL queries.

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'mercator-jupiter'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
    scrape_timeout: 10s
```

### Viewing Metrics

```bash
# View all metrics
curl http://localhost:8080/metrics

# Filter specific metrics
curl http://localhost:8080/metrics | grep mercator_requests_total

# View in Prometheus UI
open http://localhost:9090
```

---

## Distributed Tracing

### Configuration

```yaml
telemetry:
  tracing:
    enabled: true
    sampler: ratio         # always, never, ratio
    sample_ratio: 0.1      # 10% sampling
    exporter: otlp         # otlp, jaeger, zipkin
    endpoint: localhost:4317
    service_name: mercator-jupiter
```

### Sampling Strategies

| Strategy | Use Case | Overhead |
|----------|----------|----------|
| `always` | Development, debugging | High (all requests traced) |
| `never` | Tracing disabled | Minimal (~26ns) |
| `ratio` | Production | Configurable (sample %) |

**Production Recommendation**: Use `ratio` with 5-10% sampling.

### Trace Hierarchy

A typical request creates this span hierarchy:

```
mercator.proxy.request (10s)
├── mercator.processing.request (5ms)
├── mercator.policy.evaluate (2ms)
├── mercator.limits.check (1ms)
├── mercator.provider.call (9.9s)
│   ├── mercator.provider.connect (100ms)
│   ├── mercator.provider.send (50ms)
│   └── mercator.provider.receive (9.75s)
├── mercator.processing.response (3ms)
└── mercator.evidence.generate (10ms)
```

### Span Attributes

Each span includes relevant attributes:

| Span | Attributes |
|------|------------|
| `proxy.request` | `request_id`, `api_key`, `user` |
| `provider.call` | `provider`, `model`, `tokens`, `cost` |
| `policy.evaluate` | `policy_id`, `rule_id`, `action` |
| `limits.check` | `identifier`, `allowed`, `reason` |

### W3C Trace Context

Mercator Jupiter supports W3C Trace Context for distributed tracing:

```http
# Incoming request with trace context
GET /v1/chat/completions HTTP/1.1
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
tracestate: congo=t61rcWkgMzE

# Outgoing request (trace context propagated)
POST https://api.openai.com/v1/chat/completions HTTP/1.1
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-b7ad6b7169203331-01
tracestate: congo=t61rcWkgMzE
```

### Viewing Traces

#### Jaeger

```bash
# Run Jaeger locally
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest

# View UI
open http://localhost:16686
```

#### Query Examples

- **Find slow requests**: Duration > 10s
- **Find errors**: Status = error
- **Find by user**: Tag `user=user@example.com`
- **Find by cost**: Tag `cost>1.0`

---

## Health Check Endpoints

### Configuration

```yaml
telemetry:
  health:
    enabled: true
    liveness_path: /health
    readiness_path: /ready
    version_path: /version
    check_timeout: 5s
    min_healthy_providers: 1
```

### Endpoints

#### GET /health (Liveness Probe)

Indicates if the process is alive.

**Response** (200 OK):
```json
{
  "status": "ok",
  "timestamp": "2025-11-20T10:30:00Z"
}
```

**Use**: Kubernetes liveness probe (restart pod if unhealthy)

#### GET /ready (Readiness Probe)

Indicates if the system can serve traffic.

**Response** (200 OK):
```json
{
  "status": "ready",
  "checks": {
    "config": {"status": "ok", "duration_ms": 0.1},
    "providers": {"status": "ok", "duration_ms": 5.2},
    "policy": {"status": "ok", "duration_ms": 0.3}
  },
  "timestamp": "2025-11-20T10:30:00Z"
}
```

**Response** (503 Service Unavailable):
```json
{
  "status": "degraded",
  "checks": {
    "config": {"status": "ok"},
    "providers": {"status": "unhealthy", "message": "no healthy providers"},
    "policy": {"status": "ok"}
  },
  "timestamp": "2025-11-20T10:30:00Z"
}
```

**Use**: Kubernetes readiness probe (route traffic only when ready)

#### GET /version (Version Information)

Returns build and version information.

**Response** (200 OK):
```json
{
  "version": "1.0.0",
  "commit": "abc123def456",
  "build_time": "2025-11-20T00:00:00Z",
  "go_version": "go1.21.5"
}
```

### Kubernetes Integration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mercator-jupiter
spec:
  template:
    spec:
      containers:
      - name: jupiter
        image: mercator-hq/jupiter:latest
        ports:
        - containerPort: 8080
          name: http

        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3

        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 5
          failureThreshold: 2
```

---

## Integration Examples

### Complete Stack

#### 1. Run Observability Stack

```bash
# Start Prometheus
docker run -d --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Start Jaeger
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest

# Start Loki (for logs)
docker run -d --name loki \
  -p 3100:3100 \
  grafana/loki:latest

# Start Grafana
docker run -d --name grafana \
  -p 3000:3000 \
  grafana/grafana:latest
```

#### 2. Configure Mercator Jupiter

```yaml
# config.yaml
telemetry:
  logging:
    level: info
    format: json
    redact_pii: true

  metrics:
    enabled: true
    path: /metrics

  tracing:
    enabled: true
    sampler: ratio
    sample_ratio: 0.1
    exporter: otlp
    endpoint: localhost:4317

  health:
    enabled: true
    liveness_path: /health
    readiness_path: /ready
    version_path: /version
```

#### 3. Access UIs

- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686
- **Grafana**: http://localhost:3000 (import dashboard from `docs/grafana-dashboard.json`)

---

## Troubleshooting

### Logs Not Appearing

**Problem**: Logs not being written or appearing delayed.

**Solutions**:
1. Check log level: Ensure level is not too restrictive (use `debug` temporarily)
2. Check buffer: Flush logs on shutdown or reduce buffer size
3. Check permissions: Ensure write permissions for log file
4. Check format: Validate JSON output with `jq`

```bash
# Force log flush
kill -USR1 $(pidof mercator-jupiter)

# Test log output
echo '{"level":"info","msg":"test"}' | jq .
```

### Metrics Not Scraped

**Problem**: Prometheus not scraping metrics.

**Solutions**:
1. Check Prometheus config: Validate `scrape_configs`
2. Check endpoint: `curl http://localhost:8080/metrics`
3. Check firewall: Ensure port 8080 is accessible
4. Check Prometheus logs: `docker logs prometheus`

```bash
# Verify metrics endpoint
curl -v http://localhost:8080/metrics

# Check Prometheus targets
open http://localhost:9090/targets
```

### Traces Not Appearing

**Problem**: Traces not showing up in Jaeger.

**Solutions**:
1. Check sampling: Increase sample_ratio to 1.0 temporarily
2. Check exporter: Verify OTLP endpoint is accessible
3. Check Jaeger: Ensure Jaeger is running and accepting traces
4. Check logs: Look for tracing errors in application logs

```bash
# Test OTLP endpoint
telnet localhost 4317

# Check Jaeger logs
docker logs jaeger

# Force sampling
# In config: sampler: always, sample_ratio: 1.0
```

### Health Checks Failing

**Problem**: `/ready` returns 503.

**Solutions**:
1. Check components: Review which component check is failing
2. Check providers: Ensure at least one provider is healthy
3. Check dependencies: Verify external dependencies (DB, storage)
4. Check timeouts: Increase `check_timeout` if checks are slow

```bash
# Check readiness status
curl -i http://localhost:8080/ready | jq .

# Check individual components
curl http://localhost:8080/ready | jq '.checks'

# Check liveness (should always be OK)
curl http://localhost:8080/health
```

---

## Best Practices

### Logging

✅ **DO**:
- Use structured logging with key-value pairs
- Enable PII redaction in production
- Use `info` level for production, `debug` for troubleshooting
- Include context (request_id, api_key, user) in all logs

❌ **DON'T**:
- Don't log sensitive data without redaction
- Don't use `fmt.Printf` for logging
- Don't set level to `debug` in production long-term
- Don't log excessively in hot paths

### Metrics

✅ **DO**:
- Use Prometheus for metrics (industry standard)
- Keep cardinality low (avoid high-cardinality labels)
- Use histograms for latencies
- Use counters for event counts

❌ **DON'T**:
- Don't use unbounded label values
- Don't create metrics in tight loops
- Don't use gauges for counters
- Don't forget to scrape metrics regularly

### Tracing

✅ **DO**:
- Use sampling in production (5-10%)
- Include relevant attributes in spans
- Propagate trace context across services
- Use OTLP exporter (standard)

❌ **DON'T**:
- Don't trace 100% of requests in production
- Don't include PII in span attributes
- Don't create too many spans (>50 per request)
- Don't forget to end spans

### Health Checks

✅ **DO**:
- Use liveness for process health
- Use readiness for traffic routing
- Keep checks fast (<100ms)
- Check critical dependencies

❌ **DON'T**:
- Don't fail liveness unless process is dead
- Don't make expensive calls in health checks
- Don't check optional dependencies in readiness
- Don't set aggressive timeouts

---

## Further Reading

- [Metrics Query Guide](metrics-queries.md) - PromQL examples
- [Grafana Dashboard](grafana-dashboard.json) - Pre-built dashboard
- [Configuration Reference](../examples/observability-config.yaml) - Full config example
- [API Documentation](../pkg/telemetry/doc.go) - Package documentation

---

**Need Help?**
- GitHub Issues: https://github.com/anthropics/mercator-jupiter/issues
- Documentation: https://docs.mercator-hq.com
