# Mercator Jupiter Documentation

Welcome to the Mercator Jupiter documentation! This guide will help you understand, deploy, and use Mercator Jupiter for LLM governance.

## 📚 Documentation Index

### Getting Started

New to Mercator Jupiter? Start here:

- **[Quick Start Guide](getting-started/quick-start.md)** - Deploy Jupiter in 15 minutes
- **[First Policy Guide](getting-started/first-policy.md)** - Create and test your first policy
- **[Configuration Basics](getting-started/configuration-basics.md)** - Understanding configuration
- **[How to Run](HOW-TO-RUN.md)** - Detailed setup instructions

### Configuration

- **[Configuration Reference](configuration/reference.md)** - Complete configuration documentation
- **[Proxy Configuration](configuration/proxy.md)** - HTTP proxy server settings
- **[Provider Configuration](configuration/providers.md)** - LLM provider setup
- **[Policy Configuration](configuration/policy.md)** - Policy loading options
- **[Evidence Configuration](configuration/evidence.md)** - Evidence storage setup
- **[Telemetry Configuration](configuration/telemetry.md)** - Logging, metrics, tracing
- **[Security Configuration](configuration/security.md)** - TLS/mTLS and API keys
- **[Security Guide](SECURITY.md)** - Security best practices
- **[Certificates Guide](CERTIFICATES.md)** - TLS certificate management

### Mercator Policy Language (MPL)

- **[MPL Overview](mpl/README.md)** - Introduction to MPL
- **[Language Specification](mpl/SPECIFICATION.md)** - Complete MPL specification
- **[Syntax Guide](mpl/SYNTAX.md)** - MPL syntax reference
- **[Best Practices](mpl/BEST_PRACTICES.md)** - Writing effective policies
- **[Policy Examples](mpl/examples/)** - 22 example policies

### Policy Cookbook

Real-world policy examples:

- **[Policy Cookbook](policies/cookbook.md)** - Index of all policy examples
- **[Content Safety](policies/content-safety.md)** - PII, profanity, sensitive content
- **[Budget & Limits](policies/budget-limits.md)** - Cost and rate limiting
- **[Routing Policies](policies/routing.md)** - Provider routing strategies
- **[Compliance](policies/compliance.md)** - HIPAA, GDPR, SOC2 examples
- **[Development Workflows](policies/development.md)** - Test, staging, production

### Provider Setup

- **[OpenAI Setup](providers/openai.md)** - Configure OpenAI provider
- **[Anthropic Setup](providers/anthropic.md)** - Configure Claude/Anthropic
- **[Ollama Setup](providers/ollama.md)** - Local model deployment
- **[Custom Providers](providers/custom.md)** - Integrate custom providers

### Deployment

- **[Docker Deployment](deployment/docker.md)** - Single container deployment
- **[Docker Compose](deployment/docker-compose.md)** - Multi-container setup
- **[Kubernetes Deployment](deployment/kubernetes.md)** - K8s deployment guide
- **[Systemd Service](deployment/systemd.md)** - Linux service deployment
- **[Bare Metal](deployment/bare-metal.md)** - Manual binary deployment
- **[High Availability](deployment/high-availability.md)** - HA and load balancing

### Command-Line Interface

- **[CLI Overview](cli/overview.md)** - Command-line interface overview
- **[CLI Reference](CLI.md)** - Complete CLI reference
- **[CLI Cookbook](CLI-COOKBOOK.md)** - Practical CLI recipes
- **[run Command](cli/run.md)** - Start proxy server
- **[lint Command](cli/lint.md)** - Validate policies
- **[test Command](cli/test.md)** - Test policies
- **[evidence Command](cli/evidence.md)** - Query evidence records
- **[benchmark Command](cli/benchmark.md)** - Load testing
- **[validate Command](cli/validate.md)** - Validate signatures
- **[keys Command](cli/keys.md)** - Key management

### HTTP API

- **[API Overview](api/overview.md)** - HTTP API introduction
- **[Chat Completions](api/chat-completions.md)** - Chat completions endpoint
- **[Streaming Responses](api/streaming.md)** - SSE streaming guide
- **[Authentication](api/authentication.md)** - API key management

### Troubleshooting

- **[Common Errors](troubleshooting/common-errors.md)** - Common issues and solutions
- **[Provider Issues](troubleshooting/provider-issues.md)** - Connection problems
- **[Policy Errors](troubleshooting/policy-errors.md)** - Validation errors
- **[TLS/Certificate Issues](troubleshooting/tls-certificates.md)** - Certificate problems
- **[Performance Tuning](troubleshooting/performance.md)** - Optimization guide
- **[Debugging](troubleshooting/debugging.md)** - Debug logging

### Observability

- **[Observability Guide](observability-guide.md)** - Metrics, logs, and tracing
- **[Limits & Usage Guide](limits-usage-guide.md)** - Budget and rate limiting
- **[Metrics Queries](metrics-queries.md)** - Prometheus query examples

### Architecture

- **[Architecture Overview](architecture/overview.md)** - System architecture
- **[Design Decisions](architecture/design-decisions.md)** - Key design choices
- **[Data Flow](architecture/data-flow.md)** - Request/response flow
- **[Security Model](architecture/security-model.md)** - Security architecture

### Contributing

- **[Contributing Guide](../CONTRIBUTING.md)** - How to contribute
- **[Development Setup](contributing/development-setup.md)** - Dev environment
- **[Testing Guide](contributing/testing.md)** - Testing requirements
- **[Code Style](contributing/code-style.md)** - Code conventions
- **[Pull Request Process](contributing/pull-requests.md)** - PR workflow

## 🚀 Quick Links

- **[GitHub Repository](https://github.com/mercator-hq/jupiter)**
- **[Issues](https://github.com/mercator-hq/jupiter/issues)**
- **[Discussions](https://github.com/mercator-hq/jupiter/discussions)**
- **[Changelog](../CHANGELOG.md)**
- **[License](../LICENSE)**

## 📦 Examples

Check out the [examples/](../examples/) directory for:

- Configuration examples
- Policy examples
- Deployment examples (Docker, Kubernetes, systemd)
- CI/CD examples

## 🔍 Search

Looking for something specific? Use GitHub's search or browse the documentation by section above.

## 📝 Documentation Version

- **Documentation Version**: 1.0
- **Jupiter Version**: 0.1.0
- **Last Updated**: 2025-11-21

## 💡 Getting Help

- **Documentation**: Browse this docs directory
- **Issues**: [Report a bug](https://github.com/mercator-hq/jupiter/issues/new)
- **Discussions**: [Ask questions](https://github.com/mercator-hq/jupiter/discussions/new)

---

**Mercator Jupiter** - Trustworthy LLM governance for the enterprise.
