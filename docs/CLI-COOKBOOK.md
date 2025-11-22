# CLI Cookbook - Common Recipes

Practical recipes for common Mercator Jupiter tasks.

## Table of Contents

1. [Local Development Setup](#recipe-1-local-development-setup)
2. [CI/CD Policy Validation](#recipe-2-cicd-policy-validation)
3. [Evidence Audit Workflow](#recipe-3-evidence-audit-workflow)
4. [Key Generation and Rotation](#recipe-4-key-generation-and-rotation)
5. [Load Testing and Performance Tuning](#recipe-5-load-testing-and-performance-tuning)
6. [Policy Testing Workflow](#recipe-6-policy-testing-workflow)
7. [Multi-Environment Deployment](#recipe-7-multi-environment-deployment)
8. [Evidence Export for SIEM Integration](#recipe-8-evidence-export-for-siem-integration)
9. [Troubleshooting with Verbose Logs](#recipe-9-troubleshooting-with-verbose-logs)
10. [Quick Policy Validation Loop](#recipe-10-quick-policy-validation-loop)

---

## Recipe 1: Local Development Setup

Set up Mercator for local development with minimal configuration.

### Step 1: Create minimal config

```bash
cat > config.yaml <<EOF
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "\${OPENAI_API_KEY}"
    timeout: 30s
    max_retries: 3

policy:
  mode: "file"
  file_path: "policies.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "evidence.db"

telemetry:
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: false
  tracing:
    enabled: false
EOF
```

### Step 2: Create sample policy

```bash
cat > policies.yaml <<EOF
version: "1.0"
policies:
  - name: "development-policy"
    description: "Basic policy for local development"
    rules:
      - condition: "true"
        action: "allow"
EOF
```

### Step 3: Validate configuration

```bash
# Dry run to validate config
mercator run --dry-run

# Expected output:
# ✓ Configuration valid
# ✓ Policy file loaded successfully
# ✓ Evidence backend initialized
# ✓ All checks passed
```

### Step 4: Start server

```bash
# Set API key
export OPENAI_API_KEY="sk-your-api-key-here"

# Start server
mercator run

# Or with verbose logging
mercator run --log-level debug --verbose
```

### Step 5: Test with curl

```bash
# Test health endpoint
curl http://localhost:8080/health

# Send test request
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${OPENAI_API_KEY}" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello, world!"}]
  }'
```

### Step 6: Verify evidence

```bash
# Query evidence records
mercator evidence query --limit 10

# Check evidence database
sqlite3 evidence.db "SELECT COUNT(*) FROM evidence_records;"
```

---

## Recipe 2: CI/CD Policy Validation

Integrate policy validation into your CI/CD pipeline.

### Bash Script

```bash
#!/bin/bash
# validate-policies.sh

set -e

echo "========================================="
echo "Mercator Policy Validation Pipeline"
echo "========================================="

# Step 1: Lint all policy files
echo "Step 1: Linting policy files..."
mercator lint --dir policies/ --strict --format json > lint-results.json

# Check exit code
if [ $? -ne 0 ]; then
  echo "❌ Policy validation failed"
  cat lint-results.json | jq '.'
  exit 1
fi

echo "✅ All policies valid"

# Step 2: Run policy tests
echo "Step 2: Running policy tests..."
mercator test \
  --policy policies/main.yaml \
  --tests tests/policy-tests.yaml \
  --coverage \
  --format json > test-results.json

if [ $? -ne 0 ]; then
  echo "❌ Policy tests failed"
  cat test-results.json | jq '.tests[] | select(.status == "failed")'
  exit 1
fi

echo "✅ All tests passed"

# Step 3: Generate coverage report
echo "Step 3: Generating coverage report..."
COVERAGE=$(cat test-results.json | jq -r '.coverage.overall')
echo "Coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE < 0.8" | bc -l) )); then
  echo "⚠️  Coverage below 80% threshold"
  exit 1
fi

echo "========================================="
echo "✅ All checks passed!"
echo "========================================="
```

### GitHub Actions Workflow

```yaml
# .github/workflows/validate-policies.yml
name: Validate Policies

on:
  pull_request:
    paths:
      - 'policies/**'
      - 'tests/**'
  push:
    branches:
      - main
      - develop

jobs:
  validate:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install Mercator
        run: |
          go install mercator-hq/jupiter/cmd/mercator@latest

      - name: Lint policies
        run: |
          mercator lint --dir policies/ --strict --format json > lint-results.json
          cat lint-results.json | jq '.'

      - name: Run policy tests
        run: |
          mercator test \
            --policy policies/main.yaml \
            --tests tests/policy-tests.yaml \
            --coverage \
            --format json > test-results.json

      - name: Check coverage
        run: |
          COVERAGE=$(cat test-results.json | jq -r '.coverage.overall')
          echo "Policy coverage: ${COVERAGE}%"
          if (( $(echo "$COVERAGE < 0.8" | bc -l) )); then
            echo "Coverage below 80% threshold"
            exit 1
          fi

      - name: Upload results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: policy-validation-results
          path: |
            lint-results.json
            test-results.json

      - name: Comment PR
        uses: actions/github-script@v6
        if: github.event_name == 'pull_request'
        with:
          script: |
            const fs = require('fs');
            const results = JSON.parse(fs.readFileSync('test-results.json'));
            const coverage = results.coverage.overall * 100;
            const passed = results.summary.passed;
            const total = results.summary.total;

            const body = `## Policy Validation Results

            ✅ Tests passed: ${passed}/${total}
            📊 Coverage: ${coverage.toFixed(1)}%
            `;

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
```

### GitLab CI Pipeline

```yaml
# .gitlab-ci.yml
stages:
  - validate
  - test

validate-policies:
  stage: validate
  image: golang:1.21
  script:
    - go install mercator-hq/jupiter/cmd/mercator@latest
    - mercator lint --dir policies/ --strict --format json > lint-results.json
    - cat lint-results.json
  artifacts:
    reports:
      junit: lint-results.json
    when: always

test-policies:
  stage: test
  image: golang:1.21
  script:
    - go install mercator-hq/jupiter/cmd/mercator@latest
    - mercator test --policy policies/main.yaml --tests tests/policy-tests.yaml --coverage --format json > test-results.json
    - COVERAGE=$(cat test-results.json | jq -r '.coverage.overall')
    - echo "Coverage $COVERAGE"
    - if (( $(echo "$COVERAGE < 0.8" | bc -l) )); then exit 1; fi
  artifacts:
    reports:
      junit: test-results.json
    when: always
```

---

## Recipe 3: Evidence Audit Workflow

Query and export evidence for compliance audits.

### Audit Report Script

```bash
#!/bin/bash
# audit-report.sh

# Configuration
START_DATE="2025-11-01T00:00:00Z"
END_DATE="2025-11-30T23:59:59Z"
OUTPUT_DIR="./audit-reports"

mkdir -p "${OUTPUT_DIR}"

echo "========================================="
echo "Mercator Evidence Audit Report"
echo "Period: ${START_DATE} to ${END_DATE}"
echo "========================================="

# Step 1: Query all evidence records
echo "Step 1: Querying evidence records..."
mercator evidence query \
  --time-range "${START_DATE}/${END_DATE}" \
  --format json \
  --output "${OUTPUT_DIR}/evidence-records.json"

RECORD_COUNT=$(cat "${OUTPUT_DIR}/evidence-records.json" | jq '.total')
echo "Found ${RECORD_COUNT} evidence records"

# Step 2: Generate summary report
echo "Step 2: Generating summary report..."
mercator evidence report \
  --time-range "${START_DATE}/${END_DATE}" \
  --output "${OUTPUT_DIR}/summary-report.txt"

cat "${OUTPUT_DIR}/summary-report.txt"

# Step 3: Validate all signatures
echo "Step 3: Validating cryptographic signatures..."
mercator validate \
  --time-range "${START_DATE}/${END_DATE}" \
  --report \
  --format json \
  --output "${OUTPUT_DIR}/validation-report.json"

VALID_COUNT=$(cat "${OUTPUT_DIR}/validation-report.json" | jq '.summary.valid')
TOTAL_COUNT=$(cat "${OUTPUT_DIR}/validation-report.json" | jq '.summary.total')
echo "Validated ${VALID_COUNT}/${TOTAL_COUNT} signatures"

# Step 4: Export by user
echo "Step 4: Exporting per-user evidence..."
cat "${OUTPUT_DIR}/evidence-records.json" | \
  jq -r '.records[].user_id' | \
  sort -u | \
  while read -r user_id; do
    mercator evidence query \
      --time-range "${START_DATE}/${END_DATE}" \
      --user-id "${user_id}" \
      --format json \
      --output "${OUTPUT_DIR}/evidence-${user_id}.json"
    echo "  Exported evidence for user: ${user_id}"
  done

# Step 5: Generate compliance report
echo "Step 5: Generating compliance summary..."
cat > "${OUTPUT_DIR}/compliance-summary.md" <<EOF
# Compliance Audit Report

**Period**: ${START_DATE} to ${END_DATE}
**Generated**: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

## Summary

- **Total Requests**: ${RECORD_COUNT}
- **Signatures Validated**: ${VALID_COUNT}/${TOTAL_COUNT}
- **Validation Rate**: $(echo "scale=2; ${VALID_COUNT}*100/${TOTAL_COUNT}" | bc)%

## Files Generated

1. \`evidence-records.json\` - All evidence records
2. \`summary-report.txt\` - Human-readable summary
3. \`validation-report.json\` - Signature validation results
4. \`evidence-{user_id}.json\` - Per-user evidence exports

## Attestation

This report was generated using Mercator Jupiter v1.0.0.
All evidence records have been cryptographically verified.

---
Generated by: $(whoami)@$(hostname)
EOF

cat "${OUTPUT_DIR}/compliance-summary.md"

echo "========================================="
echo "✅ Audit reports generated in: ${OUTPUT_DIR}"
echo "========================================="
```

### Run Monthly Audits

```bash
# Create cron job for monthly audits
# Add to crontab: crontab -e

# Run on 1st of each month at 2 AM
0 2 1 * * /path/to/audit-report.sh
```

---

## Recipe 4: Key Generation and Rotation

Generate and rotate cryptographic keys for evidence signing.

### Initial Key Generation

```bash
#!/bin/bash
# generate-keys.sh

KEY_ID="prod-$(date +%Y-%m)"
OUTPUT_DIR="/etc/mercator/keys"

echo "Generating keypair: ${KEY_ID}"

# Create keys directory
sudo mkdir -p "${OUTPUT_DIR}"
sudo chmod 755 "${OUTPUT_DIR}"

# Generate keypair
sudo mercator keys generate \
  --key-id "${KEY_ID}" \
  --output "${OUTPUT_DIR}"

# Verify permissions
echo "Verifying file permissions..."
ls -la "${OUTPUT_DIR}/${KEY_ID}_"*

# Backup public key
sudo cp "${OUTPUT_DIR}/${KEY_ID}_public.pem" \
  "/backup/keys/${KEY_ID}_public.pem"

echo "✅ Keys generated and backed up"
echo ""
echo "Update your config.yaml:"
echo "  evidence:"
echo "    signing_key_path: \"${OUTPUT_DIR}/${KEY_ID}_private.pem\""
```

### Key Rotation Workflow

```bash
#!/bin/bash
# rotate-keys.sh

set -e

OLD_KEY_ID="prod-2025-08"
NEW_KEY_ID="prod-2025-11"
OUTPUT_DIR="/etc/mercator/keys"
CONFIG_FILE="/etc/mercator/config.yaml"

echo "========================================="
echo "Key Rotation Workflow"
echo "Old Key: ${OLD_KEY_ID}"
echo "New Key: ${NEW_KEY_ID}"
echo "========================================="

# Step 1: Generate new keypair
echo "Step 1: Generating new keypair..."
sudo mercator keys generate \
  --key-id "${NEW_KEY_ID}" \
  --output "${OUTPUT_DIR}"

# Step 2: Backup new public key
echo "Step 2: Backing up public key..."
sudo cp "${OUTPUT_DIR}/${NEW_KEY_ID}_public.pem" \
  "/backup/keys/${NEW_KEY_ID}_public.pem"

# Step 3: Update configuration
echo "Step 3: Updating configuration..."
sudo sed -i.bak \
  "s|${OLD_KEY_ID}_private.pem|${NEW_KEY_ID}_private.pem|g" \
  "${CONFIG_FILE}"

# Step 4: Validate new config
echo "Step 4: Validating configuration..."
mercator run --config "${CONFIG_FILE}" --dry-run

# Step 5: Restart server
echo "Step 5: Restarting Mercator server..."
sudo systemctl restart mercator

# Wait for server to be ready
sleep 5

# Step 6: Verify server is running
echo "Step 6: Verifying server status..."
curl -f http://localhost:8080/health || {
  echo "❌ Server failed to start"
  sudo systemctl status mercator
  exit 1
}

# Step 7: Archive old keys (don't delete - needed for historical verification)
echo "Step 7: Archiving old keys..."
sudo mkdir -p "${OUTPUT_DIR}/archive"
sudo mv "${OUTPUT_DIR}/${OLD_KEY_ID}_"* "${OUTPUT_DIR}/archive/"

echo "========================================="
echo "✅ Key rotation complete"
echo ""
echo "Old keys archived to: ${OUTPUT_DIR}/archive/"
echo "New key in use: ${NEW_KEY_ID}"
echo "========================================="
```

### Automated Quarterly Rotation

```bash
# Add to crontab for quarterly rotation
# Rotate keys on Jan 1, Apr 1, Jul 1, Oct 1 at 3 AM
0 3 1 1,4,7,10 * /usr/local/bin/rotate-keys.sh
```

---

## Recipe 5: Load Testing and Performance Tuning

Benchmark your Mercator deployment.

### Basic Load Test

```bash
# Basic load test - 100 req/s for 1 minute
mercator benchmark \
  --target http://localhost:8080 \
  --duration 60s \
  --rate 100 \
  --concurrency 10
```

### Progressive Load Test

```bash
#!/bin/bash
# progressive-load-test.sh

TARGET="http://localhost:8080"
DURATION="60s"

echo "Progressive Load Test"
echo "====================="

# Test 1: Baseline (100 req/s)
echo "Test 1: Baseline (100 req/s)"
mercator benchmark \
  --target "${TARGET}" \
  --duration "${DURATION}" \
  --rate 100 \
  --concurrency 10 \
  --format json \
  --output "results-100rps.json"

# Test 2: Medium load (500 req/s)
echo "Test 2: Medium load (500 req/s)"
mercator benchmark \
  --target "${TARGET}" \
  --duration "${DURATION}" \
  --rate 500 \
  --concurrency 25 \
  --format json \
  --output "results-500rps.json"

# Test 3: High load (1000 req/s)
echo "Test 3: High load (1000 req/s)"
mercator benchmark \
  --target "${TARGET}" \
  --duration "${DURATION}" \
  --rate 1000 \
  --concurrency 50 \
  --format json \
  --output "results-1000rps.json"

# Test 4: Stress test (2000 req/s)
echo "Test 4: Stress test (2000 req/s)"
mercator benchmark \
  --target "${TARGET}" \
  --duration "${DURATION}" \
  --rate 2000 \
  --concurrency 100 \
  --format json \
  --output "results-2000rps.json"

# Analyze results
echo ""
echo "Results Summary:"
echo "================"

for file in results-*.json; do
  RATE=$(basename "$file" .json | cut -d'-' -f2)
  P99=$(cat "$file" | jq '.results.latency.p99_ms')
  SUCCESS_RATE=$(cat "$file" | jq '.results.requests.success_rate')
  echo "${RATE}: p99=${P99}ms, success=${SUCCESS_RATE}"
done
```

### Analyze Performance Results

```bash
#!/bin/bash
# analyze-performance.sh

RESULT_FILE="benchmark-results.json"

echo "Performance Analysis"
echo "===================="

# Extract key metrics
TOTAL=$(cat "${RESULT_FILE}" | jq '.results.requests.total')
SUCCESS=$(cat "${RESULT_FILE}" | jq '.results.requests.successful')
SUCCESS_RATE=$(cat "${RESULT_FILE}" | jq '.results.requests.success_rate')

P50=$(cat "${RESULT_FILE}" | jq '.results.latency.p50_ms')
P90=$(cat "${RESULT_FILE}" | jq '.results.latency.p90_ms')
P99=$(cat "${RESULT_FILE}" | jq '.results.latency.p99_ms')

THROUGHPUT=$(cat "${RESULT_FILE}" | jq '.results.throughput.requests_per_second')

echo "Requests:"
echo "  Total: ${TOTAL}"
echo "  Successful: ${SUCCESS}"
echo "  Success Rate: ${SUCCESS_RATE}"
echo ""
echo "Latency:"
echo "  p50: ${P50}ms"
echo "  p90: ${P90}ms"
echo "  p99: ${P99}ms"
echo ""
echo "Throughput: ${THROUGHPUT} req/s"
echo ""

# Check against SLO targets
echo "SLO Compliance:"
if (( $(echo "$P99 < 100" | bc -l) )); then
  echo "  ✅ p99 latency: ${P99}ms (target: <100ms)"
else
  echo "  ❌ p99 latency: ${P99}ms (target: <100ms)"
fi

if (( $(echo "$SUCCESS_RATE >= 0.999" | bc -l) )); then
  echo "  ✅ Success rate: ${SUCCESS_RATE} (target: >99.9%)"
else
  echo "  ❌ Success rate: ${SUCCESS_RATE} (target: >99.9%)"
fi
```

---

## Recipe 6: Policy Testing Workflow

Create and run comprehensive policy tests.

### Step 1: Create Test Cases

```yaml
# tests/policy-tests.yaml
tests:
  # Budget enforcement tests
  - name: "Block requests over budget"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Write a long essay..."
    metadata:
      user_id: "user-123"
      estimated_cost: 5.00
      user_budget_spent: 98.00
      user_budget_limit: 100.00
    expect:
      action: "block"
      reason: "Budget limit exceeded"

  - name: "Allow requests within budget"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Hello"
    metadata:
      user_id: "user-123"
      estimated_cost: 0.01
      user_budget_spent: 10.00
      user_budget_limit: 100.00
    expect:
      action: "allow"

  # Rate limiting tests
  - name: "Block requests exceeding rate limit"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Test"
    metadata:
      user_id: "user-456"
      user_request_count: 101
    expect:
      action: "block"
      reason: "Rate limit exceeded"

  - name: "Allow requests within rate limit"
    request:
      model: "gpt-3.5-turbo"
      messages:
        - role: "user"
          content: "Test"
    metadata:
      user_id: "user-456"
      user_request_count: 50
    expect:
      action: "allow"

  # Content filtering tests
  - name: "Block requests with sensitive content"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "SENSITIVE_CONTENT_HERE"
    metadata:
      user_id: "user-789"
    expect:
      action: "block"
      reason: "Content policy violation"

  - name: "Allow requests with safe content"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "What is the weather today?"
    metadata:
      user_id: "user-789"
    expect:
      action: "allow"
```

### Step 2: Run Tests

```bash
#!/bin/bash
# run-policy-tests.sh

POLICY_FILE="policies/main.yaml"
TEST_FILE="tests/policy-tests.yaml"

echo "Running policy tests..."

# Run tests with coverage
mercator test \
  --policy "${POLICY_FILE}" \
  --tests "${TEST_FILE}" \
  --coverage \
  --format json \
  --output test-results.json

# Check results
PASSED=$(cat test-results.json | jq '.summary.passed')
FAILED=$(cat test-results.json | jq '.summary.failed')
TOTAL=$(cat test-results.json | jq '.summary.total')
COVERAGE=$(cat test-results.json | jq '.coverage.overall')

echo ""
echo "Test Results:"
echo "  Passed: ${PASSED}/${TOTAL}"
echo "  Failed: ${FAILED}"
echo "  Coverage: $(echo "${COVERAGE} * 100" | bc)%"

# Show failed tests
if [ "${FAILED}" -gt 0 ]; then
  echo ""
  echo "Failed Tests:"
  cat test-results.json | jq -r '.tests[] | select(.status == "failed") | "  - \(.name): \(.error)"'
  exit 1
fi

echo "✅ All tests passed"
```

### Step 3: Watch Mode for Development

```bash
#!/bin/bash
# watch-policy-tests.sh

# Requires: fswatch (brew install fswatch)

echo "Watching for policy changes..."
echo "Press Ctrl+C to stop"
echo ""

fswatch -o policies/ tests/ | while read -r event; do
  clear
  echo "========================================="
  echo "Change detected, running tests..."
  echo "========================================="
  echo ""

  # Lint policies
  mercator lint --dir policies/ --strict

  if [ $? -eq 0 ]; then
    echo "✅ Policies valid"
    echo ""

    # Run tests
    mercator test \
      --policy policies/main.yaml \
      --tests tests/policy-tests.yaml \
      --coverage
  else
    echo "❌ Policy validation failed"
  fi

  echo ""
  echo "========================================="
  echo "Waiting for changes..."
  echo "========================================="
done
```

---

## Recipe 7: Multi-Environment Deployment

Manage different configs for dev/staging/prod.

### Directory Structure

```
configs/
├── base.yaml              # Shared config
├── dev.yaml               # Development overrides
├── staging.yaml           # Staging overrides
└── prod.yaml              # Production overrides
```

### Base Configuration

```yaml
# configs/base.yaml
proxy:
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

policy:
  mode: "file"
  validation:
    strict: true

evidence:
  enabled: true
  backend: "sqlite"
  retention_days: 90

telemetry:
  logging:
    format: "json"
  metrics:
    enabled: true
```

### Development Configuration

```yaml
# configs/dev.yaml
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"

policy:
  file_path: "policies/dev-policies.yaml"

evidence:
  sqlite:
    path: "evidence-dev.db"

telemetry:
  logging:
    level: "debug"
  metrics:
    enabled: false
  tracing:
    enabled: false
```

### Production Configuration

```yaml
# configs/prod.yaml
proxy:
  listen_address: "0.0.0.0:8080"
  tls:
    enabled: true
    cert_file: "/etc/mercator/tls/cert.pem"
    key_file: "/etc/mercator/tls/key.pem"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
  anthropic:
    base_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_API_KEY}"

policy:
  file_path: "/etc/mercator/policies/prod-policies.yaml"

evidence:
  sqlite:
    path: "/var/lib/mercator/evidence.db"
  signing_key_path: "/etc/mercator/keys/prod-key_private.pem"

telemetry:
  logging:
    level: "warn"
  metrics:
    enabled: true
    port: 9090
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
```

### Deployment Scripts

```bash
#!/bin/bash
# deploy.sh

ENVIRONMENT=$1

if [ -z "${ENVIRONMENT}" ]; then
  echo "Usage: ./deploy.sh <environment>"
  echo "Environments: dev, staging, prod"
  exit 1
fi

CONFIG_FILE="configs/${ENVIRONMENT}.yaml"

if [ ! -f "${CONFIG_FILE}" ]; then
  echo "Config file not found: ${CONFIG_FILE}"
  exit 1
fi

echo "Deploying to: ${ENVIRONMENT}"

# Validate config
echo "Validating configuration..."
mercator run --config "${CONFIG_FILE}" --dry-run

if [ $? -ne 0 ]; then
  echo "❌ Configuration validation failed"
  exit 1
fi

echo "✅ Configuration valid"

# Deploy based on environment
case "${ENVIRONMENT}" in
  dev)
    echo "Starting development server..."
    mercator run --config "${CONFIG_FILE}" --log-level debug
    ;;

  staging)
    echo "Deploying to staging..."
    scp "${CONFIG_FILE}" staging:/etc/mercator/config.yaml
    ssh staging "sudo systemctl restart mercator"
    ;;

  prod)
    echo "Deploying to production..."
    # Blue-green deployment
    scp "${CONFIG_FILE}" prod:/etc/mercator/config-new.yaml
    ssh prod "sudo systemctl stop mercator-blue"
    ssh prod "sudo mv /etc/mercator/config-new.yaml /etc/mercator/config.yaml"
    ssh prod "sudo systemctl start mercator-green"

    # Health check
    sleep 5
    curl -f http://prod:8080/health || {
      echo "❌ Health check failed, rolling back..."
      ssh prod "sudo systemctl start mercator-blue"
      ssh prod "sudo systemctl stop mercator-green"
      exit 1
    }

    echo "✅ Deployment successful"
    ;;
esac
```

---

## Recipe 8: Evidence Export for SIEM Integration

Export evidence in SIEM-compatible formats.

### Export to Elasticsearch

```bash
#!/bin/bash
# export-to-elasticsearch.sh

ES_HOST="http://elasticsearch:9200"
ES_INDEX="mercator-evidence"
TIME_RANGE="2025-11-20T00:00:00Z/2025-11-21T00:00:00Z"

echo "Exporting evidence to Elasticsearch..."

# Query evidence
mercator evidence query \
  --time-range "${TIME_RANGE}" \
  --format json \
  --output evidence.json

# Import to Elasticsearch
cat evidence.json | jq -c '.records[]' | \
  while read -r record; do
    ID=$(echo "$record" | jq -r '.id')
    curl -X POST "${ES_HOST}/${ES_INDEX}/_doc/${ID}" \
      -H 'Content-Type: application/json' \
      -d "${record}"
  done

echo "✅ Export complete"
```

### Export to Splunk (CSV format)

```bash
#!/bin/bash
# export-to-splunk.sh

TIME_RANGE="2025-11-20T00:00:00Z/2025-11-21T00:00:00Z"
OUTPUT_FILE="evidence.csv"

echo "Exporting evidence to CSV for Splunk..."

# Query evidence as JSON
mercator evidence query \
  --time-range "${TIME_RANGE}" \
  --format json \
  --output evidence.json

# Convert to CSV
echo "timestamp,user_id,model,provider,action,cost,tokens_prompt,tokens_completion" > "${OUTPUT_FILE}"

cat evidence.json | jq -r '.records[] |
  [
    .timestamp,
    .user_id,
    .request.model,
    .metadata.provider,
    .metadata.action,
    .metadata.cost,
    .metadata.tokens.prompt,
    .metadata.tokens.completion
  ] | @csv' >> "${OUTPUT_FILE}"

echo "✅ CSV export complete: ${OUTPUT_FILE}"

# Upload to Splunk (if splunk CLI installed)
# splunk add oneshot "${OUTPUT_FILE}" -index mercator -sourcetype csv
```

### Real-time Streaming to SIEM

```bash
#!/bin/bash
# stream-evidence.sh

SIEM_ENDPOINT="http://siem:8088/collector/event"
SIEM_TOKEN="your-siem-token"

echo "Streaming evidence to SIEM..."

# Monitor evidence database for new records
sqlite3 evidence.db <<EOF | \
  while read -r record; do
    curl -X POST "${SIEM_ENDPOINT}" \
      -H "Authorization: Splunk ${SIEM_TOKEN}" \
      -H "Content-Type: application/json" \
      -d "${record}"
  done

SELECT json_object(
  'timestamp', timestamp,
  'user_id', user_id,
  'model', json_extract(request, '$.model'),
  'action', json_extract(metadata, '$.action'),
  'cost', json_extract(metadata, '$.cost')
) as json_record
FROM evidence_records
WHERE timestamp > datetime('now', '-1 hour')
ORDER BY timestamp DESC;
EOF
```

---

## Recipe 9: Troubleshooting with Verbose Logs

Debug issues with detailed logging.

### Enable Verbose Logging

```bash
# Run server with verbose output
mercator run --verbose --log-level debug

# Capture logs to file
mercator run --verbose --log-level debug 2>&1 | tee debug.log

# Filter specific components
mercator run --verbose --log-level debug 2>&1 | grep "policy"
```

### Troubleshoot Configuration Issues

```bash
#!/bin/bash
# troubleshoot-config.sh

CONFIG_FILE="config.yaml"

echo "Troubleshooting configuration: ${CONFIG_FILE}"
echo "=============================================="

# Step 1: Check file exists
if [ ! -f "${CONFIG_FILE}" ]; then
  echo "❌ Config file not found: ${CONFIG_FILE}"
  exit 1
fi
echo "✅ Config file exists"

# Step 2: Validate YAML syntax
if ! yq eval '.' "${CONFIG_FILE}" > /dev/null 2>&1; then
  echo "❌ Invalid YAML syntax"
  yq eval '.' "${CONFIG_FILE}"
  exit 1
fi
echo "✅ YAML syntax valid"

# Step 3: Dry run
echo "Running configuration validation..."
mercator run --config "${CONFIG_FILE}" --dry-run --verbose

if [ $? -ne 0 ]; then
  echo "❌ Configuration validation failed"
  exit 1
fi

echo "✅ Configuration valid"
```

### Troubleshoot Policy Issues

```bash
#!/bin/bash
# troubleshoot-policy.sh

POLICY_FILE="policies.yaml"

echo "Troubleshooting policy: ${POLICY_FILE}"
echo "======================================="

# Step 1: Lint policy
echo "Linting policy..."
mercator lint --file "${POLICY_FILE}" --verbose --strict

if [ $? -ne 0 ]; then
  echo "❌ Policy validation failed"
  exit 1
fi
echo "✅ Policy valid"

# Step 2: Test policy against sample requests
echo "Testing policy with sample requests..."
mercator test \
  --policy "${POLICY_FILE}" \
  --tests tests/smoke-tests.yaml \
  --verbose

if [ $? -ne 0 ]; then
  echo "❌ Policy tests failed"
  exit 1
fi
echo "✅ Policy tests passed"
```

### Analyze Logs

```bash
#!/bin/bash
# analyze-logs.sh

LOG_FILE="debug.log"

echo "Analyzing logs: ${LOG_FILE}"
echo "============================="

# Count errors
ERROR_COUNT=$(grep -c "ERROR" "${LOG_FILE}")
echo "Errors: ${ERROR_COUNT}"

# Show unique error messages
echo "Unique error messages:"
grep "ERROR" "${LOG_FILE}" | awk '{print $NF}' | sort -u

# Show slow requests (>1s)
echo "Slow requests:"
grep "duration" "${LOG_FILE}" | awk '$NF > 1000 {print $0}'

# Show top users by request count
echo "Top users by requests:"
grep "user_id" "${LOG_FILE}" | awk '{print $NF}' | sort | uniq -c | sort -rn | head -10
```

---

## Recipe 10: Quick Policy Validation Loop

Rapid iteration during policy development.

### File Watcher Script

```bash
#!/bin/bash
# watch-and-lint.sh

# Requires: fswatch (brew install fswatch)

echo "Watching policies/ for changes..."
echo "Press Ctrl+C to stop"
echo ""

fswatch -o policies/ | while read -r event; do
  clear
  echo "========================================="
  echo "$(date): Policy change detected"
  echo "========================================="
  echo ""

  # Lint all policies
  mercator lint --dir policies/ --strict

  if [ $? -eq 0 ]; then
    echo ""
    echo "✅ All policies valid"

    # Run quick smoke tests
    if [ -f "tests/smoke-tests.yaml" ]; then
      echo ""
      echo "Running smoke tests..."
      mercator test \
        --policy policies/main.yaml \
        --tests tests/smoke-tests.yaml
    fi
  else
    echo ""
    echo "❌ Policy validation failed"
  fi

  echo ""
  echo "Watching for changes..."
done
```

### Make Script

```makefile
# Makefile

.PHONY: lint test watch dev

lint:
	@echo "Linting policies..."
	@mercator lint --dir policies/ --strict

test:
	@echo "Running policy tests..."
	@mercator test --policy policies/main.yaml --tests tests/policy-tests.yaml --coverage

watch:
	@./watch-and-lint.sh

dev:
	@echo "Starting development server..."
	@mercator run --config configs/dev.yaml --log-level debug

validate: lint test
	@echo "✅ All validation passed"
```

**Usage:**

```bash
# Lint policies
make lint

# Run tests
make test

# Watch for changes
make watch

# Start dev server
make dev

# Run all validations
make validate
```

---

## Additional Resources

- [CLI Reference](CLI.md) - Complete command documentation
- [Policy Language Guide](mpl/README.md) - MPL syntax and examples
- [Configuration Guide](../examples/config.yaml) - Config file reference
- [Observability Guide](observability-guide.md) - Metrics and monitoring

---

**Questions or issues?** Open an issue on [GitHub](https://github.com/mercator-hq/jupiter/issues).
