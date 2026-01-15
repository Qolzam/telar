# Build stage - Use Debian-based Go image for CGO compatibility
FROM golang:1.23-bookworm AS builder

# Set working directory
WORKDIR /app

# Install build dependencies (needed for CGO and ONNX)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    git \
    gcc \
    g++ \
    libc6-dev && \
    rm -rf /var/lib/apt/lists/*

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=1 is required for ONNX Runtime Go bindings
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -o ai-engine ./cmd/api

# Runtime stage - Use Debian slim for glibc compatibility (required for ONNX Runtime)
FROM debian:bookworm-slim

# Install runtime dependencies (required for ONNX Runtime)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    libstdc++6 \
    libgcc-s1 \
    libgomp1 \
    wget && \
    rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/ai-engine .

# Copy ONNX Runtime library (keep baked in as it's a system dependency)
COPY --from=builder /app/libs ./libs

# Copy the public directory for static files
COPY --from=builder /app/public ./public

# Copy the Init Script
COPY --from=builder /app/scripts/init-models.sh ./init-models.sh

# DO NOT COPY MODELS - They will be provided via volume mount or downloaded by init script
# COPY --from=builder /app/models ./models  <-- REMOVED

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app && \
    chmod -R 755 /app && \
    chmod +x /app/ai-engine && \
    chmod +x /app/init-models.sh

# Switch to non-root user
USER appuser

# Expose port (matches PORT env var, default 9066)
EXPOSE 9066

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:${PORT:-9066}/health || exit 1

# Set Entrypoint to init script, which will exec the Go binary
ENTRYPOINT ["./init-models.sh"]
CMD ["./ai-engine"]
