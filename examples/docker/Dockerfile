# Mercator Jupiter - Production Dockerfile
# Multi-stage build for optimized image size

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty)" \
    -o mercator \
    ./cmd/mercator

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite

# Create non-root user
RUN addgroup -g 1000 mercator && \
    adduser -D -u 1000 -G mercator mercator

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mercator /app/mercator

# Create directories
RUN mkdir -p /app/data /app/policies /app/config && \
    chown -R mercator:mercator /app

# Copy default config (optional)
# COPY config.yaml /app/config/config.yaml

# Switch to non-root user
USER mercator

# Expose ports
EXPOSE 8080
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mercator", "version"]

# Default command
ENTRYPOINT ["/app/mercator"]
CMD ["run", "--config", "/app/config/config.yaml"]
