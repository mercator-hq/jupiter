# Example Policies

This directory contains example policy files demonstrating various features of the Mercator Jupiter Policy Language (MPL).

## Examples Overview

1. **[simple-logging.yaml](simple-logging.yaml)** - Basic policy with single rule for logging
2. **[model-restrictions.yaml](model-restrictions.yaml)** - Enforce allowed/blocked model lists
3. **[rate-limiting.yaml](rate-limiting.yaml)** - Rate limiting based on user/model
4. **[content-filtering.yaml](content-filtering.yaml)** - Filter requests based on content
5. **[multi-rule.yaml](multi-rule.yaml)** - Complex policy with multiple rules and priorities
6. **[production-policy.yaml](production-policy.yaml)** - Production-ready comprehensive policy

## Directory Structure for Multi-File Policies

```
examples/policies/
├── multi-file/                    # Multi-file policy example
│   ├── main.yaml                  # Main policy file
│   ├── rate-limits.yaml           # Rate limiting rules
│   └── security.yaml              # Security rules
└── single-file/                   # Single-file policy examples
    └── *.yaml                     # Individual policy files
```

## Using These Examples

### Single File Mode

```bash
# Load a single policy file
mercator --config config.yaml --policy-file examples/policies/simple-logging.yaml

# Or via configuration
# config.yaml:
# policy:
#   mode: file
#   file_path: examples/policies/simple-logging.yaml
```

### Directory Mode

```bash
# Load all policies from a directory
mercator --config config.yaml --policy-dir examples/policies/multi-file/

# Or via configuration
# config.yaml:
# policy:
#   mode: file
#   file_path: examples/policies/multi-file/
```

### Hot-Reload

Enable file watching to automatically reload policies when they change:

```yaml
# config.yaml
policy:
  mode: file
  file_path: examples/policies/production-policy.yaml
  watch: true  # Enable hot-reload
  validation:
    enabled: true
    strict: false
```

## Policy Structure

All policies follow this basic structure:

```yaml
mpl_version: "1.0"              # Required: MPL version
name: "policy-name"             # Required: Unique policy identifier
version: "1.0.0"                # Required: Semantic version
description: "Description"      # Optional: Human-readable description
metadata:                       # Optional: Additional metadata
  author: "Your Name"
  tags: ["production", "security"]

rules:                          # Required: List of policy rules
  - name: "rule-1"              # Required: Unique rule name within policy
    priority: 100               # Optional: Higher priority rules evaluated first
    conditions:                 # Required: When to apply this rule
      field: "request.model"
      operator: "=="
      value: "gpt-4"
    actions:                    # Required: What to do when conditions match
      - type: "allow"           # or "deny", "log", "transform", etc.
        message: "Explanation"
```

## Common Use Cases

### 1. Model Allowlist

Restrict to approved models only:

```yaml
rules:
  - name: "allowed-models-only"
    conditions:
      field: "request.model"
      operator: "in"
      value: ["gpt-4", "gpt-4-turbo", "claude-3-opus"]
    actions:
      - type: "allow"
```

### 2. Cost Control

Block expensive models for certain users:

```yaml
rules:
  - name: "block-expensive-for-free-tier"
    conditions:
      all:
        - field: "request.user.tier"
          operator: "=="
          value: "free"
        - field: "request.model"
          operator: "in"
          value: ["gpt-4", "claude-3-opus"]
    actions:
      - type: "deny"
        message: "Upgrade to access premium models"
```

### 3. Content Safety

Filter inappropriate content:

```yaml
rules:
  - name: "block-harmful-content"
    conditions:
      field: "request.prompt"
      operator: "contains_any"
      value: ["violence", "hate speech"]
    actions:
      - type: "deny"
        message: "Content policy violation"
      - type: "log"
        level: "warning"
```

### 4. Rate Limiting

Limit requests per time window:

```yaml
rules:
  - name: "rate-limit-per-user"
    conditions:
      field: "rate_limit.user"
      operator: ">"
      value: 100  # requests per minute
    actions:
      - type: "deny"
        message: "Rate limit exceeded. Please retry in 60 seconds."
```

## Best Practices

1. **Use Descriptive Names**: Policy and rule names should clearly indicate purpose
2. **Version Semantically**: Follow semver for policy versions (major.minor.patch)
3. **Document Rules**: Use `description` field to explain complex rules
4. **Test Policies**: Use `mercator test` to validate policies before deployment
5. **Start Strict**: Begin with restrictive policies and relax as needed
6. **Monitor Effects**: Log policy decisions to understand impact
7. **Atomic Updates**: Use directory mode for related policies that should update together

## Validation

Before deploying policies, validate them:

```bash
# Lint a policy file
mercator lint examples/policies/production-policy.yaml

# Test a policy against sample requests
mercator test examples/policies/production-policy.yaml --requests test-requests.json

# Validate all policies in a directory
mercator lint examples/policies/multi-file/
```

## Troubleshooting

### Policy Not Loading

- Check file permissions (must be readable)
- Verify YAML syntax is valid
- Ensure `mpl_version`, `name`, `version`, and `rules` are present
- Check logs for validation errors

### Hot-Reload Not Working

- Verify `watch: true` in configuration
- Check file system events are supported (may not work on network filesystems)
- Look for "File watcher started" in logs
- Ensure file changes trigger write events (some editors use atomic writes)

### Policy Not Taking Effect

- Check rule conditions are matching as expected
- Verify rule priority if multiple rules apply
- Enable debug logging to see policy evaluation
- Test with `mercator test` to isolate issues

## Further Reading

- [Policy Language Reference](../../docs/policy-language.md)
- [Policy Engine Architecture](../../docs/policy-engine.md)
- [Troubleshooting Guide](../../docs/TROUBLESHOOTING.md)
- [Configuration Guide](../../docs/configuration.md)
