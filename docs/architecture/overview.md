# Architecture Overview

Comprehensive overview of Mercator Jupiter's system architecture, components, and design.

## Table of Contents

- [System Architecture](#system-architecture)
- [Core Components](#core-components)
- [Request Flow](#request-flow)
- [Component Interaction](#component-interaction)
- [Data Flow](#data-flow)
- [Scalability](#scalability)
- [Security Architecture](#security-architecture)

---

## System Architecture

Mercator Jupiter is designed as a transparent HTTP proxy that sits between client applications and LLM providers.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Applications                          │
│  (OpenAI SDK, LangChain, Custom Apps, cURL, Web UIs)               │
└─────────────────────────────────────┬───────────────────────────────┘
                                      │
                                      │ HTTP/HTTPS
                                      │ OpenAI-compatible API
                                      │
┌─────────────────────────────────────▼───────────────────────────────┐
│                    MERCATOR JUPITER PROXY                            │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │                    HTTP Server Layer                        │   │
│  │  • Request/Response Handling                                │   │
│  │  • TLS/mTLS Termination                                     │   │
│  │  • API Key Authentication                                    │   │
│  └──────────────────────┬──────────────────────────────────────┘   │
│                         │                                            │
│  ┌──────────────────────▼──────────────────────────────────────┐   │
│  │              Request Processing Pipeline                     │   │
│  │                                                              │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │   │
│  │  │   Policy    │  │   Routing    │  │     Budget &     │  │   │
│  │  │   Engine    │  │   Engine     │  │  Rate Limiting   │  │   │
│  │  │   (MPL)     │  │              │  │                  │  │   │
│  │  └─────────────┘  └──────────────┘  └──────────────────┘  │   │
│  │                                                              │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │   │
│  │  │  Evidence   │  │  Processing  │  │   Telemetry      │  │   │
│  │  │  Recorder   │  │   (Enrich)   │  │  (Logs/Metrics)  │  │   │
│  │  └─────────────┘  └──────────────┘  └──────────────────┘  │   │
│  └──────────────────────┬──────────────────────────────────────┘   │
│                         │                                            │
│  ┌──────────────────────▼──────────────────────────────────────┐   │
│  │              Provider Manager                                 │   │
│  │  • Connection Pooling                                        │   │
│  │  • Health Checking                                           │   │
│  │  • Retry Logic                                               │   │
│  └──────────────────────┬──────────────────────────────────────┘   │
└──────────────────────────┼──────────────────────────────────────────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌──────────┐   ┌──────────┐   ┌──────────┐
    │  OpenAI  │   │ Anthropic│   │  Ollama  │
    │  API     │   │  API     │   │  Local   │
    └──────────┘   └──────────┘   └──────────┘

    ┌────────────────────────────────────┐
    │      Evidence Storage              │
    │  (SQLite / PostgreSQL / S3)        │
    └────────────────────────────────────┘
```

### Design Principles

1. **Transparency**: OpenAI-compatible API for drop-in replacement
2. **Modularity**: Pluggable components (providers, policies, storage)
3. **Observability**: Comprehensive logging, metrics, and tracing
4. **Security**: TLS, mTLS, API key authentication, cryptographic signing
5. **Performance**: Connection pooling, async processing, minimal overhead
6. **Reliability**: Health checking, automatic retry, failover

---

## Core Components

### 1. HTTP Proxy Server

**Package**: `pkg/proxy`

**Responsibilities**:
- Listen for HTTP/HTTPS requests
- Parse OpenAI-compatible requests
- Handle TLS/mTLS termination
- API key authentication
- Graceful shutdown

**Key Features**:
- OpenAI API compatibility
- Streaming response support (SSE)
- CORS handling
- Health check endpoint
- Metrics endpoint

**Implementation**:
```go
type Server struct {
    config     *config.ProxyConfig
    router     *http.ServeMux
    processing *processing.Pipeline
}
```

---

### 2. Policy Engine

**Package**: `pkg/policy/engine`

**Responsibilities**:
- Parse MPL policy files
- Evaluate policies against requests
- Execute policy actions (allow, deny, route, etc.)
- Maintain policy evaluation order (priority)

**Key Features**:
- Declarative policy language (MPL)
- Field matching, regex, content analysis
- Multiple action types
- Policy composition (multiple policies)
- Fast evaluation (<5ms p99)

**Evaluation Process**:
1. Load policies (sorted by priority)
2. Enrich request with metadata
3. Evaluate each policy's conditions
4. Execute first matching rule
5. Return decision (allow, deny, route, etc.)

---

### 3. Provider Manager

**Package**: `pkg/providers`

**Responsibilities**:
- Manage connections to LLM providers
- Abstract provider differences
- Handle provider-specific authentication
- Connection pooling and health checking
- Automatic retry with exponential backoff

**Supported Providers**:
- OpenAI (GPT models)
- Anthropic (Claude models)
- Ollama (local models)
- Custom (any OpenAI-compatible endpoint)

**Provider Interface**:
```go
type Provider interface {
    ChatCompletion(ctx context.Context, req *Request) (*Response, error)
    StreamChatCompletion(ctx context.Context, req *Request) (<-chan *Chunk, error)
    Health(ctx context.Context) error
}
```

---

### 4. Routing Engine

**Package**: `pkg/routing`

**Responsibilities**:
- Select provider for requests
- Implement routing strategies
- Failover to backup providers
- Load balancing

**Routing Strategies**:
- **Policy-based**: Route based on policy decisions
- **Round-robin**: Distribute evenly
- **Least-latency**: Route to fastest provider
- **Least-cost**: Route to cheapest provider
- **Weighted**: Distribute by weights
- **Sticky**: Route same user to same provider

---

### 5. Evidence Recorder

**Package**: `pkg/evidence`

**Responsibilities**:
- Record all LLM interactions
- Generate cryptographic signatures
- Store evidence in database
- Prune old evidence (retention)
- Query and export evidence

**Evidence Record**:
```go
type Evidence struct {
    ID         string
    Timestamp  time.Time
    Request    *Request
    Response   *Response
    Decision   *PolicyDecision
    Provider   string
    Cost       float64
    Latency    time.Duration
    Signature  []byte
}
```

**Storage Backends**:
- SQLite (default, single instance)
- PostgreSQL (production, multi-instance)
- S3 (archival, compliance)

---

### 6. Budget & Rate Limiter

**Package**: `pkg/limits`

**Responsibilities**:
- Track spending per user/team
- Enforce budget limits
- Track request rates (RPM, TPM)
- Enforce rate limits

**Limit Types**:
- Per-user budgets
- Per-team budgets
- Global budgets
- Per-user rate limits
- Per-model rate limits

**Implementation**:
- Sliding window rate limiter
- Token bucket algorithm
- In-memory tracking (Redis for multi-instance)

---

### 7. Telemetry

**Package**: `pkg/telemetry`

**Responsibilities**:
- Structured logging (slog)
- Prometheus metrics
- OpenTelemetry traces
- Performance monitoring

**Metrics Categories**:
- Request metrics (count, latency, errors)
- Provider metrics (health, latency, costs)
- Policy metrics (evaluations, denials)
- Evidence metrics (writes, queries)
- Resource metrics (CPU, memory, disk)

---

### 8. Configuration System

**Package**: `pkg/config`

**Responsibilities**:
- Load YAML configuration
- Apply environment variable overrides
- Validate configuration
- Provide singleton access
- Hot-reload support

**Configuration Sections**:
- Proxy (server settings)
- Providers (LLM connections)
- Policy (policy loading)
- Evidence (storage settings)
- Limits (budgets, rates)
- Telemetry (observability)
- Security (TLS, auth)

---

## Request Flow

### Non-Streaming Request

```
Client Request
     │
     ├─► 1. HTTP Server receives request
     │      • Parse OpenAI format
     │      • Authenticate API key
     │
     ├─► 2. Processing Pipeline
     │      • Enrich request metadata
     │      • Estimate tokens & cost
     │
     ├─► 3. Policy Engine
     │      • Evaluate policies
     │      • Check: deny? allow? route?
     │      • Apply modifications
     │
     ├─► 4. Budget & Rate Limiter
     │      • Check budget remaining
     │      • Check rate limits
     │      • Update counters
     │
     ├─► 5. Routing Engine
     │      • Select provider
     │      • Check provider health
     │      • Get provider connection
     │
     ├─► 6. Provider Manager
     │      • Send request to provider
     │      • Handle retries
     │      • Parse provider response
     │
     ├─► 7. Evidence Recorder
     │      • Generate evidence record
     │      • Sign with private key
     │      • Store asynchronously
     │
     ├─► 8. Telemetry
     │      • Log request/response
     │      • Update metrics
     │      • Record trace
     │
     └─► 9. HTTP Server returns response
            • Send to client
            • Close connection
```

**Latency Breakdown** (typical):
- Receive & parse: <1ms
- Policy evaluation: <5ms
- Provider request: 200-2000ms (varies by model)
- Evidence recording: <2ms (async)
- Total overhead: <10ms

### Streaming Request

```
Client Request (stream: true)
     │
     ├─► 1-5. Same as non-streaming
     │
     ├─► 6. Provider Manager (streaming)
     │      • Open streaming connection
     │      • Receive chunks in real-time
     │
     ├─► 7. For each chunk:
     │      • Parse SSE event
     │      • Forward to client immediately
     │      • Accumulate for evidence
     │
     ├─► 8. Stream complete:
     │      • Close stream
     │      • Record evidence
     │      • Update metrics
     │
     └─► 9. Connection closed
```

---

## Component Interaction

### Policy-Driven Routing

```
Request → Policy Engine → Decision: "route to provider X"
              │
              ├─► If route action:
              │   • Routing Engine uses specified provider
              │
              ├─► If deny action:
              │   • Return error to client
              │   • Record evidence of denial
              │
              └─► If allow action:
                  • Routing Engine uses default strategy
```

### Failover Flow

```
Provider Request → Provider A (Primary)
     │
     ├─► Success: Return response
     │
     └─► Failure:
         ├─► Check failover enabled
         ├─► Select Provider B (Backup)
         ├─► Retry request
         ├─► Log failover event
         └─► Return response or error
```

---

## Data Flow

### Request Data Flow

```
Client JSON → Request Struct → Enriched Request → Provider Request → Provider API
                  │                   │                                      │
                  │                   │                                      │
                  ▼                   ▼                                      ▼
             Parse & Validate   Add Metadata                        HTTP Request
                               (user, team, cost)                  (OpenAI format)
```

### Response Data Flow

```
Provider API → Provider Response → Processing → Evidence → Client JSON
      │              │                │            │            │
      │              │                │            │            │
      ▼              ▼                ▼            ▼            ▼
  HTTP Response  Parse & Map    Content Filter  Record &   Format
  (JSON/SSE)     to Standard      (optional)     Sign      Response
```

### Evidence Data Flow

```
Request + Response → Evidence Record → Signature → Storage
                          │               │           │
                          │               │           │
                          ▼               ▼           ▼
                   Serialize JSON    Ed25519 Sign  SQLite/
                   + Metadata        with Key      Postgres/S3
```

---

## Scalability

### Horizontal Scaling

Mercator Jupiter can be scaled horizontally:

```
         Load Balancer
              │
    ┌─────────┼─────────┐
    │         │         │
    ▼         ▼         ▼
Jupiter-1 Jupiter-2 Jupiter-3
    │         │         │
    └─────────┼─────────┘
              │
        Shared Storage
     (PostgreSQL + Redis)
```

**Requirements for multi-instance**:
- Shared PostgreSQL for evidence
- Redis for rate limiting state
- Shared policy storage (Git mode)
- Load balancer (NGINX, HAProxy, K8s Service)

### Vertical Scaling

**CPU**: Policy evaluation, request processing
- Recommendation: 2-4 cores minimum
- Scales linearly with request volume

**Memory**: Connection pools, policy cache, evidence buffer
- Recommendation: 512MB-2GB
- ~100MB baseline + ~1MB per 1000 requests/sec

**Network**: Provider API requests
- Recommendation: 1Gbps minimum
- Scales with request volume and model size

### Performance Targets

- **Throughput**: 1,000+ req/sec per instance
- **Latency**: <10ms proxy overhead (p99)
- **Policy Evaluation**: <5ms (p99)
- **Evidence Recording**: <2ms (async)
- **Memory**: <1GB under load

---

## Security Architecture

### Defense in Depth

```
Layer 1: Network
  • TLS 1.3 encryption
  • mTLS client authentication
  • Firewall rules

Layer 2: Application
  • API key authentication
  • Rate limiting (DoS protection)
  • Input validation

Layer 3: Policy
  • Content filtering (PII, injection)
  • Access control (RBAC)
  • Budget enforcement

Layer 4: Provider
  • Separate API keys per environment
  • Connection encryption
  • Credential rotation

Layer 5: Data
  • Evidence encryption at rest
  • Cryptographic signatures (Ed25519)
  • Secure key storage

Layer 6: Audit
  • Comprehensive logging
  • Evidence trail
  • Signature verification
```

### Trust Boundaries

```
Untrusted → Trusted → Provider → External
             │           │
             │           └─► Provider API (HTTPS)
             │
             └─► Evidence Storage (Encrypted)
```

**Trust Boundaries**:
1. **Client → Jupiter**: TLS + API key
2. **Jupiter → Provider**: HTTPS + provider API key
3. **Jupiter → Storage**: Encrypted connection + signed evidence

---

## See Also

- [Design Decisions](design-decisions.md) - Key architectural choices
- [Data Flow](data-flow.md) - Detailed data flow diagrams
- [Security Model](security-model.md) - Security architecture deep-dive
- [Configuration Reference](../configuration/reference.md) - All configuration options

---

## Package Structure

```
pkg/
├── config/         # Configuration management
├── proxy/          # HTTP proxy server
├── policy/         # Policy engine (MPL)
│   ├── engine/     # Policy evaluation
│   ├── parser/     # MPL parser
│   └── manager/    # Policy loading (file/git)
├── providers/      # LLM provider adapters
│   ├── openai/     # OpenAI implementation
│   ├── anthropic/  # Anthropic implementation
│   ├── ollama/     # Ollama implementation
│   └── registry.go # Provider registry
├── processing/     # Request/response processing
├── routing/        # Request routing
├── evidence/       # Evidence recording
│   ├── recorder/   # Evidence generation
│   ├── storage/    # Storage backends
│   └── crypto/     # Cryptographic signing
├── limits/         # Budget & rate limiting
├── telemetry/      # Logging, metrics, tracing
└── cli/            # CLI command implementations
```
