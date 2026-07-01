# Google Kubernetes Engine (GKE) Deployment Guide

This guide covers deploying Cryptex to Google Kubernetes Engine (GKE) on Google Cloud Platform.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Setup and Configuration](#setup-and-configuration)
- [Building and Pushing Container Images](#building-and-pushing-container-images)
- [Creating a GKE Cluster](#creating-a-gke-cluster)
- [Deploying to GKE](#deploying-to-gke)
- [Configuring Google Cloud Services](#configuring-google-cloud-services)
- [Monitoring and Logging](#monitoring-and-logging)
- [Scaling and Management](#scaling-and-management)
- [Cost Optimization](#cost-optimization)
- [Security](#security)

## Prerequisites

### Google Cloud Project Setup

```bash
# Install Google Cloud SDK
# macOS
brew install --cask google-cloud-sdk

# Linux
curl https://sdk.cloud.google.com | bash

# Initialize gcloud
gcloud init

# Set your project
gcloud config set project YOUR_PROJECT_ID

# Enable required APIs
gcloud services enable \
  container.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  redis.googleapis.com \
  cloudmonitoring.googleapis.com \
  logging.googleapis.com
```

### Tools Installation

```bash
# Install kubectl
gcloud components install kubectl

# Install gke-gcloud-auth-plugin (for GKE authentication)
gcloud components install gke-gcloud-auth-plugin

# Verify installation
gcloud version
kubectl version --client
```

### Required Permissions

Ensure you have the following IAM roles:
- `roles/container.admin`
- `roles/cloudbuild.builds.builder`
- `roles/artifactregistry.writer`
- `roles/redis.admin`
- `roles/monitoring.viewer`
- `roles/logging.viewer`

## Setup and Configuration

### Project and Region Configuration

```bash
# Set your preferred region
export REGION=us-central1
export ZONE=us-central1-a

gcloud config set compute/region $REGION
gcloud config set compute/zone $ZONE
```

### Create Artifact Registry Repository

```bash
# Create a Docker repository
gcloud artifacts repositories create cryptex \
  --repository-format=docker \
  --location=$REGION \
  --description="Cryptex container images"

# Verify repository creation
gcloud artifacts repositories list --location=$REGION
```

### Configure Docker Authentication

```bash
# Configure Docker to use gcloud as a credential helper
gcloud auth configure-docker $REGION-docker.pkg.dev

# Verify authentication
docker pull $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex:latest || true
```

## Building and Pushing Container Images

### Option 1: Using Cloud Build (Recommended)

```bash
# Build and push using Cloud Build
gcloud builds submit --tag $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex:latest .

# Build with specific tag
gcloud builds submit --tag $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex:v1.0.0 .

# Build with Dockerfile context
gcloud builds submit --tag $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex:latest --config cloudbuild.yaml .
```

### Option 2: Using Docker Locally

```bash
# Build the image
docker build -t cryptex:latest .

# Tag for Artifact Registry
docker tag cryptex:latest $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex:latest

# Push to Artifact Registry
docker push $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex:latest
```

### Cloud Build Configuration

Create `cloudbuild.yaml`:

```yaml
steps:
  # Build the container image
  - name: 'gcr.io/cloud-builders/docker'
    args: ['build', '-t', '$REGION-docker.pkg.dev/$PROJECT_ID/cryptex/cryptex:$COMMIT_SHA', '.']

  # Push the container image
  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', '$REGION-docker.pkg.dev/$PROJECT_ID/cryptex/cryptex:$COMMIT_SHA']

  # Deploy to GKE
  - name: 'gcr.io/cloud-builders/gke-deploy'
    args:
      - run
      - --filename=deploy/gke-production.yaml
      - --image=$REGION-docker.pkg.dev/$PROJECT_ID/cryptex/cryptex:$COMMIT_SHA
      - --location=$REGION
      - --cluster=cryptex-cluster

images:
  - '$REGION-docker.pkg.dev/$PROJECT_ID/cryptex/cryptex:$COMMIT_SHA'
```

## Creating a GKE Cluster

### Standard Cluster (Recommended for Production)

```bash
# Create a standard GKE cluster
gcloud container clusters create cryptex-cluster \
  --zone=$ZONE \
  --num-nodes=3 \
  --machine-type=e2-medium \
  --disk-type=pd-balanced \
  --disk-size=100GB \
  --network=default \
  --subnetwork=default \
  --enable-autoscaling \
  --min-nodes=3 \
  --max-nodes=10 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-ip-alias \
  --enable-private-nodes \
  --enable-master-authorized-networks \
  --master-authorized-networks=$(curl -s ifconfig.me)/32 \
  --enable-vertical-pod-autoscaling \
  --enable-shielded-nodes \
  --shielded-secure-boot \
  --workload-pool=$(gcloud config get-value project).svc.id.goog \
  --labels=environment=production,application=cryptex
```

### Cost-Optimized Cluster

```bash
# Create a cost-optimized cluster using Spot VMs
gcloud container clusters create cryptex-cluster \
  --zone=$ZONE \
  --num-nodes=3 \
  --machine-type=e2-small \
  --spot \
  --enable-autoscaling \
  --min-nodes=1 \
  --max-nodes=5 \
  --disk-size=50GB \
  --enable-autorepair \
  --enable-autoupgrade
```

### Get Cluster Credentials

```bash
# Get credentials for the cluster
gcloud container clusters get-credentials cryptex-cluster --zone=$ZONE

# Verify cluster connection
kubectl cluster-info
kubectl get nodes
```

## Deploying to GKE

### Update Deployment Configuration

Update `deploy/gke-production.yaml` to use your Artifact Registry image:

```yaml
# In the Cryptex deployment, update the image reference
spec:
  template:
    spec:
      containers:
      - name: cryptex
        image: REGION-docker.pkg.dev/PROJECT_ID/cryptex/cryptex:latest
        imagePullPolicy: Always
```

### Create Namespace and Secrets

```bash
# Create namespace
kubectl create namespace cryptex-prod

# Create Redis password secret
kubectl create secret generic redis-secret \
  --from-literal=password=$(openssl rand -base64 32) \
  -n cryptex-prod

# Verify secrets
kubectl get secrets -n cryptex-prod
```

### Deploy Infrastructure Components

```bash
# Deploy Redis and NATS first
kubectl apply -f deploy/production.yaml -l 'app in (redis,nats)' -n cryptex-prod

# Wait for infrastructure to be ready
kubectl wait --for=condition=ready pod -l app=redis -n cryptex-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=nats -n cryptex-prod --timeout=300s

# Verify infrastructure
kubectl get pods -n cryptex-prod
kubectl get pvc -n cryptex-prod
```

### Deploy Application

```bash
# Deploy Cryptex application
kubectl apply -f deploy/production.yaml -l 'app=cryptex' -n cryptex-prod

# Wait for deployment to be ready
kubectl rollout status deployment/cryptex -n cryptex-prod --timeout=300s

# Verify deployment
kubectl get pods -n cryptex-prod
kubectl get services -n cryptex-prod
```

### Configure Ingress (Optional)

For production, use Google Cloud Load Balancing with Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cryptex-ingress
  namespace: cryptex-prod
  annotations:
    kubernetes.io/ingress.class: "gce"
    kubernetes.io/ingress.global-static-ip-name: "cryptex-ip"
    networking.gke.io/managed-certificates: "cryptex-cert"
spec:
  rules:
  - host: cryptex.example.com
    http:
      paths:
      - path: /*
        pathType: ImplementationSpecific
        backend:
          service:
            name: cryptex-service
            port:
              number: 80
```

## Configuring Google Cloud Services

### Using Memorystore for Redis (Recommended)

Instead of deploying Redis in the cluster, use Google Cloud Memorystore:

```bash
# Create Memorystore Redis instance
gcloud redis instances create cryptex-redis \
  --region=$REGION \
  --tier=standard \
  --memory-size=1GB \
  --redis-version=redis_7_0 \
  --auth-enabled \
  --transit-encryption-mode=SERVER_AUTHENTICATION

# Get Redis connection details
gcloud redis instances describe cryptex-redis --region=$REGION

# Update ConfigMap with Redis endpoint
kubectl patch configmap cryptex-config -n cryptex-prod --type=json \
  -p='[{"op": "replace", "path": "/data/REDIS_ADDR", "value":"REDIS_IP:6379"}]'
```

### Using Cloud NATS

For NATS, you can either:
1. Deploy NATS in the cluster (as in the current configuration)
2. Use Google Cloud Pub/Sub as an alternative
3. Use a managed NATS service

### Using Cloud Monitoring

```bash
# Install Google Cloud Ops Agent for advanced monitoring
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/gke-monitoring-operator/main/config/monitoring-operator.yaml

# Create monitoring configuration
kubectl apply -f - <<EOF
apiVersion: monitoring.googleapis.com/v1
kind: ScrapeConfig
metadata:
  name: cryptex-scrape
  namespace: cryptex-prod
spec:
  targets:
  - relabelings:
    - sourceLabels: [__address__]
      regex: (.*)
      targetLabel: __address__
      replacement: \1:9090
  endpoints:
  - port: metrics
    interval: 30s
  selector:
    matchLabels:
      app: cryptex
EOF
```

## Monitoring and Logging

### Cloud Logging Integration

```bash
# Install Fluentd for log collection
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/fluentd-gcp-exporter/main/config/fluentd-gcp.yaml

# View logs in Cloud Console
gcloud logging read "resource.type=k8s_container" --limit=50

# View logs for specific deployment
gcloud logging read "resource.labels.namespace_name=cryptex-prod AND resource.labels.deployment_name=cryptex" --limit=50
```

### Cloud Monitoring Dashboards

```bash
# Create a custom dashboard in Cloud Console
# Navigate to: Monitoring > Dashboards > Create Dashboard

# Key metrics to monitor:
# - Container CPU/Memory usage
# - Request latency
# - Error rates
# - Redis connection health
# - Pod restart counts
```

### Alerting

```bash
# Create alerting policies
gcloud alpha monitoring policies create --policy-from-file=alerts.yaml

# Example alerts.yaml:
# - Alert on high error rate (>1%)
# - Alert on high latency (>100ms)
# - Alert on pod restarts
# - Alert on Redis connection failures
```

## Scaling and Management

### Cluster Autoscaler

The cluster is configured with autoscaling. You can adjust the settings:

```bash
# Update cluster autoscaling
gcloud container clusters update cryptex-cluster \
  --zone=$ZONE \
  --enable-autoscaling \
  --min-nodes=3 \
  --max-nodes=20

# Check autoscaling status
gcloud container clusters describe cryptex-cluster --zone=$ZONE
```

### Pod Autoscaling

The Horizontal Pod Autoscaler is already configured. Adjust as needed:

```bash
# Update HPA
kubectl autoscale deployment cryptex \
  --min=3 --max=20 \
  --cpu-percent=70 \
  --memory-percent=80 \
  -n cryptex-prod

# Check HPA status
kubectl get hpa -n cryptex-prod
kubectl describe hpa cryptex-hpa -n cryptex-prod
```

### Vertical Pod Autoscaling

```bash
# Enable VPA
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/vertical-pod-autoscaler/deploy/vpa.yaml

# Create VPA for Cryptex
kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: cryptex-vpa
  namespace: cryptex-prod
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cryptex
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: cryptex
      minAllowed:
        cpu: "250m"
        memory: "256Mi"
      maxAllowed:
        cpu: "2000m"
        memory: "2Gi"
EOF
```

## Cost Optimization

### Right-Sizing Resources

```bash
# Analyze resource usage
kubectl top pods -n cryptex-prod
kubectl top nodes

# Adjust resource limits in deployment
kubectl edit deployment cryptex -n cryptex-prod
```

### Use Preemptible/Spot VMs

```bash
# Create node pool with Spot VMs
gcloud container node-pools create spot-pool \
  --cluster=cryptex-cluster \
  --zone=$ZONE \
  --machine-type=e2-medium \
  --num-nodes=0 \
  --spot \
  --enable-autoscaling \
  --min-nodes=0 \
  --max-nodes=5

# Add node selector to deployment
kubectl patch deployment cryptex -n cryptex-prod -p '{"spec":{"template":{"spec":{"nodeSelector":{"cloud.google.com/gke-spot":"true"}}}}}'
```

### Enable Network Policies

```bash
# Enable network policy engine
gcloud container clusters update cryptex-cluster \
  --zone=$ZONE \
  --enable-network-policy

# Network policies are already defined in production.yaml
```

## Security

### Workload Identity

```bash
# Enable Workload Identity (already enabled in cluster creation)
gcloud iam service-accounts create cryptex-sa \
  --display-name="Cryptex Service Account"

# Grant IAM roles
gcloud iam service-accounts add-iam-policy-binding cryptex-sa@$(gcloud config get-value project).iam.gserviceaccount.com \
  --role=roles/iam.workloadIdentityUser \
  --member="serviceAccount:$(gcloud config get-value project).svc.id.goog[cryptex-prod/cryptex-service-account]"

# Annotate Kubernetes service account
kubectl annotate serviceaccount cryptex-service-account \
  iam.gke.io/gcp-service-account=cryptex-sa@$(gcloud config get-value project).iam.gserviceaccount.com \
  -n cryptex-prod
```

### Private Cluster

The cluster is configured as private. To access it:

```bash
# Configure authorized networks
gcloud container clusters update cryptex-cluster \
  --zone=$ZONE \
  --enable-master-authorized-networks \
  --master-authorized-networks=$(curl -s ifconfig.me)/32

# Use Cloud IAP for secure access
gcloud compute backend-services create cryptex-backend \
  --global \
  --load-balancing-scheme=EXTERNAL
```

### Secret Management

Consider using Google Secret Manager:

```bash
# Enable Secret Manager API
gcloud services enable secretmanager.googleapis.com

# Create secret
echo -n "your-redis-password" | gcloud secrets create redis-password --data-file=-

# Grant access
gcloud secrets add-iam-policy-binding redis-password \
  --member="serviceAccount:cryptex-sa@$(gcloud config get-value project).iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Use External Secrets Operator to sync secrets to Kubernetes
# https://external-secrets.io/
```

## Cleanup

### Delete Deployment

```bash
# Delete all resources in namespace
kubectl delete namespace cryptex-prod

# Or delete specific resources
kubectl delete -f deploy/production.yaml -n cryptex-prod
```

### Delete Cluster

```bash
# Delete GKE cluster
gcloud container clusters delete cryptex-cluster --zone=$ZONE

# Delete Artifact Registry repository
gcloud artifacts repositories delete cryptex --location=$REGION
```

## Troubleshooting

### Cluster Connection Issues

```bash
# Re-authenticate
gcloud auth login
gcloud container clusters get-credentials cryptex-cluster --zone=$ZONE

# Check cluster status
gcloud container clusters describe cryptex-cluster --zone=$ZONE
```

### Image Pull Issues

```bash
# Verify image exists
gcloud artifacts docker images list $REGION-docker.pkg.dev/$(gcloud config get-value project)/cryptex/cryptex

# Check image pull secrets
kubectl get secrets -n cryptex-prod

# Create image pull secret if needed
kubectl create secret docker-registry artifact-registry \
  --docker-server=$REGION-docker.pkg.dev \
  --docker-username=_json_key \
  --docker-password="$(gcloud auth print-access-token)" \
  --docker-email=your-email@example.com \
  -n cryptex-prod
```

### Resource Issues

```bash
# Check node resources
kubectl describe nodes

# Check pod events
kubectl get events -n cryptex-prod --sort-by=.metadata.creationTimestamp

# Check resource quotas
kubectl get resourcequota -n cryptex-prod
```

## Additional Resources

- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [Artifact Registry Documentation](https://cloud.google.com/artifact-registry/docs)
- [Cloud Build Documentation](https://cloud.google.com/build/docs)
- [Google Cloud Operations Suite](https://cloud.google.com/stackdriver)

---

For general Kubernetes deployment information, see [KUBERNETES.md](KUBERNETES.md).
For general deployment information, see [DEPLOYMENT.md](../DEPLOYMENT.md).
