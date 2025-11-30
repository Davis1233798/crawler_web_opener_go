#!/bin/bash

# Usage: ./deploy.sh [PROJECT_ID]

PROJECT_ID=$1

if [ -z "$PROJECT_ID" ]; then
    echo "Usage: ./deploy.sh <PROJECT_ID>"
    exit 1
fi

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

IMAGE_NAME="us-central1-docker.pkg.dev/$PROJECT_ID/crawler-repo/crawler-no-proxy:latest"

# 0. Sync latest code
log "Pulling latest code..."
git pull origin no-proxy-docker-gcp


# 1. Build and Push Docker Image
log "Building and pushing Docker image to $IMAGE_NAME..."
# Ensure we are in the root directory
cd ..
# Configure docker credential helper for gcloud
gcloud auth configure-docker us-central1-docker.pkg.dev --quiet

docker build -t $IMAGE_NAME .
docker push $IMAGE_NAME
cd gcp

# 2. List Regions (or define a subset)
# To deploy to ALL regions (beware of quotas!):
# REGIONS=$(gcloud compute regions list --format="value(name)")
# For testing, let's pick a few diverse ones:
REGIONS=("us-central1" "europe-west1" "asia-northeast1")

log "Deploying to regions: ${REGIONS[@]}"

# 3. Deploy to each region
for REGION in "${REGIONS[@]}"; do
    INSTANCE_NAME="crawler-$REGION-$(date +%s)"
    ZONE="$REGION-a" # Simple zone selection

    log "Creating instance $INSTANCE_NAME in $ZONE..."

    gcloud compute instances create $INSTANCE_NAME \
        --project=$PROJECT_ID \
        --zone=$ZONE \
        --image-family=cos-stable \
        --image-project=cos-cloud \
        --machine-type=e2-micro \
        --scopes=https://www.googleapis.com/auth/cloud-platform \
        --metadata=startup-script="#!/bin/bash
        # Configure Docker Auth for COS
        docker-credential-gcr configure-docker --registries=us-central1-docker.pkg.dev
        
        # Run Crawler
        docker run --rm -e THREADS=2 -e DURATION=60 -e HEADLESS=true $IMAGE_NAME
        
        echo 'Crawler finished. Self-destructing in 60s...'
        sleep 60
        
        # Self Destruct via API
        TOKEN=\$(curl -s -H 'Metadata-Flavor: Google' 'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token' | grep -o '\"access_token\":\"[^\"]*\"' | cut -d'\"' -f4)
        PROJECT=\$(curl -s -H 'Metadata-Flavor: Google' 'http://metadata.google.internal/computeMetadata/v1/project/project-id')
        ZONE=\$(curl -s -H 'Metadata-Flavor: Google' 'http://metadata.google.internal/computeMetadata/v1/instance/zone' | awk -F/ '{print \$NF}')
        NAME=\$(curl -s -H 'Metadata-Flavor: Google' 'http://metadata.google.internal/computeMetadata/v1/instance/name')
        
        curl -X DELETE -H \"Authorization: Bearer \$TOKEN\" \"https://compute.googleapis.com/compute/v1/projects/\$PROJECT/zones/\$ZONE/instances/\$NAME\"
        " \
        --tags=crawler \
        --preemptible # Use preemptible for lower cost

    if [ $? -eq 0 ]; then
        log "Successfully deployed to $REGION. Instance: $INSTANCE_NAME"
    else
        log "Failed to deploy to $REGION"
    fi
done

log "Deployment round complete. Instances will self-destruct after running."
