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
# Stage 2: Development (hot-reload with Air)
# =============================================================================
FROM deps AS dev

# Install runtime + dev tools
RUN apk add --no-cache \
    graphviz \
    rsvg-convert \
    ttf-dejavu

# Install Air for hot-reloading (pinned to version compatible with Go 1.24)
RUN go install github.com/air-verse/air@v1.61.5

WORKDIR /app

EXPOSE 8080

# Air watches for changes and rebuilds
CMD ["air", "-c", ".air.toml"]

# =============================================================================
# Stage 3: Build (production binary)
# =============================================================================
FROM deps AS builder

ARG VERSION=dev
ARG COMMIT=unknown

# Copy source code
COPY . .

# Build with cache mounts for faster incremental builds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X github.com/matzehuels/stacktower/pkg/buildinfo.Version=${VERSION} -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=${COMMIT}" \
    -o /stacktowerd ./cmd/stacktowerd

# =============================================================================
# Stage 4: Runtime Base (shared by prod and release)
# =============================================================================
FROM alpine:3.20 AS runtime-base

# Install runtime dependencies
# - ca-certificates: for HTTPS requests to package registries
# - git: for cloning repositories during analysis
# - graphviz: for nodelink visualization
# - rsvg-convert: for SVG to PNG/PDF conversion
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

# Create data directory (for any local storage needs)
RUN mkdir -p /data && chown -R stacktower:stacktower /data

# =============================================================================
# Stage 5: Production (builds from source)
# Usage: docker build --target prod .
# =============================================================================
FROM runtime-base AS prod

# Copy binary from builder stage
COPY --from=builder /stacktowerd /usr/local/bin/stacktowerd

USER stacktower
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/v1/health || exit 1

# Entrypoint - mode is determined by STACKTOWER_MODE env var or flags
# Examples:
#   docker run stacktower                     # API server (default)
#   docker run stacktower --worker            # Worker only
#   docker run stacktower --all               # API + Worker
#   docker run stacktower standalone          # Standalone mode
ENTRYPOINT ["/usr/local/bin/stacktowerd"]
CMD []

# =============================================================================
# Stage 6: Release (for GoReleaser - uses pre-built binary)
# Usage: GoReleaser copies binary to context, then builds with --target release
# =============================================================================
FROM runtime-base AS release

# Copy pre-built binary from GoReleaser build context
COPY stacktowerd /usr/local/bin/stacktowerd

USER stacktower
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/usr/local/bin/stacktowerd"]
CMD []
