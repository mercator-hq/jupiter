# GCP Deployment Guide

Deploy Mercator Jupiter on Google Cloud Platform.

## Deployment Options

| Option | Use Case | Complexity | Cost |
|--------|----------|------------|------|
| **Cloud Run** | Serverless containers | Low | Low |
| **GKE** | Kubernetes | Medium | Medium |
| **Compute Engine** | VMs | Low | Medium |

---

## Cloud Run Deployment

### 1. Build and Push Image

```bash
# Configure Docker for GCR
gcloud auth configure-docker

# Build image
docker build -t gcr.io/PROJECT_ID/mercator-jupiter:latest -f examples/docker/Dockerfile .

# Push to Container Registry
docker push gcr.io/PROJECT_ID/mercator-jupiter:latest
```

### 2. Deploy to Cloud Run

```bash
gcloud run deploy mercator-jupiter \
  --image gcr.io/PROJECT_ID/mercator-jupiter:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 512Mi \
  --cpu 1 \
  --max-instances 10 \
  --set-env-vars MERCATOR_PROXY_LISTEN_ADDRESS=0.0.0.0:8080 \
  --set-secrets OPENAI_API_KEY=openai-key:latest,ANTHROPIC_API_KEY=anthropic-key:latest \
  --ingress all
```

---

## GKE Deployment

### 1. Create GKE Cluster

```bash
gcloud container clusters create mercator-cluster \
  --region us-central1 \
  --num-nodes 3 \
  --machine-type n1-standard-2 \
  --enable-autoscaling \
  --min-nodes 2 \
  --max-nodes 10 \
  --enable-stackdriver-kubernetes
```

### 2. Deploy with Helm

```bash
# Get credentials
gcloud container clusters get-credentials mercator-cluster --region us-central1

# Deploy
helm install jupiter examples/kubernetes/helm \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set ingress.enabled=true
```

---

## Secrets Management

```bash
# Create secrets
echo -n "$OPENAI_API_KEY" | gcloud secrets create openai-key --data-file=-
echo -n "$ANTHROPIC_API_KEY" | gcloud secrets create anthropic-key --data-file=-

# Grant access
gcloud secrets add-iam-policy-binding openai-key \
  --member serviceAccount:PROJECT_NUMBER-compute@developer.gserviceaccount.com \
  --role roles/secretmanager.secretAccessor
```

---

## Cloud SQL for Evidence

```bash
# Create PostgreSQL instance
gcloud sql instances create mercator-evidence \
  --database-version=POSTGRES_15 \
  --tier=db-f1-micro \
  --region=us-central1 \
  --backup \
  --database-flags=max_connections=100

# Create database
gcloud sql databases create mercator_evidence --instance=mercator-evidence

# Create user
gcloud sql users create mercator \
  --instance=mercator-evidence \
  --password="$DB_PASSWORD"
```

---

## Monitoring

```bash
# Enable Cloud Monitoring
gcloud services enable monitoring.googleapis.com

# Create uptime check
gcloud alpha monitoring uptime create http \
  --display-name="Mercator Jupiter Health" \
  --resource-type=gae \
  --http-check-path="/health"
```

---

## Best Practices

1. **Use Cloud Run** for auto-scaling and cost efficiency
2. **Store secrets in Secret Manager**
3. **Enable Cloud SQL encryption**
4. **Use Cloud Load Balancing** for HTTPS
5. **Cloud Logging** for centralized logs
6. **Cloud Monitoring** for metrics and alerts
7. **VPC** for network isolation
8. **Service Accounts** for least-privilege
9. **Multi-region** for high availability
10. **Cloud Armor** for DDoS protection

---

## See Also

- [Kubernetes Deployment](kubernetes.md)
- [AWS Deployment](aws.md)
- [High Availability](high-availability.md)
