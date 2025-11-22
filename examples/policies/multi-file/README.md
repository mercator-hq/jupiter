# Multi-File Policy Example

This directory demonstrates how to organize policies across multiple files for better maintainability and team collaboration.

## Structure

```
multi-file/
├── README.md              # This file
├── security.yaml          # Security-focused rules
├── rate-limits.yaml       # Rate limiting rules
├── content-safety.yaml    # Content filtering rules
└── logging.yaml           # Observability rules
```

## Benefits of Multi-File Policies

1. **Separation of Concerns**: Each file focuses on a specific governance domain
2. **Team Collaboration**: Different teams can own different policy files
3. **Easier Reviews**: Changes to security policies don't mix with rate limiting changes
4. **Atomic Updates**: All files reload together, ensuring consistency
5. **Modular Testing**: Test each policy domain independently

## Loading Multi-File Policies

Point the policy manager to this directory:

```yaml
# config.yaml
policy:
  mode: file
  file_path: examples/policies/multi-file/
  watch: true
  validation:
    enabled: true
```

All `.yaml` and `.yml` files in the directory will be loaded automatically.

## Priority Ranges by Domain

To avoid conflicts, each domain uses a specific priority range:

- **Security** (200-299): Highest priority, evaluated first
- **Access Control** (150-199): Authentication, authorization
- **Rate Limiting** (100-149): Quotas and rate limits
- **Content Safety** (50-99): Content filtering, compliance
- **Logging/Transform** (1-49): Observability and transformations

## File Descriptions

### security.yaml
Critical security checks including:
- Prompt injection detection
- PII protection
- Attack pattern blocking

### rate-limits.yaml
Cost control and abuse prevention:
- Per-tier rate limits
- Token quotas
- Burst protection

### content-safety.yaml
Content governance:
- Harmful content filtering
- Compliance logging
- Sensitive topic detection

### logging.yaml
Observability and monitoring:
- Request logging
- Audit trails
- Analytics tracking
