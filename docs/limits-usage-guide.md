# Budget & Rate Limiting - Usage Guide

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Metrics](#metrics)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The Budget & Rate Limiting system provides comprehensive cost control and usage governance for LLM requests. It supports:

- **Multi-dimensional limits**: Per-API key, per-user, per-team
- **Budget tracking**: Rolling hourly, daily, and monthly budgets
- **Rate limiting**: Request-based, token-based, and concurrent limits
- **Smart enforcement**: Block, queue, downgrade, or alert actions
- **High performance**: Sub-millisecond limit checks
- **Persistent storage**: Memory or SQLite backends

## Quick Start

### 1. Basic Configuration

Create a `config.yaml` file:

```yaml
limits:
  budgets:
    enabled: true
    alert_threshold: 0.8  # Alert at 80% usage
    by_api_key:
      "my-api-key":
        daily: 100.00     # $100/day
        monthly: 2500.00  # $2500/month

  rate_limits:
    enabled: true
    by_api_key:
      "my-api-key":
        requests_per_second: 10
        tokens_per_minute: 100000

  enforcement:
    action: block  # Or: queue, downgrade, alert

  storage:
    backend: memory  # Or: sqlite
```

### 2. Initialize Manager

```go
import (
    "mercator-hq/jupiter/pkg/config"
    "mercator-hq/jupiter/pkg/limits"
)

// Load configuration
cfg, err := config.LoadConfigWithEnvOverrides("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Create limits manager
manager := createLimitsManager(cfg.Limits)
defer manager.Close()
```

### 3. Check Limits Before Request

```go
// Check if request is allowed
result, err := manager.CheckLimits(
    ctx,
    apiKey,           // Identifier
    estimatedTokens,  // Estimated token count
    estimatedCost,    // Estimated cost in USD
    model,            // Model name
)

if !result.Allowed {
    // Handle limit exceeded
    http.Error(w, result.Reason, http.StatusTooManyRequests)
    return
}
```

### 4. Record Actual Usage

```go
// After request completes
err = manager.RecordUsage(ctx, &limits.UsageRecord{
    Identifier:     apiKey,
    Dimension:      limits.DimensionAPIKey,
    RequestTokens:  actualPromptTokens,
    ResponseTokens: actualCompletionTokens,
    TotalTokens:    totalTokens,
    Cost:           actualCost,
    Provider:       "openai",
    Model:          "gpt-4",
})
```

## Configuration

### Budget Configuration

```yaml
budgets:
  enabled: true
  alert_threshold: 0.8  # Trigger alert at 80% usage

  # Per-API key budgets
  by_api_key:
    "production-key":
      hourly: 10.00
      daily: 200.00
      monthly: 5000.00

  # Per-user budgets
  by_user:
    "alice@example.com":
      daily: 50.00
      monthly: 1000.00

  # Per-team budgets
  by_team:
    "engineering":
      daily: 500.00
      monthly: 10000.00
```

### Rate Limit Configuration

```yaml
rate_limits:
  enabled: true

  by_api_key:
    "production-key":
      requests_per_second: 10    # Max 10 req/s
      requests_per_minute: 500   # Max 500 req/min
      requests_per_hour: 10000   # Max 10K req/hour
      tokens_per_minute: 100000  # Max 100K tokens/min
      tokens_per_hour: 1000000   # Max 1M tokens/hour
      max_concurrent: 20         # Max 20 simultaneous requests
```

### Enforcement Configuration

```yaml
enforcement:
  # Action when limits exceeded
  action: block  # Options: block, queue, downgrade, alert

  # Queue configuration (if action=queue)
  queue_depth: 100
  queue_timeout: 30s

  # Model downgrade mapping (if action=downgrade)
  model_downgrades:
    "gpt-4": "gpt-4-turbo"
    "gpt-4-turbo": "gpt-3.5-turbo"
    "claude-3-opus": "claude-3-sonnet"
```

### Storage Configuration

```yaml
storage:
  # Backend: memory (fast, no persistence) or sqlite (persistent)
  backend: sqlite

  sqlite:
    path: /var/lib/mercator/limits.db
    snapshot_interval: 5m

  memory:
    max_entries: 100000
    cleanup_interval: 1m
```

### Environment Variables

Override any configuration with environment variables:

```bash
# Budget settings
export MERCATOR_LIMITS_BUDGETS_ENABLED=true
export MERCATOR_LIMITS_BUDGETS_ALERT_THRESHOLD=0.8

# Rate limit settings
export MERCATOR_LIMITS_RATE_LIMITS_ENABLED=true

# Enforcement
export MERCATOR_LIMITS_ENFORCEMENT_ACTION=downgrade

# Storage
export MERCATOR_LIMITS_STORAGE_BACKEND=sqlite
export MERCATOR_LIMITS_STORAGE_SQLITE_PATH=/data/limits.db
```

## Usage Examples

### Example 1: Simple Rate Limiting

```go
// Configure rate limits only (no budgets)
config := limits.Config{
    RateLimits: map[string]ratelimit.Config{
        "api-key-123": {
            RequestsPerSecond: 10,
            MaxConcurrent:     5,
        },
    },
}

manager := limits.NewManager(config)
defer manager.Close()

// Check limits
result, err := manager.CheckLimits(ctx, "api-key-123", 0, 0, "gpt-4")
if !result.Allowed {
    log.Printf("Rate limit exceeded: %s", result.Reason)
    return
}
```

### Example 2: Budget Tracking with Alerts

```go
// Configure budgets with alert threshold
config := limits.Config{
    Budgets: map[string]budget.Config{
        "api-key-123": {
            Daily:          100.00,  // $100/day
            AlertThreshold: 0.8,     // Alert at 80%
        },
    },
}

manager := limits.NewManager(config)
defer manager.Close()

// Check limits
result, _ := manager.CheckLimits(ctx, "api-key-123", 1000, 0.05, "gpt-4")

// Handle alert
if result.Action == limits.ActionAlert {
    log.Printf("Budget alert: %.1f%% of daily budget used",
        result.Budget.Percentage * 100)
    // Send alert notification (email, Slack, etc.)
}
```

### Example 3: Model Downgrade

```go
// Configure automatic model downgrade
config := limits.Config{
    Budgets: map[string]budget.Config{
        "api-key-123": {Daily: 50.00},
    },
    Enforcement: enforcement.Config{
        DefaultAction: enforcement.ActionDowngrade,
        ModelDowngrades: map[string]string{
            "gpt-4":         "gpt-3.5-turbo",
            "claude-3-opus": "claude-3-sonnet",
        },
    },
}

manager := limits.NewManager(config)
defer manager.Close()

// When budget exceeded, automatically downgrade
result, _ := manager.CheckLimits(ctx, "api-key-123", 0, 0, "gpt-4")

if result.Action == limits.ActionDowngrade {
    log.Printf("Downgrading from %s to %s due to budget",
        "gpt-4", result.DowngradeTo)
    // Use downgraded model instead
    model = result.DowngradeTo
}
```

### Example 4: Concurrent Request Limiting

```go
// Acquire concurrent slot before processing
if manager.AcquireConcurrent(apiKey) {
    defer manager.ReleaseConcurrent(apiKey)

    // Process request
    response := processLLMRequest(request)

    // Record usage
    manager.RecordUsage(ctx, usageRecord)
} else {
    // Too many concurrent requests
    http.Error(w, "Too many concurrent requests", 429)
}
```

### Example 5: HTTP Middleware Integration

```go
import "mercator-hq/jupiter/pkg/proxy/middleware"

// Create limits middleware
limitsHandler := middleware.LimitsMiddleware(manager)

// Add to handler chain
mux := http.NewServeMux()
mux.Handle("/v1/chat/completions",
    limitsHandler(chatCompletionHandler))
```

## Metrics

The limits system exports Prometheus metrics for monitoring.

### Available Metrics

**Rate Limit Metrics:**
- `mercator_limits_rate_limit_checks_total{identifier, result}` - Total rate limit checks
- `mercator_limits_rate_limit_hits_total{identifier, limit_type}` - Rate limit violations

**Budget Metrics:**
- `mercator_limits_budget_checks_total{identifier, result}` - Total budget checks
- `mercator_limits_budget_hits_total{identifier, window}` - Budget limit violations
- `mercator_limits_budget_usage_percentage{identifier, window}` - Current budget usage %

**Enforcement Metrics:**
- `mercator_limits_enforcement_actions_total{identifier, action}` - Enforcement actions taken

**Performance Metrics:**
- `mercator_limits_check_duration_seconds{operation}` - Duration of limit checks
- `mercator_limits_concurrent_requests{identifier}` - Current concurrent requests

### Grafana Dashboard Example

```promql
# Rate limit hit rate
rate(mercator_limits_rate_limit_hits_total[5m])

# Budget usage by API key
mercator_limits_budget_usage_percentage{window="daily"}

# Average check latency
histogram_quantile(0.99,
  rate(mercator_limits_check_duration_seconds_bucket[5m]))
```

## Best Practices

### 1. Set Conservative Limits Initially

Start with conservative limits and increase based on actual usage:

```yaml
budgets:
  by_api_key:
    "new-customer":
      daily: 10.00    # Start low
      monthly: 200.00 # Increase after monitoring
```

### 2. Use Alert Thresholds

Always configure alert thresholds to get early warnings:

```yaml
budgets:
  alert_threshold: 0.8  # Alert at 80%
```

### 3. Combine Multiple Limit Types

Use both rate limits and budgets for comprehensive control:

```yaml
by_api_key:
  "production-key":
    # Rate limits prevent abuse
    requests_per_second: 10

    # Budget limits prevent cost overruns
    daily: 100.00
```

### 4. Configure Model Downgrades

Set up intelligent fallbacks to maintain service:

```yaml
enforcement:
  action: downgrade
  model_downgrades:
    "gpt-4": "gpt-3.5-turbo"  # Cheaper fallback
```

### 5. Monitor Metrics

Set up Prometheus alerts for limit violations:

```yaml
# prometheus.rules.yml
groups:
  - name: limits
    rules:
      - alert: HighBudgetUsage
        expr: mercator_limits_budget_usage_percentage > 0.9
        for: 5m
        annotations:
          summary: "Budget usage above 90%"
```

### 6. Use SQLite for Production

Enable persistence for production deployments:

```yaml
storage:
  backend: sqlite
  sqlite:
    path: /var/lib/mercator/limits.db
```

## Troubleshooting

### Issue: Limits not being enforced

**Check:**
1. Verify limits are enabled in config:
   ```yaml
   budgets:
     enabled: true  # Must be true
   ```

2. Verify identifier exists in config:
   ```yaml
   by_api_key:
     "your-api-key":  # Must match exactly
       daily: 100.00
   ```

3. Check logs for limit check results

### Issue: Too many false positives

**Solution:** Adjust limit thresholds:

```yaml
# Increase limits
rate_limits:
  by_api_key:
    "key":
      requests_per_second: 20  # Was 10
```

### Issue: Performance degradation

**Check:**
1. Storage backend (memory is faster than SQLite)
2. Number of tracked identifiers (use cleanup)
3. Monitor check duration metrics

**Solution:**
```yaml
storage:
  backend: memory  # Faster
  memory:
    max_entries: 100000      # Limit memory usage
    cleanup_interval: 1m     # Clean up regularly
```

### Issue: Budget not resetting

**Remember:** Budgets use **rolling windows**, not fixed periods:
- "Daily" = last 24 hours (not calendar day)
- "Monthly" = last 30 days (not calendar month)

Budget decreases as old spending expires from the window.

### Issue: SQLite database locked

**Solution:** Ensure only one instance writes to the database:

```yaml
storage:
  sqlite:
    # Use different paths for different instances
    path: /var/lib/mercator/limits-instance-1.db
```

Or use Redis for distributed setups (future enhancement).

## Response Headers

When limits are checked, the following headers are set:

### Rate Limit Headers (RFC 6585)

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 456
X-RateLimit-Reset: 1638360000
Retry-After: 60
```

### Budget Headers (Custom)

```
X-Budget-Limit: 100.00
X-Budget-Used: 78.50
X-Budget-Remaining: 21.50
X-Budget-Reset: 1638360000
```

## API Reference

See [pkg/limits documentation](../pkg/limits/doc.go) for complete API reference.

### Key Functions

- `NewManager(config Config) *Manager` - Create limits manager
- `CheckLimits(ctx, identifier, tokens, cost, model) (*LimitCheckResult, error)` - Check limits
- `RecordUsage(ctx, record *UsageRecord) error` - Record usage
- `AcquireConcurrent(identifier) bool` - Acquire concurrent slot
- `ReleaseConcurrent(identifier)` - Release concurrent slot
- `Close() error` - Cleanup resources

## Support

For issues or questions:
- GitHub Issues: https://github.com/mercator-hq/jupiter/issues
- Documentation: https://docs.mercator.com/jupiter/limits
