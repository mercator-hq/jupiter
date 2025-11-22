# Development Workflow Policies

Guide to implementing policies for different environments, development workflows, and testing scenarios.

## Table of Contents

- [Environment-Based Policies](#environment-based-policies)
- [Time-Based Access Control](#time-based-access-control)
- [Feature Flags & A/B Testing](#feature-flags--ab-testing)
- [Testing & Staging](#testing--staging)
- [Department-Based Policies](#department-based-policies)
- [Developer Experience](#developer-experience)
- [Best Practices](#best-practices)

---

## Environment-Based Policies

**Use Case**: Different policies for development, staging, and production environments.

### Multi-Environment Policy

**File**: [docs/mpl/examples/14-environment.yaml](../mpl/examples/14-environment.yaml)

```yaml
version: "1.0"

policies:
  - name: "environment-based-policies"
    description: "Adapt policies based on deployment environment"
    priority: 100
    rules:
      # Development: Very lenient
      - condition: |
          request.metadata.environment == "development"
        action: "log"
        log_level: "debug"
        message: "DEV: {{request.metadata.user_id}} - {{request.model}}"
        # No budget limits, no rate limits in dev

      # Staging: Moderate enforcement
      - condition: |
          request.metadata.environment == "staging" and
          request.metadata.user_budget_spent >= 50.0
        action: "log"
        log_level: "warn"
        message: "STAGING: Budget warning for {{request.metadata.user_id}}"
        # Soft limits in staging

      # Production: Strict enforcement
      - condition: |
          request.metadata.environment == "production" and
          request.metadata.user_budget_spent >= 100.0
        action: "deny"
        reason: "Production budget limit ($100/day) exceeded"

      # Production: Require approved models only
      - condition: |
          request.metadata.environment == "production" and
          request.model not in ["gpt-3.5-turbo", "gpt-4", "claude-3-sonnet"]
        action: "deny"
        reason: "Model {{request.model}} not approved for production use"

      # Staging: Warn on unapproved models
      - condition: |
          request.metadata.environment == "staging" and
          request.model not in ["gpt-3.5-turbo", "gpt-4", "claude-3-sonnet"]
        action: "log"
        log_level: "warn"
        message: "Unapproved model in staging: {{request.model}}"

      # Development: Allow all models
      - condition: |
          request.metadata.environment == "development"
        action: "allow"
```

### Configuration

Set environment via metadata or configuration:

```yaml
# config.yaml
processing:
  default_metadata:
    environment: "production"  # or "staging", "development"
```

Or pass in request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Environment: production" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [...]
  }'
```

### Environment-Specific Configurations

#### Development Environment

```yaml
policies:
  - name: "dev-environment"
    rules:
      # No budgets in dev
      - condition: |
          request.metadata.environment == "development"
        action: "allow"
        metadata:
          bypass_budget: true
          bypass_rate_limit: true

      # Verbose logging
      - condition: |
          request.metadata.environment == "development"
        action: "log"
        log_level: "debug"
        message: "DEV Request: {{request}}"
```

#### Staging Environment

```yaml
policies:
  - name: "staging-environment"
    rules:
      # Production-like but lenient
      - condition: |
          request.metadata.environment == "staging"
        action: "log"
        log_level: "info"
        message: "STAGING: Testing production policies"

      # Soft budget limits (warn, don't block)
      - condition: |
          request.metadata.environment == "staging" and
          request.metadata.user_budget_spent >= 50.0
        action: "log"
        log_level: "warn"
        message: "Budget warning (soft limit)"
```

#### Production Environment

```yaml
policies:
  - name: "production-environment"
    rules:
      # Strict enforcement
      - condition: |
          request.metadata.environment == "production" and
          request.metadata.user_budget_spent >= 100.0
        action: "deny"
        reason: "Budget exceeded"

      # Approved models only
      - condition: |
          request.metadata.environment == "production" and
          request.model not in approved_models
        action: "deny"
        reason: "Model not approved for production"

      # PII protection
      - condition: |
          request.metadata.environment == "production" and
          request.contains_pii == true
        action: "deny"
        reason: "PII detected in production request"
```

---

## Time-Based Access Control

**Use Case**: Restrict LLM access based on time of day, day of week, or business hours.

### Business Hours Policy

**File**: [docs/mpl/examples/13-time-based.yaml](../mpl/examples/13-time-based.yaml)

```yaml
version: "1.0"

policies:
  - name: "business-hours-access"
    description: "Restrict access to business hours"
    priority: 200
    rules:
      # Allow during business hours (9 AM - 5 PM)
      - condition: |
          time.hour >= 9 and time.hour < 17 and
          time.weekday >= 1 and time.weekday <= 5
        action: "allow"
        log_message: "Access during business hours"

      # Deny outside business hours
      - condition: |
          time.hour < 9 or time.hour >= 17 or
          time.weekday == 0 or time.weekday == 6
        action: "deny"
        reason: "LLM access is only available during business hours (Mon-Fri, 9 AM - 5 PM {{time.timezone}})"

      # Exception: On-call staff
      - condition: |
          request.metadata.user_role == "on_call" and
          (time.hour < 9 or time.hour >= 17)
        action: "allow"
        log_level: "warn"
        log_message: "After-hours access by on-call staff: {{request.metadata.user_id}}"
```

### Time-Based Rate Limits

```yaml
policies:
  - name: "time-based-rate-limits"
    rules:
      # Peak hours: Stricter limits (9 AM - 5 PM)
      - condition: |
          time.hour >= 9 and time.hour < 17 and
          request.metadata.user_rpm_current >= 30
        action: "deny"
        reason: "Peak hours rate limit: 30 RPM"

      # Off-peak: Relaxed limits
      - condition: |
          (time.hour < 9 or time.hour >= 17) and
          request.metadata.user_rpm_current >= 60
        action: "deny"
        reason: "Off-peak rate limit: 60 RPM"
```

### Time-Based Cost Optimization

```yaml
policies:
  - name: "time-based-routing"
    rules:
      # Business hours: Premium provider
      - condition: |
          time.hour >= 9 and time.hour < 17
        action: "route"
        provider: "openai-premium"
        log_message: "Business hours - premium provider"

      # After hours: Economy provider
      - condition: |
          time.hour < 9 or time.hour >= 17
        action: "route"
        provider: "openai-economy"
        log_message: "After hours - economy provider"
```

### Weekend Policies

```yaml
policies:
  - name: "weekend-policies"
    rules:
      # Weekend: Block expensive models
      - condition: |
          (time.weekday == 0 or time.weekday == 6) and
          request.model == "gpt-4"
        action: "deny"
        reason: "GPT-4 access restricted on weekends. Use GPT-3.5 or wait until Monday."

      # Weekend: Lower budgets
      - condition: |
          (time.weekday == 0 or time.weekday == 6) and
          request.metadata.user_weekend_spend >= 20.0
        action: "deny"
        reason: "Weekend budget ($20) exceeded"
```

---

## Feature Flags & A/B Testing

**Use Case**: Gradually roll out new policies or test policy changes.

### Feature Flag Policy

```yaml
version: "1.0"

policies:
  - name: "feature-flags"
    description: "Feature flag control for gradual rollout"
    rules:
      # Feature flag: New content filter
      - condition: |
          feature_flags.new_content_filter == true and
          request.messages[-1].content matches "(?i)(test|beta)"
        action: "deny"
        reason: "New content filter (beta)"

      # Feature flag: Cost optimization
      - condition: |
          feature_flags.cost_optimization == true and
          request.estimated_cost > 1.0
        action: "modify"
        set:
          model: "gpt-3.5-turbo"
        log_message: "Cost optimization: downgraded model"

      # Feature flag: A/B test
      - condition: |
          feature_flags.ab_test_enabled == true and
          hash(request.metadata.user_id) % 2 == 0
        action: "route"
        provider: "provider_a"
        metadata:
          ab_group: "control"

      - condition: |
          feature_flags.ab_test_enabled == true
        action: "route"
        provider: "provider_b"
        metadata:
          ab_group: "treatment"
```

### Gradual Rollout

```yaml
policies:
  - name: "gradual-rollout"
    rules:
      # Week 1: 5% of users
      - condition: |
          rollout.week == 1 and
          hash(request.metadata.user_id) % 100 < 5
        action: "modify"
        set:
          use_new_policy: true

      # Week 2: 25%
      - condition: |
          rollout.week == 2 and
          hash(request.metadata.user_id) % 100 < 25
        action: "modify"
        set:
          use_new_policy: true

      # Week 3: 50%
      - condition: |
          rollout.week == 3 and
          hash(request.metadata.user_id) % 100 < 50
        action: "modify"
        set:
          use_new_policy: true

      # Week 4: 100%
      - condition: |
          rollout.week >= 4
        action: "modify"
        set:
          use_new_policy: true
```

### Canary Testing

```yaml
policies:
  - name: "canary-testing"
    rules:
      # Canary: Internal users test new policy
      - condition: |
          request.metadata.user_email matches "@company\\.com$" and
          canary.enabled == true
        action: "log"
        log_level: "info"
        message: "CANARY: Internal user testing new policy"
        metadata:
          use_canary_policy: true

      # Beta users
      - condition: |
          request.metadata.user_beta_tester == true
        action: "log"
        log_level: "info"
        message: "BETA: Beta tester using new features"
```

---

## Testing & Staging

### Test Request Detection

```yaml
policies:
  - name: "test-request-detection"
    rules:
      # Detect test requests
      - condition: |
          request.messages[-1].content matches "(?i)(test|testing|qa|staging)"
        action: "log"
        log_level: "info"
        message: "Test request detected"
        metadata:
          is_test_request: true

      # Route test requests to test provider
      - condition: |
          request.metadata.is_test_request == true
        action: "route"
        provider: "test-provider"
        log_message: "Routing test request to test provider"

      # Don't charge for test requests
      - condition: |
          request.metadata.is_test_request == true
        action: "modify"
        set:
          bypass_budget: true
```

### Staging Environment Safeguards

```yaml
policies:
  - name: "staging-safeguards"
    rules:
      # Prevent production data in staging
      - condition: |
          request.metadata.environment == "staging" and
          request.messages[-1].content matches "(?i)(production|prod|live)"
        action: "log"
        log_level: "warn"
        message: "STAGING_WARNING: Possible production data in staging"

      # Limit staging costs
      - condition: |
          request.metadata.environment == "staging" and
          global.staging_daily_spend >= 50.0
        action: "deny"
        reason: "Staging daily budget ($50) exceeded"

      # Auto-cleanup old staging evidence
      # (configured in evidence retention, not policy)
```

---

## Department-Based Policies

**Use Case**: Different policies for different departments or teams.

### Department Policy Bundle

**File**: [docs/mpl/examples/21-department-based.yaml](../mpl/examples/21-department-based.yaml)

```yaml
version: "1.0"

policies:
  - name: "department-policies"
    description: "Per-department access control and budgets"
    priority: 200
    rules:
      # Engineering: Full access
      - condition: |
          request.metadata.department == "engineering"
        action: "allow"
        log_message: "Engineering: Full LLM access"

      # Sales: Limited to approved models
      - condition: |
          request.metadata.department == "sales" and
          request.model not in ["gpt-3.5-turbo"]
        action: "deny"
        reason: "Sales team limited to GPT-3.5. Contact IT for higher tier access."

      # Marketing: Content generation only
      - condition: |
          request.metadata.department == "marketing" and
          request.metadata.use_case != "content_generation"
        action: "deny"
        reason: "Marketing approved for content generation only"

      # Finance: Stricter compliance
      - condition: |
          request.metadata.department == "finance" and
          request.contains_pii == true
        action: "deny"
        reason: "Finance: PII detected (stricter compliance)"

      # HR: Enhanced privacy
      - condition: |
          request.metadata.department == "hr"
        action: "log"
        log_level: "info"
        message: "HR_AUDIT: {{request.metadata.user_id}} - {{request.action}}"

      # Department budgets
      - condition: |
          request.metadata.department == "sales" and
          department.sales.daily_spend >= 50.0
        action: "deny"
        reason: "Sales department daily budget ($50) exceeded"

      - condition: |
          request.metadata.department == "engineering" and
          department.engineering.daily_spend >= 500.0
        action: "deny"
        reason: "Engineering department daily budget ($500) exceeded"
```

### Team-Based Access

```yaml
policies:
  - name: "team-based-access"
    rules:
      # Team leads: Higher limits
      - condition: |
          request.metadata.user_role == "team_lead"
        action: "modify"
        set:
          budget_multiplier: 2.0
        log_message: "Team lead: 2x budget multiplier"

      # Junior team members: Lower limits
      - condition: |
          request.metadata.user_level == "junior"
        action: "modify"
        set:
          max_tokens: 1000
        log_message: "Junior: Limited to 1000 tokens"
```

---

## Developer Experience

### Developer-Friendly Errors

```yaml
policies:
  - name: "dev-friendly-errors"
    rules:
      # Development: Detailed error messages
      - condition: |
          request.metadata.environment == "development"
        action: "deny"
        reason: |
          Policy violation: {{policy.violation}}

          Debug info:
          - Request ID: {{request.id}}
          - User: {{request.metadata.user_id}}
          - Model: {{request.model}}
          - Estimated cost: ${{request.estimated_cost}}

          Suggestion: {{policy.suggestion}}

      # Production: Generic error messages (security)
      - condition: |
          request.metadata.environment == "production"
        action: "deny"
        reason: "Request denied by policy. Contact support with request ID: {{request.id}}"
```

### Debug Mode

```yaml
policies:
  - name: "debug-mode"
    rules:
      # Debug mode: Log everything
      - condition: |
          request.metadata.debug_mode == true
        action: "log"
        log_level: "debug"
        message: |
          DEBUG:
          Request: {{request}}
          Metadata: {{request.metadata}}
          Policies evaluated: {{policy.evaluated}}
          Decision: {{policy.decision}}

      # Debug mode: Skip budgets (dev only)
      - condition: |
          request.metadata.environment == "development" and
          request.metadata.debug_mode == true
        action: "allow"
        metadata:
          bypass_all_limits: true
```

### Testing Utilities

```yaml
policies:
  - name: "testing-utilities"
    rules:
      # Test mode: Mark requests
      - condition: |
          request.metadata.test_mode == true
        action: "modify"
        set:
          is_test: true
          bypass_billing: true

      # Dry-run mode: Don't call provider
      - condition: |
          request.metadata.dry_run == true
        action: "log"
        log_level: "info"
        message: "DRY-RUN: Would send to {{provider.selected.name}}"
        metadata:
          skip_provider_call: true
```

---

## Best Practices

### 1. Environment Parity

Keep policies similar across environments, varying only in enforcement:

```yaml
# Same policy structure, different enforcement
policies:
  # Development: Log violations
  - condition: |
      request.metadata.environment == "development" and
      budget_exceeded
    action: "log"

  # Staging: Warn on violations
  - condition: |
      request.metadata.environment == "staging" and
      budget_exceeded
    action: "log"
    log_level: "warn"

  # Production: Block violations
  - condition: |
      request.metadata.environment == "production" and
      budget_exceeded
    action: "deny"
```

### 2. Progressive Enhancement

Start lenient, tighten over time:

```yaml
# Week 1: Log only
- condition: |
    deployment.week == 1 and policy_violation
  action: "log"
  log_level: "warn"

# Week 2: Warn users
- condition: |
    deployment.week == 2 and policy_violation
  action: "log"
  log_level: "error"
  notify: true

# Week 3: Soft enforcement
- condition: |
    deployment.week == 3 and policy_violation
  action: "deny"
  grace_period: true

# Week 4+: Full enforcement
- condition: |
    deployment.week >= 4 and policy_violation
  action: "deny"
```

### 3. Testing in Production

```yaml
# Shadow mode: Run new policy without blocking
- condition: |
    feature_flags.shadow_new_policy == true
  action: "log"
  log_level: "info"
  message: "SHADOW: New policy would {{hypothetical_action}}"
  metadata:
    shadow_result: "{{hypothetical_action}}"
```

### 4. Environment Detection

Automatically detect environment:

```yaml
# Detect based on domain
- condition: |
    request.metadata.origin matches "localhost|127\\.0\\.0\\.1"
  action: "modify"
  set:
    environment: "development"

- condition: |
    request.metadata.origin matches "staging\\."
  action: "modify"
  set:
    environment: "staging"

- condition: |
    request.metadata.origin matches "app\\."
  action: "modify"
  set:
    environment: "production"
```

### 5. Documentation

Document environment-specific behavior:

```yaml
# Clear documentation in policy
- name: "environment-docs"
  description: |
    Environment behavior:
    - Development: No limits, verbose logging
    - Staging: Soft limits, warning alerts
    - Production: Hard limits, error alerts

    To change environment, set X-Environment header
    or configure in deployment config.
```

### 6. Monitoring

Monitor policy effectiveness per environment:

```bash
# Compare policy decisions across environments
mercator evidence query \
  --time-range "last 7 days" \
  --format json | \
  jq 'group_by(.request.metadata.environment) |
      map({
        environment: .[0].request.metadata.environment,
        total: length,
        denied: [.[] | select(.policy_decision.action == "deny")] | length,
        denial_rate: (([.[] | select(.policy_decision.action == "deny")] | length) / length)
      })'
```

---

## See Also

- [Policy Cookbook](cookbook.md) - All policy examples
- [Environment Policy Example](../mpl/examples/14-environment.yaml)
- [Time-Based Policy Example](../mpl/examples/13-time-based.yaml)
- [Department Policy Example](../mpl/examples/21-department-based.yaml)
- [Configuration Reference](../configuration/reference.md)

---

## Testing Environment Policies

```yaml
# environment-tests.yaml
tests:
  - name: "Dev: Should allow all requests"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
      metadata:
        environment: "development"
    expected:
      action: "allow"

  - name: "Production: Should enforce limits"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
      metadata:
        environment: "production"
        user_budget_spent: 100.0
        user_budget_limit: 100.0
    expected:
      action: "deny"
      reason_contains: "budget"

  - name: "Business hours: Should allow"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Test"
    time:
      hour: 10
      weekday: 2  # Tuesday
    expected:
      action: "allow"

  - name: "After hours: Should deny"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Test"
    time:
      hour: 22
      weekday: 2
    expected:
      action: "deny"
      reason_contains: "business hours"
```

Run tests:
```bash
mercator test --policy environment-policies.yaml --tests environment-tests.yaml
```
