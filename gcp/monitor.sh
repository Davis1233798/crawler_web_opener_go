#!/bin/bash

# Usage: ./monitor.sh [INTERVAL_SECONDS]
INTERVAL=${1:-60}

echo "Starting Monitor Loop. Interval: $INTERVAL seconds"

while true; do
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Checking active crawler instances..."
    
    # List instances with tag 'crawler'
    # We use --format to get a clean list of names and zones
    INSTANCES=$(gcloud compute instances list --filter="tags:crawler" --format="table(name, zone, status)")
    
    if [ -z "$INSTANCES" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] No active crawler instances found."
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Active Instances:"
        echo "$INSTANCES"
    fi
    
    sleep $INTERVAL
done
