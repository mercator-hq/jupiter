# Mercator Jupiter

**GitOps-native LLM governance runtime and policy engine**

Mercator Jupiter is an open-source HTTP proxy that provides policy enforcement, multi-provider LLM routing, and cryptographic evidence generation for Large Language Model (LLM) requests. It enables organizations to govern LLM usage through declarative policies while maintaining a complete audit trail with cryptographic signatures.

## Features

- **Policy-driven governance**: Define LLM usage policies in declarative YAML using Mercator Policy Language (MPL)
- **Multi-provider routing**: Route requests across OpenAI, Anthropic, and other LLM providers
- **Cryptographic evidence**: Generate signed evidence records for all LLM interactions
- **GitOps workflow**: Manage policies through Git with validation and testing tools
- **Observability**: Built-in metrics, logging, and distributed tracing with OpenTelemetry
- **Rate limiting & budgets**: Control usage and costs per user, team, or organization
- **Content filtering**: Inspect and modify prompts and responses based on policy rules

## Architecture

```
┌─────────────┐         ┌──────────────────────────────────┐         ┌─────────────┐
│   Client    │────────▶│     Mercator Jupiter Proxy       │────────▶│ LLM Provider│
│ Application │         │  ┌────────────────────────────┐  │         │  (OpenAI,   │
└─────────────┘         │  │   Policy Engine (MPL)      │  │         │  Anthropic) │
                        │  │   - Budget enforcement      │  │         └─────────────┘
                        │  │   - Content filtering       │  │
                        │  │   - Rate limiting           │  │
                        │  └────────────────────────────┘  │
                        │  ┌────────────────────────────┐  │
                        │  │   Evidence Generation      │  │
                        │  │   - Request/response logs  │  │
                        │  │   - Cryptographic signing  │  │
                        │  └────────────────────────────┘  │
                        └──────────────────────────────────┘
                                       │
                                       ▼
                                ┌─────────────┐
                                │  Evidence   │
                                │   Storage   │
                                │  (SQLite)   │
                                └─────────────┘
```

## Quick Start

### Installation

**Download pre-built binary (recommended):**

Download the latest release for your platform from the [releases page](https://github.com/mercator-hq/jupiter/releases/latest).

```bash
# macOS (arm64)
curl -L https://github.com/mercator-hq/jupiter/releases/latest/download/mercator_Darwin_arm64.tar.gz | tar xz
sudo mv mercator /usr/local/bin/

# macOS (amd64)
curl -L https://github.com/mercator-hq/jupiter/releases/latest/download/mercator_Darwin_x86_64.tar.gz | tar xz
sudo mv mercator /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/mercator-hq/jupiter/releases/latest/download/mercator_Linux_x86_64.tar.gz | tar xz
sudo mv mercator /usr/local/bin/

# Linux (arm64)
curl -L https://github.com/mercator-hq/jupiter/releases/latest/download/mercator_Linux_arm64.tar.gz | tar xz
sudo mv mercator /usr/local/bin/

# Verify installation
mercator version
```

**Using Go install:**

```bash
go install mercator-hq/jupiter/cmd/mercator@latest
```

**From source:**

```bash
# Clone the repository
git clone https://github.com/mercator-hq/jupiter.git
cd jupiter

# Build the binary
go build -o mercator ./cmd/mercator

# Install to PATH
sudo mv mercator /usr/local/bin/
```

### Running Mercator

**1. Create a configuration file** (`config.yaml`):

```yaml
proxy:
  listen_address: "127.0.0.1:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
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
```

**2. Create a policy file** (`policies.yaml`):

```yaml
version: "1.0"
policies:
  - name: "budget-enforcement"
    description: "Enforce user spending limits"
    rules:
      - condition: "request.metadata.user_budget_spent + request.estimated_cost > request.metadata.user_budget_limit"
        action: "block"
        reason: "User has exceeded their budget limit"

  - name: "rate-limiting"
    description: "Limit requests per user"
    rules:
      - condition: "request.metadata.user_request_count > 100"
        action: "block"
        reason: "Rate limit exceeded (100 requests/hour)"
```

**3. Start the proxy server:**

```bash
export OPENAI_API_KEY="your-api-key"
mercator run --config config.yaml
```

**4. Send requests through the proxy:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello, world!"}]
  }'
```

## CLI Usage

Mercator Jupiter provides a comprehensive command-line interface for all operations.

### Available Commands

```bash
mercator run         # Start the proxy server
mercator lint        # Validate MPL policy files
mercator test        # Run policy unit tests
mercator evidence    # Query and export evidence records
mercator benchmark   # Load test the proxy
mercator validate    # Validate evidence signatures
mercator keys        # Manage cryptographic keys
mercator version     # Print version information
mercator completion  # Generate shell completion scripts
```

### Command Examples

**Start the proxy server:**

```bash
# Start with default config
mercator run

# Start with custom config
mercator run --config /etc/mercator/config.yaml

# Override listen address
mercator run --listen 0.0.0.0:8080

# Validate config without starting
mercator run --dry-run
```

**Validate policy files:**

```bash
# Lint single file
mercator lint --file policies.yaml

# Lint directory of policies
mercator lint --dir policies/

# Strict mode (warnings as errors)
mercator lint --file policies.yaml --strict

# JSON output for CI/CD
mercator lint --file policies.yaml --format json
```

**Run policy tests:**

```bash
# Run tests against a policy
mercator test --policy policies.yaml --tests policy-tests.yaml

# Run with coverage report
mercator test --policy policies.yaml --tests tests.yaml --coverage

# JSON output
mercator test --policy policies.yaml --tests tests.yaml --format json
```

**Query evidence records:**

```bash
# Query by time range
mercator evidence query \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

# Filter by user
mercator evidence query --user-id "user-123" --limit 50

# Export to JSON
mercator evidence query \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z" \
  --format json \
  --output evidence.json
```

**Load testing:**

```bash
# Basic benchmark
mercator benchmark --target http://localhost:8080 --duration 60s

# High-load test
mercator benchmark \
  --target http://localhost:8080 \
  --duration 300s \
  --rate 500 \
  --concurrency 50
```

**Validate evidence signatures:**

```bash
# Validate all evidence in time range
mercator validate \
  --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

# Validate with detailed report
mercator validate --report --format json
```

**Manage cryptographic keys:**

```bash
# Generate new keypair
mercator keys generate --key-id "prod-2025-11" --output ./keys

# The command creates:
# - prod-2025-11_private.pem (mode 0600)
# - prod-2025-11_public.pem (mode 0644)
```

### Global Flags

All commands support these global flags:

- `--config, -c` - Path to config file (default: `config.yaml`)
- `--verbose, -v` - Enable verbose logging
- `--help, -h` - Show help for any command

### Shell Completion

Generate shell completion for faster CLI usage:

```bash
# Bash
mercator completion bash > /etc/bash_completion.d/mercator

# Zsh
mercator completion zsh > "${fpath[1]}/_mercator"

# Fish
mercator completion fish > ~/.config/fish/completions/mercator.fish

# PowerShell
mercator completion powershell > mercator.ps1
```

### Exit Codes

All commands follow consistent exit code conventions:

- `0` - Success
- `1` - General error (configuration, validation)
- `2` - Initialization error (server start, backend connection)
- `3` - Operation-specific error (test failure, validation error)
- `130` - User interrupted (SIGINT/Ctrl+C)

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- [CLI Reference](docs/CLI.md) - Complete command-line reference
- [CLI Cookbook](docs/CLI-COOKBOOK.md) - Practical recipes for common tasks
- [Policy Language (MPL)](docs/mpl/README.md) - Policy syntax and examples
- [Observability Guide](docs/observability-guide.md) - Metrics, logs, and tracing
- [How to Run](docs/HOW-TO-RUN.md) - Detailed setup instructions

## Development

### Prerequisites

- Go 1.21 or later
- SQLite 3.x (for evidence storage)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/mercator-hq/jupiter.git
cd jupiter

# Install dependencies
go mod download

# Build
go build -o mercator ./cmd/mercator

# Run tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Project Structure

```
mercator-hq/jupiter/
├── cmd/
│   └── mercator/              # CLI entrypoint
├── pkg/                       # Public API packages
│   ├── config/                # Configuration management
│   ├── proxy/                 # HTTP proxy server
│   ├── policy/                # Policy engine (MPL)
│   ├── providers/             # LLM provider adapters
│   ├── evidence/              # Evidence generation & storage
│   ├── routing/               # Request routing logic
│   ├── crypto/                # Cryptographic signing
│   ├── cli/                   # CLI command implementations
│   └── telemetry/             # Observability (logs, metrics, traces)
├── internal/                  # Private packages
│   └── util/                  # Shared utilities
├── examples/                  # Sample configs and policies
├── docs/                      # Documentation
├── test/                      # Integration tests
└── Task-Notes/                # Implementation notes
```

### Running Tests

```bash
# Unit tests
go test ./pkg/... ./cmd/... -v

# Integration tests
go test ./test -tags=integration -v

# Benchmarks
go test ./pkg/... -bench=. -benchmem
```

### Code Style

This project follows [Effective Go](https://go.dev/doc/effective_go) guidelines and the conventions documented in [CLAUDE.md](CLAUDE.md). Key principles:

- Use `gofmt` for formatting
- Follow standard Go naming conventions
- Write table-driven tests
- Document all exported functions
- Prefer composition over inheritance
- Handle errors explicitly

## Contributing

Contributions are welcome! Please read our contributing guidelines and code of conduct before submitting pull requests.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Run linter (`go vet ./...`)
6. Commit with conventional commits (`feat: add amazing feature`)
7. Push to your fork
8. Open a Pull Request

### Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

## Releases

Mercator Jupiter uses automated releases via [GoReleaser](https://goreleaser.com/) and GitHub Actions.

### Creating a Release

Releases are triggered by pushing a semantic version tag:

```bash
# Create and push a new tag
git tag -a v0.2.0 -m "Release v0.2.0: Add new features"
git push origin v0.2.0
```

The GitHub Actions workflow will automatically:
1. Run all tests
2. Build binaries for multiple platforms (Linux/macOS/Windows, amd64/arm64)
3. Create a GitHub release
4. Upload binaries and checksums
5. Generate a changelog from conventional commits

### Release Artifacts

Each release includes:
- **Binaries**: Pre-compiled for all major platforms
- **Archives**: `.tar.gz` (Linux/macOS) and `.zip` (Windows)
- **Checksums**: SHA256 checksums for verification
- **Changelog**: Auto-generated from commit history
- **Documentation**: Complete docs included in archives

### Version Numbering

Mercator Jupiter follows [Semantic Versioning](https://semver.org/):
- **Major (v1.0.0)**: Breaking changes
- **Minor (v0.2.0)**: New features (backwards compatible)
- **Patch (v0.1.1)**: Bug fixes (backwards compatible)

### Pre-releases

Pre-release versions can be tagged with suffixes:
```bash
git tag -a v0.2.0-rc.1 -m "Release candidate 1"
git push origin v0.2.0-rc.1
```

## Security

### Reporting Security Issues

Please report security vulnerabilities to security@mercator.io. Do not create public issues for security problems.

### Security Features

- **Cryptographic signing**: All evidence records are signed with Ed25519
- **Key rotation**: Support for key rotation with historical verification
- **TLS/mTLS**: Optional TLS for proxy server and provider connections
- **Secret management**: Integration with environment variables and KMS
- **Input validation**: Comprehensive validation of all external inputs

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/mercator-hq/jupiter/issues)
- **Discussions**: [GitHub Discussions](https://github.com/mercator-hq/jupiter/discussions)

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [OpenTelemetry](https://opentelemetry.io/) - Observability
- [SQLite](https://www.sqlite.org/) - Evidence storage

---

**Mercator Jupiter** - Trustworthy LLM governance for the enterprise.
