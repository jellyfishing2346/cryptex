# Kubernetes Deployment Guide

This guide covers deploying Cryptex on Kubernetes, from local development to production environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Environment Configurations](#environment-configurations)
- [Deployment Manifests](#deployment-manifests)
- [Production Deployment](#production-deployment)
- [Scaling and Autoscaling](#scaling-and-autoscaling)
- [Monitoring and Observability](#monitoring-and-observability)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured with cluster access
- At least 3 nodes for production
- 10GB+ storage available
- LoadBalancer support (or Ingress controller)

## Quick Start

### Local Development (Minikube/Kind)

```bash
# Start local cluster
minikube start
# or
kind create cluster --name cryptex

# Load Docker image
minikube image load cryptex:latest
# or
kind load docker-image cryptex:latest --name cryptex

# Deploy development configuration
kubectl apply -f deploy/development.yaml

# Access the service
minikube service cryptex-service -n cryptex-dev
# or
kubectl port-forward -n cryptex-dev svc/cryptex-service 8080:8080
```

### Production Deployment

```bash
# Apply production manifests
kubectl apply -f deploy/production.yaml

# Check deployment status
kubectl get pods -n cryptex-prod
kubectl get services -n cryptex-prod

# Get LoadBalancer IP
kubectl get svc cryptex-service -n cryptex-prod
```

## Environment Configurations

### Development Environment

The `deploy/development.yaml` configuration includes:
- Single replica deployment
- Resource limits optimized for development
- NodePort service for local access
- EmptyDir volumes for state (ephemeral)
- Basic health checks

**Use for**: Local development, testing, feature validation

### Staging Environment

The `deploy/staging.yaml` configuration includes:
- 2 replicas for high availability
- Persistent volumes for Redis and NATS
- Horizontal Pod Autoscaler (2-5 replicas)
- Resource quotas and limits
- Pod Disruption Budget

**Use for**: Pre-production testing, integration testing, load testing

### Production Environment

The `deploy/production.yaml` configuration includes:
- 3 replicas minimum
- Persistent volumes with 10Gi Redis storage
- Horizontal Pod Autoscaler (3-10 replicas)
- ConfigMap for configuration management
- Service accounts and RBAC
- Network policies for security
- Resource quotas and limit ranges
- Security hardening (non-root, read-only filesystem)

**Use for**: Production workloads

## Deployment Manifests

### Namespace Structure

Each environment uses its own namespace:
- `cryptex-dev` - Development
- `cryptex-staging` - Staging
- `cryptex-prod` - Production

### Components

#### Redis
- **Deployment**: Single replica with persistence
- **Service**: ClusterIP for internal access
- **Storage**: PersistentVolumeClaim with configurable size
- **Configuration**: Password authentication, memory limits, LRU eviction

#### NATS
- **Deployment**: Single replica with JetStream enabled
- **Service**: ClusterIP for internal access
- **Storage**: PersistentVolumeClaim for JetStream data
- **Configuration**: Monitoring port exposed

#### Cryptex
- **Deployment**: Multiple replicas (varies by environment)
- **Service**: LoadBalancer for external access
- **Configuration**: ConfigMap for env vars, Secret for sensitive data
- **Autoscaling**: HPA based on CPU and memory
- **Security**: Non-root user, read-only filesystem, dropped capabilities

## Production Deployment

### Step 1: Create Namespace

```bash
kubectl create namespace cryptex-prod
```

### Step 2: Create Secrets

```bash
# Create Redis password secret
kubectl create secret generic redis-secret \
  --from-literal=password=your-secure-password \
  -n cryptex-prod

# Create TLS certificates (if using HTTPS)
kubectl create secret tls cryptex-tls \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key \
  -n cryptex-prod
```

### Step 3: Create ConfigMap

```bash
# Create custom configuration
kubectl create configmap cryptex-config \
  --from-literal=TRADING_PAIR=BTC-USD \
  --from-literal=MAX_POSITION_SIZE=1000.0 \
  --from-literal=ENABLE_METRICS=true \
  -n cryptex-prod
```

### Step 4: Deploy Infrastructure

```bash
# Deploy Redis and NATS first
kubectl apply -f deploy/production.yaml -l 'app in (redis,nats)'

# Wait for infrastructure to be ready
kubectl wait --for=condition=ready pod -l app=redis -n cryptex-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=nats -n cryptex-prod --timeout=300s
```

### Step 5: Deploy Application

```bash
# Deploy Cryptex
kubectl apply -f deploy/production.yaml -l 'app=cryptex'

# Verify deployment
kubectl get pods -n cryptex-prod
kubectl get services -n cryptex-prod
```

### Step 6: Configure Ingress (Optional)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cryptex-ingress
  namespace: cryptex-prod
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - cryptex.example.com
    secretName: cryptex-tls
  rules:
  - host: cryptex.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: cryptex-service
            port:
              number: 80
```

## Scaling and Autoscaling

### Manual Scaling

```bash
# Scale deployment
kubectl scale deployment cryptex --replicas=5 -n cryptex-prod

# Check scale status
kubectl get hpa cryptex-hpa -n cryptex-prod
```

### Horizontal Pod Autoscaler

The production configuration includes HPA with:
- **Min replicas**: 3
- **Max replicas**: 10
- **CPU target**: 70% utilization
- **Memory target**: 80% utilization
- **Scale down**: Stabilization window of 5 minutes
- **Scale up**: Immediate response

```bash
# Check HPA status
kubectl get hpa -n cryptex-prod

# View HPA details
kubectl describe hpa cryptex-hpa -n cryptex-prod

# Update HPA
kubectl autoscale deployment cryptex \
  --min=3 --max=15 \
  --cpu-percent=70 \
  -n cryptex-prod
```

### Vertical Pod Autoscaler (Optional)

```yaml
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
```

## Monitoring and Observability

### Health Checks

```bash
# Check pod health
kubectl get pods -n cryptex-prod

# Describe pod for detailed health
kubectl describe pod <pod-name> -n cryptex-prod

# Check logs
kubectl logs -f deployment/cryptex -n cryptex-prod

# Check specific pod logs
kubectl logs <pod-name> -n cryptex-prod
```

### Metrics Collection

The production deployment includes Prometheus annotations:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

Configure Prometheus ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cryptex
  namespace: cryptex-prod
spec:
  selector:
    matchLabels:
      app: cryptex
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Logging

```bash
# Stream logs
kubectl logs -f deployment/cryptex -n cryptex-prod

# View previous logs
kubectl logs --previous deployment/cryptex -n cryptex-prod

# View logs with selector
kubectl logs -l app=cryptex -n cryptex-prod

# Export logs
kubectl logs deployment/cryptex -n cryptex-prod > cryptex.log
```

### Distributed Tracing

For distributed tracing, consider integrating with Jaeger or Zipkin:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cryptex-config
  namespace: cryptex-prod
data:
  JAEGER_AGENT_HOST: "jaeger-agent"
  JAEGER_AGENT_PORT: "6831"
  JAEGER_SAMPLER_TYPE: "probabilistic"
  JAEGER_SAMPLER_PARAM: "0.1"
```

## Security Best Practices

### Network Policies

The production configuration includes network policies that:
- Restrict ingress to namespace only
- Allow egress to Redis and NATS
- Allow DNS resolution
- Block all other traffic

```bash
# Check network policies
kubectl get networkpolicies -n cryptex-prod

# Describe network policy
kubectl describe networkpolicy cryptex-network-policy -n cryptex-prod
```

### Pod Security

The production deployment uses:
- **Non-root user**: Run as UID 1000
- **Read-only filesystem**: Prevent runtime modifications
- **Dropped capabilities**: Remove all Linux capabilities
- **No privilege escalation**: Prevent privilege escalation

```bash
# Check security context
kubectl describe pod <pod-name> -n cryptex-prod | grep -A 10 SecurityContext
```

### Secrets Management

```bash
# List secrets
kubectl get secrets -n cryptex-prod

# Describe secret
kubectl describe secret redis-secret -n cryptex-prod

# Create secret from file
kubectl create secret generic cryptex-config \
  --from-file=config.yaml=config.yaml \
  -n cryptex-prod

# Update secret
kubectl create secret generic redis-secret \
  --from-literal=password=new-password \
  --dry-run=client -o yaml | kubectl apply -f -
```

### RBAC

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cryptex-role
  namespace: cryptex-prod
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cryptex-rolebinding
  namespace: cryptex-prod
subjects:
- kind: ServiceAccount
  name: cryptex-service-account
  namespace: cryptex-prod
roleRef:
  kind: Role
  name: cryptex-role
  apiGroup: rbac.authorization.k8s.io
```

## Troubleshooting

### Pod Issues

```bash
# Check pod status
kubectl get pods -n cryptex-prod

# Describe pod for details
kubectl describe pod <pod-name> -n cryptex-prod

# Check pod events
kubectl get events -n cryptex-prod --sort-by=.metadata.creationTimestamp

# View pod logs
kubectl logs <pod-name> -n cryptex-prod

# Execute into pod
kubectl exec -it <pod-name> -n cryptex-prod -- /bin/sh
```

### Service Issues

```bash
# Check service endpoints
kubectl get endpoints cryptex-service -n cryptex-prod

# Test service connectivity
kubectl run -it --rm debug --image=busybox --restart=Never -n cryptex-prod -- \
  wget -O- http://cryptex-service:8080/healthz

# Check service configuration
kubectl describe service cryptex-service -n cryptex-prod
```

### Storage Issues

```bash
# Check PVC status
kubectl get pvc -n cryptex-prod

# Describe PVC
kubectl describe pvc redis-pvc -n cryptex-prod

# Check PV status
kubectl get pv

# Check storage class
kubectl get storageclass
```

### Network Issues

```bash
# Check network policies
kubectl get networkpolicies -n cryptex-prod

# Test DNS resolution
kubectl run -it --rm debug --image=busybox --restart=Never -n cryptex-prod -- \
  nslookup redis-service

# Test connectivity
kubectl run -it --rm debug --image=busybox --restart=Never -n cryptex-prod -- \
  nc -zv redis-service 6379
```

### Resource Issues

```bash
# Check resource usage
kubectl top pods -n cryptex-prod
kubectl top nodes

# Check resource limits
kubectl describe pod <pod-name> -n cryptex-prod | grep -A 5 Limits

# Check resource quotas
kubectl get resourcequota -n cryptex-prod
kubectl describe resourcequota cryptex-resource-quota -n cryptex-prod
```

## Maintenance

### Rolling Updates

```bash
# Update image
kubectl set image deployment/cryptex \
  cryptex=cryptex:v2.0.0 \
  -n cryptex-prod

# Check rollout status
kubectl rollout status deployment/cryptex -n cryptex-prod

# View rollout history
kubectl rollout history deployment/cryptex -n cryptex-prod

# Rollback if needed
kubectl rollout undo deployment/cryptex -n cryptex-prod
```

### Backup and Restore

```bash
# Backup namespace
kubectl get all -n cryptex-prod -o yaml > cryptex-backup.yaml

# Backup specific resources
kubectl get configmaps -n cryptex-prod -o yaml > configmaps-backup.yaml
kubectl get secrets -n cryptex-prod -o yaml > secrets-backup.yaml

# Restore from backup
kubectl apply -f cryptex-backup.yaml
```

### Draining Nodes

```bash
# Cordon node (mark as unschedulable)
kubectl cordon <node-name>

# Drain node (evict pods)
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data

# Uncordon node (mark as schedulable)
kubectl uncordon <node-name>
```

## Advanced Configuration

### Custom Resource Definitions

For advanced configuration, consider using CRDs:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: cryptexconfigs.cryptex.dev
spec:
  group: cryptex.dev
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              tradingPair:
                type: string
              maxPositionSize:
                type: number
  scope: Namespaced
  names:
    plural: cryptexconfigs
    singular: cryptexconfig
    kind: CryptexConfig
```

### Operator Pattern

For complex deployments, consider using Kubernetes Operators:

```bash
# Install Cryptex Operator
kubectl apply -f https://github.com/cryptex/cryptex-operator/releases/latest/download/operator.yaml

# Create Cryptex instance
kubectl apply -f - <<EOF
apiVersion: cryptex.dev/v1alpha1
kind: Cryptex
metadata:
  name: my-cryptex
  namespace: cryptex-prod
spec:
  replicas: 3
  tradingPair: BTC-USD
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "1Gi"
      cpu: "1000m"
EOF
```

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Production Patterns](https://kubernetes.io/docs/concepts/cluster-administration/)

---

For general deployment information, see [DEPLOYMENT.md](../DEPLOYMENT.md).
For Docker deployment, see [DOCKER.md](DOCKER.md).
