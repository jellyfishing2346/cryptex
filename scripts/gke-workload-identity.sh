#!/bin/bash

# GKE Workload Identity Setup Script for Cryptex
# This script sets up Workload Identity for secure access to Google Cloud services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=$(gcloud config get-value project)
NAMESPACE="cryptex-prod"
K8S_SA_NAME="cryptex-service-account"
GCP_SA_NAME="cryptex-sa"

echo -e "${GREEN}=== Cryptex GKE Workload Identity Setup ===${NC}"

# Check if project is set
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}Error: Project ID not set${NC}"
    echo "Run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

# Create Google Cloud service account
echo -e "${GREEN}Creating Google Cloud service account...${NC}"
if ! gcloud iam service-accounts describe "$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com" &> /dev/null; then
    gcloud iam service-accounts create "$GCP_SA_NAME" \
        --display-name="Cryptex Service Account"
else
    echo -e "${YELLOW}Service account $GCP_SA_NAME already exists${NC}"
fi

# Grant necessary roles to the service account
echo -e "${GREEN}Granting IAM roles to service account...${NC}"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/monitoring.viewer" \
    --quiet

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/logging.viewer" \
    --quiet

# Enable Workload Identity on the service account
echo -e "${GREEN}Enabling Workload Identity...${NC}"
gcloud iam service-accounts add-iam-policy-binding "$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/iam.workloadIdentityUser" \
    --member="serviceAccount:$PROJECT_ID.svc.id.goog[$NAMESPACE/$K8S_SA_NAME]" \
    --quiet

# Annotate the Kubernetes service account
echo -e "${GREEN}Annotating Kubernetes service account...${NC}"
kubectl annotate serviceaccount "$K8S_SA_NAME" \
    iam.gke.io/gcp-service-account="$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com" \
    -n "$NAMESPACE" \
    --overwrite

echo -e "${GREEN}=== Workload Identity Setup Complete ===${NC}"
echo -e "${YELLOW}Kubernetes Service Account: $K8S_SA_NAME${NC}"
echo -e "${YELLOW}Google Cloud Service Account: $GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com${NC}"
