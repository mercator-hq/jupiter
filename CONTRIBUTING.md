# Contributing to Mercator Jupiter

Thank you for your interest in contributing to Mercator Jupiter! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Documentation](#documentation)
- [Reporting Issues](#reporting-issues)

## Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Be respectful, professional, and constructive in all interactions.

### Expected Behavior

- Use welcoming and inclusive language
- Respect differing viewpoints and experiences
- Accept constructive criticism gracefully
- Focus on what is best for the community
- Show empathy towards other community members

### Unacceptable Behavior

- Harassment, discriminatory language, or personal attacks
- Publishing others' private information
- Trolling or inflammatory comments
- Other conduct that would be inappropriate in a professional setting

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.21 or later** - [Installation Guide](https://go.dev/doc/install)
- **Git** - [Installation Guide](https://git-scm.com/downloads)
- **SQLite 3.x** - Usually pre-installed on most systems
- **Make** (optional) - For using Makefile commands

### Fork and Clone

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/YOUR-USERNAME/jupiter.git
   cd jupiter
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/mercator-hq/jupiter.git
   ```
4. **Verify remotes**:
   ```bash
   git remote -v
   # origin    https://github.com/YOUR-USERNAME/jupiter.git (fetch)
   # origin    https://github.com/YOUR-USERNAME/jupiter.git (push)
   # upstream  https://github.com/mercator-hq/jupiter.git (fetch)
   # upstream  https://github.com/mercator-hq/jupiter.git (push)
   ```

## Development Environment

### Building from Source

```bash
# Install dependencies
go mod download

# Build the binary
go build -o mercator ./cmd/mercator

# Verify build
./mercator version
```

### Running Tests

```bash
# Run all unit tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package tests
go test ./pkg/policy/... -v

# Run integration tests
go test ./test -tags=integration -v

# Run benchmarks
go test ./pkg/... -bench=. -benchmem
```

### Code Quality Tools

```bash
# Format code
gofmt -s -w .

# Run Go vet
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run ./...
```

### IDE Setup

**VS Code** recommended extensions:
- Go (golang.go)
- YAML (redhat.vscode-yaml)

**GoLand/IntelliJ IDEA**:
- Built-in Go support works out of the box

## Development Workflow

### 1. Create a Feature Branch

```bash
# Update main branch
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/your-feature-name
```

**Branch naming conventions**:
- Feature: `feature/policy-caching`
- Bug fix: `fix/evidence-signature-validation`
- Documentation: `docs/update-cli-guide`
- Refactor: `refactor/proxy-middleware`

### 2. Make Your Changes

- Write clean, idiomatic Go code
- Follow the [Code Standards](#code-standards) below
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run tests
go test ./...

# Check formatting
gofmt -d .

# Run linter
go vet ./...
```

### 4. Commit Your Changes

Follow our [Commit Message Guidelines](#commit-message-guidelines):

```bash
git add .
git commit -m "feat(policy): add caching for compiled policies"
```

### 5. Push to Your Fork

```bash
git push origin feature/your-feature-name
```

### 6. Open a Pull Request

- Go to the [Mercator Jupiter repository](https://github.com/mercator-hq/jupiter)
- Click "New Pull Request"
- Select your fork and branch
- Fill out the PR template
- Submit for review

## Code Standards

### Go Style Guidelines

This project follows [Effective Go](https://go.dev/doc/effective_go) and the conventions documented in [CLAUDE.md](CLAUDE.md).

#### Key Principles

1. **Use `gofmt`** - All code must be formatted with `gofmt`
2. **Follow naming conventions** - PascalCase for exported, camelCase for unexported
3. **Write idiomatic Go** - Prefer standard library over dependencies
4. **Handle errors explicitly** - Never ignore errors
5. **Document exports** - All exported functions, types, and methods must have godoc comments

#### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Packages | lowercase, single word | `policy`, `evidence`, `proxy` |
| Files | lowercase with underscores | `policy_engine.go`, `evidence_store.go` |
| Exported identifiers | PascalCase | `PolicyEngine`, `EvidenceRecord` |
| Unexported identifiers | camelCase | `policyCache`, `evaluateRule` |
| Constants | PascalCase or camelCase | `MaxRetries`, `defaultTimeout` |

#### Function Design

```go
// Good - context first, error last
func EvaluatePolicy(ctx context.Context, policy Policy, req *Request) (*Decision, error)

// Bad - no context, error not last
func EvaluatePolicy(req *Request, policy Policy) (*Decision, error)
```

#### Error Handling

```go
// Good - wrap errors with context
if err != nil {
    return nil, fmt.Errorf("failed to load policy %s: %w", policyID, err)
}

// Good - define sentinel errors
var ErrPolicyNotFound = errors.New("policy not found")

// Bad - generic error
if err != nil {
    return nil, err
}
```

#### Documentation

```go
// PolicyEngine evaluates incoming requests against compiled policy bundles.
// It maintains an in-memory cache of compiled policies for performance.
//
// The engine is safe for concurrent use by multiple goroutines.
type PolicyEngine struct {
    cache sync.Map
    mu    sync.RWMutex
}

// Evaluate processes a request through all active policies and returns
// the aggregated policy decision. If multiple policies match, the most
// restrictive action is applied.
//
// The context is used for cancellation and timeout control. If the context
// is cancelled, Evaluate returns immediately with context.Canceled.
func (e *PolicyEngine) Evaluate(ctx context.Context, req *Request) (*Decision, error) {
    // Implementation...
}
```

## Testing Requirements

### Unit Tests

- **Coverage Target**: 80%+ for all packages
- **Test File Naming**: `*_test.go` in same package
- **Use table-driven tests** for multiple scenarios

```go
func TestPolicyEngine_Evaluate(t *testing.T) {
    tests := []struct {
        name    string
        policy  Policy
        request *Request
        want    *Decision
        wantErr bool
    }{
        {
            name:    "allow request",
            policy:  testPolicy,
            request: &Request{Model: "gpt-4"},
            want:    &Decision{Action: Allow},
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            e := NewPolicyEngine()
            got, err := e.Evaluate(context.Background(), tt.policy, tt.request)
            if (err != nil) != tt.wantErr {
                t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Assertions...
        })
    }
}
```

### Integration Tests

- Place in `test/` directory with `//go:build integration` tag
- Test end-to-end workflows
- Run with: `go test -tags=integration ./test/...`

### Benchmarks

- Add benchmarks for performance-critical code
- File naming: `*_bench_test.go` or add to `*_test.go`
- Run with: `go test -bench=. -benchmem`

```go
func BenchmarkPolicyEngine_Evaluate(b *testing.B) {
    engine := NewPolicyEngine()
    policy := loadTestPolicy()
    req := &Request{Model: "gpt-4"}
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := engine.Evaluate(ctx, policy, req)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation only changes
- **test**: Adding or updating tests
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **chore**: Maintenance tasks (dependencies, build, etc.)
- **ci**: CI/CD configuration changes

### Scope (Optional)

The scope specifies the package or component affected:
- `config` - Configuration system
- `policy` - Policy engine
- `proxy` - HTTP proxy server
- `providers` - LLM provider adapters
- `evidence` - Evidence generation and storage
- `routing` - Request routing
- `cli` - Command-line interface
- `crypto` - Cryptographic functions
- `telemetry` - Observability (logs, metrics, traces)

### Examples

```
feat(policy): add caching for compiled policies
fix(proxy): handle connection timeouts correctly
docs(mpl): add examples for content filtering
test(routing): add provider selection tests
refactor(providers): simplify OpenAI adapter
perf(policy): optimize rule evaluation loop
chore(deps): update dependencies
```

### Breaking Changes

If your change breaks backward compatibility, add `BREAKING CHANGE:` to the footer:

```
feat(config)!: remove deprecated configuration fields

BREAKING CHANGE: The `old_field` configuration option has been removed.
Use `new_field` instead.
```

## Pull Request Process

### Before Submitting

1. ✅ All tests pass (`go test ./...`)
2. ✅ Code is formatted (`gofmt -d .` shows no output)
3. ✅ No linter warnings (`go vet ./...`)
4. ✅ Documentation updated (if applicable)
5. ✅ Commit messages follow guidelines
6. ✅ Branch is up-to-date with `main`

### PR Checklist

When you open a PR, ensure:

- [ ] Clear description of what and why
- [ ] Tests added for new functionality
- [ ] Documentation updated
- [ ] No breaking changes (or clearly documented)
- [ ] CI checks pass
- [ ] Code coverage doesn't decrease

### PR Template

```markdown
## Description
Brief description of what this PR does and why.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that causes existing functionality to change)
- [ ] Documentation update

## Testing
Describe the tests you added or ran:
- Unit tests: ...
- Integration tests: ...
- Manual testing: ...

## Checklist
- [ ] Code follows project style guidelines
- [ ] Tests added and passing
- [ ] Documentation updated
- [ ] Commit messages follow guidelines
```

### Review Process

1. **Automated checks** run on all PRs (tests, linting, coverage)
2. **Code review** by at least one maintainer
3. **Changes requested** - address feedback and push updates
4. **Approval** - once approved, a maintainer will merge

### Review Timeline

- Initial review: 2-3 business days
- Follow-up reviews: 1-2 business days
- **Large PRs** (>500 lines) may take longer

## Documentation

### When to Update Documentation

Update documentation when you:
- Add a new feature
- Change existing behavior
- Add new configuration options
- Add new CLI commands or flags
- Change API endpoints
- Fix bugs that affect documented behavior

### Documentation Types

1. **Code documentation** - Godoc comments for all exported functions
2. **README updates** - For major features
3. **User guides** - In `docs/` directory
4. **Examples** - Working code in `examples/` directory
5. **API documentation** - For HTTP endpoints
6. **CLI documentation** - For new commands

### Documentation Style

- Use clear, concise language
- Provide examples for every feature
- Include both success and error cases
- Use code blocks with syntax highlighting
- Test all code examples

## Reporting Issues

### Before Creating an Issue

1. **Search existing issues** - Your issue may already be reported
2. **Check documentation** - The answer might be in the docs
3. **Try latest version** - Bug may already be fixed

### Issue Types

**Bug Report** - Something isn't working:
- Clear description of the bug
- Steps to reproduce
- Expected vs. actual behavior
- Environment details (OS, Go version, Jupiter version)
- Relevant logs or error messages

**Feature Request** - Suggest a new feature:
- Use case and problem it solves
- Proposed solution
- Alternative solutions considered
- Willingness to contribute implementation

**Documentation** - Docs are unclear or incomplete:
- Link to documentation
- What's unclear or missing
- Suggested improvements

### Security Issues

**DO NOT** create public issues for security vulnerabilities. Instead:
- Email security@mercator.io
- Include detailed description and reproduction steps
- We will respond within 48 hours

## Getting Help

- **Documentation**: Check [docs/](docs/) directory
- **Issues**: [GitHub Issues](https://github.com/mercator-hq/jupiter/issues)
- **Discussions**: [GitHub Discussions](https://github.com/mercator-hq/jupiter/discussions)

## Recognition

All contributors are recognized in our release notes. Significant contributions may be highlighted in the README.

Thank you for contributing to Mercator Jupiter! 🚀
