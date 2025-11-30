#!/bin/bash

# Usage: ./mother_vm_setup.sh <PROJECT_ID>

PROJECT_ID=$1
ZONE="us-central1-a"
VM_NAME="crawler-mother-vm"

if [ -z "$PROJECT_ID" ]; then
    echo "Usage: ./mother_vm_setup.sh <PROJECT_ID>"
    exit 1
fi

echo "Creating Mother VM ($VM_NAME) in $ZONE..."

# Create the VM
# We give it full cloud-platform scope so it can create other VMs and push to Artifact Registry
gcloud compute instances create $VM_NAME \
    --project=$PROJECT_ID \
    --zone=$ZONE \
    --machine-type=e2-micro \
    --image-family=debian-11 \
    --image-project=debian-cloud \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --tags=controller \
    --metadata=startup-script="#!/bin/bash
    # Install Git and Docker
    apt-get update
    apt-get install -y git docker.io

    # Clone the repo (You might need to make the repo public or configure auth token manually inside)
    # For simplicity, assuming public or user will config auth after logging in.
    # Here we just prepare the directory.
    mkdir -p /home/crawler
    cd /home/crawler
    git clone https://github.com/Davis1233798/crawler_web_opener.git
    
    # Fix permissions
    chown -R google-sudoers:google-sudoers /home/crawler
    "

echo "Mother VM created!"
echo "To access it and start the loop:"
echo "1. SSH into the VM: gcloud compute ssh $VM_NAME --zone=$ZONE"
echo "2. Go to the directory: cd /home/crawler/crawler_web_opener/crawler-go/gcp"
echo "3. (If repo is private) git pull to ensure latest code"
echo "4. Run the loop: nohup ./controller_loop.sh $PROJECT_ID 300 &"
