#!/bin/bash

# GKE Deployment Script for Cryptex
# This script deploys Cryptex to GKE

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=$(gcloud config get-value project)
REGION="us-central1"
REPO_NAME="cryptex"
IMAGE_NAME="cryptex"
NAMESPACE="cryptex-prod"

echo -e "${GREEN}=== Cryptex GKE Deployment Script ===${NC}"

# Check if project is set
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}Error: Project ID not set${NC}"
    echo "Run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Error: Cannot connect to cluster${NC}"
    echo "Run: gcloud container clusters get-credentials CLUSTER_NAME --zone=ZONE"
    exit 1
fi

# Update the deployment file with the correct image
FULL_IMAGE_NAME="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/$IMAGE_NAME:latest"
echo -e "${GREEN}Updating deployment with image: $FULL_IMAGE_NAME${NC}"

# Create namespace
echo -e "${GREEN}Creating namespace...${NC}"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Create Redis secret
echo -e "${GREEN}Creating Redis secret...${NC}"
if ! kubectl get secret redis-secret -n "$NAMESPACE" &> /dev/null; then
    kubectl create secret generic redis-secret \
        --from-literal=password=$(openssl rand -base64 32) \
        -n "$NAMESPACE"
fi

# Update the deployment file with the correct image
sed "s|us-central1-docker.pkg.dev/credit-risk-499502/cryptex/cryptex:latest|$FULL_IMAGE_NAME|g" deploy/gke-production.yaml > /tmp/gke-production.yaml

# Remove ingress and managed certificate if you don't have a custom domain
# Uncomment the following lines if you want to skip ingress setup
# sed -i '/^---$/,+1d' /tmp/gke-production.yaml
# sed -i '/^apiVersion: networking.k8s.io\/v1$/,/^kind: Ingress$/d' /tmp/gke-production.yaml
# sed -i '/^apiVersion: networking.gke.io\/v1$/,/^kind: ManagedCertificate$/d' /tmp/gke-production.yaml

# Deploy infrastructure (Redis and NATS)
echo -e "${GREEN}Deploying infrastructure (Redis and NATS)...${NC}"
kubectl apply -f /tmp/gke-production.yaml -l 'app in (redis,nats)' -n "$NAMESPACE"

# Wait for infrastructure to be ready
echo -e "${GREEN}Waiting for infrastructure to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=redis -n "$NAMESPACE" --timeout=300s
kubectl wait --for=condition=ready pod -l app=nats -n "$NAMESPACE" --timeout=300s

# Deploy application
echo -e "${GREEN}Deploying Cryptex application...${NC}"
kubectl apply -f /tmp/gke-production.yaml -l 'app=cryptex' -n "$NAMESPACE"

# Wait for deployment to be ready
echo -e "${GREEN}Waiting for deployment to be ready...${NC}"
kubectl rollout status deployment/cryptex -n "$NAMESPACE" --timeout=300s

# Get service information
echo -e "${GREEN}=== Deployment Complete ===${NC}"
kubectl get pods -n "$NAMESPACE"
kubectl get services -n "$NAMESPACE"

# Get LoadBalancer IP
echo -e "${GREEN}Waiting for LoadBalancer IP...${NC}"
sleep 10
EXTERNAL_IP=$(kubectl get service cryptex-service -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [ -n "$EXTERNAL_IP" ]; then
    echo -e "${GREEN}=== Access Information ===${NC}"
    echo -e "${YELLOW}LoadBalancer IP: $EXTERNAL_IP${NC}"
    echo -e "${YELLOW}Dashboard: http://$EXTERNAL_IP${NC}"
    echo -e "${YELLOW}Health Check: http://$EXTERNAL_IP/healthz${NC}"
else
    echo -e "${YELLOW}LoadBalancer IP is still being provisioned${NC}"
    echo -e "${YELLOW}Check with: kubectl get service cryptex-service -n $NAMESPACE${NC}"
fi

# Cleanup
rm -f /tmp/gke-production.yaml
