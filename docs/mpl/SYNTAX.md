# MPL Syntax Quick Reference

**Version:** 1.0.0

---

## Policy Structure

```yaml
mpl_version: "1.0"              # Required: MPL schema version
name: "policy-name"             # Required: Policy identifier (kebab-case)
version: "1.0.0"                # Required: Semantic version
description: "..."              # Optional: Human-readable description
author: "team@example.com"      # Optional: Author/owner
created: "2025-11-16"           # Optional: Creation date (ISO 8601)
updated: "2025-11-16"           # Optional: Last update (ISO 8601)
tags: ["tag1", "tag2"]          # Optional: Tags for organization

variables:                      # Optional: Reusable variables
  variable_name: value

rules:                          # Required: Array of rules
  - name: "rule-name"           # Required: Rule identifier
    description: "..."          # Optional: Rule description
    enabled: true               # Optional: Enable/disable (default: true)
    conditions: [...]           # Required: Condition array
    actions: [...]              # Required: Action array
```

---

## Conditions

### Simple Condition

```yaml
conditions:
  - field: "field.path"         # Field to evaluate
    operator: "=="              # Comparison operator
    value: "expected-value"     # Value to compare
```

### Function Condition

```yaml
conditions:
  - function: "function_name"   # Function to call
    args: ["arg1", "arg2"]      # Function arguments
    operator: "=="              # Comparison operator (optional)
    value: true                 # Value to compare (optional)
```

### Logical Operators

```yaml
# Implicit AND (all must be true)
conditions:
  - field: "field1"
    operator: "=="
    value: "value1"
  - field: "field2"
    operator: ">"
    value: 5

# Explicit AND
conditions:
  - all:
      - field: "field1"
        operator: "=="
        value: "value1"
      - field: "field2"
        operator: ">"
        value: 5

# OR (any can be true)
conditions:
  - any:
      - field: "field1"
        operator: "=="
        value: "value1"
      - field: "field2"
        operator: "=="
        value: "value2"

# NOT (negation)
conditions:
  - not:
      field: "field1"
      operator: "=="
      value: "value1"

# Nested conditions
conditions:
  - all:
      - field: "field1"
        operator: "=="
        value: "value1"
      - any:
          - field: "field2"
            operator: ">"
            value: 5
          - field: "field3"
            operator: "=="
            value: true
```

---

## Comparison Operators

| Operator | Description | Types |
|----------|-------------|-------|
| `==` | Equal to | All |
| `!=` | Not equal to | All |
| `<` | Less than | number |
| `>` | Greater than | number |
| `<=` | Less than or equal | number |
| `>=` | Greater than or equal | number |
| `contains` | String contains substring | string |
| `matches` | String matches regex | string |
| `starts_with` | String starts with | string |
| `ends_with` | String ends with | string |
| `in` | Value in array | All (value), array (target) |
| `not_in` | Value not in array | All (value), array (target) |

---

## Field Access

```yaml
# Simple field
field: "request.model"

# Nested field
field: "processing.content_analysis.pii_detection.has_pii"

# Array indexing
field: "request.messages[0].content"

# Array wildcard (for functions)
field: "request.messages[*].role"
```

---

## Actions

### Allow

```yaml
- type: "allow"
```

### Deny

```yaml
- type: "deny"
  message: "Error message"      # Required
  code: "error_code"            # Optional
```

### Log

```yaml
- type: "log"
  level: "info"                 # Required: debug, info, warn, error
  message: "Log message"        # Required (supports {{ template }})
```

### Redact

```yaml
- type: "redact"
  fields: ["field.path"]        # Required: Array of fields
  method: "mask"                # Required: mask, remove, replace
  replacement: "[REDACTED]"     # Optional: For "replace" method
```

### Modify

```yaml
- type: "modify"
  field: "field.path"           # Required
  value: "new-value"            # Required
```

### Route

```yaml
- type: "route"
  provider: "provider-name"     # Optional
  model: "model-name"           # Optional
  reason: "Routing reason"      # Optional
```

### Alert

```yaml
- type: "alert"
  webhook: "https://..."        # Optional
  message: "Alert message"      # Required
  severity: "high"              # Optional: low, medium, high, critical
```

### Rate Limit

```yaml
- type: "rate_limit"
  key: "{{ request.user }}"     # Required: Rate limit key
  limit: 100                    # Required: Request limit
  window: "1h"                  # Required: Time window
```

### Budget

```yaml
- type: "budget"
  key: "{{ request.user }}"     # Required: Budget key
  limit: 1000000                # Required: Budget limit
  window: "1d"                  # Required: Time window
  type: "tokens"                # Required: tokens or cost
```

---

## Built-in Functions

### String Functions

```yaml
# Length
- function: "len"
  args: ["field.path"]
  operator: ">"
  value: 10

# Lowercase
- function: "lower"
  args: ["field.path"]
  operator: "=="
  value: "lowercase-string"

# Uppercase
- function: "upper"
  args: ["field.path"]
  operator: "=="
  value: "UPPERCASE-STRING"
```

### Content Analysis Functions

```yaml
# PII detection
- function: "has_pii"
  args: ["request.messages[0].content"]
  operator: "=="
  value: true

# Prompt injection detection
- function: "has_injection"
  args: ["request.messages[0].content"]
  operator: "=="
  value: true

# Sensitive content detection
- function: "has_sensitive"
  args: ["request.messages[0].content"]
  operator: "=="
  value: true
```

### Array Functions

```yaml
# Contains
- function: "contains"
  args: ["{{ variables.allowed_models }}", "request.model"]
  operator: "=="
  value: true
```

---

## Variables

### Definition

```yaml
variables:
  # Scalar
  max_tokens: 4000
  api_version: "v1"

  # Array
  allowed_models:
    - "gpt-4"
    - "gpt-3.5-turbo"
    - "claude-3-sonnet"

  # Object
  tier_limits:
    free: 10
    premium: 100
    enterprise: 1000
```

### Reference

```yaml
# In conditions
conditions:
  - field: "processing.token_estimate.total_tokens"
    operator: ">"
    value: "{{ variables.max_tokens }}"

# In actions
actions:
  - type: "deny"
    message: "Exceeds limit of {{ variables.max_tokens }}"

# In array checks
conditions:
  - field: "request.model"
    operator: "in"
    value: "{{ variables.allowed_models }}"
```

---

## Data Model

### Request Fields

```yaml
request.model                          # string
request.messages                       # array
request.messages[N].role               # string
request.messages[N].content            # string
request.temperature                    # number
request.max_tokens                     # number
request.top_p                          # number
request.stream                         # boolean
request.user                           # string
request.tools                          # array
```

### Response Fields

```yaml
response.content                       # string
response.finish_reason                 # string
response.usage.prompt_tokens           # number
response.usage.completion_tokens       # number
response.usage.total_tokens            # number
```

### Processing Fields

```yaml
# Token and cost estimates
processing.token_estimate.total_tokens              # number
processing.token_estimate.prompt_tokens             # number
processing.cost_estimate.total_cost                 # number

# Content analysis
processing.content_analysis.pii_detection.has_pii                    # boolean
processing.content_analysis.pii_detection.pii_types                  # array
processing.content_analysis.sensitive_content.has_sensitive_content  # boolean
processing.content_analysis.sensitive_content.severity               # string
processing.content_analysis.prompt_injection.has_prompt_injection    # boolean
processing.content_analysis.prompt_injection.confidence              # number

# Risk and complexity
processing.risk_score                              # number (1-10)
processing.complexity_score                        # number (1-10)

# Conversation context
processing.conversation_context.turn_count         # number
processing.conversation_context.context_window_percent  # number
```

### Context Fields

```yaml
# Time context
context.time.hour                      # number (0-23)
context.time.day_of_week               # string (Monday, Tuesday, ...)
context.time.date                      # string (YYYY-MM-DD)

# Environment
context.environment                    # string (production, staging, development)

# User attributes
context.user_attributes.user_id        # string
context.user_attributes.tier           # string (free, premium, enterprise)
context.user_attributes.department     # string
context.user_attributes.requests_this_hour  # number
context.user_attributes.daily_token_usage   # number
context.user_attributes.daily_cost          # number
```

---

## Type System

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | `"gpt-4"` |
| `number` | Integer or float | `42`, `3.14` |
| `boolean` | True or false | `true`, `false` |
| `array` | Ordered list | `["a", "b", "c"]` |
| `object` | Key-value map | `{key: value}` |
| `null` | Absence of value | `null` |

**Type checking:** All comparisons are type-checked. Incompatible types result in errors.

---

## Operator Precedence

1. Field access, function calls (highest)
2. Comparison operators (`==`, `!=`, `<`, `>`, etc.)
3. `not`
4. `all` (AND)
5. `any` (OR) (lowest)

---

## Complete Example

```yaml
mpl_version: "1.0"
name: "example-policy"
version: "1.0.0"
description: "Example policy demonstrating syntax"
author: "team@example.com"
tags: ["example"]

variables:
  max_tokens: 4000
  allowed_models: ["gpt-4", "gpt-3.5-turbo"]

rules:
  - name: "enforce-token-limit"
    description: "Block requests exceeding token limit"
    enabled: true
    conditions:
      - field: "processing.token_estimate.total_tokens"
        operator: ">"
        value: "{{ variables.max_tokens }}"
    actions:
      - type: "log"
        level: "warn"
        message: "Token limit exceeded: {{ processing.token_estimate.total_tokens }}"
      - type: "deny"
        message: "Request exceeds token limit"
        code: "token_limit_exceeded"

  - name: "model-allowlist"
    description: "Only allow approved models"
    enabled: true
    conditions:
      - field: "request.model"
        operator: "not_in"
        value: "{{ variables.allowed_models }}"
    actions:
      - type: "deny"
        message: "Model not in allowlist"
        code: "model_not_allowed"

  - name: "default-allow"
    description: "Default behavior"
    conditions: []
    actions:
      - type: "allow"
```

---

## Common Patterns

### Match All

```yaml
conditions: []  # Empty array matches all requests
```

### Multiple Conditions (AND)

```yaml
conditions:
  - field: "field1"
    operator: "=="
    value: "value1"
  - field: "field2"
    operator: ">"
    value: 5
# Both must be true
```

### Multiple Conditions (OR)

```yaml
conditions:
  - any:
      - field: "field1"
        operator: "=="
        value: "value1"
      - field: "field2"
        operator: "=="
        value: "value2"
# At least one must be true
```

### Range Check

```yaml
conditions:
  - field: "processing.risk_score"
    operator: ">="
    value: 5
  - field: "processing.risk_score"
    operator: "<="
    value: 7
# Between 5 and 7 (inclusive)
```

### Null Check

```yaml
# Check if field exists
conditions:
  - field: "request.user"
    operator: "!="
    value: null

# Check if field is null
conditions:
  - field: "request.user"
    operator: "=="
    value: null
```

### Template Variables in Messages

```yaml
actions:
  - type: "log"
    message: "User {{ request.user }} requested model {{ request.model }}"
  - type: "deny"
    message: "Token limit {{ variables.max_tokens }} exceeded"
```

---

**End of Syntax Reference**
