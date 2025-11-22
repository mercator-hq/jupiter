# Security Guide

## Overview

Mercator Jupiter provides comprehensive security features designed for production deployments of LLM-powered applications. This guide covers all security aspects including transport security (TLS/mTLS), secret management, API key authentication, and best practices for secure deployment.

**Security Features:**
- **TLS 1.3 Transport Security** - Encrypted communication with automatic certificate management
- **Mutual TLS (mTLS)** - Client certificate authentication for zero-trust architectures
- **Secret Management** - Pluggable secret providers for secure credential storage
- **API Key Authentication** - Flexible authentication with multiple extraction sources
- **Certificate Auto-Reload** - Zero-downtime certificate renewal for Let's Encrypt

## Quick Start

Minimal secure configuration for production:

```yaml
security:
  # TLS Configuration
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt/live/example.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/example.com/privkey.pem"
    min_version: "1.3"
    cert_reload_interval: "5m"

  # Secret Management
  secrets:
    providers:
      - type: "file"
        path: "/var/secrets"
        watch: true
    cache:
      enabled: true
      ttl: "5m"

  # API Key Authentication
  authentication:
    enabled: true
    sources:
      - type: "header"
        name: "Authorization"
        scheme: "Bearer"

# Use secrets in provider configuration
providers:
  openai:
    api_key: "${secret:openai-api-key}"
```

## Table of Contents

1. [Transport Security (TLS/mTLS)](#transport-security)
2. [Secret Management](#secret-management)
3. [API Key Authentication](#api-key-authentication)
4. [Production Deployment Checklist](#production-deployment-checklist)
5. [Security Best Practices](#security-best-practices)
6. [Common Security Scenarios](#common-security-scenarios)
7. [Troubleshooting](#troubleshooting)
8. [Security Hardening](#security-hardening)

---

## Transport Security

### Basic TLS Configuration

**Always use TLS in production** to encrypt traffic between clients and Mercator Jupiter:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/certs/server.crt"
    key_file: "/etc/mercator/certs/server.key"
    min_version: "1.3"  # Always use TLS 1.3 in production
    cert_reload_interval: "5m"
```

⚠️ **Security Warning**: Never use self-signed certificates in production! Use Let's Encrypt or a commercial CA.

### TLS Version Selection

Mercator Jupiter supports TLS 1.2 and TLS 1.3:

```yaml
security:
  tls:
    min_version: "1.3"  # Recommended (most secure)
    # min_version: "1.2"  # Only if compatibility with older clients is required
```

**Security Implications:**
- **TLS 1.3** (Recommended): Fastest, most secure, removes obsolete cryptographic algorithms
- **TLS 1.2**: Supported for compatibility, but less efficient than TLS 1.3
- **TLS 1.0/1.1**: Rejected automatically (insecure, deprecated)

### Certificate Sources

#### Let's Encrypt (Recommended for Production)

Let's Encrypt provides free, automated certificates with 90-day validity:

```bash
# Install Certbot
sudo apt-get install certbot

# Obtain certificate
sudo certbot certonly --standalone -d your-domain.com

# Certificates are stored in:
# /etc/letsencrypt/live/your-domain.com/fullchain.pem
# /etc/letsencrypt/live/your-domain.com/privkey.pem
```

Configuration:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt/live/your-domain.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/your-domain.com/privkey.pem"
    min_version: "1.3"
    cert_reload_interval: "5m"  # Automatic reload when renewed
```

**Automatic Renewal:**

```bash
# Add cron job for automatic renewal
sudo crontab -e

# Renew twice daily (recommended by Let's Encrypt)
0 0,12 * * * certbot renew --quiet --deploy-hook "systemctl reload mercator"
```

With certificate auto-reload enabled, Mercator will automatically pick up renewed certificates without restart.

#### Commercial Certificate Authorities

For commercial CAs (DigiCert, GlobalSign, etc.):

1. Generate Certificate Signing Request (CSR)
2. Submit CSR to CA
3. Receive signed certificate
4. Configure Mercator with certificate and key

See [CERTIFICATES.md](CERTIFICATES.md) for detailed certificate management instructions.

### Cipher Suite Configuration

Mercator uses secure cipher suites by default. For custom configuration:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    min_version: "1.3"
    cipher_suites:
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"
```

⚠️ **Warning**: Only modify cipher suites if you have specific compliance requirements. The defaults are secure.

### Certificate Auto-Reload

Zero-downtime certificate renewal:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt/live/example.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/example.com/privkey.pem"
    cert_reload_interval: "5m"  # Check for changes every 5 minutes
```

**How it works:**
1. Mercator checks certificate file modification times every 5 minutes
2. If files have changed, certificates are reloaded automatically
3. New TLS connections use the updated certificate
4. Existing connections continue with the old certificate until they close
5. No server restart required

**Benefits:**
- Zero downtime during certificate renewal
- Automatic Let's Encrypt renewal support
- No manual intervention required

### Mutual TLS (mTLS)

mTLS provides client certificate authentication for service-to-service communication:

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/certs/server.crt"
    key_file: "/etc/mercator/certs/server.key"

    mtls:
      enabled: true
      client_ca_file: "/etc/mercator/certs/client-ca.pem"
      client_auth_type: "require"  # require|request|verify_if_given
      verify_client_cert: true
      identity_source: "subject.CN"  # subject.CN|subject.OU|subject.O|SAN
```

#### Client Authentication Modes

**1. `require` - Strict Mode (Recommended for Zero-Trust)**

```yaml
mtls:
  client_auth_type: "require"
  verify_client_cert: true
```

- Client certificate is mandatory
- Requests without valid client certificate are rejected
- Use for service-to-service communication in zero-trust networks

**2. `request` - Optional Mode**

```yaml
mtls:
  client_auth_type: "request"
  verify_client_cert: true
```

- Client certificate is requested but not required
- Allows both authenticated and unauthenticated requests
- Use when migrating to mTLS gradually

**3. `verify_if_given` - Permissive Mode**

```yaml
mtls:
  client_auth_type: "verify_if_given"
  verify_client_cert: true
```

- Client certificate is verified only if provided
- No certificate required
- Use for mixed environments (public + internal traffic)

#### Identity Extraction

Extract client identity from certificates for authorization:

```yaml
mtls:
  identity_source: "subject.CN"  # Options: subject.CN, subject.OU, subject.O, SAN
```

**Identity Sources:**
- `subject.CN` - Common Name (e.g., "service-a")
- `subject.OU` - Organizational Unit (e.g., "engineering")
- `subject.O` - Organization (e.g., "Acme Corp")
- `SAN` - First DNS name from Subject Alternative Names

**Example Certificate:**

```
Subject: CN=service-a, OU=engineering, O=Acme Corp
SAN: DNS:service-a.internal.example.com
```

- `subject.CN` → "service-a"
- `subject.OU` → "engineering"
- `subject.O` → "Acme Corp"
- `SAN` → "service-a.internal.example.com"

#### Use Cases for mTLS

**1. Zero-Trust Architecture**

```yaml
mtls:
  enabled: true
  client_auth_type: "require"  # No unauthenticated access
  verify_client_cert: true
```

**2. Service Mesh Integration**

```yaml
mtls:
  enabled: true
  client_auth_type: "require"
  identity_source: "subject.CN"  # Service name from cert
```

**3. Policy-Based Authorization**

Use client identity in policy rules:

```yaml
policies:
  engineering_only:
    conditions:
      - client_identity: "engineering"  # From OU field
    actions:
      - allow
```

**4. Compliance Requirements (PCI-DSS, HIPAA)**

```yaml
mtls:
  enabled: true
  client_auth_type: "require"
  verify_client_cert: true
```

For detailed mTLS setup instructions, see [CERTIFICATES.md](CERTIFICATES.md#client-certificates-mtls).

---

## Secret Management

Mercator Jupiter provides a pluggable secret management framework for securely storing and accessing sensitive credentials.

### Secret Providers

#### Environment Variables (Development)

Best for development and testing:

```bash
export MERCATOR_SECRET_OPENAI_API_KEY="sk-..."
export MERCATOR_SECRET_ANTHROPIC_API_KEY="sk-ant-..."
```

Configuration:

```yaml
security:
  secrets:
    providers:
      - type: "env"
        prefix: "MERCATOR_SECRET_"
    cache:
      enabled: true
      ttl: "5m"

providers:
  openai:
    api_key: "${secret:openai-api-key}"
  anthropic:
    api_key: "${secret:anthropic-api-key}"
```

**Name Conversion:**
- Secret name: `openai-api-key` (lowercase with hyphens)
- Environment variable: `MERCATOR_SECRET_OPENAI_API_KEY` (uppercase with underscores)

**Use Cases:**
- Local development
- CI/CD pipelines
- Container environments (Docker, Kubernetes)

#### File-Based Secrets (Production/Kubernetes)

Best for production and Kubernetes deployments:

```bash
# Create secrets directory
mkdir -p /var/secrets
chmod 700 /var/secrets

# Create secret files
echo "sk-..." > /var/secrets/openai-api-key
chmod 0600 /var/secrets/openai-api-key
```

Configuration:

```yaml
security:
  secrets:
    providers:
      - type: "file"
        path: "/var/secrets"
        watch: true  # Auto-reload on changes
    cache:
      enabled: true
      ttl: "5m"

providers:
  openai:
    api_key: "${secret:openai-api-key}"
```

⚠️ **Security Critical**: Secret files MUST have `0600` or `0400` permissions! Mercator will reject files with world-readable permissions.

**Directory Structure:**

```
/var/secrets/
├── openai-api-key      # 0600
├── anthropic-api-key   # 0600
└── custom-api-key      # 0600
```

**Kubernetes Secret Mounting:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mercator-secrets
type: Opaque
data:
  openai-api-key: c2stLi4u  # Base64 encoded
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: mercator
        volumeMounts:
        - name: secrets
          mountPath: /var/secrets
          readOnly: true
      volumes:
      - name: secrets
        secret:
          secretName: mercator-secrets
          defaultMode: 0600
```

**Use Cases:**
- Production deployments
- Kubernetes environments
- Docker Compose with mounted volumes
- File-based secret rotation

#### Cloud Provider Secrets (Future)

Mercator includes stubs for cloud secret managers:

```yaml
security:
  secrets:
    providers:
      - type: "aws-kms"
        region: "us-east-1"
        # Full implementation planned for future release

      - type: "gcp-kms"
        project: "my-project"
        # Full implementation planned for future release

      - type: "vault"
        address: "https://vault.example.com"
        # Full implementation planned for future release
```

Currently, these providers return "not implemented" errors. Use environment variables or file-based secrets.

### Secret Reference Syntax

Use `${secret:name}` syntax in configuration:

```yaml
providers:
  openai:
    api_key: "${secret:openai-api-key}"
    organization_id: "${secret:openai-org-id}"

  anthropic:
    api_key: "${secret:anthropic-api-key}"
```

**Resolution Process:**
1. Parse configuration file
2. Find all `${secret:name}` references
3. Query secret providers in order (first match wins)
4. Replace references with secret values
5. Cache secrets with TTL

### Secret Caching

Reduce secret provider load with caching:

```yaml
security:
  secrets:
    cache:
      enabled: true
      ttl: "5m"       # Cache duration
      max_size: 1000  # Maximum cached secrets
```

**Performance:**
- Cache hit: <1ms
- Cache miss (env): ~5ms
- Cache miss (file): ~10ms

**Cache Behavior:**
- LRU eviction when `max_size` is reached
- TTL-based expiration
- Thread-safe concurrent access
- Automatic invalidation on file changes (when watch enabled)

### Secret Rotation

**Zero-Downtime Rotation:**

1. Update secret file or environment variable
2. Wait for cache TTL to expire (or force refresh)
3. New requests use updated secret
4. No service restart required

**File-Based Rotation (with watch enabled):**

```bash
# Update secret file
echo "sk-new-key..." > /var/secrets/openai-api-key

# Mercator detects change and invalidates cache automatically
```

**Environment Variable Rotation:**

Environment variable changes require service restart:

```bash
export MERCATOR_SECRET_OPENAI_API_KEY="sk-new-key..."
systemctl restart mercator
```

**Best Practices:**
- Rotate secrets every 90 days
- Use file-based secrets for automated rotation
- Monitor secret access logs
- Test rotation process in staging first

---

## API Key Authentication

### Overview

API key authentication provides flexible, token-based authentication for incoming requests.

### Configuration

```yaml
security:
  authentication:
    enabled: true

    # Extraction sources (tried in order)
    sources:
      - type: "header"
        name: "Authorization"
        scheme: "Bearer"
      - type: "header"
        name: "X-API-Key"
        scheme: ""
      # - type: "query"    # NOT recommended for production
      #   name: "api_key"

    # API keys with metadata
    keys:
      - key: "sk-prod-1234567890abcdef"
        user_id: "service-account-eng"
        team_id: "team-engineering"
        enabled: true
        rate_limit: "10000/hour"

      - key: "sk-dev-abcdef1234567890"
        user_id: "dev-user-1"
        team_id: "team-engineering"
        enabled: true
        rate_limit: "1000/hour"
```

### Extraction Sources

#### Bearer Token (Recommended)

```yaml
sources:
  - type: "header"
    name: "Authorization"
    scheme: "Bearer"
```

Usage:

```bash
curl -H "Authorization: Bearer sk-prod-1234567890abcdef" \
  https://api.example.com/v1/chat/completions
```

**Best Practice**: Use Bearer tokens for public APIs and mobile apps.

#### Custom Header

```yaml
sources:
  - type: "header"
    name: "X-API-Key"
    scheme: ""
```

Usage:

```bash
curl -H "X-API-Key: sk-prod-1234567890abcdef" \
  https://api.example.com/v1/chat/completions
```

**Best Practice**: Use custom headers for service-to-service communication.

#### Query Parameter (NOT Recommended)

```yaml
sources:
  - type: "query"
    name: "api_key"
```

Usage:

```bash
curl "https://api.example.com/v1/chat/completions?api_key=sk-prod-1234567890abcdef"
```

⚠️ **Security Warning**: Query parameters are:
- Visible in URLs
- Logged by proxies and servers
- Cached by browsers
- Included in referrer headers

**Never use query parameters in production!** Use headers instead.

### Generating Secure API Keys

Always generate cryptographically random API keys:

```bash
# Linux/macOS
openssl rand -hex 32 | awk '{print "sk-" $0}'

# Output: sk-a1b2c3d4e5f6...

# Python
python3 -c "import secrets; print('sk-' + secrets.token_hex(32))"

# Go
go run -c 'package main; import ("crypto/rand"; "encoding/hex"; "fmt"); func main() { b := make([]byte, 32); rand.Read(b); fmt.Printf("sk-%s\n", hex.EncodeToString(b)) }'
```

**Key Requirements:**
- Minimum 32 bytes (256 bits) of entropy
- Cryptographically random (not pseudo-random)
- Unique per user/service
- Prefixed for easy identification (e.g., `sk-`, `prod-`, `dev-`)

### Key Management

#### User and Team Association

```yaml
keys:
  - key: "sk-prod-engineering-abc123"
    user_id: "service-account-eng"
    team_id: "team-engineering"
    enabled: true
    rate_limit: "10000/hour"
```

Use user and team IDs for:
- Authorization decisions in policies
- Rate limiting per user/team
- Usage tracking and billing
- Audit logging

#### Enabling/Disabling Keys

```yaml
keys:
  - key: "sk-revoked-key"
    user_id: "old-user"
    enabled: false  # Key is disabled
```

Disabled keys are rejected immediately without checking other metadata.

#### Dynamic Key Management

Mercator supports runtime key management:

1. **Add new key** - Add to configuration and reload
2. **Revoke key** - Set `enabled: false` and reload
3. **Remove key** - Remove from configuration and reload

For production deployments, consider implementing a key management API.

### Integration with Secret Management

Load API keys from secrets:

```yaml
security:
  authentication:
    enabled: true
    sources:
      - type: "header"
        name: "Authorization"
        scheme: "Bearer"

    keys:
      - key: "${secret:api-key-engineering}"
        user_id: "service-account-eng"
        team_id: "team-engineering"
        enabled: true
        rate_limit: "10000/hour"
```

Benefits:
- Keep sensitive keys out of configuration files
- Centralized secret management
- Automated rotation

### Security Best Practices

1. **Always use HTTPS** - API keys transmitted over HTTP can be intercepted
2. **Rotate keys every 90 days** - Limit exposure window
3. **Use different keys per environment** - Prevent production key leakage
4. **Monitor authentication failures** - Detect brute force attacks
5. **Implement rate limiting** - Prevent abuse
6. **Audit key usage** - Track who accessed what
7. **Revoke compromised keys immediately** - Set `enabled: false`

---

## Production Deployment Checklist

### Pre-Deployment

- [ ] **TLS Configuration**
  - [ ] Certificate from trusted CA (Let's Encrypt or commercial)
  - [ ] TLS 1.3 enabled (or TLS 1.2 minimum)
  - [ ] Certificate auto-reload configured (5-minute interval)
  - [ ] Certificate expiration monitoring enabled

- [ ] **Secret Management**
  - [ ] All secrets stored securely (not in config files)
  - [ ] File permissions validated (0600 for keys, 0400 for secrets)
  - [ ] Secret provider configured (file or env)
  - [ ] Secret caching enabled with appropriate TTL

- [ ] **API Key Authentication**
  - [ ] API keys are cryptographically random (32+ bytes)
  - [ ] Keys stored in secrets manager (not hardcoded)
  - [ ] User and team IDs configured for all keys
  - [ ] Disabled/test keys removed from configuration

- [ ] **Network Security**
  - [ ] HTTPS enforced for all endpoints (no HTTP)
  - [ ] Firewall rules configured
  - [ ] Rate limiting enabled
  - [ ] DDoS protection in place (CloudFlare, AWS Shield, etc.)

### Post-Deployment Validation

- [ ] **TLS Testing**
  ```bash
  # Test TLS connection
  curl -v https://your-domain.com

  # Validate certificate
  mercator certs validate --cert /path/to/cert

  # Check TLS version and cipher suite
  openssl s_client -connect your-domain.com:443 -tls1_3
  ```

- [ ] **API Key Authentication Testing**
  ```bash
  # Test valid key
  curl -H "Authorization: Bearer $VALID_KEY" https://your-domain.com/api

  # Test invalid key (should return 401)
  curl -H "Authorization: Bearer invalid-key" https://your-domain.com/api
  ```

- [ ] **Secret Resolution Testing**
  ```bash
  # Verify secrets are resolved correctly
  mercator config validate

  # Check logs for secret access errors
  journalctl -u mercator | grep -i secret
  ```

- [ ] **Certificate Expiration Monitoring**
  ```bash
  # Add monitoring script
  mercator certs info /path/to/cert | grep "days_remaining"

  # Set up alerts for <30 days
  ```

### Ongoing Maintenance

- [ ] **Daily**
  - Monitor authentication logs for failures
  - Check application error logs

- [ ] **Weekly**
  - Review rate limiting metrics
  - Check certificate expiration status

- [ ] **Monthly**
  - Review API key usage
  - Test secret rotation procedure
  - Verify backup and recovery procedures

- [ ] **Quarterly**
  - Rotate API keys
  - Update TLS configuration for new vulnerabilities
  - Test certificate renewal process
  - Review security audit logs

---

## Security Best Practices

### Defense in Depth

Implement multiple layers of security:

1. **Network Layer** - Firewall, VPC, security groups
2. **Transport Layer** - TLS 1.3, mTLS
3. **Application Layer** - API key authentication, rate limiting
4. **Data Layer** - Encryption at rest, secret management
5. **Audit Layer** - Logging, monitoring, alerting

### Principle of Least Privilege

Grant minimum necessary permissions:

```yaml
# Good: Specific permissions
policies:
  engineering_team:
    conditions:
      - team_id: "team-engineering"
    actions:
      - allow_models: ["gpt-4", "claude-3-opus"]

# Bad: Overly permissive
policies:
  everyone:
    actions:
      - allow_all
```

### Zero-Trust Architecture

Never trust, always verify:

```yaml
security:
  tls:
    mtls:
      enabled: true
      client_auth_type: "require"  # No unauthenticated access

  authentication:
    enabled: true  # All requests must authenticate
```

### Secrets Management Best Practices

1. **Never commit secrets to version control**
   ```bash
   # .gitignore
   *.key
   *.pem
   secrets/
   .env
   ```

2. **Rotate secrets regularly**
   - API keys: Every 90 days
   - TLS certificates: Every 365 days (or Let's Encrypt 90 days)
   - Signing keys: Every 180 days

3. **Use secret managers in production**
   - File-based secrets (Kubernetes)
   - AWS KMS / Secrets Manager
   - GCP Secret Manager
   - HashiCorp Vault

4. **Audit secret access**
   ```yaml
   # Enable secret access logging
   logging:
     level: "info"
     audit_secret_access: true
   ```

### TLS/mTLS Best Practices

1. **Always use TLS 1.3 in production**
   ```yaml
   tls:
     min_version: "1.3"
   ```

2. **Use strong key sizes**
   - RSA: 2048+ bits (prefer 4096 for long-lived keys)
   - ECDSA: P-256 or P-384

3. **Enable certificate auto-reload**
   ```yaml
   tls:
     cert_reload_interval: "5m"
   ```

4. **Monitor certificate expiration**
   ```bash
   # Alert when <30 days remaining
   mercator certs info /path/to/cert --format json | jq '.validity.days_remaining'
   ```

5. **Use mTLS for service-to-service communication**
   ```yaml
   tls:
     mtls:
       enabled: true
       client_auth_type: "require"
   ```

### Authentication Best Practices

1. **Strong API keys**
   - Minimum 32 bytes (256 bits)
   - Cryptographically random
   - Unique per user/service

2. **Key rotation**
   ```bash
   # Rotate every 90 days
   openssl rand -hex 32 | awk '{print "sk-" $0}'
   ```

3. **Monitor failed attempts**
   ```yaml
   logging:
     audit_auth_failures: true
   ```

4. **Implement rate limiting**
   ```yaml
   keys:
     - key: "sk-..."
       rate_limit: "1000/hour"
   ```

### Monitoring and Logging

1. **Log authentication attempts**
   - Successful authentications (user, IP, timestamp)
   - Failed authentications (reason, IP, timestamp)

2. **Alert on security events**
   - Certificate expiring soon (<30 days)
   - High authentication failure rate (>10% in 1 hour)
   - TLS handshake failures
   - Secret access errors

3. **Never log secrets or keys**
   ```go
   // Good: Redact secrets
   log.Info("loaded secret", "name", "openai-api-key", "value", "[REDACTED]")

   // Bad: Log secret values
   log.Info("loaded secret", "name", "openai-api-key", "value", secretValue)
   ```

### Compliance Considerations

**PCI-DSS:**
- TLS 1.2+ required
- Strong cryptography (2048-bit RSA minimum)
- Key rotation every 365 days
- Audit logging enabled

**HIPAA:**
- Encryption in transit (TLS 1.2+)
- Access controls (API keys, mTLS)
- Audit trails
- Automatic log-off (token expiration)

**SOC 2:**
- Encryption at rest and in transit
- Access controls and authentication
- Monitoring and alerting
- Incident response procedures

---

## Common Security Scenarios

### Scenario 1: Development Environment

**Requirements:**
- Local development
- Self-signed certificates acceptable
- Simple secret management
- Minimal authentication

**Configuration:**

```yaml
security:
  # TLS with self-signed cert
  tls:
    enabled: true
    cert_file: "./dev-certs/cert.pem"
    key_file: "./dev-certs/key.pem"
    min_version: "1.2"

  # Environment variable secrets
  secrets:
    providers:
      - type: "env"
        prefix: "MERCATOR_SECRET_"

  # Simple API key auth
  authentication:
    enabled: false  # Optional for local dev
```

**Setup:**

```bash
# Generate self-signed certificate
mercator certs generate --host localhost --output dev-certs/

# Set secrets
export MERCATOR_SECRET_OPENAI_API_KEY="sk-..."

# Run
mercator start --config dev-config.yaml
```

### Scenario 2: Staging Environment

**Requirements:**
- Let's Encrypt staging certificates
- File-based secrets
- Full authentication
- Similar to production

**Configuration:**

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt-staging/live/staging.example.com/fullchain.pem"
    key_file: "/etc/letsencrypt-staging/live/staging.example.com/privkey.pem"
    min_version: "1.3"
    cert_reload_interval: "5m"

  secrets:
    providers:
      - type: "file"
        path: "/var/secrets-staging"
        watch: true

  authentication:
    enabled: true
    sources:
      - type: "header"
        name: "Authorization"
        scheme: "Bearer"
```

### Scenario 3: Production Environment

**Requirements:**
- Production Let's Encrypt certificates
- File-based secrets (Kubernetes)
- Full authentication and authorization
- Maximum security

**Configuration:**

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt/live/api.example.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/api.example.com/privkey.pem"
    min_version: "1.3"
    cert_reload_interval: "5m"

  secrets:
    providers:
      - type: "file"
        path: "/var/secrets"
        watch: true
    cache:
      enabled: true
      ttl: "5m"
      max_size: 1000

  authentication:
    enabled: true
    sources:
      - type: "header"
        name: "Authorization"
        scheme: "Bearer"
```

### Scenario 4: High-Security Environment (Zero-Trust)

**Requirements:**
- mTLS everywhere
- Client certificate authentication
- Strong authentication and authorization
- Full audit logging

**Configuration:**

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/certs/server.crt"
    key_file: "/etc/mercator/certs/server.key"
    min_version: "1.3"

    mtls:
      enabled: true
      client_ca_file: "/etc/mercator/certs/client-ca.pem"
      client_auth_type: "require"  # Mandatory client certificates
      verify_client_cert: true
      identity_source: "subject.CN"

  secrets:
    providers:
      - type: "file"  # Or AWS KMS, GCP KMS, Vault
        path: "/var/secrets"
        watch: true

  authentication:
    enabled: true
    sources:
      - type: "header"
        name: "Authorization"
        scheme: "Bearer"

# Enable audit logging
logging:
  level: "info"
  audit_auth: true
  audit_secrets: true
```

### Scenario 5: Kubernetes Deployment

**Requirements:**
- Certificate from cert-manager
- Secrets from Kubernetes Secrets
- Service mesh integration
- Auto-scaling support

**Kubernetes Manifests:**

```yaml
# Certificate (cert-manager)
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: mercator-tls
spec:
  secretName: mercator-tls-secret
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - api.example.com

---
# Secrets
apiVersion: v1
kind: Secret
metadata:
  name: mercator-secrets
type: Opaque
data:
  openai-api-key: c2stLi4u
  anthropic-api-key: c2stYW50LWQuLi4=

---
# Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mercator
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mercator
  template:
    metadata:
      labels:
        app: mercator
    spec:
      containers:
      - name: mercator
        image: mercator-hq/jupiter:latest
        volumeMounts:
        - name: tls-certs
          mountPath: /etc/mercator/certs
          readOnly: true
        - name: secrets
          mountPath: /var/secrets
          readOnly: true
      volumes:
      - name: tls-certs
        secret:
          secretName: mercator-tls-secret
          defaultMode: 0600
      - name: secrets
        secret:
          secretName: mercator-secrets
          defaultMode: 0600
```

**Mercator Configuration:**

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/certs/tls.crt"
    key_file: "/etc/mercator/certs/tls.key"
    min_version: "1.3"
    cert_reload_interval: "5m"

  secrets:
    providers:
      - type: "file"
        path: "/var/secrets"
        watch: true
```

---

## Troubleshooting

### TLS Handshake Failures

**Symptom:**
```
TLS handshake error: tls: first record does not look like a TLS handshake
```

**Causes:**
1. Client connecting with HTTP instead of HTTPS
2. Incompatible TLS version
3. Certificate validation failure

**Solutions:**

```bash
# 1. Verify HTTPS is used
curl -v https://your-domain.com  # NOT http://

# 2. Check TLS version compatibility
openssl s_client -connect your-domain.com:443 -tls1_3

# 3. Validate certificate
mercator certs validate --cert /path/to/cert --key /path/to/key
```

### Certificate Not Reloading

**Symptom:**
New certificate not picked up after renewal.

**Causes:**
1. Certificate reload interval too long
2. File modification time not updated
3. Invalid new certificate

**Solutions:**

```bash
# 1. Check reload interval in config
grep cert_reload_interval config.yaml

# 2. Touch certificate file to update mtime
touch /etc/letsencrypt/live/example.com/fullchain.pem

# 3. Validate new certificate
mercator certs validate --cert /path/to/new/cert

# 4. Check Mercator logs
journalctl -u mercator | grep -i certificate
```

### Secret Not Found Errors

**Symptom:**
```
failed to get secret: secret not found: openai-api-key
```

**Causes:**
1. Secret file doesn't exist
2. Incorrect secret name
3. File permissions too restrictive
4. Secret provider not configured

**Solutions:**

```bash
# 1. Verify secret file exists
ls -la /var/secrets/openai-api-key

# 2. Check secret name (lowercase with hyphens)
# Config: ${secret:openai-api-key}
# File: /var/secrets/openai-api-key

# 3. Fix file permissions
chmod 0600 /var/secrets/openai-api-key

# 4. Verify secret provider configuration
grep -A 5 "secrets:" config.yaml
```

### API Key Authentication Failures

**Symptom:**
```
401 Unauthorized: invalid or missing API key
```

**Causes:**
1. Missing Authorization header
2. Incorrect API key
3. Disabled key
4. Wrong extraction source

**Solutions:**

```bash
# 1. Verify Authorization header format
curl -H "Authorization: Bearer sk-your-key" https://api.example.com

# 2. Check key is configured and enabled
grep -A 2 "sk-your-key" config.yaml
# enabled: true

# 3. Test with verbose output
curl -v -H "Authorization: Bearer sk-your-key" https://api.example.com

# 4. Check Mercator logs
journalctl -u mercator | grep -i "authentication failed"
```

### mTLS Client Certificate Errors

**Symptom:**
```
TLS handshake error: remote error: tls: bad certificate
```

**Causes:**
1. Client certificate not trusted by CA
2. Client certificate expired
3. Client certificate revoked
4. Wrong client CA file

**Solutions:**

```bash
# 1. Validate client certificate against CA
openssl verify -CAfile /etc/mercator/certs/client-ca.pem /path/to/client-cert.pem

# 2. Check certificate expiration
mercator certs info /path/to/client-cert.pem

# 3. Test mTLS connection
curl --cert /path/to/client-cert.pem \
     --key /path/to/client-key.pem \
     --cacert /etc/mercator/certs/ca-cert.pem \
     https://api.example.com

# 4. Verify server mTLS configuration
grep -A 5 "mtls:" config.yaml
```

### File Permission Errors

**Symptom:**
```
failed to load secret: file has insecure permissions: 0644
```

**Cause:**
Secret file is world-readable (security violation).

**Solution:**

```bash
# Fix file permissions
chmod 0600 /var/secrets/*

# Verify
ls -la /var/secrets/
# Should show: -rw------- (0600)
```

---

## Security Hardening

### System-Level Security

1. **Run as non-root user**
   ```bash
   useradd -r -s /bin/false mercator
   chown -R mercator:mercator /etc/mercator
   ```

2. **Restrict file permissions**
   ```bash
   chmod 0600 /etc/mercator/certs/*.key
   chmod 0644 /etc/mercator/certs/*.crt
   chmod 0600 /var/secrets/*
   ```

3. **Enable SELinux/AppArmor**
   ```bash
   # SELinux
   semanage fcontext -a -t httpd_sys_content_t "/etc/mercator(/.*)?"
   restorecon -Rv /etc/mercator
   ```

4. **Configure firewall**
   ```bash
   # Allow HTTPS only
   ufw allow 443/tcp
   ufw deny 80/tcp
   ufw enable
   ```

### Network Security

1. **Use private networks**
   - Deploy in VPC/private subnet
   - Use security groups to restrict access
   - Enable VPC flow logs

2. **Implement rate limiting**
   ```yaml
   rate_limiting:
     enabled: true
     global: "1000/minute"
     per_key: "100/minute"
   ```

3. **Enable DDoS protection**
   - Use CloudFlare, AWS Shield, or similar
   - Configure connection limits
   - Implement request throttling

4. **Restrict IP addresses**
   ```yaml
   # firewall or reverse proxy
   allow:
     - 10.0.0.0/8      # Internal network
     - 172.16.0.0/12   # Internal network
   deny:
     - 0.0.0.0/0       # Everything else
   ```

### Application Security

1. **Keep Mercator updated**
   ```bash
   # Check for updates
   mercator version --check

   # Update to latest
   go install mercator-hq/jupiter/cmd/mercator@latest
   ```

2. **Enable audit logging**
   ```yaml
   logging:
     level: "info"
     audit_auth: true
     audit_secrets: true
     audit_policy_decisions: true
   ```

3. **Implement security headers** (via reverse proxy)
   ```nginx
   add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
   add_header X-Content-Type-Options "nosniff" always;
   add_header X-Frame-Options "DENY" always;
   add_header X-XSS-Protection "1; mode=block" always;
   ```

4. **Regular security audits**
   ```bash
   # Run security scan
   gosec ./...

   # Check dependencies
   go list -m -u all

   # Audit configuration
   mercator config validate --strict
   ```

### Backup and Recovery

1. **Backup certificates and keys**
   ```bash
   tar -czf mercator-backup.tar.gz \
     /etc/mercator/certs/ \
     /var/secrets/ \
     /etc/mercator/config.yaml
   ```

2. **Test restore procedure**
   ```bash
   # Extract backup
   tar -xzf mercator-backup.tar.gz -C /tmp/restore

   # Validate certificates
   mercator certs validate --cert /tmp/restore/etc/mercator/certs/server.crt
   ```

3. **Document recovery process**
   - Certificate renewal steps
   - Secret rotation procedure
   - Configuration rollback process
   - Incident response plan

---

## Additional Resources

- **Certificate Management**: [CERTIFICATES.md](CERTIFICATES.md)
- **Configuration Reference**: [Configuration Guide](../README.md#configuration)
- **API Documentation**: [API Reference](../README.md#api)
- **GitHub Issues**: [Report Security Issues](https://github.com/mercator-hq/jupiter/security)
- **Community**: [Discord](https://discord.gg/mercator) | [Discussions](https://github.com/mercator-hq/jupiter/discussions)

---

## Reporting Security Vulnerabilities

If you discover a security vulnerability, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email: security@mercator.dev
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

We will respond within 48 hours and work with you to address the issue.

---

**Last Updated**: November 21, 2025
**Version**: 1.0.0
