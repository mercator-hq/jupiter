# High Availability Deployment Guide

Configure Mercator Jupiter for high availability and fault tolerance.

## Architecture Overview

```
                 ┌─────────────────────┐
                 │   Load Balancer     │
                 │   (HAProxy/ALB)     │
                 └──────────┬──────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
    ┌─────▼─────┐     ┌─────▼─────┐     ┌─────▼─────┐
    │ Jupiter 1 │     │ Jupiter 2 │     │ Jupiter 3 │
    │ (Zone A)  │     │ (Zone B)  │     │ (Zone C)  │
    └─────┬─────┘     └─────┬─────┘     └─────┬─────┘
          │                 │                 │
          └─────────────────┼─────────────────┘
                            │
                   ┌────────▼────────┐
                   │   PostgreSQL    │
                   │  (Primary +     │
                   │   Read Replica) │
                   └─────────────────┘
```

---

## Key Principles

1. **No Single Point of Failure**
2. **Automatic Failover**
3. **Data Redundancy**
4. **Geographic Distribution**
5. **Health-Based Routing**

---

## Multi-Instance Deployment

### Kubernetes (Recommended)

```yaml
# high-availability-values.yaml
replicaCount: 3

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app
              operator: In
              values:
                - mercator-jupiter
        topologyKey: "kubernetes.io/hostname"

topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app: mercator-jupiter

podDisruptionBudget:
  enabled: true
  minAvailable: 2

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

Deploy:

```bash
helm install jupiter . -f high-availability-values.yaml \
  --set persistence.enabled=false \  # Use external DB
  --set config.evidence.backend=postgres
```

---

## Load Balancing

### HAProxy Configuration

```haproxy
# /etc/haproxy/haproxy.cfg
global
    maxconn 4096
    log stdout format raw local0

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    timeout connect 5000
    timeout client  300000
    timeout server  300000

frontend http-in
    bind *:80
    default_backend mercator_backend

backend mercator_backend
    balance roundrobin
    option httpchk GET /health
    http-check expect status 200

    server jupiter1 10.0.1.10:8080 check inter 5s fall 3 rise 2
    server jupiter2 10.0.1.11:8080 check inter 5s fall 3 rise 2
    server jupiter3 10.0.1.12:8080 check inter 5s fall 3 rise 2

frontend metrics
    bind *:9090
    default_backend metrics_backend

backend metrics_backend
    balance roundrobin
    server jupiter1 10.0.1.10:9090
    server jupiter2 10.0.1.11:9090
    server jupiter3 10.0.1.12:9090
```

---

## Database High Availability

### PostgreSQL with Replication

```yaml
# Primary configuration
evidence:
  backend: postgres
  postgres:
    host: postgres-primary.example.com
    port: 5432
    database: mercator_evidence
    user: mercator
    password: "${DB_PASSWORD}"
    ssl_mode: require
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: "1h"
```

### Read Replicas for Queries

```yaml
# Add read replica endpoint for evidence queries
evidence:
  postgres:
    host: postgres-primary.example.com
    read_replica_hosts:
      - postgres-replica-1.example.com
      - postgres-replica-2.example.com
```

---

## Multi-Region Deployment

### Active-Active Configuration

```
Region US-East             Region EU-West
┌─────────────┐           ┌─────────────┐
│ Jupiter x3  │           │ Jupiter x3  │
│ PostgreSQL  │◄─────────►│ PostgreSQL  │
│  Primary    │  Sync     │  Primary    │
└─────────────┘           └─────────────┘
       │                         │
       └─────────┬───────────────┘
                 │
          ┌──────▼──────┐
          │  Global LB  │
          │  (Route53)  │
          └─────────────┘
```

### DNS Failover (Route 53)

```json
{
  "Comment": "Mercator Jupiter multi-region",
  "Changes": [
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "jupiter.example.com",
        "Type": "A",
        "SetIdentifier": "US-East",
        "Failover": "PRIMARY",
        "TTL": 60,
        "ResourceRecords": [{"Value": "us-east-lb-ip"}],
        "HealthCheckId": "health-check-us"
      }
    },
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "jupiter.example.com",
        "Type": "A",
        "SetIdentifier": "EU-West",
        "Failover": "SECONDARY",
        "TTL": 60,
        "ResourceRecords": [{"Value": "eu-west-lb-ip"}],
        "HealthCheckId": "health-check-eu"
      }
    }
  ]
}
```

---

## Health Checks

### Deep Health Check

```go
// Implement comprehensive health check
GET /health/deep

Response:
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "providers": {
      "openai": "ok",
      "anthropic": "ok"
    },
    "policy_engine": "ok",
    "evidence_storage": "ok"
  },
  "uptime_seconds": 3600
}
```

### Load Balancer Configuration

```yaml
# ALB Target Group
HealthCheckProtocol: HTTP
HealthCheckPath: /health
HealthCheckIntervalSeconds: 30
HealthCheckTimeoutSeconds: 5
HealthyThresholdCount: 2
UnhealthyThresholdCount: 3
```

---

## Failover Strategy

### Automatic Failover

1. **Health Check Failure** → Instance marked unhealthy
2. **Traffic Rerouted** → Load balancer removes instance
3. **Replacement Started** → Auto-scaling launches new instance
4. **Health Check Success** → New instance receives traffic

### Manual Failover

```bash
# Drain instance
kubectl cordon node-1
kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data

# Scale up in other zone
kubectl scale deployment mercator-jupiter --replicas=4

# After maintenance
kubectl uncordon node-1
```

---

## Disaster Recovery

### Backup Strategy

```bash
# Automated backups
0 2 * * * /usr/local/bin/backup-evidence.sh
0 3 * * * /usr/local/bin/backup-policies.sh

# Retention:
# - Daily: 7 days
# - Weekly: 4 weeks
# - Monthly: 12 months
```

### Recovery Time Objectives

| Component | RTO | RPO |
|-----------|-----|-----|
| Application | 5 min | 0 |
| Evidence Data | 15 min | 5 min |
| Policies | 1 min | 0 |

---

## Monitoring for HA

### Critical Metrics

```yaml
alerts:
  - name: InstanceDown
    condition: up == 0
    severity: critical
    duration: 5m

  - name: HighErrorRate
    condition: rate(errors_total[5m]) > 0.05
    severity: warning

  - name: DatabaseUnreachable
    condition: db_up == 0
    severity: critical
    duration: 1m
```

---

## Testing HA

### Chaos Engineering

```bash
# Kill random pod
kubectl delete pod -l app=mercator-jupiter --random

# Simulate zone failure
kubectl cordon -l topology.kubernetes.io/zone=us-east-1a

# Network partition
# Use tools like Chaos Mesh or Litmus
```

### Load Testing

```bash
# Test failover under load
mercator benchmark \
  --target http://jupiter.example.com \
  --duration 300s \
  --rate 1000 \
  --concurrency 100

# During test, kill instances and observe
```

---

## Best Practices

1. **Minimum 3 instances** across availability zones
2. **Database replication** with automatic failover
3. **Health checks** every 30 seconds
4. **Pod anti-affinity** to spread across nodes
5. **Pod disruption budgets** to maintain availability
6. **Horizontal autoscaling** for traffic spikes
7. **Multi-region** for disaster recovery
8. **Regular failover drills** to verify readiness
9. **Monitoring and alerting** for all components
10. **Documented runbooks** for incident response

---

## Checklist

- [ ] Minimum 3 instances deployed
- [ ] Instances spread across availability zones
- [ ] Load balancer with health checks configured
- [ ] Database with replication enabled
- [ ] Automated backups configured
- [ ] Monitoring and alerts set up
- [ ] Failover tested successfully
- [ ] Recovery procedures documented
- [ ] On-call rotation established
- [ ] Disaster recovery plan reviewed

---

## See Also

- [Kubernetes Deployment](kubernetes.md)
- [AWS Deployment](aws.md)
- [GCP Deployment](gcp.md)
- [Azure Deployment](azure.md)
