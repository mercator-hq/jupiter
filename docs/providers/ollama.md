# Ollama Provider Setup

Complete guide to configuring Ollama for running local LLM models with Mercator Jupiter.

## Table of Contents

- [What is Ollama?](#what-is-ollama)
- [Installation](#installation)
- [Basic Configuration](#basic-configuration)
- [Model Management](#model-management)
- [Connection Settings](#connection-settings)
- [Performance Tuning](#performance-tuning)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## What is Ollama?

**Ollama** is a tool for running large language models locally on your machine. Benefits:

- ✅ **No API costs** - Run models for free
- ✅ **Data privacy** - Data never leaves your machine
- ✅ **Offline operation** - No internet required
- ✅ **Low latency** - Local processing
- ✅ **Customization** - Fine-tune models for your use case

**Limitations**:
- ❌ Requires powerful hardware (GPU recommended)
- ❌ Slower than cloud APIs (depending on hardware)
- ❌ Limited to open-source models
- ❌ Requires local storage (models are 4GB-70GB+)

---

## Installation

### Install Ollama

#### macOS

```bash
# Download and install from website
curl https://ollama.ai/install.sh | sh

# Or use Homebrew
brew install ollama
```

#### Linux

```bash
curl https://ollama.ai/install.sh | sh
```

#### Windows

Download from: https://ollama.ai/download/windows

#### Docker

```bash
docker pull ollama/ollama

docker run -d \
  --name ollama \
  -p 11434:11434 \
  -v ollama:/root/.ollama \
  ollama/ollama
```

### Start Ollama Server

```bash
# Start Ollama service
ollama serve

# Ollama will listen on http://localhost:11434
```

### Verify Installation

```bash
# Check Ollama version
ollama --version

# Test with a small model
ollama run llama2
```

---

## Basic Configuration

### Minimal Ollama Setup

```yaml
# config.yaml
providers:
  ollama:
    base_url: "http://localhost:11434"
    timeout: "120s"  # Local models can be slow
```

### Full Ollama Configuration

```yaml
providers:
  ollama:
    # API endpoint (local)
    base_url: "http://localhost:11434"

    # No API key needed for local Ollama
    # api_key: ""

    # Timeouts (local models are slower)
    timeout: "180s"

    # Retry configuration
    max_retries: 1  # Don't retry local failures

    # Connection settings
    connection_pool:
      max_idle_conns: 10
      idle_timeout: "60s"

    # Health checking
    health_check_interval: "30s"
    health_check_timeout: "5s"
```

---

## Model Management

### Downloading Models

```bash
# Pull a model
ollama pull llama2

# Pull specific version
ollama pull llama2:13b

# Pull latest version
ollama pull llama2:latest
```

### Available Models

Popular open-source models:

| Model | Size | Parameters | Context | Use Case |
|-------|------|------------|---------|----------|
| `llama2` | 3.8GB | 7B | 4K | General purpose |
| `llama2:13b` | 7.3GB | 13B | 4K | Better quality |
| `llama2:70b` | 39GB | 70B | 4K | Highest quality |
| `mistral` | 4.1GB | 7B | 8K | Fast, efficient |
| `mixtral` | 26GB | 8x7B | 32K | Large context |
| `codellama` | 3.8GB | 7B | 16K | Code generation |
| `phi` | 1.6GB | 2.7B | 2K | Tiny, fast |

See all models: https://ollama.ai/library

### List Installed Models

```bash
# List local models
ollama list

# Output:
# NAME              SIZE    MODIFIED
# llama2:latest     3.8GB   2 days ago
# mistral:latest    4.1GB   5 days ago
```

### Remove Models

```bash
# Remove a model to free space
ollama rm llama2:13b
```

### Model Routing Policy

```yaml
# policies.yaml
version: "1.0"

policies:
  - name: "ollama-model-routing"
    description: "Route local models to Ollama"
    rules:
      # Route Llama models to Ollama
      - condition: 'request.model matches "^llama"'
        action: "route"
        provider: "ollama"

      # Route Mistral to Ollama
      - condition: 'request.model matches "^mistral"'
        action: "route"
        provider: "ollama"

      # Route CodeLlama to Ollama
      - condition: 'request.model matches "^codellama"'
        action: "route"
        provider: "ollama"
```

---

## Connection Settings

### Timeout Configuration

Local models are slower than cloud APIs:

```yaml
providers:
  ollama:
    timeout: "180s"  # 3 minutes for large models
```

**Recommendations by model size**:
- Small (< 5GB): 60-120 seconds
- Medium (5-15GB): 120-180 seconds
- Large (> 15GB): 180-300 seconds

### Remote Ollama Server

If Ollama is running on a different machine:

```yaml
providers:
  ollama:
    base_url: "http://ollama-server:11434"
    timeout: "120s"
```

### Ollama with Docker

```yaml
providers:
  ollama:
    base_url: "http://ollama-container:11434"
    timeout: "120s"
```

---

## Performance Tuning

### Hardware Requirements

**Minimum**:
- CPU: 4+ cores
- RAM: 8GB
- Storage: 20GB free

**Recommended**:
- CPU: 8+ cores
- RAM: 16GB+ (32GB for large models)
- GPU: NVIDIA GPU with 8GB+ VRAM
- Storage: 100GB+ SSD

### GPU Acceleration

Ollama automatically uses GPU if available:

```bash
# Check if GPU is being used
ollama ps

# Force CPU mode (for testing)
OLLAMA_NUM_GPU=0 ollama serve
```

### Parallel Requests

Configure concurrent model loading:

```bash
# Allow 2 models loaded simultaneously
OLLAMA_NUM_PARALLEL=2 ollama serve

# Control memory allocation
OLLAMA_MAX_LOADED_MODELS=2 ollama serve
```

### Model-Specific Configuration

```yaml
policies:
  - name: "ollama-performance-tuning"
    rules:
      # Use smaller models for simple tasks
      - condition: |
          request.metadata.task_complexity == "low" and
          request.model == "llama2"
        action: "modify"
        set:
          model: "phi"  # Smaller, faster
        log_message: "Using Phi model for simple task"

      # Limit context for faster response
      - condition: |
          request.estimated_prompt_tokens > 2000
        action: "modify"
        set:
          max_tokens: 500
        log_message: "Limiting response length for performance"
```

---

## Best Practices

### 1. Model Selection

```yaml
policies:
  - name: "optimal-model-selection"
    rules:
      # Simple queries -> small fast model
      - condition: |
          request.estimated_prompt_tokens < 500
        action: "modify"
        set:
          model: "phi"

      # Code generation -> CodeLlama
      - condition: |
          request.metadata.task_type == "code"
        action: "modify"
        set:
          model: "codellama"

      # General tasks -> Mistral (good balance)
      - condition: "true"
        action: "modify"
        set:
          model: "mistral"
```

### 2. Development Workflow

Use Ollama for development, cloud for production:

```yaml
policies:
  - name: "dev-prod-routing"
    rules:
      # Development: Use free local Ollama
      - condition: |
          request.metadata.environment == "development"
        action: "route"
        provider: "ollama"
        log_message: "Dev environment - using local Ollama"

      # Production: Use reliable cloud API
      - condition: |
          request.metadata.environment == "production"
        action: "route"
        provider: "openai"
        log_message: "Production - using OpenAI"
```

### 3. Cost Optimization

```yaml
policies:
  - name: "cost-optimization"
    rules:
      # Use Ollama when budget low
      - condition: |
          request.metadata.user_budget_remaining < 5.0
        action: "route"
        provider: "ollama"
        log_message: "Low budget - routing to free Ollama"

      # Use cloud APIs for critical requests
      - condition: |
          request.metadata.priority == "high"
        action: "route"
        provider: "openai"
```

### 4. Privacy-Sensitive Data

```yaml
policies:
  - name: "privacy-routing"
    rules:
      # Sensitive data stays local
      - condition: |
          request.metadata.contains_pii == true or
          request.metadata.data_classification == "confidential"
        action: "route"
        provider: "ollama"
        log_message: "Sensitive data - using local Ollama"
```

### 5. Monitoring Performance

```bash
# Check Ollama status
curl http://localhost:11434/api/tags

# Monitor model loading
ollama ps

# Check system resources
# macOS
top -pid $(pgrep ollama)

# Linux
htop -p $(pgrep ollama)
```

### 6. Pre-loading Models

Pre-load frequently used models:

```bash
# Keep model loaded
ollama run llama2 "test"

# Or use API
curl http://localhost:11434/api/generate -d '{
  "model": "llama2",
  "keep_alive": -1
}'
```

---

## Advanced Configuration

### Custom Models

Create custom model configurations:

```bash
# Create Modelfile
cat > Modelfile << 'EOF'
FROM llama2

# Set temperature
PARAMETER temperature 0.7

# Set system prompt
SYSTEM You are a helpful coding assistant.
EOF

# Create custom model
ollama create my-code-assistant -f Modelfile

# Use in Mercator
```

```yaml
providers:
  ollama:
    base_url: "http://localhost:11434"

# In policy
- condition: 'request.metadata.assistant_type == "code"'
  action: "modify"
  set:
    model: "my-code-assistant"
```

### Multiple Ollama Instances

Run multiple Ollama servers for isolation:

```bash
# Start Ollama on different ports
OLLAMA_HOST=0.0.0.0:11435 ollama serve &
OLLAMA_HOST=0.0.0.0:11436 ollama serve &
```

```yaml
providers:
  ollama-1:
    base_url: "http://localhost:11434"
  ollama-2:
    base_url: "http://localhost:11435"
  ollama-3:
    base_url: "http://localhost:11436"

routing:
  strategy: "round-robin"
```

### Ollama with Kubernetes

```yaml
# kubernetes/ollama-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ollama
  template:
    metadata:
      labels:
        app: ollama
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
        resources:
          requests:
            memory: "8Gi"
            nvidia.com/gpu: 1  # Request GPU
          limits:
            memory: "16Gi"
            nvidia.com/gpu: 1
        volumeMounts:
        - name: ollama-data
          mountPath: /root/.ollama
      volumes:
      - name: ollama-data
        persistentVolumeClaim:
          claimName: ollama-pvc
```

---

## Troubleshooting

### Issue: "Connection Refused"

**Symptoms**: Cannot connect to Ollama

**Solutions**:
1. Check if Ollama is running:
   ```bash
   pgrep ollama
   ```
2. Start Ollama:
   ```bash
   ollama serve
   ```
3. Verify port:
   ```bash
   lsof -i :11434
   ```
4. Check base_url in config

### Issue: "Model Not Found"

**Symptoms**: 404 errors

**Solutions**:
1. List installed models:
   ```bash
   ollama list
   ```
2. Pull the model:
   ```bash
   ollama pull llama2
   ```
3. Check model name spelling

### Issue: Slow Responses

**Symptoms**: Timeouts, high latency

**Solutions**:
1. Use smaller model:
   ```bash
   ollama pull phi  # 1.6GB, fast
   ```
2. Enable GPU acceleration (check with `ollama ps`)
3. Increase RAM allocation
4. Reduce max_tokens
5. Pre-load models

### Issue: Out of Memory

**Symptoms**: Ollama crashes, OOM errors

**Solutions**:
1. Use smaller model
2. Limit concurrent models:
   ```bash
   OLLAMA_MAX_LOADED_MODELS=1 ollama serve
   ```
3. Increase system RAM
4. Close other applications

### Issue: GPU Not Detected

**Symptoms**: Ollama uses CPU instead of GPU

**Solutions**:
1. Check NVIDIA drivers:
   ```bash
   nvidia-smi
   ```
2. Install CUDA toolkit
3. Restart Ollama after driver installation
4. Check Docker GPU access (if using Docker)

---

## Example Configurations

### Development Configuration

```yaml
providers:
  ollama:
    base_url: "http://localhost:11434"
    timeout: "60s"
```

### Production Configuration

```yaml
providers:
  ollama:
    base_url: "http://ollama-prod:11434"
    timeout: "120s"
    max_retries: 1

    health_check_interval: "30s"
```

### Hybrid Configuration

Use both local and cloud:

```yaml
providers:
  ollama:
    base_url: "http://localhost:11434"
    timeout: "120s"

  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"

# Route based on requirements
routing:
  strategy: "policy"
```

---

## See Also

- [Provider Configuration Reference](../configuration/reference.md#provider-configuration)
- [Routing Guide](../policies/routing.md)
- [Ollama Documentation](https://github.com/ollama/ollama)
- [Ollama Model Library](https://ollama.ai/library)

---

## Quick Reference

### Ollama Commands

```bash
# Start server
ollama serve

# Pull model
ollama pull llama2

# List models
ollama list

# Remove model
ollama rm llama2

# Run model interactively
ollama run llama2

# Show model info
ollama show llama2

# Check running models
ollama ps
```

### Environment Variables

```bash
OLLAMA_HOST="0.0.0.0:11434"      # Bind address
OLLAMA_NUM_GPU=1                  # Number of GPUs to use
OLLAMA_NUM_PARALLEL=1             # Parallel requests
OLLAMA_MAX_LOADED_MODELS=1        # Max models in memory
```

### Monitoring

```bash
# Check Ollama health
curl http://localhost:11434/api/tags

# Test model
curl http://localhost:11434/api/generate -d '{
  "model": "llama2",
  "prompt": "Hello"
}'

# Check via Mercator
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "llama2", "messages": [{"role": "user", "content": "test"}]}'
```

### Model Comparison

| Model | Size | Speed | Quality | Context | Best For |
|-------|------|-------|---------|---------|----------|
| phi | 1.6GB | ⚡⚡⚡⚡⚡ | ⭐⭐⭐ | 2K | Simple queries |
| llama2 | 3.8GB | ⚡⚡⚡⚡ | ⭐⭐⭐⭐ | 4K | General use |
| mistral | 4.1GB | ⚡⚡⚡⚡ | ⭐⭐⭐⭐ | 8K | Balanced |
| codellama | 3.8GB | ⚡⚡⚡ | ⭐⭐⭐⭐ | 16K | Code generation |
| mixtral | 26GB | ⚡⚡ | ⭐⭐⭐⭐⭐ | 32K | Complex tasks |
| llama2:70b | 39GB | ⚡ | ⭐⭐⭐⭐⭐ | 4K | Highest quality |
