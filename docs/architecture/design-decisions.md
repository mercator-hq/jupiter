# Design Decisions

Key architectural and design decisions made in Mercator Jupiter.

## Core Principles

1. **GitOps-First** - Policies as code, version controlled
2. **Zero Trust** - Validate every request, trust nothing
3. **Transparency** - Cryptographically signed audit trail
4. **Simplicity** - Minimal dependencies, easy deployment
5. **Performance** - Sub-10ms policy evaluation

---

## Language Choice: Go

**Decision**: Implement Mercator Jupiter in Go

**Rationale**:
- **Performance**: Compiled language with excellent concurrency
- **Deployment**: Single binary, no runtime dependencies
- **Ecosystem**: Strong HTTP, networking, and cloud libraries
- **Team expertise**: Go is widely known in infrastructure teams
- **Cross-platform**: Easy compilation for Linux, macOS, Windows

**Alternatives Considered**:
- **Rust**: Better performance but steeper learning curve
- **Python**: Easier development but slower runtime
- **TypeScript**: Good for web but not ideal for proxy workloads

---

## Proxy Architecture

**Decision**: HTTP reverse proxy with middleware pipeline

**Rationale**:
- **OpenAI compatibility**: Drop-in replacement for existing applications
- **Transparency**: No client changes required
- **Control**: Full visibility into requests and responses
- **Flexibility**: Can modify, route, or deny requests

**Architecture**:
```
Request → Middleware Pipeline → Provider → Middleware Pipeline → Response
          [Auth, Policy, Evidence, Routing, Limits]
```

**Alternatives Considered**:
- **SDK/Library**: Requires client changes, harder adoption
- **Sidecar**: More complex deployment, higher latency
- **API Gateway**: Too generic, lacks LLM-specific features

---

## Policy Language (MPL)

**Decision**: Custom declarative policy language in YAML

**Rationale**:
- **Declarative**: What, not how
- **Human-readable**: Easy for non-developers
- **Version-controllable**: Plain text, Git-friendly
- **Extensible**: Can add new conditions/actions
- **Safe**: No arbitrary code execution

**Example**:
```yaml
rules:
  - condition: request.model == "gpt-4"
    action: deny
    reason: "GPT-4 not approved"
```

**Alternatives Considered**:
- **Rego (OPA)**: Too complex for simple policies
- **CEL**: Not expressive enough for LLM use cases
- **JavaScript**: Security risk, requires sandboxing
- **WASM**: Planned for Phase 2, too complex for MVP

---

## Evidence Generation

**Decision**: Cryptographic signatures on all LLM interactions

**Rationale**:
- **Non-repudiation**: Proof of what happened
- **Tamper-proof**: Can't modify historical records
- **Compliance**: Meets audit requirements (SOC2, HIPAA)
- **Trust**: Independent verification possible

**Implementation**:
- **Algorithm**: Ed25519 (fast, secure, standardized)
- **Format**: JSON with detached signature
- **Storage**: SQLite (MVP) or PostgreSQL (production)

**Alternatives Considered**:
- **HMAC**: Symmetric, doesn't provide non-repudiation
- **RSA**: Slower than Ed25519
- **No signatures**: Fails compliance requirements

---

## Storage Backend

**Decision**: SQLite for MVP, PostgreSQL for production

**Rationale**:

**SQLite**:
- **Zero configuration**: No separate database server
- **Simple deployment**: Single file
- **Sufficient for MVP**: Handles moderate load
- **Easy backup**: Just copy the file

**PostgreSQL**:
- **Scalability**: Handles high volume
- **Replication**: High availability support
- **Rich queries**: Complex evidence analysis
- **Industry standard**: Well-understood operations

**Alternatives Considered**:
- **MySQL**: Less feature-rich than PostgreSQL
- **MongoDB**: Overkill for structured data
- **S3**: Planned for archival, not primary storage

---

## Configuration System

**Decision**: YAML with environment variable overrides

**Rationale**:
- **Readable**: YAML is human-friendly
- **Flexible**: Environment variables for secrets
- **Validatable**: Schema validation at startup
- **12-Factor**: Separate config from code

**Example**:
```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"  # From environment
```

**Alternatives Considered**:
- **TOML**: Less popular than YAML
- **JSON**: Not human-friendly for config
- **Environment only**: Too many variables for complex config

---

## Multi-Provider Routing

**Decision**: Abstract provider interface with strategy pattern

**Rationale**:
- **Flexibility**: Easy to add new providers
- **Strategies**: Round-robin, least-latency, cost-optimized
- **Failover**: Automatic retry with different provider
- **Vendor independence**: Not locked to any provider

**Interface**:
```go
type Provider interface {
    ChatCompletion(ctx context.Context, req *Request) (*Response, error)
    HealthCheck(ctx context.Context) error
}
```

**Strategies**:
- **Round-robin**: Distribute load evenly
- **Least-latency**: Route to fastest provider
- **Cost-optimized**: Choose cheapest provider
- **Sticky**: Same user → same provider

---

## Policy Evaluation

**Decision**: Interpreted MPL evaluation (Phase 1)

**Rationale**:
- **Simplicity**: Easy to implement and debug
- **Flexibility**: Can modify policies without recompilation
- **Performance**: <5ms evaluation meets requirements
- **Safety**: No code execution risks

**Future**: WASM compilation (Phase 2) for <1ms evaluation

**Alternatives Considered**:
- **WASM from start**: More complex, not needed for MVP
- **Embedded V8**: Security concerns, larger binary
- **Native compilation**: Requires compiler infrastructure

---

## Observability

**Decision**: OpenTelemetry for metrics and tracing

**Rationale**:
- **Standard**: Industry-standard observability
- **Vendor-neutral**: Works with any backend
- **Complete**: Metrics, logs, and traces
- **Integration**: Easy Prometheus/Grafana setup

**Metrics Exposed**:
- Request counts and latencies
- Policy evaluation results
- Provider health and performance
- Evidence generation stats
- Error rates and types

---

## Security Model

**Decision**: Defense in depth with multiple layers

**Layers**:
1. **Network**: TLS/mTLS encryption
2. **Authentication**: API key validation
3. **Authorization**: Policy-based access control
4. **Input validation**: Sanitize all inputs
5. **Output validation**: Check provider responses
6. **Audit**: Cryptographically signed evidence

**Principles**:
- **Least privilege**: Minimal permissions
- **Fail secure**: Deny by default
- **No secrets in logs**: Redaction everywhere
- **Regular rotation**: Keys, credentials, certificates

---

## Deployment Model

**Decision**: Cloud-native with multiple deployment options

**Supported**:
- **Docker**: Single container deployment
- **Kubernetes**: Helm chart for orchestration
- **Systemd**: Traditional Linux services
- **Cloud**: ECS, GKE, AKS native support

**Rationale**:
- **Flexibility**: Users choose what fits their environment
- **Portability**: Runs anywhere
- **Scalability**: From single instance to multi-region
- **Operations**: Standard tooling (kubectl, docker)

---

## API Compatibility

**Decision**: OpenAI-compatible API

**Rationale**:
- **Drop-in replacement**: No client changes
- **Ecosystem**: Works with all OpenAI-compatible tools
- **Familiarity**: Developers know the API
- **Standards**: De facto standard for LLM APIs

**Supported Endpoints**:
- `/v1/chat/completions`
- `/v1/completions`
- Streaming with Server-Sent Events

---

## Error Handling

**Decision**: Explicit error handling with context

**Principles**:
- **Never panic**: Return errors, don't crash
- **Wrap errors**: Add context at each layer
- **Structured logging**: JSON logs with context
- **User-friendly**: Clear error messages

**Example**:
```go
if err != nil {
    return nil, fmt.Errorf("failed to evaluate policy %s: %w", policyID, err)
}
```

---

## Testing Strategy

**Decision**: Table-driven tests with 80%+ coverage

**Rationale**:
- **Comprehensive**: Cover all code paths
- **Maintainable**: Easy to add new test cases
- **Fast**: Unit tests run in <10 seconds
- **Confidence**: Safe to refactor

**Test Types**:
- **Unit**: Individual function testing
- **Integration**: Component interaction testing
- **Benchmark**: Performance validation
- **E2E**: Full workflow testing

---

## Performance Targets

| Component | Target | Actual | Status |
|-----------|--------|--------|--------|
| Config Load | <10ms | ~3ms | ✅ |
| Policy Eval | <10ms | ~2ms | ✅ |
| Proxy Overhead | <5ms | ~1ms | ✅ |
| Evidence Gen | <5ms | ~1ms | ✅ |
| Throughput | 1000 req/s | 1500+ req/s | ✅ |

---

## Future Design Decisions

### Planned for Phase 2

1. **WASM Policy Compilation** - Sub-millisecond evaluation
2. **PostgreSQL Evidence Backend** - High-volume storage
3. **S3 Evidence Archival** - Long-term retention
4. **Content Analysis** - PII detection, sentiment analysis
5. **Multi-tenancy** - Namespace isolation
6. **Policy Playground** - Web-based policy testing
7. **Advanced Routing** - ML-based provider selection

---

## Lessons Learned

### What Worked

- **Simple deployment**: Single binary adoption is high
- **GitOps policies**: Developers love version control
- **OpenAI compatibility**: Zero-friction integration
- **Comprehensive docs**: Users succeed independently

### What We'd Do Differently

- **WASM earlier**: Would've started with WASM compilation
- **PostgreSQL from start**: SQLite limits hit quickly at scale
- **More providers**: Add Cohere, AI21 sooner
- **Web UI earlier**: Visual policy editor is requested

---

## Decision Log

Significant decisions are tracked in this document. For detailed technical discussions, see:
- [Architecture Overview](overview.md)
- [Data Flow](data-flow.md)
- [Security Model](security-model.md)
