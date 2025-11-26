FROM golang:1.21-bullseye AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./
# COPY go.sum ./ # Uncomment when go.sum exists
# RUN go mod download

COPY . .

# Build the application
RUN go build -o crawler ./cmd/crawler

# Final stage
FROM mcr.microsoft.com/playwright:v1.40.0-jammy

WORKDIR /app

# Install Go (optional, if we just run the binary we might not need full go, 
# but we need to ensure glibc compatibility. The builder stage handles compilation)
# Actually, we just need the binary and runtime deps.
# Playwright image is based on Ubuntu, so it should be fine.

COPY --from=builder /app/crawler .
COPY --from=builder /app/target_site.txt .
# COPY --from=builder /app/proxies.txt . # If you want to bundle proxies

# Install dependencies if needed (Playwright image has browsers)

CMD ["./crawler"]
