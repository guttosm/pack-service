# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

# Target architecture (provided by Docker buildx)
ARG TARGETARCH
ARG TARGETOS=linux

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /app/pack-service \
    ./cmd/main.go

# Final stage - minimal alpine for debugging support
FROM alpine:3.19

LABEL org.opencontainers.image.title="pack-service" \
      org.opencontainers.image.description="Pack calculation service API" \
      org.opencontainers.image.source="https://github.com/guttosm/pack-service" \
      maintainer="pack-service"

# Install minimal tools for health checks and debugging
RUN apk add --no-cache ca-certificates tzdata wget && \
    rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary
COPY --from=builder /app/pack-service /pack-service

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run as non-root user
USER appuser:appgroup

ENTRYPOINT ["/pack-service"]
