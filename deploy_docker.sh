#!/bin/bash

APP_NAME="crawler-app"
IMAGE_NAME="crawler"

echo "🚀 Starting Docker Deployment for $APP_NAME..."

# 1. Update Code
echo "📥 Pulling latest code..."
git pull origin main

# 2. Cleanup Old Container
echo "🧹 Cleaning up old container..."
if [ "$(docker ps -q -f name=$APP_NAME)" ]; then
    docker stop $APP_NAME
fi
if [ "$(docker ps -aq -f name=$APP_NAME)" ]; then
    docker rm $APP_NAME
fi

# 3. Fix ip_usage.json (handle directory issue)
echo "🔧 Checking ip_usage.json..."
if [ -d "ip_usage.json" ]; then
    echo "⚠️ ip_usage.json is a directory! Removing..."
    rm -rf ip_usage.json
fi

if [ ! -f "ip_usage.json" ]; then
    echo "📄 Creating empty ip_usage.json..."
    echo "{}" > ip_usage.json
else
    echo "✅ ip_usage.json exists."
fi

# Ensure proxies.txt exists
if [ ! -f "proxies.txt" ]; then
    touch proxies.txt
fi
# Ensure vless.txt exists
if [ ! -f "vless.txt" ]; then
    touch vless.txt
fi

# 4. Build Image
echo "🔨 Building Docker image..."
docker build -t $IMAGE_NAME .

if [ $? -ne 0 ]; then
    echo "❌ Build failed! Aborting."
    exit 1
fi

# 5. Run Container
echo "🏃 Running container..."
docker run -d --name $APP_NAME \
  --restart always \
  -e THREADS=10 \
  -e HEADLESS=true \
  -v $(pwd)/proxies.txt:/app/proxies.txt \
  -v $(pwd)/vless.txt:/app/vless.txt \
  -v $(pwd)/ip_usage.json:/app/ip_usage.json \
  $IMAGE_NAME

echo "✅ Deployment complete! Container is running in background (Detached mode)."
echo "📜 Recent logs:"
sleep 2
docker logs --tail 20 $APP_NAME

echo ""
echo "💡 To follow logs in real-time, run: docker logs -f $APP_NAME"
echo "You can now safely exit the remote session without stopping the crawler."
