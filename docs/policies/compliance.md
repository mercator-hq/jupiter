# Compliance Policies

Guide to implementing policies for regulatory compliance including HIPAA, GDPR, SOC2, and other frameworks.

## Table of Contents

- [HIPAA Compliance](#hipaa-compliance)
- [GDPR Compliance](#gdpr-compliance)
- [SOC2 Compliance](#soc2-compliance)
- [Data Residency](#data-residency)
- [Industry-Specific](#industry-specific)
- [Audit & Logging](#audit--logging)
- [Best Practices](#best-practices)

---

## HIPAA Compliance

**Use Case**: Healthcare applications handling Protected Health Information (PHI).

### HIPAA Policy Bundle

**File**: [docs/mpl/examples/08-compliance.yaml](../mpl/examples/08-compliance.yaml)

```yaml
version: "1.0"

policies:
  - name: "hipaa-compliance"
    description: "HIPAA compliance for healthcare applications"
    priority: 400
    rules:
      # 1. Block PHI in requests
      - condition: |
          request.messages[-1].content matches "(?i)(medical record|diagnosis|prescription|patient|SSN|date of birth)"
        action: "deny"
        reason: "Potential PHI detected. Healthcare data must be de-identified before processing."

      # 2. Require authorized users only
      - condition: |
          request.metadata.user_role not in ["healthcare_provider", "authorized_personnel"]
        action: "deny"
        reason: "HIPAA compliance: Only authorized healthcare personnel can submit requests."

      # 3. Comprehensive audit logging (required by HIPAA)
      - condition: "true"
        action: "log"
        log_level: "info"
        message: |
          HIPAA_AUDIT:
          User: {{request.metadata.user_id}}
          Role: {{request.metadata.user_role}}
          Action: {{request.messages[-1].role}}
          Timestamp: {{time.now}}
          Request_ID: {{request.id}}

      # 4. Enforce 7-year retention (HIPAA requirement)
      # This is configured in evidence storage, not policy

      # 5. Require encryption in transit (TLS)
      - condition: |
          request.metadata.connection_secure != true
        action: "deny"
        reason: "HIPAA compliance: Encrypted connection (TLS) required"
```

### HIPAA Requirements Checklist

| Requirement | Implementation | Policy |
|-------------|---------------|--------|
| PHI Protection | Block PHI in requests | PII detection |
| Access Control | Role-based authorization | User role check |
| Audit Logging | Log all access | Comprehensive logging |
| Encryption | TLS/mTLS | Connection security |
| Data Retention | 7 years minimum | Evidence retention |
| Business Associate Agreement | Provider contracts | Documentation |

### Configuration

```yaml
# config.yaml
evidence:
  retention_days: 2555  # 7 years for HIPAA

  signing_key_path: "/secure/path/signing-key.pem"  # Required

security:
  tls:
    enabled: true  # Required for HIPAA
    min_version: "1.3"

  mtls:
    enabled: true  # Recommended
```

---

## GDPR Compliance

**Use Case**: Applications processing EU personal data.

### GDPR Policy Bundle

```yaml
version: "1.0"

policies:
  - name: "gdpr-compliance"
    description: "GDPR compliance for EU user data"
    priority: 400
    rules:
      # 1. Data Minimization (Article 5)
      - condition: |
          request.messages[-1].content matches "\\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}\\b"
        action: "deny"
        reason: "GDPR Article 5: Personal data detected. Please anonymize before submitting."

      # 2. Purpose Limitation (Article 5)
      - condition: |
          request.metadata.processing_purpose not in ["service_delivery", "analytics", "support"]
        action: "deny"
        reason: "GDPR Article 5: Processing purpose must be specified and lawful."

      # 3. Data Subject Consent (Article 6 & 7)
      - condition: |
          request.metadata.user_consent != true
        action: "deny"
        reason: "GDPR Article 6: User consent required for data processing."

      # 4. Right to Erasure (Article 17)
      - condition: |
          request.metadata.erasure_request == true
        action: "log"
        log_level: "info"
        message: "GDPR_ERASURE_REQUEST: {{request.metadata.user_id}}"

      # 5. Data Portability (Article 20)
      - condition: |
          request.metadata.export_request == true
        action: "log"
        log_level: "info"
        message: "GDPR_EXPORT_REQUEST: {{request.metadata.user_id}}"

      # 6. Geographic Data Residency (EU only)
      - condition: |
          request.metadata.user_region == "EU" and
          provider.selected.region != "EU"
        action: "deny"
        reason: "GDPR: EU user data must be processed in EU"

      # 7. Audit Trail (Article 30)
      - condition: "true"
        action: "log"
        log_level: "info"
        message: |
          GDPR_AUDIT:
          User: {{request.metadata.user_id}}
          Purpose: {{request.metadata.processing_purpose}}
          Consent: {{request.metadata.user_consent}}
          Region: {{request.metadata.user_region}}
          Provider_Region: {{provider.selected.region}}
```

### GDPR Requirements Checklist

| GDPR Article | Requirement | Implementation |
|--------------|-------------|----------------|
| Article 5 | Data minimization | Block PII detection |
| Article 5 | Purpose limitation | Require purpose metadata |
| Article 6 | Lawful basis | User consent tracking |
| Article 17 | Right to erasure | Evidence deletion API |
| Article 20 | Data portability | Evidence export |
| Article 25 | Privacy by design | Secure defaults |
| Article 30 | Records of processing | Audit logging |
| Article 32 | Security measures | Encryption (TLS) |
| Article 44-50 | Cross-border transfers | Data residency |

### Configuration

```yaml
evidence:
  retention_days: 90  # GDPR: Reasonable period, not excessive

  # Support for erasure requests
  enable_deletion_api: true

security:
  tls:
    enabled: true  # Article 32: Security measures

routing:
  # Enforce EU data residency
  geographic_restrictions:
    eu_users: ["azure-eu-west", "gcp-eu-west"]
```

---

## SOC2 Compliance

**Use Case**: Service organizations demonstrating security controls.

### SOC2 Policy Bundle

```yaml
version: "1.0"

policies:
  - name: "soc2-compliance"
    description: "SOC2 Trust Service Criteria compliance"
    priority: 350
    rules:
      # CC1: Control Environment - Access Control
      - condition: |
          request.metadata.user_authenticated != true
        action: "deny"
        reason: "SOC2 CC1: Authentication required"

      # CC2: Communication - Audit Logging
      - condition: "true"
        action: "log"
        log_level: "info"
        message: |
          SOC2_AUDIT:
          User: {{request.metadata.user_id}}
          IP: {{request.metadata.client_ip}}
          Action: {{request.action}}
          Timestamp: {{time.now}}

      # CC3: Risk Assessment - Rate Limiting
      - condition: |
          request.metadata.user_rpm_current > 100
        action: "deny"
        reason: "SOC2 CC3: Rate limit exceeded (abuse prevention)"

      # CC6: Logical Access - Role-Based Access
      - condition: |
          request.model in ["gpt-4", "claude-3-opus"] and
          request.metadata.user_role != "admin"
        action: "deny"
        reason: "SOC2 CC6: Insufficient privileges for premium models"

      # CC7: System Operations - Monitoring
      - condition: |
          request.estimated_cost > 10.0
        action: "log"
        log_level: "warn"
        message: "SOC2_ALERT: High-cost request (${{request.estimated_cost}})"

      # CC8: Change Management - Policy Versioning
      # Tracked via Git integration

      # A1: Availability - Uptime Monitoring
      - condition: |
          provider.selected.health_status != "healthy"
        action: "deny"
        reason: "SOC2 A1: Provider unavailable"
```

### SOC2 Trust Service Criteria

| Criteria | Description | Implementation |
|----------|-------------|----------------|
| CC1 | Control Environment | Authentication, authorization |
| CC2 | Communication | Audit logs, notifications |
| CC3 | Risk Assessment | Rate limiting, anomaly detection |
| CC4 | Monitoring | Real-time metrics, alerts |
| CC5 | Control Activities | Policy enforcement |
| CC6 | Logical Access | RBAC, API keys |
| CC7 | System Operations | Health checks, failover |
| CC8 | Change Management | Git-based policies |
| A1 | Availability | High availability, SLA monitoring |
| C1 | Confidentiality | Encryption, access control |
| P1 | Privacy | PII protection, consent |

---

## Data Residency

**Use Case**: Ensure data stays within specific geographic regions.

### Data Residency Policy

**File**: [docs/mpl/examples/09-data-residency.yaml](../mpl/examples/09-data-residency.yaml)

```yaml
version: "1.0"

policies:
  - name: "data-residency"
    description: "Enforce geographic data residency requirements"
    priority: 400
    rules:
      # EU Data Residency (GDPR)
      - condition: |
          request.metadata.user_region == "EU"
        action: "route"
        provider: "azure-eu-west"
        log_message: "EU user routed to EU provider (data residency compliance)"

      # Block cross-border data transfer for EU
      - condition: |
          request.metadata.user_region == "EU" and
          provider.selected.region != "EU"
        action: "deny"
        reason: "Data residency violation: EU data cannot be processed outside EU (GDPR Article 44-50)"

      # US Data Residency
      - condition: |
          request.metadata.user_region == "US" and
          request.metadata.requires_us_residency == true
        action: "route"
        provider: "openai-us-east"
        log_message: "US data residency enforced"

      # China Data Residency
      - condition: |
          request.metadata.user_region == "CN"
        action: "route"
        provider: "local-china-provider"
        log_message: "China data residency enforced (Cybersecurity Law)"

      # Log all cross-border data transfers
      - condition: |
          request.metadata.user_region != provider.selected.region
        action: "log"
        log_level: "warn"
        message: "CROSS_BORDER_TRANSFER: User in {{request.metadata.user_region}}, data processed in {{provider.selected.region}}"
```

### Regional Compliance Matrix

| Region | Laws | Provider Requirements | Data Residency |
|--------|------|----------------------|----------------|
| EU | GDPR | EU-based or adequacy decision | Strict |
| US | CCPA, HIPAA | Varies by state/industry | Moderate |
| China | Cybersecurity Law | Local data centers mandatory | Strict |
| Russia | Data Localization Law | Russian servers required | Strict |
| Brazil | LGPD | Similar to GDPR | Moderate |
| India | IT Act, DPDP | Local storage for sensitive data | Moderate |

---

## Industry-Specific

### Financial Services (PCI-DSS, GLBA)

```yaml
policies:
  - name: "financial-compliance"
    rules:
      # Block credit card numbers (PCI-DSS)
      - condition: |
          request.messages[-1].content matches "\\b\\d{4}[\\s-]?\\d{4}[\\s-]?\\d{4}[\\s-]?\\d{4}\\b"
        action: "deny"
        reason: "PCI-DSS: Credit card numbers detected"

      # Require MFA for financial operations (GLBA)
      - condition: |
          request.metadata.operation_type == "financial" and
          request.metadata.mfa_verified != true
        action: "deny"
        reason: "GLBA: Multi-factor authentication required"

      # Log all financial operations
      - condition: |
          request.metadata.operation_type == "financial"
        action: "log"
        log_level: "info"
        message: "FINANCIAL_AUDIT: User {{request.metadata.user_id}}, Operation: {{request.metadata.operation_details}}"
```

### Government (FedRAMP, FISMA)

```yaml
policies:
  - name: "government-compliance"
    rules:
      # FedRAMP: Require government-approved providers
      - condition: |
          request.metadata.classification == "government" and
          provider.selected.fedramp_authorized != true
        action: "deny"
        reason: "FedRAMP: Only authorized providers can process government data"

      # FISMA: Log all access
      - condition: |
          request.metadata.classification == "government"
        action: "log"
        log_level: "info"
        message: "FISMA_AUDIT: {{request.metadata.user_id}} accessed {{request.metadata.system}}"
```

### Education (FERPA, COPPA)

```yaml
policies:
  - name: "education-compliance"
    rules:
      # FERPA: Block student records
      - condition: |
          request.messages[-1].content matches "(?i)(student ID|grade|transcript|disciplinary)"
        action: "deny"
        reason: "FERPA: Student educational records detected"

      # COPPA: Enhanced protection for minors
      - condition: |
          request.metadata.user_age < 13
        action: "log"
        log_level: "warn"
        message: "COPPA: Request from minor, enhanced protection applied"
```

---

## Audit & Logging

### Comprehensive Audit Trail

**File**: [docs/mpl/examples/20-audit-trail.yaml](../mpl/examples/20-audit-trail.yaml)

```yaml
version: "1.0"

policies:
  - name: "comprehensive-audit-trail"
    description: "Complete audit logging for compliance"
    priority: 50  # Low priority - log after other policies
    rules:
      # Log every request with full context
      - condition: "true"
        action: "log"
        log_level: "info"
        message: |
          AUDIT_TRAIL:
          RequestID: {{request.id}}
          Timestamp: {{time.now}}
          User: {{request.metadata.user_id}}
          Role: {{request.metadata.user_role}}
          Organization: {{request.metadata.org_id}}
          Model: {{request.model}}
          Provider: {{provider.selected.name}}
          Tokens: {{request.estimated_total_tokens}}
          Cost: ${{request.estimated_cost}}
          PolicyDecision: {{policy.decision.action}}
          MatchedPolicies: {{policy.decision.matched_policies}}
          ClientIP: {{request.metadata.client_ip}}
          UserAgent: {{request.metadata.user_agent}}
```

### Evidence Configuration

```yaml
evidence:
  enabled: true  # Required for compliance

  # Signing for tamper-evidence
  signing_key_path: "/secure/signing-key.pem"

  # Retention based on compliance requirements
  retention:
    days: 2555  # HIPAA: 7 years
    # days: 365  # SOC2: 1 year
    # days: 90   # GDPR: Reasonable period

    # Archive before deletion
    archive_before_delete: true
    archive_path: "/archive/evidence/"

  # Immutable evidence
  recorder:
    hash_request: true
    hash_response: true
```

### Query Evidence for Compliance

```bash
# Export all evidence for audit
mercator evidence query \
  --time-range "2025-01-01/2025-12-31" \
  --format json \
  --output compliance-audit-2025.json

# Query specific user activity
mercator evidence query \
  --user-id "user-123" \
  --time-range "last 30 days" \
  --format csv \
  --output user-123-activity.csv

# Verify evidence signatures
mercator validate \
  --time-range "last 7 days" \
  --report \
  --format json
```

---

## Best Practices

### 1. Layered Compliance

```yaml
policies:
  # Layer 1: Legal requirements (highest priority)
  - name: "gdpr-compliance"
    priority: 400

  # Layer 2: Industry standards
  - name: "pci-dss-compliance"
    priority: 350

  # Layer 3: Organizational policies
  - name: "company-security-policy"
    priority: 300

  # Layer 4: Audit logging (lowest priority)
  - name: "audit-trail"
    priority: 50
```

### 2. Regular Compliance Audits

```bash
# Monthly compliance report
mercator evidence query \
  --time-range "last 30 days" \
  --format json | \
  jq '{
    total_requests: length,
    denied_requests: [.[] | select(.policy_decision.action == "deny")] | length,
    policy_violations: [.[] | select(.policy_decision.violated_policies | length > 0)],
    users: [.[].request.metadata.user_id] | unique | length
  }'
```

### 3. Documentation

Maintain compliance documentation:
- Policy justifications
- Risk assessments
- Data flow diagrams
- Vendor assessments (LLM providers)
- Incident response procedures

### 4. Testing Compliance Policies

```yaml
# compliance-tests.yaml
tests:
  - name: "HIPAA: Should block PHI"
    request:
      messages:
        - role: "user"
          content: "Patient John Doe, DOB 01/01/1980"
    expected:
      action: "deny"
      reason_contains: "PHI"

  - name: "GDPR: Should enforce EU data residency"
    request:
      messages:
        - role: "user"
          content: "Hello"
      metadata:
        user_region: "EU"
    expected:
      provider_region: "EU"

  - name: "SOC2: Should require authentication"
    request:
      messages:
        - role: "user"
          content: "Test"
      metadata:
        user_authenticated: false
    expected:
      action: "deny"
      reason_contains: "Authentication required"
```

### 5. Compliance Checklist

- [ ] PII detection and blocking implemented
- [ ] Role-based access control configured
- [ ] Audit logging enabled and comprehensive
- [ ] Evidence retention meets regulatory requirements
- [ ] Evidence signing enabled for tamper-evidence
- [ ] TLS/mTLS enabled for encryption in transit
- [ ] Data residency enforced for applicable regions
- [ ] User consent tracking implemented (GDPR)
- [ ] Data subject rights supported (access, erasure, portability)
- [ ] Vendor contracts (Business Associate Agreements) signed
- [ ] Incident response procedures documented
- [ ] Regular compliance audits scheduled
- [ ] Staff training completed
- [ ] Compliance policies tested

---

## See Also

- [Content Safety Guide](content-safety.md) - PII protection
- [Data Residency Policy](../mpl/examples/09-data-residency.yaml)
- [Audit Trail Policy](../mpl/examples/20-audit-trail.yaml)
- [Security Guide](../SECURITY.md) - TLS/mTLS setup
- [Evidence Configuration](../configuration/reference.md#evidence-configuration)

---

## Compliance Resources

- **HIPAA**: https://www.hhs.gov/hipaa
- **GDPR**: https://gdpr.eu/
- **SOC2**: https://www.aicpa.org/soc2
- **PCI-DSS**: https://www.pcisecuritystandards.org/
- **FedRAMP**: https://www.fedramp.gov/
- **NIST**: https://www.nist.gov/cyberframework

**Disclaimer**: This documentation provides technical guidance for implementing compliance controls. It does not constitute legal advice. Consult with legal counsel and compliance experts for your specific regulatory requirements.
