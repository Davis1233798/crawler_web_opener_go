# Build Stage
FROM golang:1.23-bullseye AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Install Playwright driver
# This populates /root/.cache/ms-playwright-go with the driver binary required by the Go library
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@v0.4001.0 install --with-deps

# Copy source
COPY . .

# Build
RUN go build -o crawler ./cmd/crawler

# Final Stage
# Using v1.40.0 to match the Go library version (v0.4001.0 ~= v1.40.1)
FROM mcr.microsoft.com/playwright:v1.40.0-jammy

WORKDIR /app

# Install Xray (Method: Download binary)
RUN apt-get update && apt-get install -y unzip curl \
    && curl -L -o xray.zip https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip \
    && unzip xray.zip -d /usr/bin/ \
    && chmod +x /usr/bin/xray \
    && rm xray.zip \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy Playwright driver from builder cache
COPY --from=builder /root/.cache/ms-playwright-go /root/.cache/ms-playwright-go

# Copy binary and assets
COPY --from=builder /app/crawler .
COPY --from=builder /app/target_site.txt .
COPY --from=builder /app/.env.example .
# Copy empty proxies.txt/vless.txt if they exist in context, handled by user mounting usually
# We copy them just in case they are needed for default startup
COPY proxies.txt . 
COPY vless.txt .

# Ensure permissions
RUN chmod +x crawler

# Environment variables
ENV HEADLESS=true
ENV THREADS=10

CMD ["./crawler"]
