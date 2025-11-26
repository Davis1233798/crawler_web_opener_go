#!/bin/bash

# Define the binary name
BINARY_NAME="crawler"
LOG_FILE="crawler.log"

# Ensure we have the latest code
echo "Pulling latest changes..."
git pull origin feature/streamruby-watcher

# Install Go dependencies
echo "Tidying Go modules..."
go mod tidy

# Install Playwright browsers and system dependencies
echo "Installing Playwright dependencies..."
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps

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
