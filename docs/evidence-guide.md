# Evidence Recording Guide

## Overview

Mercator Jupiter provides cryptographic evidence generation for LLM requests and responses, creating an immutable audit trail for compliance and governance.

## Current Implementation Status

**⚠️ Note**: Evidence recording infrastructure is implemented but not yet integrated with the proxy request flow. This feature is currently under development.

### What's Implemented

- ✅ Evidence storage backend (SQLite, Memory)
- ✅ Evidence recorder with async write buffering
- ✅ Retention and pruning scheduler
- ✅ Evidence query CLI (`mercator evidence query`)
- ✅ Cryptographic signing infrastructure

### What's In Progress

- ⏳ Integration with proxy request handlers
- ⏳ Automatic evidence creation for all proxied requests
- ⏳ Response data capture and hashing
- ⏳ Policy decision recording

## Configuration

Evidence recording is configured in `config.yaml`:

```yaml
evidence:
  enabled: true
  backend: "sqlite"  # Options: "sqlite", "memory"

  sqlite:
    path: "evidence.db"
    max_open_conns: 10
    max_idle_conns: 5
    wal_mode: true
    busy_timeout: 5s

  recorder:
    async_buffer: 1000
    write_timeout: 5s
    hash_request: true
    hash_response: true
    redact_api_keys: true
    max_field_length: 500

  retention:
    days: 90
    prune_schedule: "0 3 * * *"  # Daily at 3 AM
    archive_before_delete: false
    archive_path: ""
    max_records: 0  # 0 = unlimited
```

## Evidence Record Schema

Each evidence record contains:

```json
{
  "id": "uuid-v4",
  "timestamp": "2025-11-23T10:30:00Z",
  "request_id": "request-uuid",
  "user_id": "user-123",
  "provider": "openai",
  "model": "gpt-4",
  "operation": "chat.completions",

  "request": {
    "method": "POST",
    "path": "/v1/chat/completions",
    "headers": {
      "content-type": "application/json"
    },
    "body_hash": "sha256:abc123...",
    "metadata": {
      "user_tier": "pro",
      "estimated_cost": 0.05
    }
  },

  "response": {
    "status_code": 200,
    "headers": {
      "content-type": "application/json"
    },
    "body_hash": "sha256:def456...",
    "latency_ms": 1234,
    "tokens_used": 150
  },

  "policy": {
    "decision": "allow",
    "policies_evaluated": ["rate-limiting", "budget-enforcement"],
    "rules_matched": ["rule-1", "rule-2"]
  },

  "signature": {
    "algorithm": "Ed25519",
    "public_key_id": "key-2025-11",
    "signature": "base64-encoded-signature"
  }
}
```

## Querying Evidence

### Basic Queries

```bash
# Query all evidence records
mercator evidence query

# Limit number of results
mercator evidence query --limit 100

# Query by time range
mercator evidence query \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

# Query by user
mercator evidence query --user-id "user-123"

# Query by request ID
mercator evidence query --request-id "req-abc123"

# Query by provider
mercator evidence query --provider "openai"

# Query by model
mercator evidence query --model "gpt-4"
```

### Output Formats

```bash
# Text output (default, human-readable)
mercator evidence query --format text

# JSON output (machine-readable)
mercator evidence query --format json

# JSON Lines (streaming)
mercator evidence query --format jsonl

# CSV output (for analysis)
mercator evidence query --format csv
```

### Export to File

```bash
# Export to JSON file
mercator evidence query \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z" \
  --format json \
  --output evidence-export.json

# Export to CSV
mercator evidence query \
  --user-id "user-123" \
  --format csv \
  --output user-123-evidence.csv
```

## Retention and Pruning

Evidence records are automatically pruned based on retention policy:

```yaml
evidence:
  retention:
    days: 90  # Keep records for 90 days
    prune_schedule: "0 3 * * *"  # Run daily at 3 AM
    max_records: 1000000  # Maximum total records
    archive_before_delete: true  # Archive to file before deletion
    archive_path: "/var/lib/mercator/evidence-archive"
```

### Manual Pruning

```bash
# View retention status
mercator evidence retention status

# Run pruning manually
mercator evidence retention prune

# Dry-run (show what would be deleted)
mercator evidence retention prune --dry-run
```

## Cryptographic Signing

### Generate Signing Keys

```bash
# Generate new Ed25519 keypair
mercator keys generate --key-id "prod-2025-11" --output ./keys

# This creates:
# - prod-2025-11_private.pem (mode 0600)
# - prod-2025-11_public.pem (mode 0644)
```

### Configure Signing

```yaml
evidence:
  signing:
    enabled: true
    algorithm: "Ed25519"
    private_key_file: "/etc/mercator/keys/prod-2025-11_private.pem"
    public_key_id: "prod-2025-11"
```

### Validate Signatures

```bash
# Validate all evidence in database
mercator validate --all

# Validate specific time range
mercator validate \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

# Validate with detailed report
mercator validate --report --format json

# Validate exported evidence file
mercator validate --file evidence-export.json
```

## Integration Examples

### API Clients

When evidence recording is enabled, the response includes an evidence ID header:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{"model": "gpt-4", "messages": [...]}'

# Response headers include:
# X-Evidence-ID: uuid-of-evidence-record
# X-Request-ID: uuid-of-request
```

You can then query this evidence:

```bash
mercator evidence query --id "uuid-of-evidence-record"
```

### Compliance Automation

```bash
#!/bin/bash
# Daily compliance report

DATE=$(date -u +"%Y-%m-%d")
START="${DATE}T00:00:00Z"
END="${DATE}T23:59:59Z"

# Export evidence for the day
mercator evidence query \
  --time-range "${START}/${END}" \
  --format json \
  --output "evidence-${DATE}.json"

# Validate all signatures
mercator validate --file "evidence-${DATE}.json" \
  --report \
  --format json \
  > "validation-${DATE}.json"

# Upload to compliance archive
aws s3 cp "evidence-${DATE}.json" \
  "s3://compliance-bucket/evidence/${DATE}/"
```

## Performance Considerations

### Async Writing

Evidence recording uses async buffering to avoid blocking proxy requests:

```yaml
evidence:
  recorder:
    async_buffer: 1000  # Buffer up to 1000 records
    write_timeout: 5s   # Timeout for storage writes
```

### Database Sizing

SQLite storage recommendations:

| Request Volume | Retention | Estimated Size |
|----------------|-----------|----------------|
| 1K req/day | 30 days | ~100 MB |
| 10K req/day | 30 days | ~1 GB |
| 100K req/day | 30 days | ~10 GB |
| 1M req/day | 30 days | ~100 GB |

For high-volume deployments (>100K req/day), consider:
- Shorter retention periods
- More aggressive pruning schedules
- External archival system
- Database partitioning (future feature)

## Troubleshooting

### No Evidence Records

If no evidence is being created:

1. Check evidence is enabled in config:
   ```yaml
   evidence:
     enabled: true
   ```

2. Check database file exists and is writable:
   ```bash
   ls -l evidence.db
   ```

3. Check recorder logs:
   ```bash
   mercator run --verbose 2>&1 | grep evidence
   ```

### Storage Full

If evidence storage is filling up:

1. Check current size:
   ```bash
   du -sh evidence.db
   ```

2. Count records:
   ```bash
   mercator evidence query --format json | wc -l
   ```

3. Run manual pruning:
   ```bash
   mercator evidence retention prune
   ```

4. Adjust retention period:
   ```yaml
   evidence:
     retention:
       days: 30  # Reduce from 90
   ```

### Signature Validation Failures

If signature validation fails:

1. Check key configuration:
   ```bash
   ls -l /etc/mercator/keys/
   ```

2. Verify key permissions (private key must be 0600):
   ```bash
   chmod 600 /etc/mercator/keys/*_private.pem
   ```

3. Check key ID matches config:
   ```bash
   mercator keys list
   ```

## Roadmap

Planned enhancements for evidence recording:

- [ ] Complete integration with proxy handlers
- [ ] Distributed tracing correlation
- [ ] PostgreSQL backend support
- [ ] S3 backend for archival
- [ ] Evidence streaming to external systems
- [ ] Advanced query capabilities (full-text search)
- [ ] Evidence tampering detection
- [ ] Multi-signature support
- [ ] Hardware security module (HSM) integration

## See Also

- [CLI Reference](CLI.md#evidence-commands)
- [Configuration Reference](configuration/reference.md#evidence)
- [Security Best Practices](security-guide.md)
