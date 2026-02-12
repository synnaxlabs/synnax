# syntax=docker/dockerfile:1

# Stage 1: Download dependencies
FROM golang:1.25.6 AS deps
WORKDIR /workspace

# Copy go.work files first
COPY go.work go.work.sum ./

# Copy go.mod and go.sum files from all modules for better caching
COPY core/go.mod core/go.sum ./core/
COPY freighter/go/go.mod freighter/go/go.sum ./freighter/go/
COPY freighter/integration/go.mod freighter/integration/go.sum ./freighter/integration/
COPY alamos/go/go.mod alamos/go/go.sum ./alamos/go/
COPY x/go/go.mod x/go/go.sum ./x/go/
COPY cesium/go.mod cesium/go.sum ./cesium/
COPY aspen/go.mod aspen/go.sum ./aspen/
COPY arc/go/go.mod arc/go/go.sum ./arc/go/
COPY oracle/go.mod oracle/go.sum ./oracle/

# Download dependencies with cache mount for go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Stage 2: Build the binary
FROM deps AS builder
WORKDIR /workspace

# Copy source code from all modules
COPY freighter/go ./freighter/go
COPY alamos/go ./alamos/go
COPY x/go ./x/go
COPY cesium ./cesium
COPY aspen ./aspen
COPY arc/go ./arc/go
COPY oracle ./oracle
COPY core ./core

# Build with cache mounts for both go modules and build cache
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG DRIVER_ENABLED=false
ARG CONSOLE_ENABLED=false
ARG VERSION=
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /workspace/core

RUN if [ "$DRIVER_ENABLED" = "true" ]; then \
      if [ ! -s "pkg/driver/assets/driver" ]; then \
        echo "ERROR: DRIVER_ENABLED=true but driver binary missing or empty"; \
        ls -la pkg/driver/assets/ 2>/dev/null || echo "assets directory does not exist"; \
        exit 1; \
      fi; \
      echo "Driver binary found:"; \
      file pkg/driver/assets/driver || ls -la pkg/driver/assets/driver; \
    fi

RUN if [ "$CONSOLE_ENABLED" = "true" ]; then \
      if [ ! -d "pkg/console/dist" ] || [ -z "$(ls -A pkg/console/dist 2>/dev/null)" ]; then \
        echo "ERROR: CONSOLE_ENABLED=true but console assets missing or empty"; \
        ls -la pkg/console/dist/ 2>/dev/null || echo "dist directory does not exist"; \
        exit 1; \
      fi; \
      echo "Console assets found:"; \
      ls pkg/console/dist/ | head -10; \
    fi

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    TAGS=""; \
    if [ "$DRIVER_ENABLED" = "true" ]; then \
    TAGS="${TAGS}driver"; \
    fi; \
    if [ "$CONSOLE_ENABLED" = "true" ]; then \
    if [ -n "$TAGS" ]; then TAGS="${TAGS},"; fi; \
    TAGS="${TAGS}console"; \
    fi; \
    LDFLAGS="-w -s -X github.com/synnaxlabs/synnax/pkg/version.Version=${VERSION} -X github.com/synnaxlabs/synnax/pkg/version.GitCommit=${GIT_COMMIT} -X github.com/synnaxlabs/synnax/pkg/version.BuildDate=${BUILD_DATE}"; \
    if [ -n "$TAGS" ]; then \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -tags "$TAGS" -ldflags="${LDFLAGS}" -o synnax .; \
    else \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="${LDFLAGS}" -o synnax .; \
    fi

# Stage 3: Minimal runtime image with glibc for driver support
FROM ubuntu:24.04

# Pass build args to final stage
ARG VERSION=
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# OCI standard labels for image metadata
LABEL org.opencontainers.image.title="Synnax"
LABEL org.opencontainers.image.description="Horizontally-scalable observability and control platform for hardware telemetry systems"
LABEL org.opencontainers.image.vendor="Synnax Labs, Inc."
LABEL org.opencontainers.image.source="https://github.com/synnaxlabs/synnax"
LABEL org.opencontainers.image.documentation="https://docs.synnaxlabs.com"
LABEL org.opencontainers.image.licenses="BSL-1.1"
LABEL org.opencontainers.image.url="https://synnaxlabs.com"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Create nonroot user
RUN groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/false -m nonroot

WORKDIR /synnax

# Copy binary and entrypoint script from builder
COPY --from=builder /workspace/core/synnax /synnax/synnax
COPY --from=builder /workspace/core/docker-entrypoint.sh /docker-entrypoint.sh

# Install gosu for dropping privileges and curl for health checks
RUN apt-get update && \
    apt-get install -y --no-install-recommends gosu curl && \
    rm -rf /var/lib/apt/lists/* && \
    chmod +x /docker-entrypoint.sh

# Expose default Synnax port (documentation only)
EXPOSE 9090

# Health check - verify connectivity endpoint responds
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f -X POST http://localhost:9090/api/v1/connectivity/check -H "Content-Type: application/json" -d '{}' || exit 1

# Use SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

ENTRYPOINT ["/docker-entrypoint.sh", "/synnax/synnax", "start"]
