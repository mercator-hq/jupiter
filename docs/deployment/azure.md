# Azure Deployment Guide

Deploy Mercator Jupiter on Microsoft Azure.

## Deployment Options

| Option | Use Case | Complexity | Cost |
|--------|----------|------------|------|
| **Container Instances** | Simple containers | Low | Low |
| **AKS** | Kubernetes | Medium | Medium |
| **App Service** | PaaS | Low | Medium |
| **VMs** | Full control | Low | Medium |

---

## Azure Container Instances

### 1. Create Resource Group

```bash
az group create --name mercator-rg --location eastus
```

### 2. Deploy Container

```bash
az container create \
  --resource-group mercator-rg \
  --name mercator-jupiter \
  --image mercator-hq/jupiter:latest \
  --cpu 1 \
  --memory 2 \
  --ports 8080 9090 \
  --dns-name-label mercator-jupiter \
  --environment-variables \
    MERCATOR_PROXY_LISTEN_ADDRESS=0.0.0.0:8080 \
  --secure-environment-variables \
    OPENAI_API_KEY="$OPENAI_API_KEY" \
    ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY"
```

---

## AKS Deployment

### 1. Create AKS Cluster

```bash
az aks create \
  --resource-group mercator-rg \
  --name mercator-cluster \
  --node-count 3 \
  --node-vm-size Standard_D2s_v3 \
  --enable-addons monitoring \
  --enable-managed-identity \
  --enable-cluster-autoscaler \
  --min-count 2 \
  --max-count 10
```

### 2. Deploy with Helm

```bash
# Get credentials
az aks get-credentials --resource-group mercator-rg --name mercator-cluster

# Deploy
helm install jupiter examples/kubernetes/helm \
  --set secrets.openaiApiKey="$OPENAI_API_KEY" \
  --set ingress.enabled=true
```

---

## Secrets Management

```bash
# Create Key Vault
az keyvault create \
  --name mercator-vault \
  --resource-group mercator-rg \
  --location eastus

# Store secrets
az keyvault secret set \
  --vault-name mercator-vault \
  --name openai-key \
  --value "$OPENAI_API_KEY"

az keyvault secret set \
  --vault-name mercator-vault \
  --name anthropic-key \
  --value "$ANTHROPIC_API_KEY"
```

---

## Azure Database for PostgreSQL

```bash
az postgres flexible-server create \
  --resource-group mercator-rg \
  --name mercator-evidence \
  --location eastus \
  --admin-user mercator \
  --admin-password "$DB_PASSWORD" \
  --sku-name Standard_B1ms \
  --tier Burstable \
  --storage-size 32 \
  --version 15 \
  --backup-retention 7
```

---

## Application Gateway

```bash
az network application-gateway create \
  --name mercator-gateway \
  --resource-group mercator-rg \
  --location eastus \
  --sku Standard_v2 \
  --http-settings-port 8080 \
  --http-settings-protocol Http \
  --frontend-port 80 \
  --priority 100
```

---

## Monitoring

```bash
# Enable Application Insights
az monitor app-insights component create \
  --app mercator-insights \
  --location eastus \
  --resource-group mercator-rg \
  --application-type web

# Create alert
az monitor metrics alert create \
  --name high-cpu \
  --resource-group mercator-rg \
  --scopes "/subscriptions/SUBSCRIPTION_ID/resourceGroups/mercator-rg" \
  --condition "avg Percentage CPU > 80" \
  --description "High CPU usage"
```

---

## Best Practices

1. **Use AKS** for production workloads
2. **Key Vault** for secrets management
3. **Azure Database** for managed PostgreSQL
4. **Application Gateway** for load balancing
5. **Monitor** with Application Insights
6. **Azure AD** for authentication
7. **RBAC** for access control
8. **Private endpoints** for security
9. **Availability Zones** for HA
10. **Azure Backup** for data protection

---

## See Also

- [Kubernetes Deployment](kubernetes.md)
- [AWS Deployment](aws.md)
- [GCP Deployment](gcp.md)
- [High Availability](high-availability.md)
