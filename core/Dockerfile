# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# Stage 1: Download dependencies
# The tag must match the go line in go.mod.
FROM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS deps
WORKDIR /workspace

# Copy go.mod and go.sum files from all modules for better caching
COPY core/go.mod core/go.sum ./core/
COPY freighter/go/go.mod freighter/go/go.sum ./freighter/go/
COPY alamos/go/go.mod alamos/go/go.sum ./alamos/go/
COPY x/go/go.mod x/go/go.sum ./x/go/
COPY cesium/go.mod cesium/go.sum ./cesium/
COPY arc/go/go.mod arc/go/go.sum ./arc/go/
COPY gorp/go.mod gorp/go.sum ./gorp/

# The modules go into a layer, not a cache mount, because the GitHub Actions cache
# keeps layers and drops cache mounts. The proxy drops streams at times, so a failed
# fetch falls back to the source and the download runs again.
ENV GOPROXY=https://proxy.golang.org|direct
WORKDIR /workspace/core
RUN for attempt in 1 2 3; do \
      go mod download && exit 0; \
      [ "$attempt" -lt 3 ] && sleep $((attempt * 15)); \
    done; \
    exit 1
WORKDIR /workspace

# Stage 2: Build the binary
FROM deps AS builder
WORKDIR /workspace

# Copy source code from all modules
COPY freighter/go ./freighter/go
COPY alamos/go ./alamos/go
COPY x/go ./x/go
COPY cesium ./cesium
COPY arc/go ./arc/go
COPY gorp ./gorp
COPY core ./core

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

# GOPROXY=off makes a module that the download missed fail the build.
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOPROXY=off GOOS=${TARGETOS} GOARCH=${TARGETARCH} ./docker-build.sh .

# A build context named binary replaces this stage with a directory that holds a binary
# that docker-build.sh built outside Docker.
FROM scratch AS binary
COPY --from=builder /workspace/core/synnax /synnax

# Stage 3: Minimal runtime image with glibc for driver support. The base is Ubuntu 26.04
# with curl for health checks, so the build installs no packages from an Ubuntu mirror.
FROM buildpack-deps:26.04-curl@sha256:34c4efba19abf9bdf07ba9c90430a7efeb7b9e1e3c62a8c69a2a40bf8a0ebc70

# Pass build args to final stage
ARG VERSION=
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# OCI standard labels for image metadata
LABEL org.opencontainers.image.title="Synnax"
LABEL org.opencontainers.image.description="Observability and control platform for hardware telemetry systems"
LABEL org.opencontainers.image.vendor="Synnax Labs, Inc."
LABEL org.opencontainers.image.source="https://github.com/synnaxlabs/synnax"
LABEL org.opencontainers.image.documentation="https://docs.synnaxlabs.com"
LABEL org.opencontainers.image.licenses="BSL-1.1"
LABEL org.opencontainers.image.url="https://synnaxlabs.com"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Create nonroot user
RUN groupadd -g 65532 nonroot \
    && useradd -u 65532 -g nonroot -s /bin/false -m nonroot

WORKDIR /synnax

# Copy binary and entrypoint script
COPY --from=binary /synnax /synnax/synnax
COPY core/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Expose default Synnax port (documentation only)
EXPOSE 9090

# Health check - verify connectivity endpoint responds
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f -X POST http://localhost:9090/api/v1/connectivity/check -H "Content-Type: application/json" -d '{}' || exit 1

# Use SIGTERM for graceful shutdown
STOPSIGNAL SIGTERM

ENTRYPOINT ["/docker-entrypoint.sh", "/synnax/synnax", "start"]
