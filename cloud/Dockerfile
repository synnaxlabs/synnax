# syntax=docker/dockerfile:1

# Stage 1: Build the binary
FROM golang:1.26.0 AS builder
WORKDIR /workspace

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-w -s" -o cloud .

# Stage 2: Minimal runtime image
FROM ubuntu:24.04

LABEL org.opencontainers.image.title="Synnax Cloud"
LABEL org.opencontainers.image.description="Synnax Cloud auth proxy for gated downloads"
LABEL org.opencontainers.image.vendor="Synnax Labs, Inc."
LABEL org.opencontainers.image.source="https://github.com/synnaxlabs/synnax"
LABEL org.opencontainers.image.licenses="BSL-1.1"

RUN groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/false -m nonroot && \
    apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /cloud

COPY --from=builder /workspace/cloud /cloud/cloud

RUN mkdir -p /cloud/pb_data && chown -R nonroot:nonroot /cloud

USER nonroot

EXPOSE 8090

VOLUME ["/cloud/pb_data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

ENTRYPOINT ["/cloud/cloud", "serve", "--http=0.0.0.0:8090"]
