# Mercator Jupiter Helm Chart

This Helm chart deploys Mercator Jupiter, a GitOps-native LLM governance runtime and policy engine, on Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PV provisioner support in the underlying infrastructure (if persistence is enabled)

## Installing the Chart

### Quick Start

```bash
# Add the Helm repository (when published)
helm repo add mercator https://charts.mercator.io
helm repo update

# Install with release name "jupiter"
helm install jupiter mercator/mercator-jupiter \
  --set secrets.openaiApiKey="sk-..." \
  --set secrets.anthropicApiKey="sk-..."
```

### Install from Source

```bash
# From the helm chart directory
helm install jupiter . \
  --set secrets.openaiApiKey="sk-..." \
  --set secrets.anthropicApiKey="sk-..."
```

### Install with Custom Values

```bash
# Create a values file
cat <<EOF > my-values.yaml
replicaCount: 3

ingress:
  enabled: true
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
  minReplicas: 2
  maxReplicas: 10

config:
  policy:
    mode: "git"
    gitRepo: "https://github.com/yourorg/policies.git"
    gitBranch: "main"
EOF

# Install with custom values
helm install jupiter . -f my-values.yaml \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set secrets.anthropicApiKey="$ANTHROPIC_API_KEY"
```

## Uninstalling the Chart

```bash
helm uninstall jupiter
```

## Configuration

The following table lists the configurable parameters of the Mercator Jupiter chart and their default values.

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `2` |
| `image.repository` | Image repository | `mercator-hq/jupiter` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (overrides chart appVersion) | `""` |

### Service Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.metricsPort` | Metrics port | `9090` |

### Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.hosts` | Ingress hosts configuration | `[]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Resource Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `1000m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### Autoscaling Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU % | `80` |

### Secrets Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `secrets.openaiApiKey` | OpenAI API key | `""` |
| `secrets.anthropicApiKey` | Anthropic API key | `""` |
| `secrets.existingSecret` | Use existing secret | `""` |

### Persistence Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.enabled` | Enable persistence | `true` |
| `persistence.storageClass` | Storage class | `""` |
| `persistence.size` | PVC size | `10Gi` |

### Configuration Parameters

See `values.yaml` for the complete list of configuration parameters under `config.*`.

## Examples

### Production Deployment with Ingress and TLS

```yaml
# production-values.yaml
replicaCount: 3

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
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
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70

persistence:
  enabled: true
  storageClass: fast-ssd
  size: 50Gi

config:
  policy:
    mode: git
    gitRepo: https://github.com/yourorg/jupiter-policies.git
    gitBranch: production

  telemetry:
    tracing:
      enabled: true
      endpoint: http://jaeger-collector:14268/api/traces

serviceMonitor:
  enabled: true
  interval: 30s
```

```bash
helm install jupiter . -f production-values.yaml \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set secrets.anthropicApiKey="$ANTHROPIC_API_KEY"
```

### Development Deployment with File-Based Policies

```yaml
# dev-values.yaml
replicaCount: 1

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 50m
    memory: 64Mi

persistence:
  enabled: false

config:
  telemetry:
    logging:
      level: debug

policies:
  inline: |
    version: "1.0"
    mpl_version: "1.0"
    name: "dev-policy"
    policies:
      - name: "allow-all"
        description: "Development allow-all policy"
        rules:
          - condition: "true"
            action: "allow"
```

```bash
helm install jupiter-dev . -f dev-values.yaml \
  --set secrets.openaiApiKey="$OPENAI_API_KEY"
```

### Using Existing Secrets

If you already have secrets created in your cluster:

```yaml
secrets:
  existingSecret: my-jupiter-secrets
  existingSecretKeys:
    openai: my-openai-key
    anthropic: my-anthropic-key
```

## Upgrading

### To upgrade the release

```bash
helm upgrade jupiter . -f my-values.yaml
```

### Rollback

```bash
helm rollback jupiter
```

## Monitoring

The chart includes optional support for:

- **Prometheus ServiceMonitor**: Set `serviceMonitor.enabled=true`
- **Pod Disruption Budget**: Enabled by default with `minAvailable: 1`
- **Horizontal Pod Autoscaler**: Enable with `autoscaling.enabled=true`

## Troubleshooting

### Check pod status

```bash
kubectl get pods -l app.kubernetes.io/name=mercator-jupiter
```

### View logs

```bash
kubectl logs -l app.kubernetes.io/name=mercator-jupiter -f
```

### Describe pod

```bash
kubectl describe pod <pod-name>
```

### Check configuration

```bash
kubectl get configmap <release-name>-mercator-jupiter-config -o yaml
```

## Support

- Documentation: https://github.com/mercator-hq/jupiter/tree/main/docs
- Issues: https://github.com/mercator-hq/jupiter/issues
- Discussions: https://github.com/mercator-hq/jupiter/discussions
