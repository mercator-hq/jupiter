# Creating Your First Policy

This guide teaches you how to write, test, and deploy your first Mercator Policy Language (MPL) policy. You'll learn the basics of policy syntax, conditions, actions, and testing.

## What is a Policy?

A **policy** in Mercator Jupiter is a set of rules that governs how LLM requests are handled. Policies can:

- **Allow or deny** requests based on conditions
- **Log** requests for audit trails
- **Route** requests to specific providers
- **Redact** sensitive content from prompts or responses
- **Modify** request parameters
- **Enforce budgets** and rate limits

Policies are written in **Mercator Policy Language (MPL)**, a declarative YAML-based language designed for LLM governance.

## Policy Structure

Every MPL policy file has this structure:

```yaml
version: "1.0"

policies:
  - name: "policy-name"
    description: "What this policy does"
    priority: 100
    rules:
      - condition: "boolean expression"
        action: "allow | deny | log | route | redact | modify"
        # Action-specific parameters...
```

### Key Components

- **version**: MPL version (always `"1.0"` for now)
- **policies**: List of policy definitions
- **name**: Unique identifier for the policy
- **description**: Human-readable explanation
- **priority**: Evaluation order (higher = evaluated first, default: 100)
- **rules**: List of condition-action pairs

## Example 1: Content Filtering Policy

Let's create a policy that blocks requests containing profanity:

```yaml
version: "1.0"

policies:
  - name: "profanity-filter"
    description: "Block requests containing profanity"
    priority: 200  # High priority - check this first
    rules:
      - condition: |
          request.messages[-1].content matches "(?i)(badword1|badword2|offensive)"
        action: "deny"
        reason: "Request contains prohibited content"
```

**Save this as** `profanity-policy.yaml`

### Test the Policy

Use the `mercator test` command to test your policy:

```bash
# Create a test file
cat > profanity-test.yaml << 'EOF'
tests:
  - name: "Should block profanity"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "This contains badword1"
    expected:
      action: "deny"
      reason_contains: "prohibited content"

  - name: "Should allow clean content"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Hello, how are you?"
    expected:
      action: "allow"
EOF

# Run the tests
mercator test --policy profanity-policy.yaml --tests profanity-test.yaml
```

**Expected output:**

```
✓ Should block profanity
✓ Should allow clean content

Tests: 2 passed, 0 failed, 0 skipped
```

## Example 2: Budget Enforcement Policy

Create a policy that enforces spending limits:

```yaml
version: "1.0"

policies:
  - name: "budget-enforcement"
    description: "Enforce user spending limits"
    rules:
      # Deny if user has exceeded their budget
      - condition: |
          request.metadata.user_budget_spent >= request.metadata.user_budget_limit
        action: "deny"
        reason: "User {{request.metadata.user_id}} has exceeded their budget limit of ${{request.metadata.user_budget_limit}}"

      # Warn if user is approaching budget
      - condition: |
          request.metadata.user_budget_spent >= (request.metadata.user_budget_limit * 0.9)
        action: "log"
        log_level: "warn"
        message: "User {{request.metadata.user_id}} is approaching budget limit"

      # Log all spending
      - condition: "true"
        action: "log"
        log_level: "info"
        message: "User {{request.metadata.user_id}} request cost: ${{request.estimated_cost}}"
```

**Save this as** `budget-policy.yaml`

### Testing with Metadata

```yaml
tests:
  - name: "Should deny when budget exceeded"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Hello"
      metadata:
        user_id: "user-123"
        user_budget_spent: 100.0
        user_budget_limit: 100.0
    expected:
      action: "deny"

  - name: "Should allow when under budget"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Hello"
      metadata:
        user_id: "user-123"
        user_budget_spent: 50.0
        user_budget_limit: 100.0
    expected:
      action: "allow"
```

## Example 3: Model Routing Policy

Route requests to different providers based on the model:

```yaml
version: "1.0"

policies:
  - name: "model-routing"
    description: "Route requests to appropriate providers"
    rules:
      # Route GPT models to OpenAI
      - condition: 'request.model matches "^gpt-"'
        action: "route"
        provider: "openai"

      # Route Claude models to Anthropic
      - condition: 'request.model matches "^claude-"'
        action: "route"
        provider: "anthropic"

      # Route Llama models to Ollama
      - condition: 'request.model matches "^llama"'
        action: "route"
        provider: "ollama"

      # Deny unknown models
      - condition: "true"
        action: "deny"
        reason: "Unknown model: {{request.model}}"
```

**Save this as** `routing-policy.yaml`

## Example 4: PII Detection and Redaction

Block or redact personally identifiable information (PII):

```yaml
version: "1.0"

policies:
  - name: "pii-protection"
    description: "Protect PII in requests and responses"
    rules:
      # Block requests with email addresses
      - condition: |
          request.messages[-1].content matches "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
        action: "deny"
        reason: "Request contains email addresses. Please remove PII before submitting."

      # Block requests with phone numbers
      - condition: |
          request.messages[-1].content matches "\\b\\d{3}[-.]?\\d{3}[-.]?\\d{4}\\b"
        action: "deny"
        reason: "Request contains phone numbers. Please remove PII before submitting."

      # Block requests with SSN patterns
      - condition: |
          request.messages[-1].content matches "\\b\\d{3}-\\d{2}-\\d{4}\\b"
        action: "deny"
        reason: "Request contains what appears to be a Social Security Number."
```

**Save this as** `pii-policy.yaml`

### Test PII Detection

```yaml
tests:
  - name: "Should block email addresses"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Contact me at john@example.com"
    expected:
      action: "deny"
      reason_contains: "email addresses"

  - name: "Should block phone numbers"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Call me at 555-123-4567"
    expected:
      action: "deny"
      reason_contains: "phone numbers"

  - name: "Should allow clean requests"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "How do I write a Python function?"
    expected:
      action: "allow"
```

## Combining Multiple Policies

You can combine multiple policies in one file:

```yaml
version: "1.0"

policies:
  # High priority: Security checks
  - name: "pii-protection"
    description: "Block PII"
    priority: 300
    rules:
      - condition: 'request.messages[-1].content matches "\\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}\\b"'
        action: "deny"
        reason: "PII detected: email address"

  # Medium priority: Access control
  - name: "model-allowlist"
    description: "Only allow approved models"
    priority: 200
    rules:
      - condition: 'request.model not in ["gpt-3.5-turbo", "gpt-4"]'
        action: "deny"
        reason: "Model not approved"

  # Low priority: Logging
  - name: "audit-logging"
    description: "Log all requests"
    priority: 100
    rules:
      - condition: "true"
        action: "log"
        log_level: "info"
        message: "Request from {{request.metadata.user_id}}"
```

**Evaluation Order:**
1. `pii-protection` (priority 300)
2. `model-allowlist` (priority 200)
3. `audit-logging` (priority 100)

## Policy Validation

Before deploying, always validate your policies:

```bash
# Validate policy syntax
mercator lint --file my-policy.yaml

# Strict mode (warnings as errors)
mercator lint --file my-policy.yaml --strict

# JSON output for CI/CD
mercator lint --file my-policy.yaml --format json
```

**Example validation output:**

```
✓ Policy syntax is valid
✓ All conditions are valid expressions
✓ All actions have required parameters
✓ No duplicate policy names

Policies: 3
Rules: 7
Warnings: 0
```

## Deploying Your Policy

### Option 1: File Mode

Update your `config.yaml`:

```yaml
policy:
  mode: "file"
  file_path: "./my-policy.yaml"
  watch: true  # Auto-reload on file changes
```

Restart Mercator Jupiter:

```bash
mercator run --config config.yaml
```

### Option 2: Git Mode

Commit your policy to a Git repository:

```bash
git add my-policy.yaml
git commit -m "Add PII protection policy"
git push
```

Update your `config.yaml`:

```yaml
policy:
  mode: "git"
  git_repo: "https://github.com/your-org/policies.git"
  git_branch: "main"
  git_path: "my-policy.yaml"
  git_poll_interval: "60s"  # Check for updates every 60s
```

Jupiter will automatically pull and reload policies from Git.

## Testing in Production

After deploying, test your policy:

```bash
# Make a test request
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Test message"}
    ],
    "metadata": {
      "user_id": "test-user"
    }
  }'

# Check evidence records
mercator evidence query --user-id "test-user" --limit 1

# View server logs
# Look for policy evaluation messages
```

## Common Patterns

### 1. Allow by Default, Deny Specific

```yaml
policies:
  - name: "blocklist"
    rules:
      - condition: 'request.model == "dangerous-model"'
        action: "deny"
      # Everything else is allowed (no catch-all deny)
```

### 2. Deny by Default, Allow Specific

```yaml
policies:
  - name: "allowlist"
    priority: 100
    rules:
      - condition: 'request.model in ["gpt-3.5-turbo", "gpt-4"]'
        action: "allow"
      - condition: "true"
        action: "deny"
        reason: "Model not in allowlist"
```

### 3. Time-Based Policies

```yaml
policies:
  - name: "business-hours-only"
    rules:
      - condition: 'time.hour < 9 or time.hour >= 17'
        action: "deny"
        reason: "LLM access is only available during business hours (9 AM - 5 PM)"
```

### 4. User-Based Policies

```yaml
policies:
  - name: "admin-only-models"
    rules:
      - condition: 'request.model == "gpt-4" and request.metadata.user_role != "admin"'
        action: "deny"
        reason: "Only admins can use GPT-4"
```

## Next Steps

You've learned how to create basic policies! Now explore:

- **[MPL Language Reference](../mpl/SPECIFICATION.md)** - Complete syntax documentation
- **[MPL Best Practices](../mpl/BEST_PRACTICES.md)** - Writing effective policies
- **[Policy Cookbook](../policies/cookbook.md)** - 20+ real-world examples
- **[Testing Policies](../cli/test.md)** - Advanced testing techniques
- **[Configuration Basics](configuration-basics.md)** - Understanding configuration

## Troubleshooting

### Issue: Policy not taking effect

**Problem**: Changes to policy file don't seem to apply

**Solution**:
- If `watch: true`, check server logs for reload messages
- If `watch: false`, restart the server
- Verify policy file path in config.yaml is correct

### Issue: Condition syntax error

**Problem**: `mercator lint` reports "invalid condition"

**Solution**:
- Check expression syntax matches MPL spec
- Verify field names are correct (`request.model`, not `model`)
- Ensure strings are quoted: `"gpt-4"` not `gpt-4`

### Issue: Policy blocks everything

**Problem**: All requests are denied

**Solution**:
- Check policy priority (higher priority = evaluated first)
- Ensure catch-all deny rules have lowest priority
- Add test cases to verify policy logic

---

**Previous**: [Quick Start Guide](quick-start.md) ← | **Next**: [Configuration Basics](configuration-basics.md) →
