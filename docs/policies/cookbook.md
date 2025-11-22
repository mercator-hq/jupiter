# Policy Cookbook

A collection of real-world policy examples for common LLM governance use cases. Each example is production-ready and can be adapted to your needs.

## Table of Contents

- [Content Safety](#content-safety)
- [Budget & Cost Control](#budget--cost-control)
- [Routing & Load Balancing](#routing--load-balancing)
- [Compliance & Regulatory](#compliance--regulatory)
- [Development & Testing](#development--testing)
- [Security & Access Control](#security--access-control)
- [Observability & Monitoring](#observability--monitoring)

## Quick Reference

| Use Case | Policy Example | Description |
|----------|---------------|-------------|
| Block prohibited content | [01-basic-deny.yaml](../mpl/examples/01-basic-deny.yaml) | Simple deny based on conditions |
| PII detection | [02-pii-detection.yaml](../mpl/examples/02-pii-detection.yaml) | Detect and block PII in requests |
| Token limits | [03-token-limits.yaml](../mpl/examples/03-token-limits.yaml) | Enforce token usage limits |
| Model routing | [04-model-routing.yaml](../mpl/examples/04-model-routing.yaml) | Route requests to providers |
| Rate limiting | [05-rate-limiting.yaml](../mpl/examples/05-rate-limiting.yaml) | Control request rates |
| Prompt injection | [06-prompt-injection.yaml](../mpl/examples/06-prompt-injection.yaml) | Detect prompt injection attacks |
| Cost control | [07-cost-control.yaml](../mpl/examples/07-cost-control.yaml) | Budget enforcement |
| Compliance | [08-compliance.yaml](../mpl/examples/08-compliance.yaml) | HIPAA/GDPR compliance |
| Data residency | [09-data-residency.yaml](../mpl/examples/09-data-residency.yaml) | Geographic data restrictions |
| Multi-turn conversations | [10-multi-turn.yaml](../mpl/examples/10-multi-turn.yaml) | Context window management |
| Sensitive content | [11-sensitive-content.yaml](../mpl/examples/11-sensitive-content.yaml) | Filter sensitive topics |
| User attributes | [12-user-attributes.yaml](../mpl/examples/12-user-attributes.yaml) | Role-based access control |
| Time-based policies | [13-time-based.yaml](../mpl/examples/13-time-based.yaml) | Business hours restrictions |
| Environment-based | [14-environment.yaml](../mpl/examples/14-environment.yaml) | Dev/staging/prod policies |
| Model allowlist | [15-model-allowlist.yaml](../mpl/examples/15-model-allowlist.yaml) | Approved models only |
| Response filtering | [16-response-filtering.yaml](../mpl/examples/16-response-filtering.yaml) | Filter LLM responses |
| Tool calling | [17-tool-calling.yaml](../mpl/examples/17-tool-calling.yaml) | Function calling governance |
| Streaming | [18-streaming.yaml](../mpl/examples/18-streaming.yaml) | Streaming response policies |
| Multimodal | [19-multimodal.yaml](../mpl/examples/19-multimodal.yaml) | Image/vision model policies |
| Audit trail | [20-audit-trail.yaml](../mpl/examples/20-audit-trail.yaml) | Comprehensive logging |
| Department-based | [21-department-based.yaml](../mpl/examples/21-department-based.yaml) | Organizational policies |
| Request tagging | [22-request-tagging.yaml](../mpl/examples/22-request-tagging.yaml) | Metadata enrichment |

---

## Content Safety

Policies for protecting against harmful, sensitive, or inappropriate content.

### PII Detection and Blocking

**Use Case**: Prevent personally identifiable information from being sent to LLMs.

**Policy**: [02-pii-detection.yaml](../mpl/examples/02-pii-detection.yaml)

**What it does**:
- Detects email addresses, phone numbers, SSNs, credit cards
- Blocks requests containing PII
- Returns clear error messages

**When to use**:
- Healthcare applications (HIPAA compliance)
- Financial services (PCI-DSS compliance)
- Any application handling personal data

**Example**:
```yaml
- condition: |
    request.messages[-1].content matches "[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}"
  action: "deny"
  reason: "Request contains email address"
```

See: [Content Safety Guide](content-safety.md)

### Sensitive Content Filtering

**Use Case**: Block requests about sensitive topics.

**Policy**: [11-sensitive-content.yaml](../mpl/examples/11-sensitive-content.yaml)

**What it does**:
- Filters requests about violence, illegal activities, adult content
- Customizable topic blocklist
- Logging of blocked content attempts

**When to use**:
- Public-facing applications
- Educational platforms
- Enterprise tools with acceptable use policies

See: [Content Safety Guide](content-safety.md)

### Prompt Injection Detection

**Use Case**: Detect and block prompt injection attacks.

**Policy**: [06-prompt-injection.yaml](../mpl/examples/06-prompt-injection.yaml)

**What it does**:
- Detects common prompt injection patterns
- Blocks system prompt override attempts
- Logs potential attack attempts

**When to use**:
- Production applications
- User-facing chatbots
- API services with untrusted input

---

## Budget & Cost Control

Policies for managing LLM spending and resource usage.

### Cost Control and Budget Enforcement

**Use Case**: Enforce per-user, per-team, or global spending limits.

**Policy**: [07-cost-control.yaml](../mpl/examples/07-cost-control.yaml)

**What it does**:
- Tracks spending per user/team
- Hard limits on daily/monthly budgets
- Warning notifications at 80% threshold

**When to use**:
- Multi-tenant applications
- Department cost allocation
- Preventing runaway costs

**Example**:
```yaml
- condition: |
    request.metadata.user_budget_spent >= request.metadata.user_budget_limit
  action: "deny"
  reason: "Budget exceeded"
```

See: [Budget & Limits Guide](budget-limits.md)

### Token Limits

**Use Case**: Control token usage per request or user.

**Policy**: [03-token-limits.yaml](../mpl/examples/03-token-limits.yaml)

**What it does**:
- Enforces max tokens per request
- Prevents excessive token usage
- Model-specific token limits

**When to use**:
- Cost optimization
- Preventing abuse
- Quality control (concise responses)

See: [Budget & Limits Guide](budget-limits.md)

### Rate Limiting

**Use Case**: Control request frequency per user or API key.

**Policy**: [05-rate-limiting.yaml](../mpl/examples/05-rate-limiting.yaml)

**What it does**:
- Requests per minute (RPM) limits
- Tokens per minute (TPM) limits
- Sliding window rate limiting

**When to use**:
- API services
- Preventing abuse
- Fair resource allocation

See: [Budget & Limits Guide](budget-limits.md)

---

## Routing & Load Balancing

Policies for intelligent request routing across providers.

### Model-Based Routing

**Use Case**: Route requests to appropriate providers based on model.

**Policy**: [04-model-routing.yaml](../mpl/examples/04-model-routing.yaml)

**What it does**:
- Routes GPT models to OpenAI
- Routes Claude models to Anthropic
- Routes local models to Ollama
- Failover to backup providers

**When to use**:
- Multi-provider deployments
- Provider-specific features
- Cost optimization

**Example**:
```yaml
- condition: 'request.model matches "^gpt-"'
  action: "route"
  provider: "openai"
```

See: [Routing Guide](routing.md)

### Cost-Optimized Routing

**Use Case**: Route to least expensive provider for each model class.

**Policy**: [04-model-routing.yaml](../mpl/examples/04-model-routing.yaml)

**What it does**:
- Routes to cheaper providers when possible
- Falls back to premium providers if needed
- Tracks cost savings

**When to use**:
- Cost optimization
- Multiple compatible providers
- High-volume applications

See: [Routing Guide](routing.md)

---

## Compliance & Regulatory

Policies for regulatory compliance (HIPAA, GDPR, SOC2, etc.).

### HIPAA Compliance

**Use Case**: Ensure HIPAA compliance for healthcare applications.

**Policy**: [08-compliance.yaml](../mpl/examples/08-compliance.yaml)

**What it does**:
- Blocks PHI in requests
- Enforces audit logging
- Requires authorized users only
- 7-year evidence retention

**When to use**:
- Healthcare applications
- Electronic health records
- Patient communication systems

See: [Compliance Guide](compliance.md)

### GDPR Compliance

**Use Case**: Ensure GDPR compliance for EU user data.

**Policy**: [08-compliance.yaml](../mpl/examples/08-compliance.yaml)

**What it does**:
- Detects and blocks PII
- Enforces data residency (EU providers only)
- 90-day data retention
- User consent tracking

**When to use**:
- Applications with EU users
- Data processing in EU
- GDPR Article 5 compliance

See: [Compliance Guide](compliance.md)

### Data Residency

**Use Case**: Ensure data stays within specific geographic regions.

**Policy**: [09-data-residency.yaml](../mpl/examples/09-data-residency.yaml)

**What it does**:
- Routes EU users to EU providers
- Routes US users to US providers
- Blocks cross-border data transfer

**When to use**:
- Regulatory compliance
- Data sovereignty requirements
- Regional performance optimization

See: [Compliance Guide](compliance.md)

---

## Development & Testing

Policies for development workflows and testing environments.

### Environment-Based Policies

**Use Case**: Different policies for dev, staging, and production.

**Policy**: [14-environment.yaml](../mpl/examples/14-environment.yaml)

**What it does**:
- Lenient limits in development
- Stricter limits in staging
- Full enforcement in production
- Environment-specific logging

**When to use**:
- Multi-environment deployments
- Gradual policy rollout
- Testing policy changes

**Example**:
```yaml
- condition: 'request.metadata.environment == "production"'
  action: "log"
  log_level: "info"
  message: "Production request"
```

See: [Development Guide](development.md)

### Time-Based Access

**Use Case**: Restrict LLM access to business hours.

**Policy**: [13-time-based.yaml](../mpl/examples/13-time-based.yaml)

**What it does**:
- Allows access 9 AM - 5 PM
- Blocks after-hours requests
- Weekend restrictions
- Timezone-aware

**When to use**:
- Cost control
- Compliance requirements
- Preventing off-hours abuse

See: [Development Guide](development.md)

---

## Security & Access Control

Policies for security and authorization.

### Model Allowlist

**Use Case**: Only allow specific approved models.

**Policy**: [15-model-allowlist.yaml](../mpl/examples/15-model-allowlist.yaml)

**What it does**:
- Allowlist of approved models
- Blocks unapproved models
- Per-team model access

**When to use**:
- Security compliance
- Cost control (block expensive models)
- Quality control (approved models only)

### Role-Based Access Control

**Use Case**: Different permissions for different user roles.

**Policy**: [12-user-attributes.yaml](../mpl/examples/12-user-attributes.yaml)

**What it does**:
- Admin-only models
- Department-specific access
- User tier limits

**When to use**:
- Enterprise applications
- Multi-tenant systems
- Graduated feature access

**Example**:
```yaml
- condition: 'request.model == "gpt-4" and request.metadata.user_role != "admin"'
  action: "deny"
  reason: "GPT-4 requires admin role"
```

### Department-Based Policies

**Use Case**: Organizational policies per department.

**Policy**: [21-department-based.yaml](../mpl/examples/21-department-based.yaml)

**What it does**:
- Engineering: full access
- Sales: limited models
- Marketing: content generation only
- Finance: stricter compliance

**When to use**:
- Large organizations
- Department cost allocation
- Compliance by business unit

---

## Observability & Monitoring

Policies for logging, monitoring, and debugging.

### Comprehensive Audit Trail

**Use Case**: Log all requests with full context for compliance.

**Policy**: [20-audit-trail.yaml](../mpl/examples/20-audit-trail.yaml)

**What it does**:
- Logs every request/response
- Captures user metadata
- Cost tracking
- Latency monitoring

**When to use**:
- Compliance requirements
- Debugging
- Usage analytics
- Billing/chargeback

### Request Tagging and Enrichment

**Use Case**: Add metadata to requests for tracking and analysis.

**Policy**: [22-request-tagging.yaml](../mpl/examples/22-request-tagging.yaml)

**What it does**:
- Tags requests with metadata
- Tracks request source
- Enriches with user info
- Facilitates analytics

**When to use**:
- Multi-tenant systems
- Usage analytics
- Cost allocation
- A/B testing

---

## Advanced Use Cases

### Multi-Turn Conversation Management

**Use Case**: Manage context windows in long conversations.

**Policy**: [10-multi-turn.yaml](../mpl/examples/10-multi-turn.yaml)

**What it does**:
- Limits conversation length
- Warns on context window approaching
- Suggests summarization

### Tool Calling Governance

**Use Case**: Control which tools/functions LLMs can call.

**Policy**: [17-tool-calling.yaml](../mpl/examples/17-tool-calling.yaml)

**What it does**:
- Allowlist approved functions
- Blocks dangerous tools
- Logs tool usage

### Streaming Response Policies

**Use Case**: Policies for streaming responses.

**Policy**: [18-streaming.yaml](../mpl/examples/18-streaming.yaml)

**What it does**:
- Enforces streaming limits
- Monitors token usage in real-time
- Cancels runaway generations

### Multimodal Content Policies

**Use Case**: Governance for vision/image models.

**Policy**: [19-multimodal.yaml](../mpl/examples/19-multimodal.yaml)

**What it does**:
- Validates image types
- Enforces size limits
- Scans for inappropriate images

### Response Filtering

**Use Case**: Filter or modify LLM responses.

**Policy**: [16-response-filtering.yaml](../mpl/examples/16-response-filtering.yaml)

**What it does**:
- Redacts sensitive information in responses
- Blocks inappropriate content
- Enforces output formats

---

## Combining Policies

Most production deployments combine multiple policies. Here's a typical production policy bundle:

```yaml
version: "1.0"

policies:
  # Priority 400: Security (highest)
  - name: "pii-protection"
    priority: 400
    # Block PII in requests

  # Priority 300: Compliance
  - name: "gdpr-compliance"
    priority: 300
    # GDPR requirements

  # Priority 200: Cost control
  - name: "budget-enforcement"
    priority: 200
    # Budget limits

  # Priority 100: Routing
  - name: "model-routing"
    priority: 100
    # Route to providers

  # Priority 50: Observability
  - name: "audit-logging"
    priority: 50
    # Log everything
```

**Evaluation order**: Policies are evaluated from highest to lowest priority. First deny/allow wins.

---

## Policy Testing

Always test policies before deploying to production:

```bash
# Validate policy syntax
mercator lint --file my-policy.yaml

# Run policy tests
mercator test --policy my-policy.yaml --tests my-tests.yaml

# Dry-run with real requests
mercator run --config config.yaml --dry-run
```

See: [Testing Policies](../cli/test.md)

---

## Next Steps

- **[MPL Language Reference](../mpl/SPECIFICATION.md)** - Complete language spec
- **[MPL Syntax Guide](../mpl/SYNTAX.md)** - Syntax reference
- **[MPL Best Practices](../mpl/BEST_PRACTICES.md)** - Writing effective policies
- **[Content Safety Guide](content-safety.md)** - Detailed content filtering
- **[Budget & Limits Guide](budget-limits.md)** - Cost control strategies
- **[Routing Guide](routing.md)** - Advanced routing patterns
- **[Compliance Guide](compliance.md)** - Regulatory compliance
- **[Development Guide](development.md)** - Dev workflows

---

## Contributing Policy Examples

Have a useful policy to share? Contributions welcome!

1. Create your policy YAML file
2. Add comprehensive comments explaining the use case
3. Include test cases
4. Submit a pull request to [mercator-hq/jupiter](https://github.com/mercator-hq/jupiter)

See: [Contributing Guide](../../CONTRIBUTING.md)
