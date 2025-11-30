#!/bin/bash

# Usage: ./controller_loop.sh <PROJECT_ID> <INTERVAL_SECONDS>
# Example: ./controller_loop.sh my-project 300

PROJECT_ID=$1
INTERVAL=${2:-300} # Default 5 minutes

if [ -z "$PROJECT_ID" ]; then
    echo "Usage: ./controller_loop.sh <PROJECT_ID> [INTERVAL_SECONDS]"
    exit 1
fi

# Ensure scripts are executable
chmod +x deploy.sh monitor.sh

echo "Starting Controller Loop for Project: $PROJECT_ID"
echo "Interval: $INTERVAL seconds"

# Start Monitor in background
./monitor.sh 60 &
MONITOR_PID=$!
echo "Started Monitor (PID: $MONITOR_PID)"

# Cleanup function
cleanup() {
    echo "Stopping Controller Loop..."
    kill $MONITOR_PID
    exit 0
}
trap cleanup SIGINT SIGTERM

while true; do
    echo "----------------------------------------"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting deployment round"
    
    # Run the deployment script
    ./deploy.sh $PROJECT_ID
    
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Deployment round finished."
    echo "Waiting $INTERVAL seconds before next round..."
    sleep $INTERVAL
done
