# Mercator Policy Language (MPL) Specification v1.0

**Version:** 1.0.0
**Status:** Draft
**Last Updated:** 2025-11-16
**Authors:** Mercator Jupiter Team

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Language Overview](#2-language-overview)
3. [Policy Structure](#3-policy-structure)
4. [Rule Structure](#4-rule-structure)
5. [Condition Language](#5-condition-language)
6. [Type System](#6-type-system)
7. [Actions Reference](#7-actions-reference)
8. [Data Model](#8-data-model)
9. [Built-in Functions](#9-built-in-functions)
10. [Variables](#10-variables)
11. [Evaluation Semantics](#11-evaluation-semantics)
12. [Versioning](#12-versioning)
13. [Error Handling](#13-error-handling)
14. [Examples](#14-examples)

---

## 1. Introduction

### 1.1 Purpose

The Mercator Policy Language (MPL) is a declarative, YAML-based policy language designed for LLM governance. MPL enables security teams, compliance officers, and platform engineers to define rules that control LLM request and response behavior without writing code.

### 1.2 Design Goals

- **Declarative**: Express intent, not implementation
- **Readable**: Non-programmers can understand and write policies
- **Type-Safe**: Strong typing prevents ambiguous expressions
- **Composable**: Policies can reference variables and be organized modularly
- **Git-Friendly**: YAML format works well with version control
- **Extensible**: Easy to add new conditions and actions without breaking existing policies
- **Testable**: Policies can be validated independently of execution

### 1.3 Target Audience

- Security engineers implementing LLM security policies
- Compliance officers enforcing regulatory requirements
- Platform engineers managing LLM infrastructure
- DevOps teams automating LLM governance

### 1.4 Use Cases

MPL supports policy enforcement for:

- **Safety**: PII detection, prompt injection prevention, content filtering
- **Compliance**: Audit logging, data residency, regulatory compliance
- **Cost Control**: Token limits, budget enforcement, cost-based routing
- **Performance**: Model routing, load balancing, caching
- **Security**: API key validation, rate limiting, user-based access control

---

## 2. Language Overview

### 2.1 Core Concepts

**Policy**: A named collection of rules, variables, and metadata that defines governance behavior.

**Rule**: A single conditional statement that specifies when to take specific actions.

**Condition**: A boolean expression that evaluates request/response data.

**Action**: An operation to perform when a rule's conditions are met.

**Variable**: A reusable value that can be referenced in conditions and actions.

### 2.2 Design Principles

1. **Explicit over Implicit**: All behavior must be explicitly defined
2. **Fail-Safe Defaults**: Default behavior is to allow unless explicitly denied
3. **First-Match Wins**: Rules are evaluated in order; first matching rule determines action
4. **Immutable Policies**: Policies cannot modify themselves during evaluation
5. **No Side Effects**: Condition evaluation has no side effects

### 2.3 Language Format

MPL policies are written in YAML and follow a strict schema. All policies must be valid YAML and conform to the MPL schema.

**Example minimal policy:**

```yaml
mpl_version: "1.0"
name: "example-policy"
version: "1.0.0"

rules:
  - name: "block-high-risk"
    conditions:
      - field: "processing.risk_score"
        operator: ">"
        value: 7
    actions:
      - type: "deny"
        message: "Request blocked: risk score too high"
```

---

## 3. Policy Structure

### 3.1 Top-Level Policy Format

A policy document consists of metadata, optional variables, and a list of rules.

**Schema:**

```yaml
mpl_version: string          # Required: MPL schema version (currently "1.0")
name: string                 # Required: Policy name (lowercase, hyphens)
version: string              # Required: Semantic version (e.g., "1.0.0")
description: string          # Optional: Human-readable description
author: string               # Optional: Policy author/owner
created: string              # Optional: Creation date (ISO 8601)
updated: string              # Optional: Last update date (ISO 8601)
tags: array<string>          # Optional: Tags for organization
variables: object            # Optional: Variable definitions
rules: array<Rule>           # Required: Array of policy rules
```

### 3.2 Metadata Fields

**mpl_version** (required, string)
- Specifies the MPL schema version
- Must be "1.0" for this specification
- Used for version compatibility checking
- Example: `mpl_version: "1.0"`

**name** (required, string)
- Unique identifier for the policy
- Must be lowercase with hyphens (kebab-case)
- Should be descriptive and concise
- Example: `name: "production-safety-policy"`

**version** (required, string)
- Semantic version of the policy (MAJOR.MINOR.PATCH)
- Must follow semver format
- Increment MAJOR for breaking changes
- Increment MINOR for new rules
- Increment PATCH for bug fixes
- Example: `version: "1.2.3"`

**description** (optional, string)
- Human-readable description of policy purpose
- Should explain what the policy does and why
- Example: `description: "Safety and compliance policy for production LLM usage"`

**author** (optional, string)
- Policy author or team identifier
- Can be email, team name, or username
- Example: `author: "security-team@example.com"`

**created** (optional, string)
- ISO 8601 date when policy was created
- Example: `created: "2025-11-16"`

**updated** (optional, string)
- ISO 8601 date when policy was last updated
- Example: `updated: "2025-11-16"`

**tags** (optional, array of strings)
- Tags for categorization and search
- Example: `tags: ["safety", "compliance", "production"]`

### 3.3 Complete Policy Example

```yaml
mpl_version: "1.0"
name: "production-safety-policy"
version: "1.0.0"
description: "Safety and compliance policy for production LLM usage"
author: "security-team@example.com"
created: "2025-11-16"
updated: "2025-11-16"
tags: ["safety", "compliance", "production"]

variables:
  max_tokens: 4000
  allowed_models: ["gpt-4", "gpt-3.5-turbo", "claude-3-sonnet"]

rules:
  - name: "block-high-risk-requests"
    description: "Block requests with high risk scores"
    enabled: true
    conditions:
      - field: "processing.risk_score"
        operator: ">"
        value: 7
    actions:
      - type: "deny"
        message: "Request blocked due to high risk score"
```

---

## 4. Rule Structure

### 4.1 Rule Format

Each rule consists of metadata, conditions, and actions.

**Schema:**

```yaml
name: string                 # Required: Rule name
description: string          # Optional: Rule description
enabled: boolean             # Optional: Enable/disable rule (default: true)
conditions: array<Condition> # Required: Array of conditions (implicit AND)
actions: array<Action>       # Required: Actions to take when conditions match
```

### 4.2 Rule Fields

**name** (required, string)
- Unique identifier for the rule within the policy
- Should be descriptive and kebab-case
- Example: `name: "block-high-risk-requests"`

**description** (optional, string)
- Human-readable explanation of what the rule does
- Example: `description: "Block requests with high risk scores"`

**enabled** (optional, boolean)
- Whether the rule is active
- Default: `true`
- Allows temporarily disabling rules without deletion
- Example: `enabled: false`

**conditions** (required, array of Condition)
- Array of condition expressions
- All conditions must evaluate to true for the rule to match (implicit AND)
- Empty array `[]` matches all requests
- See [Section 5: Condition Language](#5-condition-language)

**actions** (required, array of Action)
- Array of actions to execute when rule matches
- Actions are executed in order
- At least one action required
- See [Section 7: Actions Reference](#7-actions-reference)

### 4.3 Rule Evaluation

Rules are evaluated in the order they appear in the policy. The first rule whose conditions all evaluate to true is considered the matching rule, and its actions are executed.

**Evaluation logic:**

1. Start with first rule
2. Evaluate all conditions (implicit AND)
3. If all conditions are true, execute actions and stop
4. If any condition is false, move to next rule
5. If no rules match, default action is allow

**Example:**

```yaml
rules:
  # Rule 1: Evaluated first
  - name: "premium-users-unlimited"
    conditions:
      - field: "context.user_attributes.tier"
        operator: "=="
        value: "premium"
    actions:
      - type: "allow"

  # Rule 2: Only evaluated if Rule 1 doesn't match
  - name: "free-users-rate-limit"
    conditions:
      - field: "context.user_attributes.tier"
        operator: "=="
        value: "free"
    actions:
      - type: "deny"
        message: "Rate limit exceeded"
```

---

## 5. Condition Language

### 5.1 Condition Structure

Conditions are boolean expressions that evaluate request/response data.

**Simple Condition Schema:**

```yaml
field: string                # Field path (dot notation)
operator: string             # Comparison operator
value: any                   # Value to compare against
```

**Function Condition Schema:**

```yaml
function: string             # Function name
args: array<any>             # Function arguments
operator: string             # Comparison operator (optional)
value: any                   # Value to compare against (optional)
```

### 5.2 Field Access

Fields are accessed using dot notation with support for array indexing.

**Syntax:**

```
field_path := segment ("." segment)*
segment := identifier | identifier "[" number "]"
```

**Examples:**

```yaml
# Simple field access
field: "request.model"

# Nested field access
field: "processing.content_analysis.pii_detection.has_pii"

# Array indexing
field: "request.messages[0].content"

# Nested array
field: "request.messages[0].content"
```

### 5.3 Comparison Operators

**Equality Operators:**

- `==` - Equal to
- `!=` - Not equal to

**Relational Operators:**

- `<` - Less than
- `>` - Greater than
- `<=` - Less than or equal to
- `>=` - Greater than or equal to

**String Operators:**

- `contains` - String contains substring (case-sensitive)
- `matches` - String matches regex pattern
- `starts_with` - String starts with prefix
- `ends_with` - String ends with suffix

**Array Operators:**

- `in` - Value is in array
- `not_in` - Value is not in array

**Examples:**

```yaml
# Equality
- field: "request.model"
  operator: "=="
  value: "gpt-4"

# Relational
- field: "processing.risk_score"
  operator: ">"
  value: 7

# String contains
- field: "request.messages[0].content"
  operator: "contains"
  value: "password"

# Regex match
- field: "request.messages[0].content"
  operator: "matches"
  value: "(?i)ignore.*instructions"

# Array membership
- field: "request.model"
  operator: "in"
  value: ["gpt-4", "gpt-3.5-turbo", "claude-3-sonnet"]
```

### 5.4 Logical Operators

**Implicit AND** (default):
Multiple conditions in the same rule are ANDed together.

```yaml
conditions:
  - field: "request.model"
    operator: "=="
    value: "gpt-4"
  - field: "processing.token_estimate.total_tokens"
    operator: "<"
    value: 4000
# Both conditions must be true
```

**Explicit OR** (any):
Use `any` to specify that at least one condition must be true.

```yaml
conditions:
  - any:
      - field: "request.model"
        operator: "=="
        value: "gpt-4"
      - field: "request.model"
        operator: "=="
        value: "gpt-3.5-turbo"
# At least one condition must be true
```

**Explicit AND** (all):
Use `all` for clarity (same as implicit AND).

```yaml
conditions:
  - all:
      - field: "request.model"
        operator: "=="
        value: "gpt-4"
      - field: "processing.token_estimate.total_tokens"
        operator: "<"
        value: 4000
# All conditions must be true
```

**NOT** (negation):
Use `not` to negate a condition.

```yaml
conditions:
  - not:
      field: "request.model"
      operator: "=="
      value: "gpt-4"
# Condition must be false
```

**Nested Conditions:**

```yaml
conditions:
  - all:
      - field: "request.model"
        operator: "=="
        value: "gpt-4"
      - any:
          - field: "processing.risk_score"
            operator: ">"
            value: 5
          - field: "processing.content_analysis.pii_detection.has_pii"
            operator: "=="
            value: true
# Model must be gpt-4 AND (risk > 5 OR has PII)
```

### 5.5 Operator Precedence

1. Field access and function calls (highest)
2. Comparison operators (==, !=, <, >, <=, >=, contains, matches, in, not_in)
3. `not`
4. `all` (AND)
5. `any` (OR) (lowest)

---

## 6. Type System

### 6.1 Supported Types

MPL supports the following types:

- `string` - Text values
- `number` - Integer or floating-point numbers
- `boolean` - `true` or `false`
- `array` - Ordered list of values
- `object` - Key-value map (for nested field access only)
- `null` - Absence of value

### 6.2 Type Checking

All comparisons are type-checked at parse time or runtime:

- Equality operators (`==`, `!=`) work with all types
- Relational operators (`<`, `>`, `<=`, `>=`) only work with numbers
- String operators (`contains`, `matches`, `starts_with`, `ends_with`) only work with strings
- Array operators (`in`, `not_in`) require array values

**Type Errors:**

```yaml
# ERROR: Cannot use < with strings
- field: "request.model"  # string
  operator: "<"
  value: "gpt-4"

# ERROR: Cannot use contains with numbers
- field: "processing.risk_score"  # number
  operator: "contains"
  value: 7
```

### 6.3 Type Coercion

MPL does NOT perform automatic type coercion. All comparisons must use compatible types.

**Examples:**

```yaml
# OK: Same types
- field: "processing.risk_score"  # number
  operator: ">"
  value: 7  # number

# ERROR: Incompatible types
- field: "processing.risk_score"  # number
  operator: ">"
  value: "7"  # string
```

### 6.4 Null Handling

- Comparing `null` with `==` or `!=` is allowed
- All other operators return false when comparing with `null`
- Missing fields evaluate to `null`

**Examples:**

```yaml
# OK: Check if field exists
- field: "request.user"
  operator: "!="
  value: null

# Returns false if field is null
- field: "request.user"
  operator: "=="
  value: "john@example.com"
```

---

## 7. Actions Reference

### 7.1 Action Structure

Actions are operations to perform when a rule's conditions match.

**Schema:**

```yaml
type: string                 # Required: Action type
# Additional fields depend on action type
```

### 7.2 Allow Action

Explicitly allow the request to proceed.

**Schema:**

```yaml
type: "allow"
```

**Example:**

```yaml
actions:
  - type: "allow"
```

**Behavior:**
- Request proceeds to LLM provider
- No further rules are evaluated
- This is the default action if no rules match

### 7.3 Deny Action

Block the request and return an error to the client.

**Schema:**

```yaml
type: "deny"
message: string              # Required: Error message
code: string                 # Optional: Error code
```

**Example:**

```yaml
actions:
  - type: "deny"
    message: "Request blocked due to high risk score"
    code: "risk_too_high"
```

**Behavior:**
- Request is blocked
- Error message is returned to client
- HTTP 403 Forbidden status code
- Optional error code for client handling

### 7.4 Log Action

Log an event for audit or debugging purposes.

**Schema:**

```yaml
type: "log"
level: string                # Required: Log level (debug, info, warn, error)
message: string              # Required: Log message
```

**Example:**

```yaml
actions:
  - type: "log"
    level: "warn"
    message: "PII detected in request from user {{ request.user }}"
```

**Behavior:**
- Event is logged to configured logging backend
- Request continues processing (non-blocking)
- Supports template variables in message (e.g., `{{ field.path }}`)

### 7.5 Redact Action

Remove or mask sensitive content from request or response.

**Schema:**

```yaml
type: "redact"
fields: array<string>        # Required: Fields to redact
method: string               # Required: Redaction method (mask, remove, replace)
replacement: string          # Optional: Replacement value (for "replace" method)
```

**Example:**

```yaml
actions:
  - type: "redact"
    fields: ["request.messages[0].content"]
    method: "mask"
    replacement: "[REDACTED]"
```

**Behavior:**
- Specified fields are redacted according to method
- `mask`: Replace with `***` or custom replacement
- `remove`: Remove field entirely
- `replace`: Replace with specified replacement value

### 7.6 Modify Action

Modify request or response fields.

**Schema:**

```yaml
type: "modify"
field: string                # Required: Field to modify
value: any                   # Required: New value
```

**Example:**

```yaml
actions:
  - type: "modify"
    field: "request.temperature"
    value: 0.7
```

**Behavior:**
- Specified field is set to new value
- Request continues processing with modified value

### 7.7 Route Action

Route request to specific provider or model.

**Schema:**

```yaml
type: "route"
provider: string             # Optional: Provider name
model: string                # Optional: Model name
reason: string               # Optional: Routing reason (for logging)
```

**Example:**

```yaml
actions:
  - type: "route"
    provider: "openai"
    model: "gpt-3.5-turbo"
    reason: "Cost optimization for simple queries"
```

**Behavior:**
- Request is routed to specified provider/model
- Overrides original request model
- Routing decision is logged

### 7.8 Alert Action

Trigger external alert via webhook or other mechanism.

**Schema:**

```yaml
type: "alert"
webhook: string              # Optional: Webhook URL
message: string              # Required: Alert message
severity: string             # Optional: Alert severity (low, medium, high, critical)
```

**Example:**

```yaml
actions:
  - type: "alert"
    webhook: "https://alerts.example.com/high-cost"
    message: "High cost request: ${{ processing.cost_estimate.total_cost }}"
    severity: "high"
```

**Behavior:**
- Alert is sent to configured webhook
- Non-blocking (request continues processing)
- Supports template variables in message

### 7.9 Rate Limit Action

Apply rate limiting based on user, API key, or other attributes.

**Schema:**

```yaml
type: "rate_limit"
key: string                  # Required: Rate limit key (e.g., "{{ request.user }}")
limit: number                # Required: Request limit
window: string               # Required: Time window (e.g., "1h", "1d")
```

**Example:**

```yaml
actions:
  - type: "rate_limit"
    key: "{{ request.user }}"
    limit: 100
    window: "1h"
```

**Behavior:**
- Request is denied if rate limit exceeded
- Rate limit is tracked by specified key
- Window is rolling time window

### 7.10 Budget Action

Enforce budget constraints (token or cost limits).

**Schema:**

```yaml
type: "budget"
key: string                  # Required: Budget key (e.g., "{{ request.user }}")
limit: number                # Required: Budget limit (tokens or USD)
window: string               # Required: Time window (e.g., "1d", "1m")
type: string                 # Required: Budget type (tokens, cost)
```

**Example:**

```yaml
actions:
  - type: "budget"
    key: "{{ request.user }}"
    limit: 1000000
    window: "1d"
    type: "tokens"
```

**Behavior:**
- Request is denied if budget exceeded
- Budget is tracked by specified key
- Window is rolling time window

---

## 8. Data Model

### 8.1 Available Fields

All fields accessible in condition expressions.

### 8.2 Request Fields

Fields from the original LLM request:

```yaml
request.model: string                           # Model name (e.g., "gpt-4")
request.messages: array                         # Array of messages
request.messages[N].role: string                # Message role (user, assistant, system)
request.messages[N].content: string             # Message content
request.temperature: number                     # Temperature (0.0-2.0)
request.max_tokens: number                      # Max completion tokens
request.top_p: number                           # Top-p sampling
request.frequency_penalty: number               # Frequency penalty
request.presence_penalty: number                # Presence penalty
request.stop: array<string>                     # Stop sequences
request.stream: boolean                         # Streaming enabled
request.user: string                            # User ID
request.tools: array                            # Tool definitions (for function calling)
```

### 8.3 Response Fields

Fields from the LLM response (available in response-phase policies):

```yaml
response.content: string                        # Response text
response.finish_reason: string                  # stop, length, content_filter, tool_calls
response.usage.prompt_tokens: number            # Actual prompt tokens
response.usage.completion_tokens: number        # Actual completion tokens
response.usage.total_tokens: number             # Total tokens
response.model: string                          # Actual model used
```

### 8.4 Processing Fields

Fields from request/response processing (Feature 4):

```yaml
# Token and Cost Estimates
processing.token_estimate.total_tokens: number  # Estimated total tokens
processing.token_estimate.prompt_tokens: number # Estimated prompt tokens
processing.token_estimate.completion_tokens: number # Estimated completion tokens
processing.cost_estimate.total_cost: number     # Estimated cost (USD)
processing.cost_estimate.prompt_cost: number    # Estimated prompt cost
processing.cost_estimate.completion_cost: number # Estimated completion cost

# Content Analysis
processing.content_analysis.pii_detection.has_pii: boolean
processing.content_analysis.pii_detection.pii_types: array<string>
processing.content_analysis.pii_detection.confidence: number

processing.content_analysis.sensitive_content.has_sensitive_content: boolean
processing.content_analysis.sensitive_content.categories: array<string>
processing.content_analysis.sensitive_content.severity: string  # low, medium, high, critical

processing.content_analysis.prompt_injection.has_prompt_injection: boolean
processing.content_analysis.prompt_injection.confidence: number
processing.content_analysis.prompt_injection.type: string  # jailbreak, instruction_override, etc.

processing.content_analysis.sentiment.score: number  # -1.0 to 1.0
processing.content_analysis.sentiment.label: string  # negative, neutral, positive

# Risk Scoring
processing.risk_score: number                   # Overall risk score (1-10)
processing.complexity_score: number             # Query complexity (1-10)

# Conversation Context
processing.conversation_context.turn_count: number
processing.conversation_context.context_window_usage: number
processing.conversation_context.context_window_percent: number
```

### 8.5 Context Fields

Runtime context fields (time, user attributes, environment):

```yaml
# Time Context
context.time.hour: number                       # Hour (0-23)
context.time.minute: number                     # Minute (0-59)
context.time.day_of_week: string                # Monday, Tuesday, etc.
context.time.day: number                        # Day of month (1-31)
context.time.month: number                      # Month (1-12)
context.time.year: number                       # Year (e.g., 2025)
context.time.date: string                       # ISO 8601 date (YYYY-MM-DD)
context.time.timestamp: number                  # Unix timestamp

# Environment Context
context.environment: string                     # production, staging, development

# User Attributes (from external systems)
context.user_attributes.user_id: string         # User ID
context.user_attributes.tier: string            # free, premium, enterprise
context.user_attributes.department: string      # User department
context.user_attributes.roles: array<string>    # User roles
context.user_attributes.requests_this_hour: number
context.user_attributes.requests_today: number
context.user_attributes.daily_token_usage: number
context.user_attributes.monthly_token_usage: number
context.user_attributes.daily_cost: number
context.user_attributes.monthly_cost: number
```

---

## 9. Built-in Functions

### 9.1 String Functions

**len(field)** - Length of string or array

```yaml
conditions:
  - function: "len"
    args: ["request.messages"]
    operator: ">"
    value: 10
```

**lower(field)** - Convert string to lowercase

```yaml
conditions:
  - function: "lower"
    args: ["request.model"]
    operator: "=="
    value: "gpt-4"
```

**upper(field)** - Convert string to uppercase

```yaml
conditions:
  - function: "upper"
    args: ["request.model"]
    operator: "=="
    value: "GPT-4"
```

### 9.2 Content Analysis Functions

**has_pii(field)** - Check if field contains PII

```yaml
conditions:
  - function: "has_pii"
    args: ["request.messages[0].content"]
    operator: "=="
    value: true
```

**has_injection(field)** - Check if field contains prompt injection

```yaml
conditions:
  - function: "has_injection"
    args: ["request.messages[0].content"]
    operator: "=="
    value: true
```

**has_sensitive(field)** - Check if field contains sensitive content

```yaml
conditions:
  - function: "has_sensitive"
    args: ["request.messages[0].content"]
    operator: "=="
    value: true
```

### 9.3 Array Functions

**contains(array, value)** - Check if array contains value

```yaml
conditions:
  - function: "contains"
    args: ["{{ variables.allowed_models }}", "request.model"]
    operator: "=="
    value: true
```

---

## 10. Variables

### 10.1 Variable Definition

Variables are reusable values defined in the `variables` section of a policy.

**Schema:**

```yaml
variables:
  variable_name: value       # Scalar value
  another_var: value
```

**Supported Types:**

- Scalar values: strings, numbers, booleans
- Arrays: `["value1", "value2"]`
- Objects: `{ key: value }`

### 10.2 Variable References

Variables are referenced using template syntax: `{{ variables.variable_name }}`

**Example:**

```yaml
variables:
  max_tokens: 4000
  allowed_models: ["gpt-4", "gpt-3.5-turbo"]

rules:
  - name: "enforce-token-limit"
    conditions:
      - field: "processing.token_estimate.total_tokens"
        operator: ">"
        value: "{{ variables.max_tokens }}"
    actions:
      - type: "deny"
        message: "Request exceeds token limit"
```

---

## 11. Evaluation Semantics

### 11.1 Rule Evaluation Order

Rules are evaluated sequentially from top to bottom. The first rule whose conditions all evaluate to true is the matching rule.

**Algorithm:**

```
for each rule in policy.rules:
  if rule.enabled == false:
    continue

  if all conditions in rule.conditions are true:
    execute rule.actions
    stop evaluation

if no rules matched:
  default action is "allow"
```

### 11.2 Condition Evaluation

Multiple conditions in a rule are combined with implicit AND (all must be true).

**Examples:**

```yaml
# Implicit AND
conditions:
  - field: "request.model"
    operator: "=="
    value: "gpt-4"
  - field: "processing.risk_score"
    operator: ">"
    value: 5
# Both must be true
```

### 11.3 Short-Circuit Evaluation

Condition evaluation uses short-circuit logic:

- For AND: If any condition is false, remaining conditions are not evaluated
- For OR: If any condition is true, remaining conditions are not evaluated

### 11.4 Action Execution

When a rule matches, all actions are executed in order:

1. Actions are executed sequentially
2. If an action fails, subsequent actions may or may not execute (depends on action type)
3. Blocking actions (deny, rate_limit, budget) stop request processing
4. Non-blocking actions (log, alert) do not stop processing

---

## 12. Versioning

### 12.1 MPL Schema Version

The `mpl_version` field specifies the MPL schema version.

**Current Version:** `"1.0"`

**Breaking Changes:**
- Changes to condition syntax
- Removal of action types
- Changes to type system

**Non-Breaking Changes:**
- New action types
- New built-in functions
- New data model fields

### 12.2 Policy Version

The `version` field specifies the policy version (semver).

**Semantic Versioning:**

- MAJOR: Breaking changes to policy behavior
- MINOR: New rules added
- PATCH: Bug fixes, description updates

**Example:**

```yaml
mpl_version: "1.0"
version: "2.1.3"
```

---

## 13. Error Handling

### 13.1 Parse Errors

Errors during policy parsing:

- Invalid YAML syntax
- Missing required fields
- Invalid field types
- Unknown action types
- Invalid operator usage

**Behavior:** Policy fails to load, error message returned

### 13.2 Evaluation Errors

Errors during condition evaluation:

- Field does not exist
- Type mismatch
- Function errors

**Behavior:** Condition evaluates to false, rule does not match

### 13.3 Action Errors

Errors during action execution:

- Invalid action parameters
- External service failures (webhook, alert)

**Behavior:** Depends on action type (fail-safe defaults)

---

## 14. Examples

See [docs/mpl/examples/](examples/) for comprehensive examples covering:

1. Basic request blocking
2. PII detection and redaction
3. Token budget enforcement
4. Intelligent model routing
5. Prompt injection detection
6. Time-based policies
7. User-based policies
8. Response content filtering
9. Compliance audit logging
10. Multi-turn conversation management
11. Cost-based routing
12. Rate limiting
13. Data residency enforcement
14. Sensitive content filtering
15. Model allowlist
16. Tool calling policies
17. Streaming-specific policies
18. Multimodal content policies
19. Environment-based policies
20. Comprehensive audit trail

---

## Appendix A: Complete Grammar (EBNF)

See [SYNTAX.md](SYNTAX.md) for complete grammar specification.

## Appendix B: JSON Schema

See [schema.json](schema.json) for JSON Schema definition.

## Appendix C: Best Practices

See [BEST_PRACTICES.md](BEST_PRACTICES.md) for policy authoring best practices.

---

**End of Specification**
