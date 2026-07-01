#!/bin/bash

# GKE Build Script for Cryptex
# This script builds and pushes the Docker image to Artifact Registry

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

echo -e "${GREEN}=== Cryptex GKE Build Script ===${NC}"

# Check if project is set
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}Error: Project ID not set${NC}"
    echo "Run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

FULL_IMAGE_NAME="$REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/$IMAGE_NAME"

echo -e "${GREEN}Building Docker image...${NC}"
docker build -t "$IMAGE_NAME:latest" .

echo -e "${GREEN}Tagging image for Artifact Registry...${NC}"
docker tag "$IMAGE_NAME:latest" "$FULL_IMAGE_NAME:latest"

echo -e "${GREEN}Pushing image to Artifact Registry...${NC}"
docker push "$FULL_IMAGE_NAME:latest"

echo -e "${GREEN}=== Build Complete ===${NC}"
echo -e "${YELLOW}Image: $FULL_IMAGE_NAME:latest${NC}"
