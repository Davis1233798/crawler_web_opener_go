# Build Stage
FROM golang:1.21-bullseye AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o crawler ./cmd/crawler

# Final Stage
FROM mcr.microsoft.com/playwright:v1.40.0-jammy

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/crawler .
COPY --from=builder /app/target_site.txt .
# We don't need proxies.txt or .env in the image necessarily if passed via env vars, 
# but copying them for safety/defaults.
COPY --from=builder /app/proxies.txt . 

# Install any additional dependencies if needed (Playwright image has most)

# Set environment variables
ENV HEADLESS=true
ENV THREADS=2
ENV DURATION=60

# Entrypoint
CMD ["./crawler"]
