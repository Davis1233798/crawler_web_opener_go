# Build Stage
FROM golang:1.23-bullseye AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN go build -o crawler ./cmd/crawler

# Final Stage
FROM mcr.microsoft.com/playwright:v1.48.0-jammy

WORKDIR /app

# Install Xray (Method: Download binary)
RUN apt-get update && apt-get install -y unzip curl \
    && curl -L -o xray.zip https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip \
    && unzip xray.zip -d /usr/bin/ \
    && chmod +x /usr/bin/xray \
    && rm xray.zip \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy binary and assets
COPY --from=builder /app/crawler .
COPY --from=builder /app/target_site.txt .
COPY --from=builder /app/.env.example .
# Create empty proxies.txt if not exists, or copy if exists (using wildcard hack if needed, but simple COPY is safer if we ensure it exists)
COPY proxies.txt . 
# Copy vless.txt if needed
COPY vless.txt .

# Ensure permissions
RUN chmod +x crawler

# Environment variables
ENV HEADLESS=true
ENV THREADS=10

CMD ["./crawler"]
