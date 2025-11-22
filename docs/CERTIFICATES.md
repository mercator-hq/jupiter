# Certificate Management Guide

This guide covers TLS certificate management for Mercator Jupiter, including obtaining certificates, validation, and troubleshooting.

## Table of Contents

- [Obtaining Production Certificates](#obtaining-production-certificates)
- [Certificate Commands](#certificate-commands)
- [Certificate Validation](#certificate-validation)
- [Certificate Information](#certificate-information)
- [Generating Test Certificates](#generating-test-certificates)
- [Client Certificates (mTLS)](#client-certificates-mtls)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Obtaining Production Certificates

### Let's Encrypt (Recommended)

Let's Encrypt provides free, automated certificates that are trusted by all major browsers and operating systems.

#### Installation

```bash
# Ubuntu/Debian
sudo apt-get install certbot

# macOS
brew install certbot

# RHEL/CentOS
sudo yum install certbot
```

#### Generate Certificate

```bash
# Standalone mode (requires port 80 to be available)
sudo certbot certonly --standalone \
  -d mercator.example.com \
  --non-interactive \
  --agree-tos \
  --email admin@example.com

# Certificates will be saved to:
# Certificate: /etc/letsencrypt/live/mercator.example.com/fullchain.pem
# Private Key: /etc/letsencrypt/live/mercator.example.com/privkey.pem
```

#### Configure Mercator

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/letsencrypt/live/mercator.example.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/mercator.example.com/privkey.pem"
    min_version: "1.3"
    cert_reload_interval: "5m"  # Automatically reload when renewed
```

### Certificate Renewal

Let's Encrypt certificates expire after 90 days. Mercator automatically reloads certificates without downtime.

#### Automatic Renewal

```bash
# Test renewal (dry run)
sudo certbot renew --dry-run

# Set up automatic renewal with cron
sudo crontab -e

# Add this line to check for renewal twice daily
0 0,12 * * * certbot renew --quiet
```

Mercator's certificate auto-reload feature (default: 5 minutes) will detect the renewed certificate and reload it automatically without requiring a restart.

### Commercial Certificate Authorities

For enterprise deployments, you may prefer commercial CAs:

- **DigiCert**: Extended validation, organization validation
- **GlobalSign**: Trusted worldwide, good for international deployments
- **Sectigo**: Cost-effective, good customer support

## Certificate Commands

Mercator provides three commands for certificate management:

### `mercator certs validate`

Validate certificate and private key, check expiration, and verify chain.

```bash
# Validate certificate and key match
mercator certs validate --cert server.crt --key server.key

# Validate certificate only (check expiration)
mercator certs validate --cert server.crt

# Validate certificate chain against CA
mercator certs validate --cert server.crt --ca ca.pem

# Validate both key and chain
mercator certs validate --cert server.crt --key server.key --ca ca.pem
```

**Output example:**
```
Validating certificate: server.crt

✓ Certificate and key match
✓ Certificate chain valid
✓ Certificate not expired (valid until 2026-11-21)

Certificate Details:
  Subject: mercator.example.com
  Organization: Acme Corp
  Issuer: Let's Encrypt Authority
  Serial: 1234567890abcdef
  Valid From: 2025-11-21T00:00:00Z
  Valid Until: 2026-11-21T00:00:00Z
  SANs (DNS): [mercator.example.com, *.mercator.example.com]
```

**Expiration warnings:**
```
⚠  Certificate expires in 29 days
```

### `mercator certs info`

Display detailed certificate information.

```bash
# Display in human-readable text format
mercator certs info server.crt

# Display in JSON format (for scripting)
mercator certs info --format json server.crt

# Save JSON output to file
mercator certs info --format json server.crt > cert-info.json
```

**Text output example:**
```
Certificate: server.crt

Subject:
  Common Name (CN): mercator.example.com
  Organization (O): Acme Corp
  Country (C): US

Issuer:
  Common Name (CN): Let's Encrypt Authority X3
  Organization (O): Let's Encrypt
  Country (C): US

Validity:
  Not Before: 2025-11-21 00:00:00 UTC
  Not After: 2026-11-21 00:00:00 UTC
  Duration: 365 days
  Status: ✓ Valid (364 days remaining)

Subject Alternative Names:
  - DNS: mercator.example.com
  - DNS: *.mercator.example.com
  - IP: 203.0.113.1

Key Usage:
  - Digital Signature
  - Key Encipherment

Extended Key Usage:
  - Server Authentication
  - Client Authentication

Algorithms:
  Signature Algorithm: SHA256-RSA
  Public Key Algorithm: RSA

Additional Information:
  Serial Number: 1234567890abcdef
  Version: 3
  Is CA: false
```

**JSON output:**
```json
{
  "subject": {
    "common_name": "mercator.example.com",
    "organization": ["Acme Corp"],
    "country": ["US"]
  },
  "issuer": {
    "common_name": "Let's Encrypt Authority X3",
    "organization": ["Let's Encrypt"],
    "country": ["US"]
  },
  "validity": {
    "not_before": "2025-11-21T00:00:00Z",
    "not_after": "2026-11-21T00:00:00Z",
    "duration_days": 365,
    "days_remaining": 364,
    "is_expired": false
  },
  "sans": {
    "dns": ["mercator.example.com", "*.mercator.example.com"],
    "ip": ["203.0.113.1"]
  },
  "key_usage": ["Digital Signature", "Key Encipherment"],
  "ext_key_usage": ["Server Authentication", "Client Authentication"],
  "signature_algorithm": "SHA256-RSA",
  "public_key_algorithm": "RSA",
  "serial_number": "1234567890abcdef",
  "version": 3,
  "is_ca": false
}
```

### `mercator certs generate`

Generate self-signed certificates for **testing only**. Do not use in production!

```bash
# Generate certificate for localhost
mercator certs generate --host localhost

# Generate with multiple hosts (DNS and IP)
mercator certs generate --host "localhost,127.0.0.1,app.local"

# Generate with custom parameters
mercator certs generate \
  --host "localhost,127.0.0.1" \
  --org "My Company" \
  --validity 365 \
  --key-size 2048 \
  --output certs/
```

**Options:**
- `--host`: Comma-separated list of DNS names and IP addresses
- `--org`: Organization name (default: "Mercator")
- `--validity`: Validity period in days (default: 365)
- `--key-size`: RSA key size - 2048, 3072, or 4096 bits (default: 2048)
- `--output`: Output directory (default: "certs")

**Output:**
```
Generating self-signed certificate...

Generating 2048-bit RSA private key...
Creating self-signed certificate...

Certificate Generation Summary:
================================
Hosts: localhost,127.0.0.1
  DNS Names: [localhost]
  IP Addresses: [127.0.0.1]
Organization: My Company
Validity: 365 days
Key Size: 2048 bits
Not Before: 2025-11-21 00:00:00 UTC
Not After: 2026-11-21 00:00:00 UTC

✓ Certificate generated: certs/cert.pem
✓ Private key generated: certs/key.pem

⚠️  WARNING: Self-signed certificates are for TESTING ONLY
    Do not use in production!

To use with Mercator, add to your config.yaml:
---
security:
  tls:
    enabled: true
    cert_file: "certs/cert.pem"
    key_file: "certs/key.pem"
    min_version: "1.3"
```

## Certificate Validation

### Validation Checks

The `mercator certs validate` command performs these checks:

1. **Certificate and Key Match**: Verifies the private key corresponds to the certificate
2. **Expiration Check**: Ensures the certificate is currently valid
3. **Chain Validation**: Verifies the certificate chain against a CA (if --ca provided)
4. **Expiration Warning**: Warns if certificate expires in less than 30 days

### Common Validation Scenarios

#### Pre-Deployment Validation

Before deploying to production, validate your certificates:

```bash
mercator certs validate \
  --cert /etc/letsencrypt/live/example.com/fullchain.pem \
  --key /etc/letsencrypt/live/example.com/privkey.pem
```

#### Scheduled Validation

Set up a cron job to check certificate expiration:

```bash
#!/bin/bash
# /usr/local/bin/check-mercator-cert.sh

mercator certs validate --cert /etc/mercator/cert.pem

if [ $? -ne 0 ]; then
  echo "Certificate validation failed!" | mail -s "Certificate Alert" admin@example.com
fi
```

Add to crontab:
```bash
# Check certificate daily at 9 AM
0 9 * * * /usr/local/bin/check-mercator-cert.sh
```

## Client Certificates (mTLS)

For service-to-service authentication or zero-trust networks, use mutual TLS (mTLS) with client certificates.

### Generating Client Certificates

#### 1. Create CA Certificate

```bash
# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate (valid for 10 years)
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
  -subj "/CN=Mercator CA/O=My Company/C=US"
```

#### 2. Generate Client Certificate

```bash
# Generate client private key
openssl genrsa -out client.key 2048

# Generate certificate signing request (CSR)
openssl req -new -key client.key -out client.csr \
  -subj "/CN=client-app-1/OU=engineering/O=My Company/C=US"

# Sign the CSR with the CA
openssl x509 -req -days 365 -in client.csr \
  -CA ca.crt -CAkey ca.key -set_serial 01 -out client.crt
```

#### 3. Configure Mercator for mTLS

```yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/mercator/certs/server.crt"
    key_file: "/etc/mercator/certs/server.key"

    mtls:
      enabled: true
      client_ca_file: "/etc/mercator/certs/ca.crt"
      client_auth_type: "require"  # require, request, or verify_if_given
      verify_client_cert: true
      identity_source: "subject.CN"  # Extract identity from CN
```

#### 4. Test Client Connection

```bash
curl --cert client.crt --key client.key --cacert ca.crt \
  https://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}'
```

### Client Authentication Types

| Type | Description | Use Case |
|------|-------------|----------|
| `require` | Client certificate required, reject if missing | High-security environments |
| `request` | Request certificate, allow if missing | Optional client auth |
| `verify_if_given` | Verify if provided, allow if not | Gradual mTLS rollout |

### Identity Extraction

Mercator can extract client identity from various certificate fields:

| Source | Description | Example |
|--------|-------------|---------|
| `subject.CN` | Common Name | "client-app-1" |
| `subject.OU` | Organizational Unit | "engineering" |
| `subject.O` | Organization | "Acme Corp" |
| `SAN` | First DNS name from SANs | "app1.example.com" |

**Policy example:**
```yaml
policies:
  - name: "client-auth"
    match:
      client_identity: "client-app-1"
    allow: true
```

## Troubleshooting

### Certificate Expired

**Error:**
```
✗ Certificate EXPIRED on 2025-10-21
Error: certificate expired
```

**Solution:**
1. Renew certificate with Let's Encrypt: `sudo certbot renew`
2. Or obtain new certificate from your CA
3. Mercator will automatically reload the new certificate (default: every 5 minutes)

### Certificate and Key Don't Match

**Error:**
```
✗ Certificate and key do NOT match
Error: tls: private key does not match public key
```

**Solution:**
- Verify you're using the correct key file for this certificate
- If using Let's Encrypt, ensure you're using `privkey.pem` (not `cert.pem`)
- Regenerate the certificate if the key was lost

### Certificate Chain Invalid

**Error:**
```
✗ Certificate chain invalid
Error: x509: certificate signed by unknown authority
```

**Solution:**
- Ensure you're using the full certificate chain (e.g., `fullchain.pem` for Let's Encrypt)
- Verify the CA certificate is correct and trusted
- For self-signed certificates, add the CA to the trust store

### File Permissions

**Error:**
```
Error: insecure permissions on /etc/mercator/server.key: 0644
Expected: 0600 or 0400
```

**Solution:**
```bash
# Fix private key permissions (owner read/write only)
chmod 0600 /etc/mercator/server.key

# Or make it read-only
chmod 0400 /etc/mercator/server.key
```

### TLS Handshake Failure

**Error:**
```
Error: tls: protocol version not supported
```

**Solution:**
- Check client TLS version support (Mercator defaults to TLS 1.3)
- If needed, lower minimum version in config:
  ```yaml
  security:
    tls:
      min_version: "1.2"  # Support TLS 1.2 and 1.3
  ```

### Certificate Not Reloading

**Issue:** Certificate renewed but Mercator still using old certificate

**Solution:**
1. Check reload interval in config:
   ```yaml
   security:
     tls:
       cert_reload_interval: "5m"  # Check every 5 minutes
   ```
2. Verify file permissions allow Mercator to read the certificate
3. Check logs for reload errors:
   ```bash
   grep "certificate reload" /var/log/mercator/mercator.log
   ```
4. Restart Mercator if necessary

## Best Practices

### Production Deployment

1. **Use Let's Encrypt** for free, automated certificates
2. **Enable automatic renewal** with certbot cron job
3. **Enable certificate auto-reload** (default: 5 minutes)
4. **Set minimum TLS version to 1.3** for best security
5. **Use strong key sizes** (RSA 2048+ bits, or ECDSA)
6. **Monitor certificate expiration** (set up alerts at 30 days)
7. **Test certificate validation** before deployment

### Security Checklist

- [ ] Certificates from trusted CA (not self-signed)
- [ ] Private key permissions: 0600 or 0400
- [ ] TLS 1.3 enabled (minimum version)
- [ ] Certificate auto-reload enabled
- [ ] Expiration monitoring configured
- [ ] Backup certificates and keys securely
- [ ] Document certificate renewal process
- [ ] Test TLS connection after deployment

### Renewal Timeline

| Days Before Expiry | Action |
|--------------------|--------|
| 90 days | Certificate issued |
| 30 days | Receive expiration warning |
| 30 days | Renew certificate |
| 0-30 days | Auto-reload new certificate |
| 0 days | Certificate expires (if not renewed) |

### Certificate Storage

**Secure locations:**
- `/etc/letsencrypt/live/` (Let's Encrypt managed)
- `/etc/mercator/certs/` (Manual management)
- `/var/secrets/` (Kubernetes)

**Permissions:**
```bash
# Certificate directory
chmod 0750 /etc/mercator/certs

# Certificate file (world-readable is OK for public certs)
chmod 0644 /etc/mercator/certs/server.crt

# Private key (owner only!)
chmod 0600 /etc/mercator/certs/server.key
```

### Backup Strategy

1. **Backup private keys** to secure, encrypted storage
2. **Never commit keys** to version control
3. **Document renewal process** for disaster recovery
4. **Test restore procedure** annually

## Additional Resources

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [OpenSSL Cookbook](https://www.feistyduck.com/books/openssl-cookbook/)
- [TLS 1.3 RFC](https://datatracker.ietf.org/doc/html/rfc8446)

## Support

For issues with Mercator certificate management:
- Open an issue: https://github.com/mercator-hq/jupiter/issues
- Documentation: https://docs.mercator.dev

For Let's Encrypt support:
- Community Forum: https://community.letsencrypt.org/
- Documentation: https://letsencrypt.org/docs/
