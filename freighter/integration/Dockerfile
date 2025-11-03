# syntax=docker/dockerfile:1

FROM golang:1.25 AS builder
WORKDIR /workspace

# Copy go.work files first
COPY go.work go.work.sum ./

# Copy go.mod and go.sum files for better caching
COPY freighter/integration/go.mod freighter/integration/go.sum ./freighter/integration/
COPY freighter/go/go.mod freighter/go/go.sum ./freighter/go/
COPY alamos/go/go.mod alamos/go/go.sum ./alamos/go/
COPY x/go/go.mod x/go/go.sum ./x/go/
COPY arc/go/go.mod arc/go/go.sum ./arc/go/
COPY aspen/go.mod aspen/go.sum ./aspen/
COPY cesium/go.mod cesium/go.sum ./cesium/
COPY core/go.mod core/go.sum ./core/

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code (only what's actually needed)
COPY freighter/go ./freighter/go
COPY freighter/integration ./freighter/integration
COPY alamos/go ./alamos/go
COPY x/go ./x/go
COPY arc ./arc
COPY aspen ./aspen
COPY cesium ./cesium
COPY core ./core

# Build with cache mounts
WORKDIR /workspace/freighter/integration

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-w -s" -o integration main.go

# Runtime image
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /workspace/freighter/integration/integration /app/integration
USER nonroot:nonroot
ENTRYPOINT ["/app/integration"]
