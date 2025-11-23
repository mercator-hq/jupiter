# Mercator Jupiter - Production Dockerfile
# Used by GoReleaser for building Docker images
# GoReleaser builds the binary and copies it into this image

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

# GoReleaser will copy the binary here
COPY mercator /app/mercator

# Create directories
RUN mkdir -p /app/data /app/policies /app/config && \
    chown -R mercator:mercator /app

# Copy default config (optional)
# COPY config.yaml /app/config/config.yaml

# Switch to non-root user
USER mercator

# Expose ports
# 8080: Main proxy port
# 9090: Metrics/health port
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mercator", "version"]

# Default command
ENTRYPOINT ["/app/mercator"]
CMD ["run", "--config", "/app/config/config.yaml"]
