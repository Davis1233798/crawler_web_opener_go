#!/bin/bash

# Define the binary name
BINARY_NAME="crawler"
LOG_FILE="crawler.log"

echo "Stopping existing $BINARY_NAME processes..."
# Find and kill existing processes
pids=$(pgrep -f "$BINARY_NAME")
if [ -n "$pids" ]; then
    echo "Found processes: $pids"
    kill -9 $pids
    echo "Processes killed."
else
    echo "No existing processes found."
fi

echo "Building $BINARY_NAME..."
# Build the Go application
go build -o $BINARY_NAME ./cmd/crawler

if [ $? -ne 0 ]; then
    echo "Build failed! Exiting."
    exit 1
fi

echo "Build successful."

echo "Starting $BINARY_NAME..."
# Run in background with nohup
nohup ./$BINARY_NAME > $LOG_FILE 2>&1 &

echo "$BINARY_NAME started. Logs are being written to $LOG_FILE."
echo "PID: $!"
