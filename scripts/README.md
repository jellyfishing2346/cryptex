# GKE Deployment Scripts

This directory contains scripts to help deploy Cryptex to Google Kubernetes Engine (GKE).

## Prerequisites

- Google Cloud SDK installed and configured
- Appropriate IAM roles for GKE, Cloud Build, and Artifact Registry
- Docker installed (for local builds)

## Script Overview

### 1. `gke-setup.sh`
Initial setup script that configures your Google Cloud environment.

**What it does:**
- Enables required Google Cloud APIs
- Creates Artifact Registry repository
- Configures Docker authentication
- Installs required gcloud components

**Usage:**
```bash
./scripts/gke-setup.sh
```

### 2. `gke-build.sh`
Builds and pushes the Docker image to Artifact Registry.

**What it does:**
- Builds the Docker image locally
- Tags it for Artifact Registry
- Pushes the image to Artifact Registry

**Usage:**
```bash
./scripts/gke-build.sh
```

**Alternative using Cloud Build:**
```bash
gcloud builds submit --tag us-central1-docker.pkg.dev/PROJECT_ID/cryptex/cryptex:latest .
```

### 3. `gke-create-cluster.sh`
Creates a GKE cluster optimized for Cryptex.

**What it does:**
- Creates a GKE cluster with production-grade settings
- Configures autoscaling, auto-repair, and auto-upgrade
- Sets up security features (shielded nodes, workload identity)
- Gets cluster credentials

**Usage:**
```bash
./scripts/gke-create-cluster.sh
```

**Cluster configuration:**
- 3 nodes minimum, 10 nodes maximum (autoscaling)
- e2-medium machine type
- 100GB disk size
- Standard network configuration
- Security features enabled

### 4. `gke-deploy.sh`
Deploys Cryptex to the GKE cluster.

**What it does:**
- Creates namespace and secrets
- Updates deployment configuration with correct image
- Deploys infrastructure (Redis, NATS)
- Deploys Cryptex application
- Waits for deployment to be ready
- Displays access information

**Usage:**
```bash
./scripts/gke-deploy.sh
```

### 5. `gke-workload-identity.sh` (Optional)
Sets up Workload Identity for secure access to Google Cloud services.

**What it does:**
- Creates Google Cloud service account
- Grants necessary IAM roles
- Enables Workload Identity
- Annotates Kubernetes service account

**Usage:**
```bash
./scripts/gke-workload-identity.sh
```

## Complete Deployment Workflow

### Step 1: Initial Setup
```bash
# Set your project ID
gcloud config set project YOUR_PROJECT_ID

# Run the setup script
./scripts/gke-setup.sh
```

### Step 2: Build and Push Image
```bash
# Build and push the Docker image
./scripts/gke-build.sh
```

### Step 3: Create Cluster
```bash
# Create the GKE cluster
./scripts/gke-create-cluster.sh
```

### Step 4: Deploy Application
```bash
# Deploy Cryptex to the cluster
./scripts/gke-deploy.sh
```

### Step 5: (Optional) Set Up Workload Identity
```bash
# Configure Workload Identity for secure access
./scripts/gke-workload-identity.sh
```

## Accessing the Application

After deployment, the script will display the LoadBalancer IP. You can also get it manually:

```bash
# Get the LoadBalancer IP
kubectl get service cryptex-service -n cryptex-prod

# Access the dashboard
http://EXTERNAL_IP

# Health check
http://EXTERNAL_IP/healthz
```

## Configuration

### Environment Variables

The scripts use the following default configuration:

- **Region**: `us-central1`
- **Zone**: `us-central1-a`
- **Cluster Name**: `cryptex-cluster`
- **Repository Name**: `cryptex`
- **Namespace**: `cryptex-prod`

To customize these, edit the respective script files.

### Custom Domain

The deployment includes optional Ingress configuration for custom domains. To use it:

1. Update `deploy/gke-production.yaml` with your domain:
   ```yaml
   spec:
     rules:
     - host: your-domain.com
   ```

2. Create a static IP:
   ```bash
   gcloud compute addresses create cryptex-ip --region=us-central1
   ```

3. Update the Ingress annotation with the IP name

## Troubleshooting

### Build Issues

If the build fails:
```bash
# Check Docker daemon is running
docker ps

# Verify gcloud authentication
gcloud auth list

# Re-authenticate
gcloud auth login
```

### Cluster Connection Issues

If you can't connect to the cluster:
```bash
# Re-authenticate
gcloud auth login

# Get credentials again
gcloud container clusters get-credentials cryptex-cluster --zone=us-central1-a

# Verify connection
kubectl cluster-info
```

### Deployment Issues

If the deployment fails:
```bash
# Check pod status
kubectl get pods -n cryptex-prod

# Check pod logs
kubectl logs -f deployment/cryptex -n cryptex-prod

# Describe pod for details
kubectl describe pod <pod-name> -n cryptex-prod
```

### Image Pull Issues

If pods can't pull the image:
```bash
# Verify image exists
gcloud artifacts docker images list us-central1-docker.pkg.dev/PROJECT_ID/cryptex/cryptex

# Check image pull secrets
kubectl get secrets -n cryptex-prod
```

## Cleanup

### Delete Deployment
```bash
# Delete all resources in namespace
kubectl delete namespace cryptex-prod
```

### Delete Cluster
```bash
# Delete the GKE cluster
gcloud container clusters delete cryptex-cluster --zone=us-central1-a
```

### Delete Repository
```bash
# Delete the Artifact Registry repository
gcloud artifacts repositories delete cryptex --location=us-central1
```

## Additional Resources

- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [Cloud Build Documentation](https://cloud.google.com/build/docs)
- [Artifact Registry Documentation](https://cloud.google.com/artifact-registry/docs)
- [GKE Deployment Guide](../docs/GKE_DEPLOYMENT.md)
