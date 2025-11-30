#!/bin/bash
apt-get update
apt-get install -y git docker.io

# Prepare directory
mkdir -p /home/crawler
cd /home/crawler

# Clone repo
if [ -d crawler_web_opener_go ]; then
    cd crawler_web_opener_go
    git pull origin no-proxy-docker-gcp
else
    git clone https://github.com/Davis1233798/crawler_web_opener_go.git
    cd crawler_web_opener_go
    git checkout no-proxy-docker-gcp
fi

# Fix permissions
chown -R google-sudoers:google-sudoers /home/crawler/crawler_web_opener_go

# Start the controller loop automatically
cd /home/crawler/crawler_web_opener_go/gcp
chmod +x controller_loop.sh deploy.sh monitor.sh

# Run in background, logging to a file
nohup ./controller_loop.sh ${project_id} 300 > /var/log/crawler_loop.log 2>&1 &
