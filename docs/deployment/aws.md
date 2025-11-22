# AWS Deployment Guide

Deploy Mercator Jupiter on Amazon Web Services.

## Deployment Options

| Option | Use Case | Complexity | Cost |
|--------|----------|------------|------|
| **ECS Fargate** | Serverless containers | Low | Medium |
| **EKS** | Kubernetes | Medium | High |
| **EC2 + Docker** | Direct control | Low | Low |
| **Lambda** | Event-driven | Low | Very Low |

---

## ECS Fargate Deployment

### 1. Create Task Definition

```json
{
  "family": "mercator-jupiter",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "containerDefinitions": [
    {
      "name": "mercator",
      "image": "mercator-hq/jupiter:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        },
        {
          "containerPort": 9090,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "MERCATOR_PROXY_LISTEN_ADDRESS",
          "value": "0.0.0.0:8080"
        }
      ],
      "secrets": [
        {
          "name": "OPENAI_API_KEY",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:mercator/openai-key"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/mercator-jupiter",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### 2. Create ECS Service

```bash
aws ecs create-service \
  --cluster mercator-cluster \
  --service-name jupiter \
  --task-definition mercator-jupiter:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-abc123],securityGroups=[sg-xyz789],assignPublicIp=ENABLED}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:...,containerName=mercator,containerPort=8080"
```

### 3. Application Load Balancer

```bash
# Create target group
aws elbv2 create-target-group \
  --name mercator-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-123456 \
  --target-type ip \
  --health-check-path /health

# Create load balancer
aws elbv2 create-load-balancer \
  --name mercator-alb \
  --subnets subnet-abc123 subnet-def456 \
  --security-groups sg-xyz789
```

---

## EKS Deployment

### 1. Create EKS Cluster

```bash
# Using eksctl
eksctl create cluster \
  --name mercator-cluster \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 2 \
  --nodes-max 5 \
  --managed
```

### 2. Deploy with Helm

```bash
# Configure kubectl
aws eks update-kubeconfig --name mercator-cluster --region us-east-1

# Install Mercator
helm install jupiter examples/kubernetes/helm \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set ingress.enabled=true \
  --set ingress.className=alb \
  --set ingress.annotations."alb\.ingress\.kubernetes\.io/scheme"=internet-facing
```

---

## Secrets Management

### AWS Secrets Manager

```bash
# Store API keys
aws secretsmanager create-secret \
  --name mercator/openai-key \
  --secret-string "$OPENAI_API_KEY"

aws secretsmanager create-secret \
  --name mercator/anthropic-key \
  --secret-string "$ANTHROPIC_API_KEY"
```

### IAM Role for ECS Task

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-east-1:123456789012:secret:mercator/*"
      ]
    }
  ]
}
```

---

## RDS for Evidence Storage

```bash
# Create RDS PostgreSQL instance
aws rds create-db-instance \
  --db-instance-identifier mercator-evidence \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --engine-version 15.3 \
  --master-username mercator \
  --master-user-password "$DB_PASSWORD" \
  --allocated-storage 100 \
  --storage-type gp3 \
  --vpc-security-group-ids sg-xyz789 \
  --db-subnet-group-name mercator-subnet-group \
  --backup-retention-period 7 \
  --storage-encrypted
```

Update configuration:

```yaml
evidence:
  enabled: true
  backend: postgres
  postgres:
    host: mercator-evidence.abc123.us-east-1.rds.amazonaws.com
    port: 5432
    database: mercator_evidence
    user: mercator
    password: "${DB_PASSWORD}"
    ssl_mode: require
```

---

## CloudWatch Monitoring

```bash
# Create log group
aws logs create-log-group --log-group-name /ecs/mercator-jupiter

# Create metric filter
aws logs put-metric-filter \
  --log-group-name /ecs/mercator-jupiter \
  --filter-name error-count \
  --filter-pattern "ERROR" \
  --metric-transformations \
    metricName=ErrorCount,metricNamespace=Mercator,metricValue=1

# Create alarm
aws cloudwatch put-metric-alarm \
  --alarm-name mercator-high-errors \
  --alarm-description "Mercator error rate" \
  --metric-name ErrorCount \
  --namespace Mercator \
  --statistic Sum \
  --period 300 \
  --threshold 10 \
  --comparison-operator GreaterThanThreshold
```

---

## Best Practices

1. **Use Fargate** for simpler ops, EC2 for cost optimization
2. **Store secrets in Secrets Manager** never in environment variables
3. **Enable RDS encryption** for evidence data
4. **Use ALB health checks** for automatic recovery
5. **CloudWatch for monitoring** logs and metrics
6. **Multi-AZ deployment** for high availability
7. **VPC security groups** restrict access
8. **IAM roles** for least-privilege access
9. **Auto Scaling** based on CPU/memory
10. **Regular backups** of RDS and EBS volumes

---

## See Also

- [Kubernetes Deployment](kubernetes.md)
- [High Availability](high-availability.md)
- [GCP Deployment](gcp.md)
