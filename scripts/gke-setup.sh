#!/bin/bash

# GKE Setup Script for Cryptex
# This script sets up the Google Cloud environment for GKE deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID=""
REGION="us-central1"
ZONE="us-central1-a"
CLUSTER_NAME="cryptex-cluster"
REPO_NAME="cryptex"

echo -e "${GREEN}=== Cryptex GKE Setup Script ===${NC}"

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}Error: gcloud is not installed${NC}"
    echo "Please install Google Cloud SDK: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    echo "Install with: gcloud components install kubectl"
    exit 1
fi

# Prompt for project ID if not set
if [ -z "$PROJECT_ID" ]; then
    echo -e "${YELLOW}Please enter your Google Cloud Project ID:${NC}"
    read -r PROJECT_ID
fi

# Set gcloud configuration
echo -e "${GREEN}Configuring gcloud...${NC}"
gcloud config set project "$PROJECT_ID"
gcloud config set compute/region "$REGION"
gcloud config set compute/zone "$ZONE"

# Enable required APIs
echo -e "${GREEN}Enabling required APIs...${NC}"
gcloud services enable \
    container.googleapis.com \
    cloudbuild.googleapis.com \
    artifactregistry.googleapis.com \
    redis.googleapis.com \
    cloudmonitoring.googleapis.com \
    logging.googleapis.com

# Create Artifact Registry repository
echo -e "${GREEN}Creating Artifact Registry repository...${NC}"
if gcloud artifacts repositories describe "$REPO_NAME" --location="$REGION" &> /dev/null; then
    echo -e "${YELLOW}Repository $REPO_NAME already exists${NC}"
else
    gcloud artifacts repositories create "$REPO_NAME" \
        --repository-format=docker \
        --location="$REGION" \
        --description="Cryptex container images"
fi

# Configure Docker authentication
echo -e "${GREEN}Configuring Docker authentication...${NC}"
gcloud auth configure-docker "$REGION-docker.pkg.dev"

# Install required components
echo -e "${GREEN}Installing required gcloud components...${NC}"
gcloud components install kubectl gke-gcloud-auth-plugin --quiet

echo -e "${GREEN}=== Setup Complete ===${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Build and push the Docker image: ./scripts/gke-build.sh"
echo "2. Create the GKE cluster: ./scripts/gke-create-cluster.sh"
echo "3. Deploy the application: ./scripts/gke-deploy.sh"
