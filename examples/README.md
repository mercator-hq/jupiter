# Mercator Jupiter Examples

This directory contains example programs demonstrating how to use various Mercator Jupiter components.

## Running Examples

Each example is in its own subdirectory with a `main.go` file. To run an example:

```bash
# From the project root
cd examples/<example-name>
go run main.go

# Or build and run
go build -o example .
./example
```

## Available Examples

### Policy Engine Example
**Directory**: `policy-engine/`
**Description**: Demonstrates how to use the interpreted policy engine to evaluate MPL policies against enriched requests.

**Features shown**:
- Creating and configuring the policy engine
- Loading policies from memory
- Evaluating requests with PII detection
- Handling policy decisions (block, allow, redact)
- Accessing matched rules and evaluation traces

**Run**:
```bash
cd examples/policy-engine
go run main.go
```

### Provider Usage Example
**Directory**: `provider-usage/`
**Description**: Demonstrates how to use LLM provider adapters to make API calls to OpenAI, Anthropic, and other providers.

**Features shown**:
- Initializing provider adapters
- Making chat completion requests
- Handling streaming responses
- Error handling and retries
- Provider-specific features (function calling, vision)

**Run**:
```bash
cd examples/provider-usage
go run main.go
```

## Configuration Examples

The `configs/` directory contains example configuration files for different use cases:

**Basic Configurations**:
- `minimal.yaml` - Minimal working configuration
- `production.yaml` - Production-ready configuration with all features
- `development.yaml` - Development/testing configuration

**Feature-Specific Configurations**:
- `basic-config.yaml` - Basic proxy and provider setup
- `providers-config.yaml` - Multi-provider configuration
- `tls-config.yaml` - TLS-enabled proxy
- `mtls-config.yaml` - Mutual TLS for client authentication
- `apikey-auth-config.yaml` - API key authentication
- `observability-config.yaml` - Comprehensive observability setup
- `dev-observability-config.yaml` - Development observability
- `limits-config.yaml` - Budget and rate limiting
- `secrets-config.yaml` - Secret management patterns
- `processing-config.yaml` - Request/response processing

## Policy Examples

The `policies/` directory contains example MPL policies:

- `simple-logging.yaml` - Basic logging policy
- `content-filtering.yaml` - Content filtering rules
- `model-restrictions.yaml` - Model access control
- `rate-limiting.yaml` - Rate limiting policies
- `production-policy.yaml` - Production-ready policy bundle
- `multi-rule.yaml` - Multiple rules in one policy
- `multi-file/` - Multi-file policy example with separate concerns

**Additional policy examples** in [../docs/mpl/examples/](../docs/mpl/examples/):
- 22 example policies covering PII detection, token limits, routing, compliance, and more

## Deployment Examples

### Docker

The `docker/` directory contains:
- `Dockerfile` - Optimized multi-stage Docker build
- `docker-compose.yaml` - Full stack with SQLite
- `docker-compose-postgres.yaml` - With PostgreSQL backend
- `.dockerignore` - Docker build exclusions

### Kubernetes

The `kubernetes/` directory contains:
- `deployment.yaml` - K8s deployment manifest
- `service.yaml` - K8s service manifest
- `configmap.yaml` - Configuration as ConfigMap
- `secret.yaml` - API keys as secrets
- `ingress.yaml` - Ingress configuration
- `helm/` - Helm chart for Jupiter deployment

### Systemd

The `systemd/` directory contains:
- `mercator.service` - Systemd unit file
- `mercator.env` - Environment variables

## CI/CD Examples

The `ci/` directory contains:
- `github-actions.yaml` - GitHub Actions workflow
- `gitlab-ci.yaml` - GitLab CI pipeline
- `jenkins.groovy` - Jenkins pipeline

## Building All Examples

To verify all examples compile correctly:

```bash
# From the project root
for example in examples/*/; do
    echo "Building $example..."
    (cd "$example" && go build -o /dev/null . && echo "✓ Success") || echo "✗ Failed"
done
```

## Creating New Examples

To add a new example:

1. Create a new subdirectory: `examples/my-example/`
2. Add a `main.go` file with `package main`
3. Import necessary Mercator Jupiter packages
4. Add documentation to this README

## Notes

- Each example is a standalone Go program with its own `main` function
- Examples use relative imports: `mercator-hq/jupiter/pkg/...`
- Configuration files in this directory can be used by examples via relative paths
- Examples are for demonstration only - production code should include proper error handling and logging
