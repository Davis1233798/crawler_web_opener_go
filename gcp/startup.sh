#!/bin/bash

# Configuration
IMAGE_NAME="gcr.io/PROJECT_ID/crawler-no-proxy:latest"

# Install Docker (if not present, though COS is recommended)
# For standard Ubuntu/Debian:
# curl -fsSL https://get.docker.com -o get-docker.sh
# sh get-docker.sh

# Authenticate Docker to GCR (if needed)
# gcloud auth configure-docker gcr.io --quiet

echo "Starting Crawler Container..."

# Run the container
# --rm: Remove container after exit
# -e: Pass environment variables
docker run --rm \
  -e THREADS=2 \
  -e DURATION=60 \
  -e HEADLESS=true \
  $IMAGE_NAME

EXIT_CODE=$?
echo "Crawler finished with exit code $EXIT_CODE"

# Wait 1 minute
echo "Waiting 60 seconds before self-destruction..."
sleep 60

# Self-destruct
echo "Destroying instance..."
ZONE=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone -s | awk -F/ '{print $4}')
NAME=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/name -s)

gcloud compute instances delete $NAME --zone=$ZONE --quiet
