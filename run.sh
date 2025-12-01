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

echo "Detecting OS..."
OS="$(uname -s)"
case "${OS}" in
    Linux*)     machine=Linux;;
    Darwin*)    machine=Mac;;
    CYGWIN*)    machine=Cygwin;;
    MINGW*)     machine=MinGW;;
    *)          machine="UNKNOWN:${OS}"
esac

echo "Detected OS: ${machine}"

echo "Installing Playwright dependencies..."
if [ "$machine" == "Linux" ]; then
    echo "Running on Linux, attempting to install dependencies (may require sudo)..."
    go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps
else
    echo "Running on $machine, installing dependencies..."
    go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps
fi

if [ $? -ne 0 ]; then
    echo "Playwright installation failed! Continuing with build..."
    # We don't exit here strictly, or should we? User asked to fix the issue.
    # If install fails, the app might fail at runtime.
    # Let's exit on failure to be safe, or just warn.
    # The user said "fix the problem", so failing if it fails is probably better.
    # But let's stick to the plan: try install, then build.
    exit 1
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
