#!/bin/bash

# Define the binary name
BINARY_NAME="crawler"
LOG_FILE="crawler.log"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# OS Detection
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
    echo "Detected OS: $OS $VER"
else
    echo "Cannot detect OS. Assuming generic Linux."
fi

# Install Go if not present
if ! command_exists go; then
    echo "Go is not installed. Attempting to install..."
    if [ "$ID" = "ubuntu" ] || [ "$ID" = "debian" ]; then
        sudo apt-get update
        sudo apt-get install -y golang-go
    else
        echo "Please install Go manually for your OS."
        exit 1
    fi
fi

# Install Playwright dependencies
echo "Installing Playwright dependencies..."
go mod download
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps

# Ubuntu 24.04 specific fix
if [ "$ID" = "ubuntu" ] && [ "$VER" = "24.04" ]; then
    echo "Applying Ubuntu 24.04 specific fixes..."
    sudo apt-get install -y libasound2t64 libicu74 libffi8 libx264-164
fi

# Check configuration files
if [ ! -f "vless.txt" ]; then
    echo "Warning: vless.txt not found. Creating empty file."
    touch vless.txt
    echo "Please add your VLESS links to vless.txt"
fi

if [ ! -f "target_site.txt" ]; then
    echo "Warning: target_site.txt not found. Creating default."
    echo "https://example.com" > target_site.txt
fi

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

echo "Cleaning up zombie processes (xray, chrome)..."
pkill -9 -f xray
pkill -9 -f chrome
pkill -9 -f chromium

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
