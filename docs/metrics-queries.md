# Mercator Jupiter - Prometheus Metrics Query Guide

This guide provides PromQL query examples for monitoring Mercator Jupiter.

## Table of Contents

1. [Request Metrics](#request-metrics)
2. [Provider Metrics](#provider-metrics)
3. [Policy Metrics](#policy-metrics)
4. [Cost Metrics](#cost-metrics)
5. [Performance Metrics](#performance-metrics)
6. [Alerts](#alerts)

---

## Request Metrics

### Request Rate

```promql
# Requests per second (overall)
rate(mercator_jupiter_requests_total[5m])

# Requests per second by provider
sum by (provider) (rate(mercator_jupiter_requests_total[5m]))

# Requests per second by model
sum by (model) (rate(mercator_jupiter_requests_total[5m]))

# Requests per second by status
sum by (status) (rate(mercator_jupiter_requests_total[5m]))
```

### Success Rate

```promql
# Overall success rate (percentage)
sum(rate(mercator_jupiter_requests_total{status="success"}[5m])) /
sum(rate(mercator_jupiter_requests_total[5m])) * 100

# Success rate by provider
sum by (provider) (rate(mercator_jupiter_requests_total{status="success"}[5m])) /
sum by (provider) (rate(mercator_jupiter_requests_total[5m])) * 100

# Error rate (percentage)
sum(rate(mercator_jupiter_requests_total{status!="success"}[5m])) /
sum(rate(mercator_jupiter_requests_total[5m])) * 100
```

### Request Duration (Latency)

```promql
# Average request duration (seconds)
rate(mercator_jupiter_request_duration_seconds_sum[5m]) /
rate(mercator_jupiter_request_duration_seconds_count[5m])

# P50 latency (median)
histogram_quantile(0.5, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))

# P95 latency
histogram_quantile(0.95, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))

# P99 latency
histogram_quantile(0.99, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))

# P99.9 latency
histogram_quantile(0.999, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))

# Latency by provider
histogram_quantile(0.95, sum by (provider, le) (
  rate(mercator_jupiter_request_duration_seconds_bucket[5m])
))
```

### Token Metrics

```promql
# Total tokens per second
sum(rate(mercator_jupiter_request_tokens_total[5m]))

# Prompt tokens per second
sum(rate(mercator_jupiter_request_tokens_total{type="prompt"}[5m]))

# Completion tokens per second
sum(rate(mercator_jupiter_request_tokens_total{type="completion"}[5m]))

# Average tokens per request
rate(mercator_jupiter_request_tokens_total[5m]) /
rate(mercator_jupiter_requests_total[5m])

# Tokens by provider
sum by (provider) (rate(mercator_jupiter_request_tokens_total[5m]))
```

### Request Size

```promql
# Average request size (bytes)
rate(mercator_jupiter_request_size_bytes_sum[5m]) /
rate(mercator_jupiter_request_size_bytes_count[5m])

# P95 request size
histogram_quantile(0.95, rate(mercator_jupiter_request_size_bytes_bucket[5m]))

# Large requests (>10KB)
count(mercator_jupiter_request_size_bytes_bucket{le="10000"} == 0)
```

---

## Provider Metrics

### Provider Health

```promql
# Provider health status (1=healthy, 0=unhealthy)
mercator_jupiter_provider_health

# Number of healthy providers
sum(mercator_jupiter_provider_health)

# Unhealthy providers
mercator_jupiter_provider_health == 0

# Provider uptime percentage (last 24h)
avg_over_time(mercator_jupiter_provider_health[24h]) * 100
```

### Provider Latency

```promql
# Average provider latency
rate(mercator_jupiter_provider_latency_seconds_sum[5m]) /
rate(mercator_jupiter_provider_latency_seconds_count[5m])

# P95 provider latency by provider
histogram_quantile(0.95, sum by (provider, le) (
  rate(mercator_jupiter_provider_latency_seconds_bucket[5m])
))

# Slowest provider
topk(1, rate(mercator_jupiter_provider_latency_seconds_sum[5m]) /
rate(mercator_jupiter_provider_latency_seconds_count[5m]))
```

### Provider Errors

```promql
# Total provider errors per second
sum(rate(mercator_jupiter_provider_errors_total[5m]))

# Provider error rate by type
sum by (error_type) (rate(mercator_jupiter_provider_errors_total[5m]))

# Rate limit errors
sum(rate(mercator_jupiter_provider_errors_total{error_type="rate_limit"}[5m]))

# Timeout errors
sum(rate(mercator_jupiter_provider_errors_total{error_type="timeout"}[5m]))

# Auth errors
sum(rate(mercator_jupiter_provider_errors_total{error_type="auth"}[5m]))

# Error percentage by provider
sum by (provider) (rate(mercator_jupiter_provider_errors_total[5m])) /
sum by (provider) (rate(mercator_jupiter_provider_requests_total[5m])) * 100
```

### Provider Usage

```promql
# Requests per provider
sum by (provider) (rate(mercator_jupiter_provider_requests_total[5m]))

# Most used provider
topk(1, sum by (provider) (rate(mercator_jupiter_provider_requests_total[5m])))

# Provider usage distribution (percentage)
sum by (provider) (rate(mercator_jupiter_provider_requests_total[5m])) /
sum(rate(mercator_jupiter_provider_requests_total[5m])) * 100

# Requests per model
sum by (model) (rate(mercator_jupiter_provider_requests_total[5m]))
```

---

## Policy Metrics

### Policy Evaluations

```promql
# Total policy evaluations per second
sum(rate(mercator_jupiter_policy_evaluations_total[5m]))

# Evaluations by action
sum by (action) (rate(mercator_jupiter_policy_evaluations_total[5m]))

# Blocked requests
sum(rate(mercator_jupiter_policy_evaluations_total{action="block"}[5m]))

# Block rate (percentage)
sum(rate(mercator_jupiter_policy_evaluations_total{action="block"}[5m])) /
sum(rate(mercator_jupiter_policy_evaluations_total[5m])) * 100

# Evaluations by rule
sum by (rule_id) (rate(mercator_jupiter_policy_evaluations_total[5m]))
```

### Policy Performance

```promql
# Average policy evaluation duration
rate(mercator_jupiter_policy_evaluation_duration_seconds_sum[5m]) /
rate(mercator_jupiter_policy_evaluation_duration_seconds_count[5m])

# P95 policy evaluation duration
histogram_quantile(0.95, rate(mercator_jupiter_policy_evaluation_duration_seconds_bucket[5m]))

# Slow policy rules (>5ms p95)
histogram_quantile(0.95, sum by (rule_id, le) (
  rate(mercator_jupiter_policy_evaluation_duration_seconds_bucket[5m])
)) > 0.005
```

### Policy Hit Rate

```promql
# Total policy hits per second
sum(rate(mercator_jupiter_policy_hits_total[5m]))

# Total policy misses per second
sum(rate(mercator_jupiter_policy_misses_total[5m]))

# Policy hit rate (percentage)
sum(rate(mercator_jupiter_policy_hits_total[5m])) /
(sum(rate(mercator_jupiter_policy_hits_total[5m])) + sum(rate(mercator_jupiter_policy_misses_total[5m]))) * 100

# Hit rate by rule
sum by (rule_id) (rate(mercator_jupiter_policy_hits_total[5m])) /
(sum by (rule_id) (rate(mercator_jupiter_policy_hits_total[5m])) +
 sum by (rule_id) (rate(mercator_jupiter_policy_misses_total[5m]))) * 100
```

---

## Cost Metrics

### Total Cost

```promql
# Total cost (USD) over time
mercator_jupiter_cost_total

# Cost rate (USD per second)
rate(mercator_jupiter_cost_total[5m])

# Cost per minute
rate(mercator_jupiter_cost_total[1m]) * 60

# Cost per hour
rate(mercator_jupiter_cost_total[1h]) * 3600

# Cost per day (projected)
rate(mercator_jupiter_cost_total[24h]) * 86400

# Cost by provider
sum by (provider) (mercator_jupiter_cost_total)

# Cost by model
sum by (model) (mercator_jupiter_cost_total)
```

### Cost per Request

```promql
# Average cost per request
rate(mercator_jupiter_cost_total[5m]) /
rate(mercator_jupiter_requests_total[5m])

# P95 cost per request
histogram_quantile(0.95, rate(mercator_jupiter_cost_per_request_bucket[5m]))

# Expensive requests (>$1)
sum(rate(mercator_jupiter_cost_per_request_bucket{le="1.0"}[5m]) == 0)

# Cost per request by model
rate(mercator_jupiter_cost_total[5m]) /
rate(mercator_jupiter_requests_total[5m])
by (model)
```

### Cost per Token

```promql
# Average cost per 1K tokens
mercator_jupiter_cost_per_token * 1000

# Cost per 1M tokens
mercator_jupiter_cost_per_token * 1000000

# Most expensive model (per token)
topk(1, mercator_jupiter_cost_per_token) by (model)

# Cheapest model (per token)
bottomk(1, mercator_jupiter_cost_per_token) by (model)
```

### Cost Optimization

```promql
# Cost savings from caching (if cache enabled)
(sum(rate(mercator_jupiter_cache_hits_total[5m])) *
 avg(rate(mercator_jupiter_cost_total[5m]) / rate(mercator_jupiter_requests_total[5m])))

# Cost per user (requires user label)
sum by (user) (rate(mercator_jupiter_cost_total[5m]))

# Top 10 most expensive users
topk(10, sum by (user) (rate(mercator_jupiter_cost_total[5m])))
```

---

## Performance Metrics

### Throughput

```promql
# Current throughput (req/s)
sum(rate(mercator_jupiter_requests_total[1m]))

# Peak throughput (last 24h)
max_over_time(sum(rate(mercator_jupiter_requests_total[1m]))[24h:])

# Average throughput (last 24h)
avg_over_time(sum(rate(mercator_jupiter_requests_total[1m]))[24h:])
```

### Cache Performance

```promql
# Cache hit rate
sum(rate(mercator_jupiter_cache_hits_total[5m])) /
(sum(rate(mercator_jupiter_cache_hits_total[5m])) + sum(rate(mercator_jupiter_cache_misses_total[5m]))) * 100

# Cache hits per second
sum(rate(mercator_jupiter_cache_hits_total[5m]))

# Cache size
sum(mercator_jupiter_cache_entries) by (cache)

# Cache eviction rate
sum(rate(mercator_jupiter_cache_evictions_total[5m]))
```

### System Resources

```promql
# Memory usage (if available from node_exporter)
process_resident_memory_bytes{job="mercator-jupiter"}

# CPU usage
rate(process_cpu_seconds_total{job="mercator-jupiter"}[5m]) * 100

# Goroutines
go_goroutines{job="mercator-jupiter"}

# GC duration
rate(go_gc_duration_seconds_sum[5m]) /
rate(go_gc_duration_seconds_count[5m])
```

---

## Alerts

### Critical Alerts

```promql
# Service Down
up{job="mercator-jupiter"} == 0

# High Error Rate (>5%)
sum(rate(mercator_jupiter_requests_total{status!="success"}[5m])) /
sum(rate(mercator_jupiter_requests_total[5m])) * 100 > 5

# No Healthy Providers
sum(mercator_jupiter_provider_health) == 0

# High Latency (P95 > 10s)
histogram_quantile(0.95, rate(mercator_jupiter_request_duration_seconds_bucket[5m])) > 10

# High Cost Burn Rate (>$100/hour)
rate(mercator_jupiter_cost_total[1h]) * 3600 > 100
```

### Warning Alerts

```promql
# Elevated Error Rate (>1%)
sum(rate(mercator_jupiter_requests_total{status!="success"}[5m])) /
sum(rate(mercator_jupiter_requests_total[5m])) * 100 > 1

# Degraded Provider
mercator_jupiter_provider_health < 1

# Slow Requests (P95 > 5s)
histogram_quantile(0.95, rate(mercator_jupiter_request_duration_seconds_bucket[5m])) > 5

# High Block Rate (>10%)
sum(rate(mercator_jupiter_policy_evaluations_total{action="block"}[5m])) /
sum(rate(mercator_jupiter_policy_evaluations_total[5m])) * 100 > 10

# Low Cache Hit Rate (<50%)
sum(rate(mercator_jupiter_cache_hits_total[5m])) /
(sum(rate(mercator_jupiter_cache_hits_total[5m])) + sum(rate(mercator_jupiter_cache_misses_total[5m]))) * 100 < 50

# High Provider Error Rate (>5%)
sum by (provider) (rate(mercator_jupiter_provider_errors_total[5m])) /
sum by (provider) (rate(mercator_jupiter_provider_requests_total[5m])) * 100 > 5
```

### AlertManager Configuration

```yaml
# alertmanager.yml
groups:
  - name: mercator-jupiter
    interval: 30s
    rules:
      # Critical: Service Down
      - alert: MercatorJupiterDown
        expr: up{job="mercator-jupiter"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Mercator Jupiter is down"
          description: "Mercator Jupiter instance {{ $labels.instance }} is down"

      # Critical: High Error Rate
      - alert: HighErrorRate
        expr: |
          sum(rate(mercator_jupiter_requests_total{status!="success"}[5m])) /
          sum(rate(mercator_jupiter_requests_total[5m])) * 100 > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }} (threshold: 5%)"

      # Critical: No Healthy Providers
      - alert: NoHealthyProviders
        expr: sum(mercator_jupiter_provider_health) == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "No healthy providers available"
          description: "All LLM providers are unhealthy"

      # Warning: High Latency
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95,
            rate(mercator_jupiter_request_duration_seconds_bucket[5m])
          ) > 10
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High request latency"
          description: "P95 latency is {{ $value | humanizeDuration }} (threshold: 10s)"

      # Warning: High Cost Burn
      - alert: HighCostBurn
        expr: rate(mercator_jupiter_cost_total[1h]) * 3600 > 100
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "High cost burn rate"
          description: "Cost burn rate is ${{ $value | humanize }}/hour (threshold: $100/hour)"
```

---

## Dashboard Queries

### Request Overview Dashboard

```promql
# Panel: Request Rate
sum(rate(mercator_jupiter_requests_total[5m]))

# Panel: Success Rate
sum(rate(mercator_jupiter_requests_total{status="success"}[5m])) /
sum(rate(mercator_jupiter_requests_total[5m])) * 100

# Panel: Latency (P50, P95, P99)
histogram_quantile(0.50, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(mercator_jupiter_request_duration_seconds_bucket[5m]))

# Panel: Requests by Provider
sum by (provider) (rate(mercator_jupiter_requests_total[5m]))

# Panel: Requests by Status
sum by (status) (rate(mercator_jupiter_requests_total[5m]))
```

### Cost Dashboard

```promql
# Panel: Total Cost
mercator_jupiter_cost_total

# Panel: Cost Rate (USD/hour)
rate(mercator_jupiter_cost_total[1h]) * 3600

# Panel: Cost by Provider
sum by (provider) (rate(mercator_jupiter_cost_total[5m]))

# Panel: Cost per Request
rate(mercator_jupiter_cost_total[5m]) /
rate(mercator_jupiter_requests_total[5m])

# Panel: Top 10 Expensive Models
topk(10, sum by (model) (rate(mercator_jupiter_cost_total[5m])))
```

### Provider Dashboard

```promql
# Panel: Provider Health
mercator_jupiter_provider_health

# Panel: Provider Request Rate
sum by (provider) (rate(mercator_jupiter_provider_requests_total[5m]))

# Panel: Provider Latency
histogram_quantile(0.95, sum by (provider, le) (
  rate(mercator_jupiter_provider_latency_seconds_bucket[5m])
))

# Panel: Provider Error Rate
sum by (provider, error_type) (rate(mercator_jupiter_provider_errors_total[5m]))

# Panel: Provider Uptime (24h)
avg_over_time(mercator_jupiter_provider_health[24h]) * 100
```

---

## Further Reading

- [Observability Guide](observability-guide.md) - Complete observability documentation
- [Grafana Dashboard](grafana-dashboard.json) - Pre-built dashboard with these queries
- [Prometheus Documentation](https://prometheus.io/docs/) - Official Prometheus docs

---

**Need more examples?** Check the [Grafana dashboard JSON](grafana-dashboard.json) for visual representations of these queries.
