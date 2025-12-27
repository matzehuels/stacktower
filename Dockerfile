# syntax=docker/dockerfile:1

# =============================================================================
# Stage 1: Dependencies (cached unless go.mod/go.sum change)
# =============================================================================
FROM golang:1.24-alpine AS deps

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Download dependencies (this layer is cached unless go.mod/sum change)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download -x

# =============================================================================
# Stage 2: Build (uses cached deps, only rebuilds on source changes)
# =============================================================================
FROM deps AS builder

# Copy source code
COPY . .

# Build with cache mounts for faster incremental builds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /stacktowerd ./cmd/server

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies
# - ca-certificates: for HTTPS requests to package registries
# - git: for cloning repositories during agent analysis
# - graphviz: for nodelink visualization
# - rsvg-convert: for SVG to PNG/PDF conversion (separate from librsvg on Alpine)
# - ttf-dejavu: fonts for rendering
RUN apk add --no-cache \
    ca-certificates \
    git \
    graphviz \
    rsvg-convert \
    ttf-dejavu

# Create non-root user
RUN addgroup -g 1000 stacktower && \
    adduser -u 1000 -G stacktower -s /bin/sh -D stacktower

WORKDIR /app

# Copy binary from builder
COPY --from=builder /stacktowerd /usr/local/bin/stacktowerd

# Create directories for local storage (optional fallback)
RUN mkdir -p /data/storage /data/cache && \
    chown -R stacktower:stacktower /data

USER stacktower

# Default environment variables
ENV STACKTOWER_STORAGE_DIR=/data/storage \
    STACKTOWER_CACHE_DIR=/data/cache \
    STACKTOWER_HOST=0.0.0.0 \
    STACKTOWER_PORT=8080

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Default command (can be overridden)
# Use --worker for worker-only mode
ENTRYPOINT ["/usr/local/bin/stacktowerd"]
CMD []

