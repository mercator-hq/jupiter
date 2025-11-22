# Docker Deployment Guide

Complete guide for deploying Mercator Jupiter using Docker.

## Table of Contents

- [Quick Start](#quick-start)
- [Single Container Deployment](#single-container-deployment)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Production Configuration](#production-configuration)
- [Networking](#networking)
- [Persistent Storage](#persistent-storage)
- [Environment Variables](#environment-variables)
- [Health Checks](#health-checks)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

```bash
# Pull the image
docker pull mercator-hq/jupiter:latest

# Run with minimal configuration
docker run -d \
  --name mercator-jupiter \
  -p 8080:8080 \
  -e OPENAI_API_KEY="$OPENAI_API_KEY" \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/policies.yaml:/app/policies/policy.yaml:ro \
  mercator-hq/jupiter:latest
```

---

## Single Container Deployment

### 1. Build from Source

```bash
# Clone the repository
git clone https://github.com/mercator-hq/jupiter.git
cd jupiter

# Build the image
docker build -t mercator-jupiter:latest -f examples/docker/Dockerfile .

# Verify the build
docker images mercator-jupiter
```

### 2. Create Configuration

```yaml
# config.yaml
proxy:
  listen_address: "0.0.0.0:8080"

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    timeout: "60s"

policy:
  mode: "file"
  file_path: "/app/policies/policy.yaml"

evidence:
  enabled: true
  backend: "sqlite"
  sqlite:
    path: "/app/data/evidence.db"

telemetry:
  logging:
    level: "info"
    format: "json"
```

### 3. Run the Container

```bash
docker run -d \
  --name mercator-jupiter \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 9090:9090 \
  -e OPENAI_API_KEY="$OPENAI_API_KEY" \
  -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/policies:/app/policies:ro \
  -v jupiter-data:/app/data \
  mercator-jupiter:latest run --config /app/config/config.yaml
```

### 4. Verify Deployment

```bash
# Check container status
docker ps | grep mercator-jupiter

# View logs
docker logs -f mercator-jupiter

# Test health endpoint
curl http://localhost:8080/health

# Test with a request
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## Docker Compose Deployment

### Complete Stack with Observability

```yaml
# docker-compose.yaml
version: '3.8'

services:
  mercator:
    image: mercator-hq/jupiter:latest
    container_name: mercator-jupiter
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - MERCATOR_PROXY_LISTEN_ADDRESS=0.0.0.0:8080
      - MERCATOR_TELEMETRY_LOGGING_LEVEL=info
    volumes:
      - ./config.yaml:/app/config/config.yaml:ro
      - ./policies:/app/policies:ro
      - mercator-data:/app/data
    healthcheck:
      test: ["CMD", "/app/mercator", "version"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
    networks:
      - mercator-network

  prometheus:
    image: prom/prometheus:latest
    container_name: mercator-prometheus
    restart: unless-stopped
    ports:
      - "9091:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - mercator-network

  grafana:
    image: grafana/grafana:latest
    container_name: mercator-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
    volumes:
      - grafana-data:/var/lib/grafana
    networks:
      - mercator-network

volumes:
  mercator-data:
    driver: local
  prometheus-data:
    driver: local
  grafana-data:
    driver: local

networks:
  mercator-network:
    driver: bridge
```

### Start the Stack

```bash
# Set environment variables
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-..."
export GRAFANA_PASSWORD="secure-password"

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f mercator

# Stop all services
docker-compose down

# Stop and remove data
docker-compose down -v
```

---

## Production Configuration

### Optimized Dockerfile

The production Dockerfile uses multi-stage builds:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mercator ./cmd/mercator

# Runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata sqlite
RUN addgroup -g 1000 mercator && adduser -D -u 1000 -G mercator mercator
WORKDIR /app
COPY --from=builder /build/mercator /app/mercator
RUN mkdir -p /app/data /app/policies /app/config && chown -R mercator:mercator /app
USER mercator
EXPOSE 8080 9090
HEALTHCHECK --interval=30s --timeout=3s CMD ["/app/mercator", "version"]
ENTRYPOINT ["/app/mercator"]
CMD ["run", "--config", "/app/config/config.yaml"]
```

### Build Production Image

```bash
docker build \
  --tag mercator-jupiter:0.1.0 \
  --tag mercator-jupiter:latest \
  --file examples/docker/Dockerfile \
  .

# Tag for registry
docker tag mercator-jupiter:0.1.0 yourregistry.com/mercator-jupiter:0.1.0
docker push yourregistry.com/mercator-jupiter:0.1.0
```

---

## Networking

### Port Mapping

| Container Port | Host Port | Purpose |
|----------------|-----------|---------|
| 8080 | 8080 | HTTP API (proxy endpoint) |
| 9090 | 9090 | Prometheus metrics |

### Custom Network

```bash
# Create dedicated network
docker network create mercator-network

# Run container on network
docker run -d \
  --name mercator-jupiter \
  --network mercator-network \
  -p 8080:8080 \
  mercator-jupiter:latest
```

### Reverse Proxy Integration

```nginx
# nginx.conf
upstream mercator {
    server localhost:8080;
}

server {
    listen 443 ssl http2;
    server_name jupiter.example.com;

    ssl_certificate /etc/ssl/certs/jupiter.crt;
    ssl_certificate_key /etc/ssl/private/jupiter.key;

    location / {
        proxy_pass http://mercator;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
    }

    location /metrics {
        deny all;
    }
}
```

---

## Persistent Storage

### Named Volumes

```bash
# Create volume
docker volume create jupiter-evidence-data

# Run with volume
docker run -d \
  --name mercator-jupiter \
  -v jupiter-evidence-data:/app/data \
  mercator-jupiter:latest

# Inspect volume
docker volume inspect jupiter-evidence-data

# Backup volume
docker run --rm \
  -v jupiter-evidence-data:/source:ro \
  -v $(pwd):/backup \
  alpine tar czf /backup/jupiter-data-backup.tar.gz -C /source .

# Restore volume
docker run --rm \
  -v jupiter-evidence-data:/target \
  -v $(pwd):/backup \
  alpine tar xzf /backup/jupiter-data-backup.tar.gz -C /target
```

### Bind Mounts

```bash
# Create directories
mkdir -p ./data ./config ./policies

# Set permissions
chmod 755 ./data ./config ./policies

# Run with bind mounts
docker run -d \
  --name mercator-jupiter \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config:/app/config:ro \
  -v $(pwd)/policies:/app/policies:ro \
  mercator-jupiter:latest
```

---

## Environment Variables

### Configuration Override

All configuration can be overridden via environment variables using the `MERCATOR_` prefix:

```bash
docker run -d \
  --name mercator-jupiter \
  -e MERCATOR_PROXY_LISTEN_ADDRESS="0.0.0.0:8080" \
  -e MERCATOR_TELEMETRY_LOGGING_LEVEL="debug" \
  -e MERCATOR_EVIDENCE_ENABLED="true" \
  -e OPENAI_API_KEY="$OPENAI_API_KEY" \
  mercator-jupiter:latest
```

### Environment File

```bash
# .env file
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-...
MERCATOR_PROXY_LISTEN_ADDRESS=0.0.0.0:8080
MERCATOR_TELEMETRY_LOGGING_LEVEL=info
MERCATOR_EVIDENCE_SQLITE_PATH=/app/data/evidence.db

# Use with docker run
docker run -d --env-file .env mercator-jupiter:latest

# Use with docker-compose
docker-compose --env-file .env up -d
```

---

## Health Checks

### Built-in Health Check

```bash
# Check health from outside container
docker exec mercator-jupiter /app/mercator version

# Or use health endpoint
curl http://localhost:8080/health
```

### Docker Health Check

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/mercator", "version"]
```

### Monitoring Health

```bash
# Check health status
docker inspect --format='{{.State.Health.Status}}' mercator-jupiter

# View health log
docker inspect --format='{{json .State.Health}}' mercator-jupiter | jq
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs mercator-jupiter

# Check last 100 lines
docker logs --tail 100 mercator-jupiter

# Follow logs
docker logs -f mercator-jupiter

# Check exit code
docker inspect --format='{{.State.ExitCode}}' mercator-jupiter
```

### Permission Issues

```bash
# Fix data directory permissions
docker run --rm \
  -v jupiter-data:/data \
  alpine chown -R 1000:1000 /data
```

### Configuration Issues

```bash
# Validate configuration
docker run --rm \
  -v $(pwd)/config.yaml:/app/config/config.yaml:ro \
  mercator-jupiter:latest run --dry-run

# Check mounted files
docker exec mercator-jupiter ls -la /app/config
docker exec mercator-jupiter cat /app/config/config.yaml
```

### Network Issues

```bash
# Check port bindings
docker port mercator-jupiter

# Test connectivity
docker exec mercator-jupiter wget -O- http://localhost:8080/health

# Check network
docker network inspect mercator-network
```

### Performance Issues

```bash
# Check resource usage
docker stats mercator-jupiter

# Set resource limits
docker run -d \
  --name mercator-jupiter \
  --cpus="2.0" \
  --memory="1g" \
  mercator-jupiter:latest
```

---

## Best Practices

1. **Use specific image tags** - Avoid `:latest` in production
2. **Enable health checks** - For automatic recovery
3. **Set resource limits** - Prevent resource exhaustion
4. **Use named volumes** - For data persistence
5. **Read-only filesystems** - Mount configs as read-only
6. **Run as non-root** - Security best practice
7. **Use environment files** - Manage secrets securely
8. **Enable logging** - Use JSON format for aggregation
9. **Monitor metrics** - Expose Prometheus endpoint
10. **Regular backups** - Backup evidence data

---

## Next Steps

- [Kubernetes Deployment](kubernetes.md) - Scale with Kubernetes
- [Systemd Deployment](systemd.md) - Run as system service
- [High Availability](high-availability.md) - Multi-instance setup
- [Security Guide](../SECURITY.md) - TLS and mTLS setup
