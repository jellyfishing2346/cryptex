#!/bin/bash

# GKE Cluster Creation Script for Cryptex
# This script creates a GKE cluster optimized for Cryptex

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=$(gcloud config get-value project)
REGION="us-central1"
ZONE="us-central1-a"
CLUSTER_NAME="cryptex-cluster"
NUM_NODES=3
MACHINE_TYPE="e2-medium"

echo -e "${GREEN}=== Cryptex GKE Cluster Creation Script ===${NC}"

# Check if project is set
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}Error: Project ID not set${NC}"
    echo "Run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

# Check if cluster already exists
if gcloud container clusters describe "$CLUSTER_NAME" --zone="$ZONE" &> /dev/null; then
    echo -e "${YELLOW}Cluster $CLUSTER_NAME already exists${NC}"
    echo -e "${YELLOW}Do you want to delete and recreate it? (y/N)${NC}"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo -e "${GREEN}Deleting existing cluster...${NC}"
        gcloud container clusters delete "$CLUSTER_NAME" --zone="$ZONE" --quiet
    else
        echo -e "${YELLOW}Using existing cluster${NC}"
        gcloud container clusters get-credentials "$CLUSTER_NAME" --zone="$ZONE"
        exit 0
    fi
fi

# Create GKE cluster
echo -e "${GREEN}Creating GKE cluster...${NC}"
gcloud container clusters create "$CLUSTER_NAME" \
    --zone="$ZONE" \
    --num-nodes="$NUM_NODES" \
    --machine-type="$MACHINE_TYPE" \
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
    --enable-vertical-pod-autoscaling \
    --enable-shielded-nodes \
    --shielded-secure-boot \
    --workload-pool="$PROJECT_ID.svc.id.goog" \
    --labels=environment=production,application=cryptex

# Get cluster credentials
echo -e "${GREEN}Getting cluster credentials...${NC}"
gcloud container clusters get-credentials "$CLUSTER_NAME" --zone="$ZONE"

# Verify cluster connection
echo -e "${GREEN}Verifying cluster connection...${NC}"
kubectl cluster-info
kubectl get nodes

echo -e "${GREEN}=== Cluster Creation Complete ===${NC}"
echo -e "${YELLOW}Cluster: $CLUSTER_NAME${NC}"
echo -e "${YELLOW}Zone: $ZONE${NC}"
echo -e "${YELLOW}Next step: Deploy the application with ./scripts/gke-deploy.sh${NC}"
