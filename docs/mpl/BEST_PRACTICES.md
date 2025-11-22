# MPL Best Practices Guide

**Version:** 1.0.0
**Last Updated:** 2025-11-16

---

## Table of Contents

1. [Policy Organization](#1-policy-organization)
2. [Rule Design](#2-rule-design)
3. [Condition Writing](#3-condition-writing)
4. [Action Selection](#4-action-selection)
5. [Variable Usage](#5-variable-usage)
6. [Performance Optimization](#6-performance-optimization)
7. [Security Considerations](#7-security-considerations)
8. [Testing Policies](#8-testing-policies)
9. [Version Management](#9-version-management)
10. [Common Patterns](#10-common-patterns)

---

## 1. Policy Organization

### 1.1 Single vs Multiple Policy Files

**When to use a single policy file:**
- Small organizations with simple governance needs
- Fewer than 10 rules
- All rules relate to a single concern (e.g., only cost control)

**When to use multiple policy files:**
- Large organizations with complex requirements
- More than 10 rules
- Rules cover different concerns (safety, cost, compliance)
- Different teams manage different policies

**Example Structure:**

```
policies/
├── safety-policy.yaml          # PII, prompt injection, content filtering
├── cost-control-policy.yaml    # Token limits, budgets, cost alerts
├── compliance-policy.yaml      # Audit logging, data residency
└── user-tier-policy.yaml       # User-based access control
```

### 1.2 Policy Naming Conventions

**Good naming:**
- `production-safety-policy`
- `cost-control-policy`
- `compliance-audit-policy`

**Bad naming:**
- `policy1`
- `new-policy`
- `final-policy`

**Guidelines:**
- Use kebab-case (lowercase with hyphens)
- Be descriptive and specific
- Include environment if applicable (`production-`, `staging-`)
- Avoid version numbers in names (use `version` field instead)

### 1.3 Rule Ordering

**First-match wins:** Rules are evaluated top-to-bottom. The first matching rule determines the action.

**Best practice: Most specific first**

```yaml
rules:
  # Specific: Premium users with high usage
  - name: "premium-high-usage-alert"
    conditions:
      - field: "context.user_attributes.tier"
        operator: "=="
        value: "premium"
      - field: "context.user_attributes.requests_today"
        operator: ">"
        value: 1000
    actions:
      - type: "alert"
        message: "Premium user high usage"

  # Less specific: All premium users
  - name: "premium-users-allow"
    conditions:
      - field: "context.user_attributes.tier"
        operator: "=="
        value: "premium"
    actions:
      - type: "allow"

  # Least specific: Default behavior
  - name: "default-allow"
    conditions: []
    actions:
      - type: "allow"
```

**Alternative: Most common first (for performance)**

If you have telemetry data showing request patterns, order rules by frequency to minimize evaluation time.

---

## 2. Rule Design

### 2.1 One Concern Per Rule

**Bad: Multiple unrelated concerns in one rule**

```yaml
- name: "catch-all-security"
  conditions:
    - any:
        - field: "processing.content_analysis.pii_detection.has_pii"
          operator: "=="
          value: true
        - field: "processing.risk_score"
          operator: ">"
          value: 7
        - field: "request.model"
          operator: "not_in"
          value: ["gpt-4", "claude-3-opus"]
  actions:
    - type: "deny"
      message: "Security violation"
```

**Good: Separate rules for separate concerns**

```yaml
- name: "block-pii"
  conditions:
    - field: "processing.content_analysis.pii_detection.has_pii"
      operator: "=="
      value: true
  actions:
    - type: "deny"
      message: "PII detected"

- name: "block-high-risk"
  conditions:
    - field: "processing.risk_score"
      operator: ">"
      value: 7
  actions:
    - type: "deny"
      message: "High risk score"

- name: "enforce-model-allowlist"
  conditions:
    - field: "request.model"
      operator: "not_in"
      value: ["gpt-4", "claude-3-opus"]
  actions:
    - type: "deny"
      message: "Model not allowed"
```

### 2.2 Descriptive Rule Names

**Good:**
- `block-high-risk-requests`
- `redact-pii-from-responses`
- `route-simple-queries-to-cheap-model`

**Bad:**
- `rule-1`
- `security-check`
- `important-rule`

### 2.3 Include Descriptions

Always add a `description` field to explain the rule's purpose:

```yaml
- name: "enforce-business-hours-premium-models"
  description: "Premium models only available 9am-5pm Mon-Fri to control costs"
  conditions:
    # ...
```

### 2.4 Use Enabled Flag for Testing

When developing new rules, set `enabled: false` to disable without deleting:

```yaml
- name: "experimental-routing"
  description: "Testing new routing logic"
  enabled: false  # Disabled during testing
  conditions:
    # ...
```

---

## 3. Condition Writing

### 3.1 Simplify Complex Conditions

**Bad: Deeply nested conditions**

```yaml
conditions:
  - all:
      - any:
          - all:
              - field: "request.model"
                operator: "=="
                value: "gpt-4"
              - field: "processing.risk_score"
                operator: ">"
                value: 5
          - field: "processing.content_analysis.pii_detection.has_pii"
            operator: "=="
            value: true
```

**Good: Split into multiple rules or simplify**

```yaml
# Rule 1: High-risk GPT-4 requests
- name: "high-risk-gpt4"
  conditions:
    - field: "request.model"
      operator: "=="
      value: "gpt-4"
    - field: "processing.risk_score"
      operator: ">"
      value: 5
  actions:
    - type: "deny"

# Rule 2: Requests with PII
- name: "pii-detected"
  conditions:
    - field: "processing.content_analysis.pii_detection.has_pii"
      operator: "=="
      value: true
  actions:
    - type: "deny"
```

### 3.2 Use Variables for Magic Numbers

**Bad: Hardcoded values**

```yaml
conditions:
  - field: "processing.token_estimate.total_tokens"
    operator: ">"
    value: 8000
```

**Good: Named variables**

```yaml
variables:
  max_tokens_per_request: 8000

rules:
  - name: "enforce-token-limit"
    conditions:
      - field: "processing.token_estimate.total_tokens"
        operator: ">"
        value: "{{ variables.max_tokens_per_request }}"
```

**Benefits:**
- Easy to update limits across multiple rules
- Self-documenting
- Consistent values

### 3.3 Validate Field Existence

Check if fields exist before comparing:

```yaml
# Check if user field exists and is not null
conditions:
  - field: "request.user"
    operator: "!="
    value: null
```

### 3.4 Use Appropriate Operators

**String operations:**

```yaml
# Check substring
- field: "request.messages[0].content"
  operator: "contains"
  value: "password"

# Regex match (case-insensitive)
- field: "request.messages[0].content"
  operator: "matches"
  value: "(?i)ignore.*instructions"
```

**Array operations:**

```yaml
# Check membership
- field: "request.model"
  operator: "in"
  value: ["gpt-4", "gpt-3.5-turbo"]
```

---

## 4. Action Selection

### 4.1 Choose Appropriate Action Types

**Blocking actions:** Stop request processing
- `deny` - Block request with error message
- `rate_limit` - Block if rate limit exceeded
- `budget` - Block if budget exceeded

**Non-blocking actions:** Log or alert but allow request
- `log` - Record event
- `alert` - Send webhook notification
- `allow` - Explicitly allow

**Modifying actions:** Change request/response
- `redact` - Remove sensitive content
- `modify` - Change field values
- `route` - Change provider/model

### 4.2 Combine Actions Thoughtfully

**Pattern: Log before deny**

```yaml
actions:
  - type: "log"
    level: "error"
    message: "Blocking high-risk request"
  - type: "deny"
    message: "Request blocked"
```

**Pattern: Alert and allow (monitoring)**

```yaml
actions:
  - type: "alert"
    webhook: "https://alerts.example.com"
    message: "Unusual activity detected"
  - type: "log"
    level: "warn"
    message: "Monitoring unusual pattern"
  - type: "allow"
```

### 4.3 Provide Actionable Error Messages

**Bad: Vague error**

```yaml
actions:
  - type: "deny"
    message: "Error"
```

**Good: Specific and actionable**

```yaml
actions:
  - type: "deny"
    message: "Request blocked: PII detected. Please remove social security numbers, credit card numbers, and email addresses before retrying."
    code: "pii_detected"
```

### 4.4 Use Error Codes for Programmatic Handling

Include error codes so clients can handle errors programmatically:

```yaml
actions:
  - type: "deny"
    message: "Daily token budget exceeded"
    code: "daily_budget_exceeded"
```

Client code can check `error.code === "daily_budget_exceeded"` and show appropriate UI.

---

## 5. Variable Usage

### 5.1 When to Use Variables

**Use variables for:**
- Values used in multiple rules
- Configuration that changes between environments
- Magic numbers that need documentation
- Lists that need to be kept in sync

**Don't use variables for:**
- Single-use values
- Values that are self-explanatory

### 5.2 Variable Naming

**Good:**
- `max_tokens_per_request`
- `allowed_models`
- `premium_tier_limit`

**Bad:**
- `value1`
- `limit`
- `config`

### 5.3 Group Related Variables

```yaml
variables:
  # Token limits
  max_tokens_per_request: 8000
  max_daily_tokens: 100000

  # Model configuration
  allowed_models: ["gpt-4", "gpt-3.5-turbo", "claude-3-sonnet"]
  premium_models: ["gpt-4", "claude-3-opus"]

  # Cost thresholds
  alert_cost_threshold: 1.00
  max_daily_cost: 100.00
```

---

## 6. Performance Optimization

### 6.1 Rule Ordering for Performance

Place most frequently matching rules first:

```yaml
# If 80% of requests are from free tier
rules:
  - name: "free-tier-limits"  # Most common
    conditions:
      - field: "context.user_attributes.tier"
        operator: "=="
        value: "free"
    actions: [...]

  - name: "premium-tier-rules"  # Less common
    conditions:
      - field: "context.user_attributes.tier"
        operator: "=="
        value: "premium"
    actions: [...]
```

### 6.2 Minimize Condition Complexity

**Slow: Complex nested conditions**

```yaml
conditions:
  - all:
      - any: [...]
      - all: [...]
      - not:
          any: [...]
```

**Fast: Simple conditions**

```yaml
conditions:
  - field: "request.model"
    operator: "=="
    value: "gpt-4"
  - field: "processing.risk_score"
    operator: ">"
    value: 5
```

### 6.3 Avoid Expensive Operations

**Expensive:**
- Regex matching with complex patterns
- Function calls on large arrays
- Nested field access on deep objects

**Use when necessary, but be aware of performance impact.**

---

## 7. Security Considerations

### 7.1 Never Hardcode Secrets

**Bad:**

```yaml
actions:
  - type: "alert"
    webhook: "https://alerts.example.com?api_key=sk_live_12345"
```

**Good:**

Configure webhooks in the application configuration, not in policies.

### 7.2 Validate User Input

Always validate untrusted data:

```yaml
- name: "validate-model-name"
  conditions:
    - field: "request.model"
      operator: "matches"
      value: "^[a-zA-Z0-9-]+$"  # Alphanumeric and hyphens only
  actions:
    - type: "allow"
```

### 7.3 Use Allowlists, Not Denylists

**Bad: Denylist (incomplete)**

```yaml
# Trying to block all dangerous models
- name: "block-banned-models"
  conditions:
    - field: "request.model"
      operator: "in"
      value: ["dangerous-model-1", "dangerous-model-2"]
  actions:
    - type: "deny"
# What if a new dangerous model is added?
```

**Good: Allowlist (complete)**

```yaml
# Only allow known-safe models
- name: "enforce-allowlist"
  conditions:
    - field: "request.model"
      operator: "not_in"
      value: ["gpt-4", "gpt-3.5-turbo", "claude-3-sonnet"]
  actions:
    - type: "deny"
      message: "Model not in allowlist"
```

### 7.4 Defense in Depth

Layer multiple security checks:

```yaml
# Layer 1: Block obvious threats
- name: "block-prompt-injection"
  conditions:
    - field: "processing.content_analysis.prompt_injection.has_prompt_injection"
      operator: "=="
      value: true
  actions:
    - type: "deny"

# Layer 2: Rate limit high-risk users
- name: "rate-limit-high-risk-users"
  conditions:
    - field: "processing.risk_score"
      operator: ">"
      value: 5
  actions:
    - type: "rate_limit"
      key: "{{ request.user }}"
      limit: 10
      window: "1h"

# Layer 3: Log all requests for audit
- name: "audit-all-requests"
  conditions: []
  actions:
    - type: "log"
      level: "info"
      message: "Request audit log"
    - type: "allow"
```

---

## 8. Testing Policies

### 8.1 Validate YAML Syntax

Use a YAML validator before deploying:

```bash
# Using yq
yq eval 'path/to/policy.yaml'

# Using Python
python -c "import yaml; yaml.safe_load(open('policy.yaml'))"
```

### 8.2 Test Against Example Requests

Create test cases for each rule:

```yaml
# Test case 1: Should be blocked
request:
  model: "gpt-4"
  messages:
    - role: "user"
      content: "My SSN is 123-45-6789"
processing:
  content_analysis:
    pii_detection:
      has_pii: true
      pii_types: ["ssn"]
expected_action: "deny"

# Test case 2: Should be allowed
request:
  model: "gpt-4"
  messages:
    - role: "user"
      content: "Hello, how are you?"
processing:
  content_analysis:
    pii_detection:
      has_pii: false
expected_action: "allow"
```

### 8.3 Gradual Rollout

When deploying new policies:

1. **Test in development** with `enabled: false`
2. **Enable in staging** with monitoring
3. **Deploy to production** during low-traffic hours
4. **Monitor logs and alerts** for unexpected behavior

### 8.4 Use Canary Rules

Test new logic alongside existing rules:

```yaml
- name: "existing-pii-rule"
  description: "Production PII blocking"
  enabled: true
  conditions:
    - field: "processing.content_analysis.pii_detection.has_pii"
      operator: "=="
      value: true
  actions:
    - type: "deny"

- name: "canary-advanced-pii-detection"
  description: "Testing new PII detection - log only"
  enabled: true
  conditions:
    - field: "processing.content_analysis.advanced_pii_detection.has_pii"
      operator: "=="
      value: true
  actions:
    - type: "log"
      level: "info"
      message: "Canary: Advanced PII detected"
    # Note: No deny action - just logging
```

---

## 9. Version Management

### 9.1 Semantic Versioning

Follow semver (MAJOR.MINOR.PATCH):

- **MAJOR**: Breaking changes (e.g., removing rules, changing behavior)
- **MINOR**: New rules added (backward compatible)
- **PATCH**: Bug fixes, description updates

**Examples:**

```yaml
# Initial version
version: "1.0.0"

# Added new rule for cost control
version: "1.1.0"

# Fixed typo in error message
version: "1.1.1"

# Changed rule behavior (breaks existing expectations)
version: "2.0.0"
```

### 9.2 Document Changes

Maintain a CHANGELOG in the policy description or separate file:

```yaml
description: |
  Production safety policy

  Changelog:
  v1.2.0 (2025-11-16): Added prompt injection detection
  v1.1.0 (2025-11-15): Added PII redaction
  v1.0.0 (2025-11-14): Initial version
```

### 9.3 Git Workflow

**Commit policies to version control:**

```bash
git add policies/production-safety-policy.yaml
git commit -m "feat(policy): add prompt injection detection (v1.2.0)"
git tag policy-v1.2.0
git push origin main --tags
```

### 9.4 Rolling Back

Keep previous versions tagged in git for rollback:

```bash
# Rollback to previous version
git checkout policy-v1.1.0 -- policies/production-safety-policy.yaml
git commit -m "chore(policy): rollback to v1.1.0 due to false positives"
```

---

## 10. Common Patterns

### 10.1 Fail-Safe Defaults

Always include a catch-all rule at the end:

```yaml
rules:
  # Specific rules here...

  # Catch-all: Default to allow (or deny for strict security)
  - name: "default-allow"
    description: "Default behavior when no other rules match"
    conditions: []
    actions:
      - type: "allow"
```

### 10.2 Tiered Access Control

```yaml
# Tier 1: Premium users - unlimited access
- name: "premium-unlimited"
  conditions:
    - field: "context.user_attributes.tier"
      operator: "=="
      value: "premium"
  actions:
    - type: "allow"

# Tier 2: Standard users - moderate limits
- name: "standard-rate-limit"
  conditions:
    - field: "context.user_attributes.tier"
      operator: "=="
      value: "standard"
  actions:
    - type: "rate_limit"
      key: "{{ request.user }}"
      limit: 100
      window: "1h"

# Tier 3: Free users - strict limits
- name: "free-rate-limit"
  conditions:
    - field: "context.user_attributes.tier"
      operator: "=="
      value: "free"
  actions:
    - type: "rate_limit"
      key: "{{ request.user }}"
      limit: 10
      window: "1h"
```

### 10.3 Progressive Enforcement

Start with logging, then escalate to blocking:

```yaml
# Phase 1: Log violations (monitoring)
- name: "detect-pii-log-only"
  enabled: true
  conditions:
    - field: "processing.content_analysis.pii_detection.has_pii"
      operator: "=="
      value: true
  actions:
    - type: "log"
      level: "warn"
      message: "PII detected (monitoring only)"
    - type: "allow"

# Phase 2: Alert + log (after monitoring period)
# - name: "detect-pii-with-alerts"
#   enabled: false  # Enable in next phase
#   conditions:
#     - field: "processing.content_analysis.pii_detection.has_pii"
#       operator: "=="
#       value: true
#   actions:
#     - type: "alert"
#       webhook: "https://security.example.com/pii"
#     - type: "log"
#       level: "error"
#       message: "PII detected (alerting)"
#     - type: "allow"

# Phase 3: Block (final enforcement)
# - name: "block-pii"
#   enabled: false  # Enable in final phase
#   conditions:
#     - field: "processing.content_analysis.pii_detection.has_pii"
#       operator: "=="
#       value: true
#   actions:
#     - type: "deny"
#       message: "PII detected"
```

### 10.4 Cost-Based Routing

```yaml
- name: "route-by-complexity"
  conditions:
    - field: "processing.complexity_score"
      operator: "<"
      value: 5
  actions:
    - type: "route"
      model: "gpt-3.5-turbo"
      reason: "Simple query, using cheaper model"
```

### 10.5 Compliance Triple-Check

```yaml
- name: "compliance-checks"
  description: "Ensure compliance with regulations"
  conditions: []
  actions:
    # 1. Log for audit trail
    - type: "log"
      level: "info"
      message: "Compliance audit: user={{ request.user }}"

    # 2. Check data residency
    - type: "route"
      provider: "openai-eu"  # GDPR compliance
      reason: "Data residency requirement"

    # 3. Alert on sensitive requests
    - type: "alert"
      webhook: "https://compliance.example.com/log"
      message: "Compliance event logged"
```

---

## Summary

**Key takeaways:**

1. **Organize policies** by concern and team ownership
2. **Write simple rules** with one concern per rule
3. **Use variables** for reusable values
4. **Order rules** by specificity or frequency
5. **Choose appropriate actions** (blocking vs non-blocking)
6. **Test thoroughly** before production deployment
7. **Version policies** using semantic versioning
8. **Layer security** with defense in depth
9. **Monitor and iterate** based on real-world usage
10. **Document everything** for maintainability

---

**End of Best Practices Guide**
