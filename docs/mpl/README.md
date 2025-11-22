# Mercator Policy Language (MPL) Documentation

**Version:** 1.0.0
**Status:** Specification Complete
**Last Updated:** 2025-11-16

---

## Overview

The Mercator Policy Language (MPL) is a declarative, YAML-based policy language designed for LLM governance. MPL enables security teams, compliance officers, and platform engineers to define rules that control LLM request and response behavior without writing code.

---

## Documentation Structure

### Core Documentation

- **[SPECIFICATION.md](SPECIFICATION.md)** - Complete MPL v1.0 specification
  - Language design and principles
  - Policy and rule structure
  - Condition language syntax
  - Type system
  - Actions reference
  - Data model (available fields)
  - Built-in functions
  - Evaluation semantics
  - Versioning

- **[SYNTAX.md](SYNTAX.md)** - Quick syntax reference
  - Condensed syntax guide
  - Common patterns
  - Field reference
  - Operator reference

- **[BEST_PRACTICES.md](BEST_PRACTICES.md)** - Policy authoring best practices
  - Policy organization
  - Rule design guidelines
  - Condition writing tips
  - Action selection
  - Variable usage
  - Performance optimization
  - Security considerations
  - Testing strategies
  - Version management

- **[schema.json](schema.json)** - JSON Schema for MPL validation
  - Enables IDE auto-completion
  - Validates policy files
  - Enforces schema compliance

### Example Policies

The [examples/](examples/) directory contains 20+ real-world policy examples:

1. [01-basic-deny.yaml](examples/01-basic-deny.yaml) - Basic request blocking
2. [02-pii-detection.yaml](examples/02-pii-detection.yaml) - PII detection and redaction
3. [03-token-limits.yaml](examples/03-token-limits.yaml) - Token budget enforcement
4. [04-model-routing.yaml](examples/04-model-routing.yaml) - Intelligent model routing
5. [05-rate-limiting.yaml](examples/05-rate-limiting.yaml) - Rate limiting by API key
6. [06-prompt-injection.yaml](examples/06-prompt-injection.yaml) - Prompt injection detection
7. [07-cost-control.yaml](examples/07-cost-control.yaml) - Cost control and budgets
8. [08-compliance.yaml](examples/08-compliance.yaml) - Compliance audit logging
9. [09-data-residency.yaml](examples/09-data-residency.yaml) - Data residency enforcement
10. [10-multi-turn.yaml](examples/10-multi-turn.yaml) - Multi-turn conversation management
11. [11-sensitive-content.yaml](examples/11-sensitive-content.yaml) - Sensitive content filtering
12. [12-user-attributes.yaml](examples/12-user-attributes.yaml) - User-based policies
13. [13-time-based.yaml](examples/13-time-based.yaml) - Time-based policies
14. [14-environment.yaml](examples/14-environment.yaml) - Environment-based policies
15. [15-model-allowlist.yaml](examples/15-model-allowlist.yaml) - Model allowlist enforcement
16. [16-response-filtering.yaml](examples/16-response-filtering.yaml) - Response content filtering
17. [17-tool-calling.yaml](examples/17-tool-calling.yaml) - Tool/function calling policies
18. [18-streaming.yaml](examples/18-streaming.yaml) - Streaming-specific policies
19. [19-multimodal.yaml](examples/19-multimodal.yaml) - Multimodal content policies
20. [20-audit-trail.yaml](examples/20-audit-trail.yaml) - Comprehensive audit trail
21. [21-department-based.yaml](examples/21-department-based.yaml) - Department-based access control

---

## Quick Start

### 1. Basic Policy Structure

```yaml
mpl_version: "1.0"
name: "my-policy"
version: "1.0.0"

rules:
  - name: "my-rule"
    conditions:
      - field: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: "allow"
```

### 2. Validate Your Policy

Use the JSON Schema to validate your policy:

```bash
# Using a JSON Schema validator
ajv validate -s docs/mpl/schema.json -d your-policy.yaml
```

### 3. Test Your Policy

See [BEST_PRACTICES.md](BEST_PRACTICES.md#8-testing-policies) for testing strategies.

---

## Key Concepts

### Policy
A named collection of rules, variables, and metadata that defines governance behavior.

### Rule
A single conditional statement that specifies when to take specific actions.

### Condition
A boolean expression that evaluates request/response data using fields, operators, and functions.

### Action
An operation to perform when a rule's conditions are met (allow, deny, log, redact, etc.).

### Variable
A reusable value that can be referenced in conditions and actions.

---

## Design Principles

1. **Declarative**: Express intent, not implementation
2. **Readable**: Non-programmers can understand and write policies
3. **Type-Safe**: Strong typing prevents ambiguous expressions
4. **Composable**: Policies can reference variables and be organized modularly
5. **Git-Friendly**: YAML format works well with version control
6. **Extensible**: Easy to add new conditions and actions without breaking existing policies
7. **Testable**: Policies can be validated independently of execution

---

## Available Actions

- **allow** - Allow request to proceed
- **deny** - Block request with error message
- **log** - Log event for audit or debugging
- **redact** - Remove or mask sensitive content
- **modify** - Change request/response fields
- **route** - Route to specific provider or model
- **alert** - Trigger external alert (webhook)
- **rate_limit** - Apply rate limiting
- **budget** - Enforce budget constraints (tokens or cost)

---

## Available Fields

### Request Fields
- `request.model` - Model name
- `request.messages` - Array of messages
- `request.temperature` - Temperature parameter
- `request.max_tokens` - Max tokens parameter
- `request.stream` - Streaming enabled
- `request.user` - User ID
- And more... (see [SPECIFICATION.md](SPECIFICATION.md#8-data-model))

### Processing Fields
- `processing.token_estimate.total_tokens` - Estimated tokens
- `processing.cost_estimate.total_cost` - Estimated cost
- `processing.risk_score` - Risk score (1-10)
- `processing.content_analysis.pii_detection.*` - PII detection results
- `processing.content_analysis.prompt_injection.*` - Prompt injection detection
- And more... (see [SPECIFICATION.md](SPECIFICATION.md#8-data-model))

### Context Fields
- `context.time.hour` - Current hour (0-23)
- `context.time.day_of_week` - Day of week
- `context.environment` - Environment (production, staging, etc.)
- `context.user_attributes.*` - User-specific attributes
- And more... (see [SPECIFICATION.md](SPECIFICATION.md#8-data-model))

---

## Common Use Cases

### Safety and Security
- PII detection and redaction
- Prompt injection prevention
- Sensitive content filtering
- Security audit logging

### Cost Control
- Token budget enforcement
- Cost-based model routing
- Cost alerts and notifications
- Department budget tracking

### Compliance
- Data residency enforcement
- Comprehensive audit logging
- Regulatory compliance checks
- Content policy enforcement

### Performance
- Intelligent model routing
- Rate limiting and throttling
- Context window management
- Streaming optimization

### Access Control
- User tier-based policies
- Department-based restrictions
- Time-based access control
- Environment-based policies

---

## Next Steps

1. **Read the Specification**: Start with [SPECIFICATION.md](SPECIFICATION.md) for complete language details
2. **Explore Examples**: Browse [examples/](examples/) for real-world policy patterns
3. **Review Best Practices**: Read [BEST_PRACTICES.md](BEST_PRACTICES.md) for authoring guidelines
4. **Write Your First Policy**: Use [SYNTAX.md](SYNTAX.md) as a quick reference
5. **Validate Your Policy**: Use [schema.json](schema.json) for validation
6. **Test Thoroughly**: Follow testing best practices before deployment

---

## Integration with Mercator Jupiter

MPL policies are consumed by:

- **MPL Parser (Feature 5B)**: Parses YAML policies into Abstract Syntax Trees (AST)
- **Policy Engine (Feature 6)**: Evaluates policies against requests/responses
- **Policy Management (Feature 8)**: Loads, validates, and hot-reloads policies

See the main Mercator Jupiter documentation for integration details.

---

## Version History

### v1.0.0 (2025-11-16)
- Initial MPL specification
- Complete syntax definition
- 21 example policies
- JSON Schema for validation
- Best practices guide

---

## Contributing

When proposing changes to the MPL specification:

1. Open an issue to discuss the change
2. Update [SPECIFICATION.md](SPECIFICATION.md) with proposed changes
3. Add examples to [examples/](examples/)
4. Update [schema.json](schema.json) if needed
5. Update [SYNTAX.md](SYNTAX.md) and [BEST_PRACTICES.md](BEST_PRACTICES.md)
6. Follow semantic versioning for breaking changes

---

## Support

For questions or issues:

- **Specification Questions**: Review [SPECIFICATION.md](SPECIFICATION.md)
- **Syntax Questions**: Check [SYNTAX.md](SYNTAX.md)
- **Best Practices**: See [BEST_PRACTICES.md](BEST_PRACTICES.md)
- **Examples**: Browse [examples/](examples/)
- **Issues**: Open an issue in the Mercator Jupiter repository

---

**Mercator Policy Language v1.0 - Specification Complete**
