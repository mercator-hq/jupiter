# Routing Policies

Guide to implementing intelligent request routing across multiple LLM providers for cost optimization, performance, and reliability.

## Table of Contents

- [Model-Based Routing](#model-based-routing)
- [Cost-Optimized Routing](#cost-optimized-routing)
- [Performance-Based Routing](#performance-based-routing)
- [Failover & High Availability](#failover--high-availability)
- [Geographic Routing](#geographic-routing)
- [Load Balancing](#load-balancing)
- [Best Practices](#best-practices)

---

## Model-Based Routing

**Use Case**: Route requests to appropriate providers based on the requested model.

### Basic Model Routing

**File**: [docs/mpl/examples/04-model-routing.yaml](../mpl/examples/04-model-routing.yaml)

```yaml
version: "1.0"

policies:
  - name: "model-routing"
    description: "Route requests to appropriate providers"
    priority: 100
    rules:
      # Route GPT models to OpenAI
      - condition: 'request.model matches "^gpt-"'
        action: "route"
        provider: "openai"
        log_message: "Routing {{request.model}} to OpenAI"

      # Route Claude models to Anthropic
      - condition: 'request.model matches "^claude-"'
        action: "route"
        provider: "anthropic"
        log_message: "Routing {{request.model}} to Anthropic"

      # Route Llama models to Ollama (local)
      - condition: 'request.model matches "^llama"'
        action: "route"
        provider: "ollama"
        log_message: "Routing {{request.model}} to local Ollama"

      # Deny unknown models
      - condition: "true"
        action: "deny"
        reason: "Unknown model: {{request.model}}. Supported: GPT (OpenAI), Claude (Anthropic), Llama (Ollama)"
```

### Configuration

```yaml
# config.yaml
providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"

  anthropic:
    base_url: "https://api.anthropic.com/v1"
    api_key: "${ANTHROPIC_API_KEY}"

  ollama:
    base_url: "http://localhost:11434"

routing:
  strategy: "policy"  # Use policy-based routing
```

### Model Family Routing

```yaml
policies:
  - name: "model-family-routing"
    rules:
      # GPT-4 family -> OpenAI primary
      - condition: 'request.model matches "^gpt-4"'
        action: "route"
        provider: "openai-premium"

      # GPT-3.5 family -> OpenAI standard or cheaper alternative
      - condition: 'request.model matches "^gpt-3.5"'
        action: "route"
        provider: "openai-standard"

      # Claude Opus -> Anthropic premium
      - condition: 'request.model == "claude-3-opus"'
        action: "route"
        provider: "anthropic-premium"

      # Claude Sonnet/Haiku -> Anthropic standard
      - condition: 'request.model matches "^claude-3-(sonnet|haiku)"'
        action: "route"
        provider: "anthropic-standard"
```

---

## Cost-Optimized Routing

**Use Case**: Route to the most cost-effective provider for each request.

### Cost-Based Routing

```yaml
version: "1.0"

policies:
  - name: "cost-optimized-routing"
    description: "Route to cheapest available provider"
    priority: 100
    rules:
      # Budget-conscious users -> cheaper providers
      - condition: |
          request.model == "gpt-3.5-turbo" and
          request.metadata.user_budget_remaining < 10.0
        action: "route"
        provider: "openrouter"  # Cheaper alternative
        log_message: "Cost optimization: routing to OpenRouter"

      # Standard routing for healthy budgets
      - condition: |
          request.model == "gpt-3.5-turbo" and
          request.metadata.user_budget_remaining >= 10.0
        action: "route"
        provider: "openai"
        log_message: "Standard routing to OpenAI"
```

### Provider Cost Tiers

```yaml
policies:
  - name: "tiered-cost-routing"
    rules:
      # Tier 1: Premium (highest quality, highest cost)
      - condition: |
          request.metadata.user_tier == "enterprise" and
          request.model == "gpt-4"
        action: "route"
        provider: "openai-premium"

      # Tier 2: Standard (balanced)
      - condition: |
          request.metadata.user_tier == "pro" and
          request.model == "gpt-4"
        action: "route"
        provider: "openai-standard"

      # Tier 3: Economy (lowest cost)
      - condition: |
          request.metadata.user_tier == "free" and
          request.model == "gpt-3.5-turbo"
        action: "route"
        provider: "openrouter"
```

### Dynamic Cost Routing

```yaml
# Route based on real-time provider costs
- condition: |
    provider.openai.current_cost_per_token < provider.azure.current_cost_per_token
  action: "route"
  provider: "openai"
  log_message: "OpenAI is cheaper right now"

- condition: "true"
  action: "route"
  provider: "azure"
  log_message: "Azure is cheaper right now"
```

---

## Performance-Based Routing

**Use Case**: Route to the fastest provider based on latency and availability.

### Latency-Based Routing

```yaml
version: "1.0"

policies:
  - name: "latency-routing"
    description: "Route to fastest available provider"
    priority: 100
    rules:
      # Route to provider with lowest latency
      - condition: |
          provider.openai.avg_latency_ms < provider.azure.avg_latency_ms
        action: "route"
        provider: "openai"
        log_message: "Routing to OpenAI (lower latency)"

      - condition: "true"
        action: "route"
        provider: "azure"
        log_message: "Routing to Azure (lower latency)"
```

### Configuration

```yaml
routing:
  strategy: "least-latency"
  health_check_interval: "30s"
```

### Performance Tiers

```yaml
policies:
  - name: "performance-tiered-routing"
    rules:
      # Low-latency requirement -> fastest provider
      - condition: |
          request.metadata.latency_requirement == "low" and
          request.model == "gpt-3.5-turbo"
        action: "route"
        provider: "openai-us-east"  # Closest region

      # Standard latency -> any available provider
      - condition: |
          request.metadata.latency_requirement == "standard"
        action: "route"
        provider: "openai-us-west"

      # Batch processing -> cheapest, latency doesn't matter
      - condition: |
          request.metadata.priority == "batch"
        action: "route"
        provider: "economy-provider"
```

---

## Failover & High Availability

**Use Case**: Automatically failover to backup providers when primary fails.

### Basic Failover

```yaml
version: "1.0"

policies:
  - name: "failover-routing"
    description: "Automatic failover to backup providers"
    priority: 100
    rules:
      # Primary: OpenAI
      - condition: |
          request.model == "gpt-4" and
          provider.openai.health_status == "healthy"
        action: "route"
        provider: "openai"

      # Failover 1: Azure OpenAI
      - condition: |
          request.model == "gpt-4" and
          provider.openai.health_status != "healthy" and
          provider.azure.health_status == "healthy"
        action: "route"
        provider: "azure"
        log_level: "warn"
        log_message: "FAILOVER: OpenAI unhealthy, routing to Azure"

      # Failover 2: OpenRouter
      - condition: |
          request.model == "gpt-4" and
          provider.openai.health_status != "healthy" and
          provider.azure.health_status != "healthy" and
          provider.openrouter.health_status == "healthy"
        action: "route"
        provider: "openrouter"
        log_level: "error"
        log_message: "FAILOVER: Primary and secondary down, routing to OpenRouter"

      # All providers down
      - condition: |
          provider.openai.health_status != "healthy" and
          provider.azure.health_status != "healthy" and
          provider.openrouter.health_status != "healthy"
        action: "deny"
        reason: "All providers are currently unavailable. Please try again later."
```

### Configuration

```yaml
routing:
  failover:
    enabled: true
    max_retries: 2
    retry_delay: "1s"
```

### Circuit Breaker Pattern

```yaml
policies:
  - name: "circuit-breaker"
    rules:
      # Open circuit breaker after 5 failures
      - condition: |
          provider.openai.consecutive_failures >= 5
        action: "route"
        provider: "azure"
        log_message: "Circuit breaker OPEN for OpenAI, routing to Azure"

      # Half-open: Try primary occasionally
      - condition: |
          provider.openai.circuit_breaker_state == "half-open" and
          time.now % 60 < 5
        action: "route"
        provider: "openai"
        log_message: "Circuit breaker HALF-OPEN, testing OpenAI"

      # Closed: Normal operation
      - condition: |
          provider.openai.circuit_breaker_state == "closed"
        action: "route"
        provider: "openai"
```

---

## Geographic Routing

**Use Case**: Route requests to providers in specific geographic regions for compliance or performance.

### Data Residency Routing

**File**: [docs/mpl/examples/09-data-residency.yaml](../mpl/examples/09-data-residency.yaml)

```yaml
version: "1.0"

policies:
  - name: "geographic-routing"
    description: "Route based on user location for data residency"
    priority: 150
    rules:
      # EU users -> EU providers only
      - condition: |
          request.metadata.user_region == "EU"
        action: "route"
        provider: "azure-eu-west"
        log_message: "EU user routed to EU provider (GDPR compliance)"

      # US users -> US providers
      - condition: |
          request.metadata.user_region == "US"
        action: "route"
        provider: "openai-us-east"
        log_message: "US user routed to US provider"

      # APAC users -> APAC providers
      - condition: |
          request.metadata.user_region == "APAC"
        action: "route"
        provider: "azure-asia-southeast"
        log_message: "APAC user routed to APAC provider"

      # Block cross-border routing for EU
      - condition: |
          request.metadata.user_region == "EU" and
          provider.selected.region != "EU"
        action: "deny"
        reason: "Data residency violation: EU data cannot leave EU"
```

### Regional Performance Optimization

```yaml
policies:
  - name: "regional-performance"
    rules:
      # Route to nearest provider
      - condition: |
          request.metadata.user_location == "us-east"
        action: "route"
        provider: "openai-us-east-1"

      - condition: |
          request.metadata.user_location == "us-west"
        action: "route"
        provider: "openai-us-west-2"

      - condition: |
          request.metadata.user_location == "eu-west"
        action: "route"
        provider: "azure-eu-west-1"
```

---

## Load Balancing

**Use Case**: Distribute requests across multiple providers for better performance and reliability.

### Round-Robin Load Balancing

```yaml
version: "1.0"

policies:
  - name: "load-balancing"
    description: "Distribute load across providers"
    priority: 100
    rules:
      # Distribute evenly
      - condition: |
          request.metadata.request_id % 3 == 0
        action: "route"
        provider: "openai-1"

      - condition: |
          request.metadata.request_id % 3 == 1
        action: "route"
        provider: "openai-2"

      - condition: |
          request.metadata.request_id % 3 == 2
        action: "route"
        provider: "openai-3"
```

### Configuration

```yaml
routing:
  strategy: "round-robin"
```

### Weighted Load Balancing

```yaml
policies:
  - name: "weighted-load-balancing"
    rules:
      # 60% to primary (highest capacity)
      - condition: |
          random() < 0.6
        action: "route"
        provider: "openai-primary"

      # 30% to secondary
      - condition: |
          random() < 0.9
        action: "route"
        provider: "openai-secondary"

      # 10% to tertiary (testing/canary)
      - condition: "true"
        action: "route"
        provider: "openai-canary"
```

### Sticky Routing

```yaml
policies:
  - name: "sticky-routing"
    description: "Route same user to same provider"
    rules:
      # Hash user_id to consistently route to same provider
      - condition: |
          hash(request.metadata.user_id) % 2 == 0
        action: "route"
        provider: "openai-a"

      - condition: "true"
        action: "route"
        provider: "openai-b"
```

### Configuration

```yaml
routing:
  sticky_routing: true
  sticky_key: "user_id"
```

---

## Best Practices

### 1. Prioritize Routing Rules

```yaml
policies:
  # Priority 400: Compliance (must be first)
  - name: "data-residency"
    priority: 400

  # Priority 300: Failover (handle failures)
  - name: "failover"
    priority: 300

  # Priority 200: Cost optimization
  - name: "cost-routing"
    priority: 200

  # Priority 100: Load balancing (default)
  - name: "load-balancing"
    priority: 100
```

### 2. Monitor Provider Health

```bash
# Check provider status
curl http://localhost:8080/metrics | grep provider_health

# View routing decisions
mercator evidence query \
  --time-range "last 1 hour" \
  --format json | \
  jq '.[] | {provider: .provider, latency: .latency_ms, cost: .cost}'
```

### 3. Test Failover

```yaml
# failover-tests.yaml
tests:
  - name: "Should failover to Azure when OpenAI down"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
    provider_state:
      openai: "unhealthy"
      azure: "healthy"
    expected:
      provider: "azure"
      logs_contain: "FAILOVER"

  - name: "Should deny when all providers down"
    request:
      model: "gpt-4"
      messages:
        - role: "user"
          content: "Test"
    provider_state:
      openai: "unhealthy"
      azure: "unhealthy"
    expected:
      action: "deny"
      reason_contains: "unavailable"
```

### 4. Gradual Traffic Shifting

```yaml
# Canary deployment: shift traffic gradually
policies:
  - name: "canary-routing"
    rules:
      # Week 1: 5% to new provider
      - condition: |
          request.metadata.deployment_week == 1 and
          random() < 0.05
        action: "route"
        provider: "new-provider"

      # Week 2: 25%
      - condition: |
          request.metadata.deployment_week == 2 and
          random() < 0.25
        action: "route"
        provider: "new-provider"

      # Week 3: 50%
      - condition: |
          request.metadata.deployment_week == 3 and
          random() < 0.50
        action: "route"
        provider: "new-provider"

      # Week 4: 100%
      - condition: |
          request.metadata.deployment_week >= 4
        action: "route"
        provider: "new-provider"
```

### 5. A/B Testing

```yaml
policies:
  - name: "ab-test-routing"
    rules:
      # Control group: OpenAI
      - condition: |
          hash(request.metadata.user_id) % 2 == 0
        action: "route"
        provider: "openai"
        metadata:
          ab_group: "control"

      # Treatment group: Azure
      - condition: "true"
        action: "route"
        provider: "azure"
        metadata:
          ab_group: "treatment"
```

### 6. Model Compatibility Mapping

```yaml
policies:
  - name: "model-compatibility"
    rules:
      # Map compatible models across providers
      - condition: |
          request.model == "gpt-3.5-turbo" and
          provider.openai.health_status != "healthy"
        action: "modify"
        set:
          model: "gpt-35-turbo"  # Azure naming
          provider: "azure"

      - condition: |
          request.model == "claude-3-opus" and
          provider.anthropic.health_status != "healthy"
        action: "modify"
        set:
          model: "claude-3-opus-20240229"  # Different API format
          provider: "bedrock"
```

### 7. Cost-Aware Failover

```yaml
# Failover with cost considerations
- condition: |
    provider.openai.health_status != "healthy"
  action: "route"
  provider: "azure"  # More expensive but available
  log_level: "warn"
  log_message: "Failover to higher-cost provider"
  metadata:
    cost_increase: true
```

---

## Advanced Routing Patterns

### Multi-Criteria Routing

```yaml
policies:
  - name: "multi-criteria-routing"
    rules:
      # High-value users get best provider
      - condition: |
          request.metadata.user_tier == "enterprise" and
          provider.openai-premium.health_status == "healthy"
        action: "route"
        provider: "openai-premium"

      # Cost-sensitive + fast required -> tradeoff
      - condition: |
          request.metadata.cost_sensitive == true and
          request.metadata.latency_requirement == "low"
        action: "route"
        provider: "azure-standard"  # Balanced

      # Batch processing -> cheapest available
      - condition: |
          request.metadata.priority == "batch"
        action: "route"
        provider: "economy-provider"
```

### Time-Based Routing

```yaml
# Route based on time of day
- condition: |
    time.hour >= 9 and time.hour < 17
  action: "route"
  provider: "openai-primary"
  log_message: "Business hours - primary provider"

- condition: |
    time.hour < 9 or time.hour >= 17
  action: "route"
  provider: "openai-economy"
  log_message: "Off-hours - economy provider"
```

---

## See Also

- [Policy Cookbook](cookbook.md) - All policy examples
- [Configuration Reference](../configuration/reference.md) - Routing configuration
- [Provider Setup](../providers/openai.md) - Provider configuration
- [Observability Guide](../observability-guide.md) - Monitoring routing

---

## Complete Example

See [docs/mpl/examples/04-model-routing.yaml](../mpl/examples/04-model-routing.yaml) for a production-ready routing policy.
