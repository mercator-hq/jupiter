# Data Flow

Complete request/response flow through Mercator Jupiter.

## High-Level Flow

```
Client → Jupiter Proxy → Policy Engine → Provider → Policy Engine → Client
          ↓                ↓                         ↓
       Auth Check      Evidence Gen            Evidence Gen
```

---

## Detailed Request Flow

### 1. Request Reception

```
┌─────────┐
│ Client  │ POST /v1/chat/completions
└────┬────┘
     │
     ▼
┌─────────────────────────────────┐
│   HTTP Server (pkg/proxy)       │
│   - Parse request                │
│   - Validate JSON                │
│   - Extract headers              │
└────────────┬────────────────────┘
             │
```

### 2. Authentication

```
             │
             ▼
┌─────────────────────────────────┐
│   Authentication Middleware     │
│   - Validate API key             │
│   - Extract user identity        │
│   - Check authorization          │
└────────────┬────────────────────┘
             │
        Valid?
         │   │
    Yes  │   │ No
         │   └──────► 401 Unauthorized
         ▼
```

### 3. Request Enrichment

```
┌─────────────────────────────────┐
│   Processing (pkg/processing)   │
│   - Estimate tokens              │
│   - Calculate cost               │
│   - Analyze content              │
│   - Add metadata                 │
└────────────┬────────────────────┘
             │
```

### 4. Policy Evaluation (Pre-Request)

```
             │
             ▼
┌─────────────────────────────────┐
│   Policy Engine (pkg/policy)    │
│   - Load active policies         │
│   - Evaluate conditions          │
│   - Determine actions            │
└────────────┬────────────────────┘
             │
        Allow?
         │   │
    Yes  │   │ No
         │   └──────► Deny + Evidence → 403 Forbidden
         ▼
```

### 5. Budget & Rate Limit Check

```
┌─────────────────────────────────┐
│   Limits (pkg/limits)            │
│   - Check budget remaining       │
│   - Check rate limits            │
│   - Update usage counters        │
└────────────┬────────────────────┘
             │
     Within limits?
         │   │
    Yes  │   │ No
         │   └──────► 429 Too Many Requests
         ▼
```

### 6. Provider Routing

```
┌─────────────────────────────────┐
│   Routing Engine (pkg/routing)  │
│   - Select provider strategy     │
│   - Choose healthy provider      │
│   - Apply routing policy         │
└────────────┬────────────────────┘
             │
```

### 7. Provider Request

```
             │
             ▼
┌─────────────────────────────────┐
│   Provider Adapter              │
│   - Transform request format     │
│   - Add provider auth            │
│   - Send HTTP request            │
│   - Handle streaming             │
└────────────┬────────────────────┘
             │
        Success?
         │   │
    Yes  │   │ No (retry or failover)
         │   └──────► Try another provider
         ▼
```

### 8. Response Processing

```
┌─────────────────────────────────┐
│   Response Processing           │
│   - Parse response               │
│   - Validate format              │
│   - Extract usage stats          │
│   - Calculate actual cost        │
└────────────┬────────────────────┘
             │
```

### 9. Policy Evaluation (Post-Response)

```
             │
             ▼
┌─────────────────────────────────┐
│   Policy Engine (Response)      │
│   - Evaluate response policies   │
│   - Apply content filtering      │
│   - Redact sensitive data        │
└────────────┬────────────────────┘
             │
        Allow?
         │   │
    Yes  │   │ No
         │   └──────► Deny + Evidence → 403 Forbidden
         ▼
```

### 10. Evidence Generation

```
┌─────────────────────────────────┐
│   Evidence (pkg/evidence)       │
│   - Create evidence record       │
│   - Sign with Ed25519            │
│   - Store in database            │
│   - Update metrics               │
└────────────┬────────────────────┘
             │
```

### 11. Response Delivery

```
             │
             ▼
┌─────────────────────────────────┐
│   HTTP Response                 │
│   - Format response              │
│   - Add headers                  │
│   - Stream or complete           │
└────────────┬────────────────────┘
             │
             ▼
        ┌─────────┐
        │ Client  │
        └─────────┘
```

---

## Streaming Flow

For streaming responses (SSE):

```
Client                  Jupiter                 Provider
  │                       │                        │
  │   POST /chat ────────►│                        │
  │   (stream=true)       │   POST /chat ─────────►│
  │                       │   (stream=true)        │
  │                       │                        │
  │ ◄────── 200 OK ───────│ ◄────── 200 OK ────────│
  │ Connection: keep-alive│                        │
  │                       │                        │
  │ ◄─── data: chunk1 ────│ ◄─── data: chunk1 ─────│
  │ ◄─── data: chunk2 ────│ ◄─── data: chunk2 ─────│
  │         ...           │         ...            │
  │ ◄─── data: [DONE] ────│ ◄─── data: [DONE] ─────│
  │                       │                        │
  │                       │ Generate Evidence      │
  │                       │ (after stream complete)│
  └───────────────────────┴────────────────────────┘
```

---

## Evidence Data Flow

```
Request → Policy Eval → Provider → Response → Evidence Record
                                                     │
                                                     ▼
                                              ┌─────────────┐
                                              │  Serialize  │
                                              │   to JSON   │
                                              └──────┬──────┘
                                                     │
                                                     ▼
                                              ┌─────────────┐
                                              │ Sign with   │
                                              │  Ed25519    │
                                              └──────┬──────┘
                                                     │
                                                     ▼
                                              ┌─────────────┐
                                              │   Store in  │
                                              │   SQLite/PG │
                                              └─────────────┘
```

---

## Configuration Loading

```
Startup
   │
   ▼
┌──────────────────────────────┐
│  Load config.yaml            │
│  - Parse YAML                │
│  - Validate schema           │
│  - Expand env vars           │
└───────────┬──────────────────┘
            │
            ▼
┌──────────────────────────────┐
│  Initialize Components       │
│  - HTTP server               │
│  - Providers                 │
│  - Policy engine             │
│  - Evidence storage          │
│  - Metrics exporter          │
└───────────┬──────────────────┘
            │
            ▼
┌──────────────────────────────┐
│  Start Services              │
│  - HTTP listener             │
│  - Metrics endpoint          │
│  - Health check endpoint     │
│  - Policy file watcher       │
└──────────────────────────────┘
```

---

## Policy Management Flow

### File Mode

```
Policy File Changed
        │
        ▼
┌──────────────────┐
│  File Watcher    │
│  detects change  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Reload Policy   │
│  - Parse YAML    │
│  - Validate      │
│  - Compile       │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Atomic Swap     │
│  (live reload)   │
└──────────────────┘
```

### Git Mode

```
┌──────────────────┐
│  Git Poll        │
│  (every 60s)     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Fetch Latest    │
│  - git pull      │
│  - Check hash    │
└────────┬─────────┘
         │
    Changed?
     │     │
Yes  │     │ No → Sleep
     ▼     │
┌──────────────────┐
│  Reload Policy   │
└──────────────────┘
```

---

## Metrics Collection Flow

```
Request/Response Event
         │
         ▼
┌──────────────────────────┐
│  Telemetry               │
│  (pkg/telemetry)         │
│  - Update counters       │
│  - Record latency        │
│  - Track errors          │
└───────────┬──────────────┘
            │
            ▼
┌──────────────────────────┐
│  Prometheus Registry     │
│  - Store metrics         │
│  - Calculate rates       │
└───────────┬──────────────┘
            │
            ▼
┌──────────────────────────┐
│  /metrics Endpoint       │
│  - Scrape by Prometheus  │
└──────────────────────────┘
```

---

## Error Flow

```
Error Occurs
     │
     ▼
┌──────────────────────┐
│  Error Handler       │
│  - Classify error    │
│  - Add context       │
│  - Log structured    │
└───────┬──────────────┘
        │
        ├─────► Metrics (error counter)
        │
        ├─────► Evidence (if applicable)
        │
        └─────► HTTP Error Response
                 - 400 Bad Request
                 - 401 Unauthorized
                 - 403 Forbidden
                 - 429 Too Many Requests
                 - 500 Internal Error
                 - 502 Bad Gateway
                 - 503 Service Unavailable
```

---

## Key Data Structures

### Request Context

```go
type RequestContext struct {
    RequestID     string
    UserID        string
    Timestamp     time.Time
    Model         string
    Provider      string
    Messages      []Message
    TokenEstimate int
    CostEstimate  float64
    Metadata      map[string]interface{}
}
```

### Evidence Record

```go
type EvidenceRecord struct {
    ID          string
    Timestamp   time.Time
    RequestID   string
    UserID      string
    Provider    string
    Model       string
    Request     []byte    // JSON
    Response    []byte    // JSON
    PolicyDecisions []PolicyDecision
    TokenUsage  TokenUsage
    Cost        float64
    Signature   []byte    // Ed25519
}
```

### Policy Decision

```go
type PolicyDecision struct {
    PolicyName  string
    RuleName    string
    Action      Action    // Allow, Deny, Modify, etc.
    Reason      string
    Timestamp   time.Time
    Evaluation  time.Duration
}
```

---

## Performance Characteristics

| Stage | Latency | Notes |
|-------|---------|-------|
| Auth | <1ms | In-memory lookup |
| Policy Eval | 1-3ms | Interpreted MPL |
| Routing | <1ms | Strategy selection |
| Provider | 100-2000ms | Network + LLM |
| Evidence | <1ms | Async write |
| **Total Overhead** | **<5ms** | Excludes provider |

---

## See Also

- [Architecture Overview](overview.md)
- [Design Decisions](design-decisions.md)
- [Security Model](security-model.md)
