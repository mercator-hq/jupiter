# Mercator CLI Reference

Complete command-line reference for Mercator Jupiter.

## Table of Contents

- [Global Flags](#global-flags)
- [Environment Variables](#environment-variables)
- [Commands](#commands)
  - [mercator run](#mercator-run)
  - [mercator lint](#mercator-lint)
  - [mercator test](#mercator-test)
  - [mercator evidence](#mercator-evidence)
  - [mercator benchmark](#mercator-benchmark)
  - [mercator validate](#mercator-validate)
  - [mercator keys](#mercator-keys)
  - [mercator version](#mercator-version)
  - [mercator completion](#mercator-completion)
- [Configuration](#configuration)
- [Exit Codes](#exit-codes)
- [Troubleshooting](#troubleshooting)

## Global Flags

These flags are available for all commands:

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--config` | `-c` | string | Path to config file (default: `config.yaml`) |
| `--verbose` | `-v` | bool | Enable verbose logging |
| `--help` | `-h` | bool | Show help for any command |

**Examples:**

```bash
# Use custom config file
mercator run --config /etc/mercator/config.yaml

# Enable verbose output
mercator lint --verbose --file policies.yaml

# Get help for any command
mercator evidence --help
```

## Environment Variables

Mercator respects the following environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `MERCATOR_CONFIG` | Override config file location | `/etc/mercator/config.yaml` |
| `MERCATOR_LOG_LEVEL` | Set log level | `debug`, `info`, `warn`, `error` |
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-...` |

**Example:**

```bash
export MERCATOR_CONFIG=/etc/mercator/config.yaml
export MERCATOR_LOG_LEVEL=debug
mercator run
```

---

## Commands

### mercator run

Start the Mercator proxy server.

**Usage:**

```bash
mercator run [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--listen` | `-l` | string | (from config) | Override listen address |
| `--log-level` | | string | `info` | Override log level (debug, info, warn, error) |
| `--dry-run` | | bool | false | Validate config without starting server |

**Examples:**

```bash
# Start with default config
mercator run

# Start with custom config
mercator run --config /etc/mercator/config.yaml

# Override listen address
mercator run --listen 0.0.0.0:8080

# Override log level
mercator run --log-level debug

# Validate config only (don't start server)
mercator run --dry-run

# Combine multiple flags
mercator run --config prod.yaml --listen 0.0.0.0:8080 --log-level warn
```

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | Server started successfully (or config validated with `--dry-run`) |
| 1 | Configuration error (invalid YAML, missing required fields) |
| 2 | Initialization error (failed to bind port, backend connection failed) |
| 3 | Server start error (runtime error during startup) |
| 130 | User interrupted (SIGINT/Ctrl+C) |

**Graceful Shutdown:**

The server handles shutdown signals gracefully:

- `SIGINT` (Ctrl+C): Graceful shutdown with 30-second timeout
- `SIGTERM`: Graceful shutdown with 30-second timeout

During shutdown:
1. Stop accepting new connections
2. Wait for in-flight requests to complete (up to 30s)
3. Close provider connections
4. Flush evidence records to storage
5. Exit

**Output Example:**

```
2025-11-20T10:00:00Z INFO Starting Mercator Jupiter proxy server
2025-11-20T10:00:00Z INFO Configuration loaded from config.yaml
2025-11-20T10:00:00Z INFO Policy engine initialized mode=file policies=5
2025-11-20T10:00:00Z INFO Evidence store initialized backend=sqlite path=evidence.db
2025-11-20T10:00:00Z INFO Providers initialized count=2 providers=[openai anthropic]
2025-11-20T10:00:00Z INFO Server listening address=127.0.0.1:8080
```

---

### mercator lint

Validate MPL policy files for syntax and semantic errors.

**Usage:**

```bash
mercator lint [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--file` | `-f` | string | | Policy file to validate (use OR `--dir`) |
| `--dir` | `-d` | string | | Directory of policy files (use OR `--file`) |
| `--strict` | | bool | false | Treat warnings as errors |
| `--format` | | string | `text` | Output format: `text`, `json` |

**Examples:**

```bash
# Lint single file
mercator lint --file policies.yaml

# Lint directory
mercator lint --dir policies/

# Strict mode (warnings become errors)
mercator lint --file policies.yaml --strict

# JSON output for CI/CD
mercator lint --file policies.yaml --format json

# Verbose output with line numbers
mercator lint --file policies.yaml --verbose
```

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | No errors (or only warnings in non-strict mode) |
| 1 | Syntax errors (invalid YAML, unknown fields) |
| 2 | Semantic errors (invalid conditions, type mismatches) |
| 3 | Rule conflicts (overlapping or contradictory rules) |

**Output Formats:**

**Text (default):**

```
Validating policies.yaml...
✓ Syntax valid
✓ YAML structure correct
✓ All policies have required fields
✓ All rules have valid conditions
⚠ Warning: Rule "rate-limit" may conflict with "budget-enforcement"

Summary: 0 error(s), 1 warning(s)
```

**JSON:**

```json
{
  "file": "policies.yaml",
  "valid": true,
  "errors": [],
  "warnings": [
    {
      "line": 15,
      "column": 3,
      "message": "Rule 'rate-limit' may conflict with 'budget-enforcement'",
      "severity": "warning"
    }
  ],
  "summary": {
    "errors": 0,
    "warnings": 1,
    "policies": 3,
    "rules": 7
  }
}
```

**Common Validation Errors:**

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid YAML syntax` | YAML parsing failed | Check for indentation, quotes, colons |
| `unknown field` | Field not in schema | Remove or rename field |
| `missing required field` | Required field missing | Add missing field (e.g., `name`, `rules`) |
| `invalid condition` | Expression syntax error | Fix condition syntax |
| `type mismatch` | Wrong type for field | Use correct type (string, number, bool) |

---

### mercator test

Run policy unit tests.

**Usage:**

```bash
mercator test [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--policy` | `-p` | string | *required* | Policy file to test |
| `--tests` | `-t` | string | *required* | Test file with test cases |
| `--coverage` | | bool | false | Generate coverage report |
| `--format` | | string | `text` | Output format: `text`, `json` |
| `--fail-fast` | | bool | false | Stop on first failure |

**Examples:**

```bash
# Run tests
mercator test --policy policies.yaml --tests policy-tests.yaml

# Run with coverage
mercator test --policy policies.yaml --tests tests.yaml --coverage

# JSON output for CI/CD
mercator test --policy policies.yaml --tests tests.yaml --format json

# Stop on first failure
mercator test --policy policies.yaml --tests tests.yaml --fail-fast
```

**Test File Format:**

```yaml
# policy-tests.yaml
tests:
  - name: "Block requests over budget"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Hello"
    metadata:
      user_id: "user-123"
      user_budget_spent: 98.00
      user_budget_limit: 100.00
      estimated_cost: 5.00
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
      user_budget_spent: 10.00
      user_budget_limit: 100.00
      estimated_cost: 0.01
    expect:
      action: "allow"
```

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | All tests passed |
| 1 | Test file or policy file invalid |
| 2 | Policy compilation failed |
| 3 | One or more tests failed |

**Output Formats:**

**Text (default):**

```
Running tests from policy-tests.yaml...

✓ Block requests over budget (12ms)
✓ Allow requests within budget (8ms)
✗ Rate limit enforcement (15ms)
  Expected: action=block
  Got:      action=allow

Summary: 2 passed, 1 failed, 3 total (35ms)

Coverage:
  budget-enforcement: 100% (2/2 rules tested)
  rate-limiting:      50%  (1/2 rules tested)
  Overall:            75%
```

**JSON:**

```json
{
  "summary": {
    "total": 3,
    "passed": 2,
    "failed": 1,
    "duration_ms": 35
  },
  "tests": [
    {
      "name": "Block requests over budget",
      "status": "passed",
      "duration_ms": 12
    },
    {
      "name": "Allow requests within budget",
      "status": "passed",
      "duration_ms": 8
    },
    {
      "name": "Rate limit enforcement",
      "status": "failed",
      "duration_ms": 15,
      "error": "Expected action=block, got action=allow"
    }
  ],
  "coverage": {
    "overall": 0.75,
    "policies": {
      "budget-enforcement": 1.0,
      "rate-limiting": 0.5
    }
  }
}
```

---

### mercator evidence

Query and export evidence records.

**Usage:**

```bash
mercator evidence <subcommand> [flags]
```

**Subcommands:**

- `query` - Query evidence records
- `report` - Generate summary report

#### mercator evidence query

Query evidence records from storage.

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--time-range` | | string | | Time range in ISO 8601 format: `START/END` |
| `--user-id` | | string | | Filter by user ID |
| `--request-id` | | string | | Filter by request ID |
| `--model` | | string | | Filter by model name |
| `--limit` | | int | 100 | Maximum number of records |
| `--offset` | | int | 0 | Offset for pagination |
| `--format` | | string | `text` | Output format: `text`, `json` |
| `--output` | `-o` | string | stdout | Output file path |

**Examples:**

```bash
# Query by time range
mercator evidence query \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

# Filter by user
mercator evidence query --user-id "user-123" --limit 50

# Filter by model
mercator evidence query --model "gpt-4" --time-range "2025-11-20T00:00:00Z/2025-11-21T00:00:00Z"

# Export to JSON file
mercator evidence query \
  --time-range "2025-11-01T00:00:00Z/2025-12-01T00:00:00Z" \
  --format json \
  --output november-evidence.json

# Pagination
mercator evidence query --limit 100 --offset 0
mercator evidence query --limit 100 --offset 100
```

**Output Format (Text):**

```
Evidence Records (2025-11-20)

ID: req-abc123
Timestamp: 2025-11-20T10:15:30Z
User: user-123
Model: gpt-4
Provider: openai
Action: allow
Cost: $0.0042
Tokens: 150 prompt, 75 completion
Signature: verified ✓

ID: req-def456
Timestamp: 2025-11-20T10:16:45Z
User: user-456
Model: claude-3-opus
Provider: anthropic
Action: block
Reason: Budget limit exceeded
Signature: verified ✓

Total: 2 records
```

**Output Format (JSON):**

```json
{
  "query": {
    "time_range": "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z",
    "filters": {},
    "limit": 100,
    "offset": 0
  },
  "records": [
    {
      "id": "req-abc123",
      "timestamp": "2025-11-20T10:15:30Z",
      "user_id": "user-123",
      "request": {
        "model": "gpt-4",
        "messages": [...]
      },
      "response": {
        "choices": [...]
      },
      "metadata": {
        "provider": "openai",
        "action": "allow",
        "cost": 0.0042,
        "tokens": {
          "prompt": 150,
          "completion": 75
        }
      },
      "signature": "base64-encoded-signature",
      "verified": true
    }
  ],
  "total": 2
}
```

#### mercator evidence report

Generate summary report of evidence records.

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--time-range` | string | Time range in ISO 8601 format |
| `--output` | string | Output file path |

**Example:**

```bash
mercator evidence report \
  --time-range "2025-11-01T00:00:00Z/2025-12-01T00:00:00Z" \
  --output november-report.txt
```

**Report Format:**

```
Evidence Summary Report
Generated: 2025-11-20T15:30:00Z
Time Range: 2025-11-01T00:00:00Z to 2025-12-01T00:00:00Z

Total Requests: 1,245
  Allowed: 1,180 (94.8%)
  Blocked: 65 (5.2%)

By Provider:
  OpenAI:     850 (68.3%)
  Anthropic:  395 (31.7%)

By Model:
  gpt-4:          450 (36.1%)
  gpt-3.5-turbo:  400 (32.1%)
  claude-3-opus:  395 (31.7%)

Total Cost: $1,234.56
  OpenAI:     $856.78
  Anthropic:  $377.78

Top Users by Requests:
  user-123: 145 requests ($123.45)
  user-456: 98 requests ($87.65)
  user-789: 87 requests ($76.54)

Policy Actions:
  Budget enforcement: 32 blocks
  Rate limiting: 23 blocks
  Content filtering: 10 blocks

Signature Verification: 1,245/1,245 verified (100%)
```

---

### mercator benchmark

Load test the Mercator proxy.

**Usage:**

```bash
mercator benchmark [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--target` | | string | *required* | Target proxy URL |
| `--duration` | | duration | `60s` | Test duration |
| `--rate` | | int | 100 | Requests per second |
| `--concurrency` | | int | 10 | Concurrent workers |
| `--format` | | string | `text` | Output format: `text`, `json` |
| `--output` | `-o` | string | stdout | Output file path |

**Examples:**

```bash
# Basic load test
mercator benchmark --target http://localhost:8080 --duration 60s

# High-load stress test
mercator benchmark \
  --target http://localhost:8080 \
  --duration 300s \
  --rate 500 \
  --concurrency 50

# Export results to JSON
mercator benchmark \
  --target http://localhost:8080 \
  --duration 60s \
  --format json \
  --output benchmark-results.json
```

**Output Format (Text):**

```
Benchmark Results
Target: http://localhost:8080
Duration: 60s
Rate: 100 req/s
Concurrency: 10

Requests:
  Total: 6,000
  Successful: 5,950 (99.2%)
  Failed: 50 (0.8%)

Latency:
  Min: 12ms
  Max: 456ms
  Mean: 45ms
  p50: 42ms
  p90: 78ms
  p95: 105ms
  p99: 234ms

Throughput: 99.2 req/s

Error Distribution:
  Timeout: 30 (0.5%)
  Connection refused: 20 (0.3%)
```

**Output Format (JSON):**

```json
{
  "config": {
    "target": "http://localhost:8080",
    "duration_seconds": 60,
    "rate": 100,
    "concurrency": 10
  },
  "results": {
    "requests": {
      "total": 6000,
      "successful": 5950,
      "failed": 50,
      "success_rate": 0.992
    },
    "latency": {
      "min_ms": 12,
      "max_ms": 456,
      "mean_ms": 45,
      "p50_ms": 42,
      "p90_ms": 78,
      "p95_ms": 105,
      "p99_ms": 234
    },
    "throughput": {
      "requests_per_second": 99.2
    },
    "errors": {
      "timeout": 30,
      "connection_refused": 20
    }
  }
}
```

---

### mercator validate

Validate cryptographic signatures of evidence records.

**Usage:**

```bash
mercator validate [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--time-range` | string | | Time range to validate |
| `--report` | bool | false | Generate detailed report |
| `--format` | string | `text` | Output format: `text`, `json` |
| `--output` | string | stdout | Output file path |

**Examples:**

```bash
# Validate all evidence in time range
mercator validate \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

# Validate with detailed report
mercator validate --report --format json --output validation-report.json

# Validate all evidence (no time range)
mercator validate --report
```

**Exit Codes:**

| Code | Meaning |
|------|---------|
| 0 | All signatures valid |
| 1 | Configuration error |
| 2 | Database connection error |
| 3 | One or more invalid signatures |

**Output Format (Text):**

```
Validating evidence signatures...

✓ req-abc123: signature valid
✓ req-def456: signature valid
✗ req-ghi789: signature invalid
✓ req-jkl012: signature valid

Summary:
  Total: 1,000
  Valid: 999 (99.9%)
  Invalid: 1 (0.1%)

Invalid Records:
  req-ghi789: signature verification failed
```

**Output Format (JSON):**

```json
{
  "summary": {
    "total": 1000,
    "valid": 999,
    "invalid": 1,
    "success_rate": 0.999
  },
  "invalid_records": [
    {
      "id": "req-ghi789",
      "timestamp": "2025-11-20T10:30:00Z",
      "error": "signature verification failed"
    }
  ]
}
```

---

### mercator keys

Manage cryptographic keys for evidence signing.

**Usage:**

```bash
mercator keys <subcommand> [flags]
```

**Subcommands:**

- `generate` - Generate new Ed25519 keypair

#### mercator keys generate

Generate a new Ed25519 keypair for signing evidence.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--key-id` | string | *required* | Key identifier (used in filename) |
| `--output` | string | `.` | Output directory for keys |

**Examples:**

```bash
# Generate keypair
mercator keys generate --key-id "prod-2025-11" --output ./keys

# Generate in specific directory
mercator keys generate --key-id "staging-key" --output /etc/mercator/keys
```

**Output:**

```
Generating Ed25519 keypair...

Keys generated:
  Private key: ./keys/prod-2025-11_private.pem (mode: 0600)
  Public key:  ./keys/prod-2025-11_public.pem (mode: 0644)

⚠ IMPORTANT:
  - Store the private key securely
  - Back up both keys
  - Never commit private keys to version control
  - Use the private key path in your config.yaml:

    evidence:
      signing_key_path: "./keys/prod-2025-11_private.pem"
```

**File Permissions:**

The command automatically sets secure permissions:

- Private key: `0600` (read/write for owner only)
- Public key: `0644` (readable by all, writable by owner)

**Key Rotation:**

To rotate keys:

1. Generate new keypair with new key-id
2. Update config to use new private key
3. Restart server
4. Keep old keys for validating historical evidence

---

### mercator version

Print version information.

**Usage:**

```bash
mercator version [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--short` | bool | Print short version only |

**Examples:**

```bash
# Full version info
mercator version

# Short version
mercator version --short
```

**Output:**

```
Mercator Jupiter v1.0.0

Build Information:
  Version:      1.0.0
  Commit:       abc123def456
  Build Date:   2025-11-20T10:00:00Z
  Go Version:   go1.21.5
  Platform:     linux/amd64
```

---

### mercator completion

Generate shell completion scripts.

**Usage:**

```bash
mercator completion <shell>
```

**Supported Shells:**

- `bash`
- `zsh`
- `fish`
- `powershell`

**Examples:**

```bash
# Bash
mercator completion bash > /etc/bash_completion.d/mercator
source /etc/bash_completion.d/mercator

# Zsh
mercator completion zsh > "${fpath[1]}/_mercator"

# Fish
mercator completion fish > ~/.config/fish/completions/mercator.fish

# PowerShell
mercator completion powershell > mercator.ps1
```

**Features:**

- Command name completion
- Flag name completion
- File path completion for `--config`, `--file`, `--output`
- Custom completions for enum flags (e.g., `--format`)

---

## Configuration

All commands respect the `--config` flag and load configuration from YAML files.

See the [Configuration Guide](../examples/config.yaml) for full configuration options.

**Minimal Config:**

```yaml
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"

policy:
  mode: "file"
  file_path: "policies.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "evidence.db"
```

---

## Exit Codes

All Mercator commands use consistent exit codes:

| Code | Meaning | Common Causes |
|------|---------|---------------|
| 0 | Success | Command completed successfully |
| 1 | General error | Config file not found, invalid YAML, missing required flags |
| 2 | Initialization error | Failed to bind port, database connection failed, provider initialization failed |
| 3 | Operation-specific error | Test failed, validation failed, signatures invalid |
| 130 | User interrupted | User pressed Ctrl+C (SIGINT) |

**Usage in Scripts:**

```bash
#!/bin/bash
set -e

mercator lint --file policies.yaml
if [ $? -ne 0 ]; then
  echo "Policy validation failed"
  exit 1
fi

mercator test --policy policies.yaml --tests tests.yaml
if [ $? -ne 0 ]; then
  echo "Tests failed"
  exit 1
fi

echo "All checks passed"
```

---

## Troubleshooting

### Common Issues

#### Problem: "Failed to load configuration"

**Cause:** Config file not found or invalid YAML

**Solution:**

```bash
# Check file exists
ls -la config.yaml

# Validate YAML syntax
mercator run --dry-run --verbose

# Use explicit path
mercator run --config /absolute/path/to/config.yaml
```

#### Problem: "Permission denied" when generating keys

**Cause:** No write access to output directory

**Solution:**

```bash
# Check directory permissions
ls -ld ./keys

# Create directory with correct permissions
mkdir -p ./keys
chmod 755 ./keys

# Generate keys
mercator keys generate --key-id prod-key --output ./keys
```

#### Problem: "Address already in use"

**Cause:** Another process is using the port

**Solution:**

```bash
# Find process using the port
lsof -i :8080

# Kill the process or use different port
mercator run --listen 127.0.0.1:8081
```

#### Problem: "Policy file not found"

**Cause:** Invalid file path in config

**Solution:**

```bash
# Use absolute path
policy:
  file_path: "/absolute/path/to/policies.yaml"

# Or relative to config file location
policy:
  file_path: "./policies.yaml"
```

#### Problem: "Evidence database locked"

**Cause:** Multiple processes accessing same SQLite database

**Solution:**

```bash
# Ensure only one server instance is running
killall mercator

# Or use separate database paths for different instances
evidence:
  sqlite:
    path: "evidence-server1.db"
```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Run with verbose output
mercator run --verbose --log-level debug

# Capture logs to file
mercator run --verbose --log-level debug 2>&1 | tee debug.log
```

### Getting Help

If you encounter issues:

1. Check this documentation
2. Review [docs/HOW-TO-RUN.md](HOW-TO-RUN.md)
3. Search [GitHub Issues](https://github.com/mercator-hq/jupiter/issues)
4. Ask in [GitHub Discussions](https://github.com/mercator-hq/jupiter/discussions)

---

## See Also

- [CLI Cookbook](CLI-COOKBOOK.md) - Practical recipes for common tasks
- [Configuration Guide](../examples/config.yaml) - Complete configuration reference
- [Policy Language (MPL)](mpl/README.md) - Policy syntax and examples
- [Observability Guide](observability-guide.md) - Metrics, logs, and tracing
