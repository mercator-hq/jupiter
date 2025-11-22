# Kubernetes Deployment Guide

Deploy Mercator Jupiter on Kubernetes for production workloads.

## Quick Start

```bash
# Using kubectl
kubectl apply -f examples/kubernetes/

# Using Helm
helm install jupiter examples/kubernetes/helm \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set secrets.anthropicApiKey="$ANTHROPIC_API_KEY"
```

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Kubectl Deployment](#kubectl-deployment)
- [Helm Deployment](#helm-deployment)
- [Configuration](#configuration)
- [Scaling](#scaling)
- [Monitoring](#monitoring)
- [Upgrading](#upgrading)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Kubernetes 1.19+ cluster
- `kubectl` configured to access your cluster
- Helm 3.0+ (for Helm deployment)
- Persistent volume provisioner (for evidence storage)

---

## Kubectl Deployment

### 1. Create Namespace

```bash
kubectl create namespace mercator-jupiter
kubectl config set-context --current --namespace=mercator-jupiter
```

### 2. Create Secrets

```bash
kubectl create secret generic mercator-secrets \
  --from-literal=openai-api-key="$OPENAI_API_KEY" \
  --from-literal=anthropic-api-key="$ANTHROPIC_API_KEY"
```

### 3. Create ConfigMaps

```bash
# Configuration
kubectl create configmap mercator-config \
  --from-file=config.yaml=examples/configs/production.yaml

# Policies
kubectl create configmap mercator-policies \
  --from-file=policy.yaml=examples/policies/production-policy.yaml
```

### 4. Deploy Manifests

```bash
# Deploy all manifests
kubectl apply -f examples/kubernetes/deployment.yaml
kubectl apply -f examples/kubernetes/service.yaml
kubectl apply -f examples/kubernetes/configmap.yaml
kubectl apply -f examples/kubernetes/secret.yaml

# Verify deployment
kubectl get pods
kubectl get svc
kubectl logs -f deployment/mercator-jupiter
```

### 5. Expose Service

```bash
# Create Ingress
kubectl apply -f examples/kubernetes/ingress.yaml

# Or use LoadBalancer
kubectl patch svc mercator-jupiter -p '{"spec":{"type":"LoadBalancer"}}'
```

---

## Helm Deployment

### Basic Installation

```bash
cd examples/kubernetes/helm

# Install with default values
helm install jupiter . \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set secrets.anthropicApiKey="$ANTHROPIC_API_KEY"
```

### Production Installation

```yaml
# production-values.yaml
replicaCount: 3

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: jupiter.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: jupiter-tls
      hosts:
        - jupiter.example.com

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10

persistence:
  enabled: true
  storageClass: fast-ssd
  size: 50Gi

resources:
  limits:
    cpu: 1000m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

```bash
helm install jupiter . -f production-values.yaml \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set secrets.anthropicApiKey="$ANTHROPIC_API_KEY"
```

### Verify Installation

```bash
# Check release status
helm status jupiter

# Get all resources
helm get all jupiter

# Check pods
kubectl get pods -l app.kubernetes.io/name=mercator-jupiter
```

---

## Configuration

### Environment Variables

Override configuration via environment variables:

```yaml
# deployment.yaml
env:
  - name: MERCATOR_PROXY_LISTEN_ADDRESS
    value: "0.0.0.0:8080"
  - name: MERCATOR_TELEMETRY_LOGGING_LEVEL
    value: "info"
```

### Git-Based Policies

```yaml
config:
  policy:
    mode: git
    gitRepo: https://github.com/yourorg/policies.git
    gitBranch: production
    gitPollInterval: "60s"
```

### PostgreSQL Evidence Storage

```yaml
config:
  evidence:
    enabled: true
    backend: postgres
    postgres:
      host: postgres-service
      port: 5432
      database: mercator_evidence
      user: mercator
      # Password from secret
```

---

## Scaling

### Manual Scaling

```bash
# Scale to 5 replicas
kubectl scale deployment mercator-jupiter --replicas=5

# With Helm
helm upgrade jupiter . --set replicaCount=5
```

### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: mercator-jupiter
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mercator-jupiter
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

```bash
kubectl apply -f hpa.yaml
kubectl get hpa
```

---

## Monitoring

### Prometheus ServiceMonitor

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: mercator-jupiter
spec:
  selector:
    matchLabels:
      app: mercator-jupiter
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

### Metrics Access

```bash
# Port-forward to metrics endpoint
kubectl port-forward svc/mercator-jupiter 9090:9090

# Access metrics
curl http://localhost:9090/metrics
```

---

## Upgrading

### Rolling Update

```bash
# Update image
kubectl set image deployment/mercator-jupiter \
  mercator=mercator-hq/jupiter:0.2.0

# Check rollout status
kubectl rollout status deployment/mercator-jupiter

# View history
kubectl rollout history deployment/mercator-jupiter

# Rollback if needed
kubectl rollout undo deployment/mercator-jupiter
```

### Helm Upgrade

```bash
# Upgrade release
helm upgrade jupiter . -f production-values.yaml

# Rollback
helm rollback jupiter

# History
helm history jupiter
```

---

## Troubleshooting

### Pod Issues

```bash
# Check pod status
kubectl get pods

# Describe pod
kubectl describe pod <pod-name>

# View logs
kubectl logs -f <pod-name>

# Previous container logs
kubectl logs <pod-name> --previous

# Execute commands in pod
kubectl exec -it <pod-name> -- /bin/sh
```

### Configuration Issues

```bash
# Check ConfigMap
kubectl get configmap mercator-config -o yaml

# Check Secrets
kubectl get secret mercator-secrets -o yaml

# Validate configuration
kubectl exec <pod-name> -- /app/mercator run --dry-run
```

### Network Issues

```bash
# Test service
kubectl run test-pod --image=curlimages/curl -it --rm -- \
  curl http://mercator-jupiter:8080/health

# Check endpoints
kubectl get endpoints mercator-jupiter

# Port forward for testing
kubectl port-forward svc/mercator-jupiter 8080:8080
```

---

## Best Practices

1. **Use Helm** for easier management and upgrades
2. **Enable autoscaling** for handling traffic spikes
3. **Set resource limits** to prevent resource exhaustion
4. **Use persistent storage** for evidence data
5. **Enable pod disruption budgets** for availability
6. **Use pod anti-affinity** for high availability
7. **Monitor with Prometheus** for observability
8. **Implement health checks** for automatic recovery
9. **Use namespaces** for multi-tenant deployments
10. **Regular backups** of persistent volumes

---

## Next Steps

- [High Availability](high-availability.md) - Multi-region setup
- [AWS Deployment](aws.md) - EKS-specific guidance
- [GCP Deployment](gcp.md) - GKE-specific guidance
- [Azure Deployment](azure.md) - AKS-specific guidance
