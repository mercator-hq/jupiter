# Budget & Rate Limiting Policies

Comprehensive guide to controlling LLM costs and usage through budgets, rate limits, and token restrictions.

## Table of Contents

- [Cost Control & Budgets](#cost-control--budgets)
- [Rate Limiting](#rate-limiting)
- [Token Limits](#token-limits)
- [Multi-Level Limits](#multi-level-limits)
- [Monitoring & Alerting](#monitoring--alerting)
- [Best Practices](#best-practices)

---

## Cost Control & Budgets

**Use Case**: Enforce spending limits to prevent runaway costs.

### Basic Budget Enforcement

**File**: [docs/mpl/examples/07-cost-control.yaml](../mpl/examples/07-cost-control.yaml)

```yaml
version: "1.0"

policies:
  - name: "budget-enforcement"
    description: "Enforce user and team spending limits"
    priority: 200
    rules:
      # Hard limit: Deny when budget exceeded
      - condition: |
          request.metadata.user_budget_spent >= request.metadata.user_budget_limit
        action: "deny"
        reason: "Daily budget of ${{request.metadata.user_budget_limit}} exceeded. Current spend: ${{request.metadata.user_budget_spent}}"

      # Warning at 90% threshold
      - condition: |
          request.metadata.user_budget_spent >= (request.metadata.user_budget_limit * 0.9)
        action: "log"
        log_level: "warn"
        message: "User {{request.metadata.user_id}} approaching budget limit: ${{request.metadata.user_budget_spent}}/${{request.metadata.user_budget_limit}}"

      # Info logging for all spending
      - condition: "true"
        action: "log"
        log_level: "info"
        message: "Request cost: ${{request.estimated_cost}} | User spend: ${{request.metadata.user_budget_spent}}/${{request.metadata.user_budget_limit}}"
```

### Configuration

Enable budget tracking in `config.yaml`:

```yaml
limits:
  budgets:
    enabled: true
    default_daily_limit: 100.0  # $100/day per user
    enforcement: "hard"  # or "soft"
```

### Budget Levels

#### 1. Per-User Budgets

```yaml
- condition: |
    request.metadata.user_budget_spent >= request.metadata.user_budget_limit
  action: "deny"
  reason: "User budget exceeded"
```

#### 2. Per-Team Budgets

```yaml
- condition: |
    request.metadata.team_budget_spent >= request.metadata.team_budget_limit
  action: "deny"
  reason: "Team '{{request.metadata.team_id}}' budget exceeded"
```

#### 3. Global Budget

```yaml
- condition: |
    global.total_daily_spend >= global.daily_budget_limit
  action: "deny"
  reason: "Organization daily budget exceeded"
```

#### 4. Model-Specific Budgets

```yaml
# Expensive models have tighter limits
- condition: |
    request.model == "gpt-4" and
    request.metadata.user_gpt4_budget_spent >= 50.0
  action: "deny"
  reason: "GPT-4 budget ($50/day) exceeded"

# Cheaper models have higher limits
- condition: |
    request.model == "gpt-3.5-turbo" and
    request.metadata.user_gpt35_budget_spent >= 200.0
  action: "deny"
  reason: "GPT-3.5 budget ($200/day) exceeded"
```

### Budget Time Windows

#### Daily Budgets

```yaml
# Resets at midnight UTC
- condition: |
    request.metadata.user_daily_spend >= request.metadata.daily_limit
  action: "deny"
  reason: "Daily budget exceeded. Resets at midnight UTC."
```

#### Monthly Budgets

```yaml
# Resets on 1st of month
- condition: |
    request.metadata.user_monthly_spend >= request.metadata.monthly_limit
  action: "deny"
  reason: "Monthly budget exceeded. Resets on {{next_month_start}}"
```

#### Rolling Window Budgets

```yaml
# Last 24 hours
- condition: |
    request.metadata.user_spend_last_24h >= 100.0
  action: "deny"
  reason: "Spending limit for rolling 24-hour window exceeded"
```

### Soft vs. Hard Enforcement

#### Hard Enforcement (Block)

```yaml
limits:
  budgets:
    enforcement: "hard"

# Policy blocks requests
- condition: |
    request.metadata.user_budget_spent >= request.metadata.user_budget_limit
  action: "deny"
```

#### Soft Enforcement (Warn)

```yaml
limits:
  budgets:
    enforcement: "soft"

# Policy logs warnings but allows requests
- condition: |
    request.metadata.user_budget_spent >= request.metadata.user_budget_limit
  action: "log"
  log_level: "error"
  message: "USER OVER BUDGET: {{request.metadata.user_id}}"
```

---

## Rate Limiting

**Use Case**: Control request frequency to prevent abuse and ensure fair resource allocation.

### Basic Rate Limiting

**File**: [docs/mpl/examples/05-rate-limiting.yaml](../mpl/examples/05-rate-limiting.yaml)

```yaml
version: "1.0"

policies:
  - name: "rate-limiting"
    description: "Enforce request rate limits"
    priority: 250
    rules:
      # Requests per minute (RPM)
      - condition: |
          request.metadata.user_rpm_current >= request.metadata.user_rpm_limit
        action: "deny"
        reason: "Rate limit exceeded: {{request.metadata.user_rpm_current}}/{{request.metadata.user_rpm_limit}} requests per minute"

      # Tokens per minute (TPM)
      - condition: |
          request.metadata.user_tpm_current >= request.metadata.user_tpm_limit
        action: "deny"
        reason: "Token rate limit exceeded: {{request.metadata.user_tpm_current}}/{{request.metadata.user_tpm_limit}} tokens per minute"

      # Warning at 80% of limit
      - condition: |
          request.metadata.user_rpm_current >= (request.metadata.user_rpm_limit * 0.8)
        action: "log"
        log_level: "warn"
        message: "User approaching rate limit: {{request.metadata.user_rpm_current}}/{{request.metadata.user_rpm_limit}} RPM"
```

### Configuration

```yaml
limits:
  rate_limiting:
    enabled: true
    default_rpm: 60        # Requests per minute
    default_tpm: 100000    # Tokens per minute
    window_size: "1m"
```

### Rate Limit Types

#### 1. Requests Per Minute (RPM)

```yaml
# Standard rate limiting
- condition: |
    request.metadata.user_rpm_current >= 60
  action: "deny"
  reason: "Maximum 60 requests per minute"
```

#### 2. Tokens Per Minute (TPM)

```yaml
# Token-based rate limiting (better for cost control)
- condition: |
    request.metadata.user_tpm_current + request.estimated_tokens >= 100000
  action: "deny"
  reason: "Token rate limit would be exceeded"
```

#### 3. Requests Per Day (RPD)

```yaml
- condition: |
    request.metadata.user_requests_today >= 1000
  action: "deny"
  reason: "Daily request limit (1000) exceeded"
```

#### 4. Concurrent Requests

```yaml
- condition: |
    request.metadata.user_concurrent_requests >= 5
  action: "deny"
  reason: "Maximum concurrent requests (5) reached"
```

### Tiered Rate Limits

Different limits for different user tiers:

```yaml
policies:
  - name: "tiered-rate-limits"
    rules:
      # Free tier: 10 RPM
      - condition: |
          request.metadata.user_tier == "free" and
          request.metadata.user_rpm_current >= 10
        action: "deny"
        reason: "Free tier limit: 10 requests/minute. Upgrade for higher limits."

      # Pro tier: 100 RPM
      - condition: |
          request.metadata.user_tier == "pro" and
          request.metadata.user_rpm_current >= 100
        action: "deny"
        reason: "Pro tier limit: 100 requests/minute"

      # Enterprise tier: 1000 RPM
      - condition: |
          request.metadata.user_tier == "enterprise" and
          request.metadata.user_rpm_current >= 1000
        action: "deny"
        reason: "Enterprise tier limit: 1000 requests/minute"
```

### Per-Model Rate Limits

```yaml
# Expensive models get lower rate limits
- condition: |
    request.model == "gpt-4" and
    request.metadata.user_gpt4_rpm_current >= 10
  action: "deny"
  reason: "GPT-4 limit: 10 requests/minute"

# Cheaper models get higher rate limits
- condition: |
    request.model == "gpt-3.5-turbo" and
    request.metadata.user_gpt35_rpm_current >= 60
  action: "deny"
  reason: "GPT-3.5 limit: 60 requests/minute"
```

---

## Token Limits

**Use Case**: Control token usage per request to manage costs and enforce quality standards.

### Basic Token Limits

**File**: [docs/mpl/examples/03-token-limits.yaml](../mpl/examples/03-token-limits.yaml)

```yaml
version: "1.0"

policies:
  - name: "token-limits"
    description: "Enforce token usage limits"
    priority: 200
    rules:
      # Maximum tokens per request
      - condition: |
          request.max_tokens > 4000
        action: "deny"
        reason: "Maximum 4000 tokens per request"

      # Estimated total tokens (prompt + completion)
      - condition: |
          request.estimated_total_tokens > 8000
        action: "deny"
        reason: "Estimated total tokens ({{request.estimated_total_tokens}}) exceeds limit (8000)"

      # Model-specific limits
      - condition: |
          request.model == "gpt-4" and request.max_tokens > 2000
        action: "deny"
        reason: "GPT-4 limit: 2000 tokens per request"

      # Warn on large requests
      - condition: |
          request.estimated_total_tokens > 6000
        action: "log"
        log_level: "warn"
        message: "Large request: {{request.estimated_total_tokens}} tokens"
```

### Token Limit Types

#### 1. Request Token Limits

```yaml
# Limit max_tokens parameter
- condition: |
    request.max_tokens > 4000
  action: "deny"
  reason: "Maximum 4000 output tokens"
```

#### 2. Prompt Token Limits

```yaml
# Limit input prompt length
- condition: |
    request.estimated_prompt_tokens > 8000
  action: "deny"
  reason: "Prompt too long: {{request.estimated_prompt_tokens}} tokens (max: 8000)"
```

#### 3. Context Window Limits

```yaml
# Prevent context window overflow
- condition: |
    request.estimated_total_tokens > model.context_window * 0.95
  action: "deny"
  reason: "Request would exceed model context window"
```

#### 4. Per-Message Token Limits

```yaml
# Limit individual message length
- condition: |
    request.messages[-1].estimated_tokens > 2000
  action: "deny"
  reason: "Single message cannot exceed 2000 tokens"
```

### Context Window Management

```yaml
policies:
  - name: "context-window-management"
    rules:
      # GPT-4: 8K context
      - condition: |
          request.model == "gpt-4" and
          request.estimated_total_tokens > 8000
        action: "deny"
        reason: "GPT-4 context limit (8K) exceeded"

      # GPT-4-turbo: 128K context
      - condition: |
          request.model == "gpt-4-turbo" and
          request.estimated_total_tokens > 128000
        action: "deny"
        reason: "GPT-4-turbo context limit (128K) exceeded"

      # Claude-3-opus: 200K context
      - condition: |
          request.model == "claude-3-opus" and
          request.estimated_total_tokens > 200000
        action: "deny"
        reason: "Claude-3-opus context limit (200K) exceeded"
```

---

## Multi-Level Limits

Combine multiple limit types for comprehensive control:

```yaml
version: "1.0"

policies:
  - name: "comprehensive-limits"
    description: "Multi-level cost and usage control"
    priority: 200
    rules:
      # Level 1: Token limits (immediate)
      - condition: |
          request.max_tokens > 4000
        action: "deny"
        reason: "Token limit exceeded"

      # Level 2: Rate limits (per minute)
      - condition: |
          request.metadata.user_rpm_current >= 60
        action: "deny"
        reason: "Rate limit: 60 RPM"

      # Level 3: Daily request limits
      - condition: |
          request.metadata.user_requests_today >= 1000
        action: "deny"
        reason: "Daily request limit: 1000"

      # Level 4: Daily budget limits
      - condition: |
          request.metadata.user_budget_spent >= 100.0
        action: "deny"
        reason: "Daily budget: $100"

      # Level 5: Monthly budget limits
      - condition: |
          request.metadata.user_monthly_budget_spent >= 1000.0
        action: "deny"
        reason: "Monthly budget: $1000"
```

---

## Monitoring & Alerting

### Real-Time Monitoring

```yaml
policies:
  - name: "usage-monitoring"
    rules:
      # Log every request with costs
      - condition: "true"
        action: "log"
        log_level: "info"
        message: |
          User: {{request.metadata.user_id}}
          Cost: ${{request.estimated_cost}}
          Tokens: {{request.estimated_total_tokens}}
          Daily spend: ${{request.metadata.user_budget_spent}}
```

### Threshold Alerts

```yaml
# Alert at 50% of budget
- condition: |
    request.metadata.user_budget_spent >= (request.metadata.user_budget_limit * 0.5)
  action: "log"
  log_level: "info"
  message: "BUDGET_ALERT_50: User {{request.metadata.user_id}}"

# Alert at 80% of budget
- condition: |
    request.metadata.user_budget_spent >= (request.metadata.user_budget_limit * 0.8)
  action: "log"
  log_level: "warn"
  message: "BUDGET_ALERT_80: User {{request.metadata.user_id}}"

# Alert at 100% (exceeded)
- condition: |
    request.metadata.user_budget_spent >= request.metadata.user_budget_limit
  action: "log"
  log_level: "error"
  message: "BUDGET_EXCEEDED: User {{request.metadata.user_id}}"
```

### Query Usage Metrics

```bash
# Total spending today
mercator evidence query \
  --time-range "today" \
  --format json | \
  jq '[.[] | .cost] | add'

# Top spenders
mercator evidence query \
  --time-range "last 7 days" \
  --format json | \
  jq 'group_by(.request.metadata.user_id) |
      map({user: .[0].request.metadata.user_id, total: ([.[] | .cost] | add)}) |
      sort_by(.total) |
      reverse |
      .[0:10]'

# Requests by model
mercator evidence query \
  --format json | \
  jq 'group_by(.request.model) |
      map({model: .[0].request.model, count: length}) |
      sort_by(.count) |
      reverse'
```

---

## Best Practices

### 1. Start Conservative, Relax Gradually

```yaml
# Week 1: Strict limits
default_daily_limit: 10.0

# Week 2: Monitor and adjust
default_daily_limit: 25.0

# Week 3: Based on patterns
default_daily_limit: 50.0

# Production: Optimal limits
default_daily_limit: 100.0
```

### 2. Different Limits for Different Environments

```yaml
# Development: Very lenient
- condition: |
    request.metadata.environment == "development"
  action: "allow"  # No limits

# Staging: Moderate limits
- condition: |
    request.metadata.environment == "staging" and
    request.metadata.user_budget_spent >= 50.0
  action: "deny"

# Production: Strict limits
- condition: |
    request.metadata.environment == "production" and
    request.metadata.user_budget_spent >= 100.0
  action: "deny"
```

### 3. Grace Periods for Exceeded Limits

```yaml
# Allow 10% overage with warning
- condition: |
    request.metadata.user_budget_spent >= request.metadata.user_budget_limit and
    request.metadata.user_budget_spent < (request.metadata.user_budget_limit * 1.1)
  action: "log"
  log_level: "error"
  message: "OVERAGE WARNING: User {{request.metadata.user_id}} in grace period"

# Hard stop at 110%
- condition: |
    request.metadata.user_budget_spent >= (request.metadata.user_budget_limit * 1.1)
  action: "deny"
  reason: "Budget exceeded (including 10% grace period)"
```

### 4. Model-Aware Cost Control

```yaml
# Route expensive models to budget-conscious users
- condition: |
    request.model == "gpt-4" and
    request.metadata.user_budget_remaining < 10.0
  action: "modify"
  set:
    model: "gpt-3.5-turbo"
  log_message: "Downgraded to GPT-3.5 due to low budget"
```

### 5. Combine with Routing for Cost Optimization

```yaml
# Route to cheaper provider when possible
- condition: |
    request.model == "gpt-3.5-turbo" and
    request.metadata.user_budget_remaining < 20.0
  action: "route"
  provider: "openrouter"  # Cheaper alternative
  log_message: "Routed to cost-effective provider"
```

### 6. Transparent Communication

Provide clear error messages:

```yaml
- action: "deny"
  reason: |
    Daily budget limit reached (${{request.metadata.user_budget_limit}}).

    Your spending today: ${{request.metadata.user_budget_spent}}
    This request cost: ${{request.estimated_cost}}

    Your budget resets at midnight UTC.
    Upgrade your plan for higher limits.
```

### 7. Testing Budget Policies

```yaml
# budget-tests.yaml
tests:
  - name: "Should deny when budget exceeded"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
      metadata:
        user_id: "user-123"
        user_budget_spent: 100.0
        user_budget_limit: 100.0
    expected:
      action: "deny"
      reason_contains: "budget"

  - name: "Should allow when under budget"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
      metadata:
        user_id: "user-123"
        user_budget_spent: 50.0
        user_budget_limit: 100.0
    expected:
      action: "allow"

  - name: "Should warn at 90% threshold"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
      metadata:
        user_id: "user-123"
        user_budget_spent: 90.0
        user_budget_limit: 100.0
    expected:
      action: "allow"
      logs_contain: "approaching budget"
```

---

## Cost Optimization Strategies

### 1. Progressive Model Downgrading

```yaml
# Start with best model, downgrade if budget low
- condition: |
    request.model == "gpt-4" and
    request.metadata.user_budget_remaining < 5.0
  action: "modify"
  set:
    model: "gpt-3.5-turbo"
```

### 2. Token-Aware Request Optimization

```yaml
# Trim prompts for budget-constrained users
- condition: |
    request.estimated_total_tokens > 2000 and
    request.metadata.user_budget_remaining < 10.0
  action: "modify"
  set:
    max_tokens: 500
  log_message: "Reduced max_tokens due to budget constraints"
```

### 3. Batching Encouragement

```yaml
# Offer discounts for batch requests
- condition: |
    request.messages.length > 5
  action: "log"
  log_level: "info"
  message: "BATCH_DISCOUNT_APPLIED"
```

---

## See Also

- [Policy Cookbook](cookbook.md) - All policy examples
- [Configuration Reference](../configuration/reference.md) - Limits configuration
- [Observability Guide](../observability-guide.md) - Monitoring costs
- [Testing Guide](../cli/test.md) - Testing budget policies

---

## Complete Example

See [docs/mpl/examples/07-cost-control.yaml](../mpl/examples/07-cost-control.yaml) for a production-ready budget enforcement policy.
