# Security Model

Comprehensive security architecture of Mercator Jupiter.

## Defense in Depth

Mercator Jupiter implements security at multiple layers:

```
┌────────────────────────────────────┐
│     Network Security (TLS/mTLS)    │
├────────────────────────────────────┤
│   Authentication (API Keys, mTLS)  │
├────────────────────────────────────┤
│   Authorization (Policy Engine)    │
├────────────────────────────────────┤
│   Input Validation (All Inputs)    │
├────────────────────────────────────┤
│   Output Validation (All Responses)│
├────────────────────────────────────┤
│   Audit Trail (Signed Evidence)    │
└────────────────────────────────────┘
```

---

## Network Security

### TLS Configuration

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/tls/cert.pem"
    key_file: "/etc/mercator/tls/key.pem"
    min_version: "1.3"  # TLS 1.3 minimum
    cipher_suites:  # Optional, secure defaults used
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"
```

### Mutual TLS (mTLS)

```yaml
security:
  mtls:
    enabled: true
    ca_file: "/etc/mercator/tls/ca.pem"
    client_auth: "require"  # Require client certificates
```

**Benefits**:
- **Authentication**: Verify client identity
- **Encryption**: Protect data in transit
- **Integrity**: Prevent tampering

---

## Authentication

### API Key Authentication

```go
// API Key structure
type APIKey struct {
    Key      string
    Name     string
    UserID   string
    Enabled  bool
    Created  time.Time
    Expires  *time.Time
    Metadata map[string]string
}
```

**Best Practices**:
- Store keys hashed (SHA-256)
- Use high-entropy keys (32+ bytes)
- Rotate keys regularly (90 days)
- Support key expiration
- Log all authentication attempts

### mTLS Certificate Authentication

Client certificates provide:
- **Strong authentication**: Cryptographic proof of identity
- **No shared secrets**: Private keys never transmitted
- **Granular access**: Per-client certificates

---

## Authorization

### Policy-Based Access Control

Every request evaluated against policies:

```yaml
- name: "department-access-control"
  rules:
    - condition: |
        request.user.department == "finance" &&
        request.model == "gpt-4"
      action: allow

    - condition: |
        request.user.department == "engineering" &&
        request.model in ["gpt-3.5-turbo", "claude-3-sonnet"]
      action: allow

    - condition: "true"  # Default deny
      action: deny
      reason: "Department not authorized for this model"
```

### Principle of Least Privilege

- **Default deny**: Nothing allowed unless explicitly permitted
- **Explicit grants**: All permissions must be declared
- **Regular review**: Audit policies for over-permissive rules

---

## Input Validation

### Request Validation

```go
// Validate all incoming requests
func ValidateRequest(req *Request) error {
    // Model validation
    if !isValidModel(req.Model) {
        return ErrInvalidModel
    }

    // Size limits
    if len(req.Prompt) > MaxPromptSize {
        return ErrPromptTooLarge
    }

    // Required fields
    if len(req.Messages) == 0 {
        return ErrNoMessages
    }

    // Content validation
    for _, msg := range req.Messages {
        if err := validateMessage(msg); err != nil {
            return err
        }
    }

    return nil
}
```

### Injection Prevention

- **Prompt injection detection**: Scan for bypass attempts
- **SQL injection**: Use parameterized queries
- **Path traversal**: Validate file paths
- **Command injection**: Never execute user input

---

## Output Validation

### Response Sanitization

```go
// Validate provider responses
func ValidateResponse(resp *Response) error {
    // Check for malformed data
    if resp.Model == "" {
        return ErrInvalidResponse
    }

    // Validate usage stats
    if resp.Usage.TotalTokens < 0 {
        return ErrInvalidUsage
    }

    // Content filtering
    for _, choice := range resp.Choices {
        if containsForbiddenContent(choice.Message.Content) {
            return ErrForbiddenContent
        }
    }

    return nil
}
```

---

## Cryptographic Evidence

### Signature Generation

```go
// Sign evidence with Ed25519
func SignEvidence(record *EvidenceRecord, privateKey ed25519.PrivateKey) ([]byte, error) {
    // Serialize to canonical JSON
    data, err := json.Marshal(record)
    if err != nil {
        return nil, err
    }

    // Generate signature
    signature := ed25519.Sign(privateKey, data)

    return signature, nil
}
```

### Verification

```go
// Verify evidence signature
func VerifyEvidence(record *EvidenceRecord, signature []byte, publicKey ed25519.PublicKey) bool {
    data, _ := json.Marshal(record)
    return ed25519.Verify(publicKey, data, signature)
}
```

### Key Management

- **Generation**: Cryptographically secure random
- **Storage**: Secure key storage (KMS, secrets manager)
- **Rotation**: Support for multiple active keys
- **Revocation**: Historical verification with old keys

---

## Secrets Management

### Environment Variables

```bash
# Never commit secrets
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-..."
export DB_PASSWORD="..."
export SIGNING_KEY_PATH="/secure/path/private.pem"
```

### Secret Redaction

```go
// Redact secrets from logs
func redactSecret(s string) string {
    if len(s) < 8 {
        return "***"
    }
    return s[:4] + "..." + s[len(s)-4:]
}

// Usage
log.Info("loaded config", "api_key", redactSecret(apiKey))
// Output: loaded config api_key=sk-1...def
```

---

## Rate Limiting & DoS Protection

### Request Rate Limiting

```yaml
limits:
  rate_limiting:
    enabled: true
    default_rpm: 60        # Requests per minute
    default_tpm: 100000    # Tokens per minute
    window_size: "1m"
```

### Protection Mechanisms

- **Token bucket**: Smooth rate limiting
- **Sliding window**: Precise limits
- **Burst allowance**: Handle traffic spikes
- **Per-user limits**: Prevent single-user abuse

---

## Audit Trail

### Evidence Records

Every request generates an immutable, signed evidence record:

```json
{
  "id": "evt_1234567890",
  "timestamp": "2025-11-22T10:30:00Z",
  "request_id": "req_abc123",
  "user_id": "user_xyz",
  "provider": "openai",
  "model": "gpt-4",
  "request": { },
  "response": { },
  "policy_decisions": [...],
  "token_usage": {...},
  "cost": 0.015,
  "signature": "base64-encoded-ed25519-signature"
}
```

### Retention & Compliance

- **Retention**: Configurable (default 90 days)
- **Immutability**: Records can't be modified
- **Verification**: Independent signature verification
- **Export**: JSON, CSV, JSONL formats
- **Archival**: Automatic old record pruning

---

## Threat Model

### Threats Addressed

| Threat | Mitigation |
|--------|------------|
| **Man-in-the-Middle** | TLS 1.3 encryption |
| **Unauthorized Access** | API key + mTLS authentication |
| **Policy Bypass** | Fail-secure design, all requests evaluated |
| **Prompt Injection** | Content analysis in policies |
| **Data Exfiltration** | Policy-based blocking, audit trail |
| **DoS Attack** | Rate limiting, resource limits |
| **Credential Theft** | Secrets not in logs, short-lived tokens |
| **Evidence Tampering** | Cryptographic signatures |
| **Insider Threat** | Audit trail, separation of duties |

### Threats Not Addressed (Out of Scope)

- **Physical security**: Deploy in secure datacenter
- **Social engineering**: Train users
- **Zero-day exploits**: Keep dependencies updated
- **Advanced persistent threats**: Deploy IDS/IPS

---

## Security Hardening

### Runtime Hardening

```yaml
# Systemd service hardening
[Service]
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadOnlyPaths=/etc /usr
ReadWritePaths=/opt/mercator/data
CapabilityBoundingSet=
AmbientCapabilities=
```

### Container Hardening

```dockerfile
# Run as non-root
USER mercator

# Read-only filesystem
--read-only --tmpfs /tmp

# Drop all capabilities
--cap-drop=ALL

# No new privileges
--security-opt=no-new-privileges:true
```

---

## Compliance

### GDPR

- **Right to access**: Evidence query API
- **Right to deletion**: Evidence pruning
- **Data minimization**: Only store necessary data
- **Encryption**: TLS + signed evidence

### HIPAA

- **Access controls**: Policy-based authorization
- **Audit trail**: Signed evidence records
- **Encryption**: At rest and in transit
- **Integrity**: Cryptographic signatures

### SOC 2

- **Security**: Multi-layer security model
- **Availability**: High availability deployment
- **Processing Integrity**: Policy enforcement
- **Confidentiality**: TLS + access controls
- **Privacy**: Data handling policies

---

## Security Checklist

- [ ] TLS 1.3 enabled for all connections
- [ ] mTLS configured for sensitive deployments
- [ ] API keys rotated regularly
- [ ] Secrets stored in vault/secrets manager
- [ ] Input validation on all requests
- [ ] Output validation on all responses
- [ ] Rate limiting enabled
- [ ] Evidence signing enabled
- [ ] Audit logs monitored
- [ ] Security updates applied
- [ ] Penetration testing performed
- [ ] Incident response plan documented

---

## Security Updates

### Keeping Secure

1. **Monitor advisories**: GitHub Security Advisories
2. **Update dependencies**: `go mod` monthly
3. **Scan for vulnerabilities**: `trivy` or `snyk`
4. **Review logs**: Look for suspicious patterns
5. **Rotate credentials**: Quarterly rotation
6. **Test disaster recovery**: Annual drills

---

## Reporting Security Issues

**DO NOT** create public GitHub issues for security vulnerabilities.

Instead:
- Email: security@mercator.io
- Include: Detailed description and reproduction steps
- Response: Within 48 hours
- Disclosure: Coordinated disclosure after patch

---

## See Also

- [Architecture Overview](overview.md)
- [Design Decisions](design-decisions.md)
- [Data Flow](data-flow.md)
- [Security Guide](../SECURITY.md)
- [Certificates Guide](../CERTIFICATES.md)
