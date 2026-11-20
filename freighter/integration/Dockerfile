# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# The tag must match the go line in go.mod.
FROM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS builder
WORKDIR /workspace

# Copy go.mod and go.sum files for better caching
COPY freighter/integration/go.mod freighter/integration/go.sum ./freighter/integration/
COPY freighter/go/go.mod freighter/go/go.sum ./freighter/go/
COPY alamos/go/go.mod alamos/go/go.sum ./alamos/go/
COPY x/go/go.mod x/go/go.sum ./x/go/

# Download dependencies with cache mount
WORKDIR /workspace/freighter/integration
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
WORKDIR /workspace

# Copy source code (only what's actually needed)
COPY freighter/go ./freighter/go
COPY freighter/integration ./freighter/integration
COPY alamos/go ./alamos/go
COPY x/go ./x/go

# Build with cache mounts
WORKDIR /workspace/freighter/integration

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-w -s" -o integration main.go

# Runtime image
FROM gcr.io/distroless/static-debian13:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931
WORKDIR /app
COPY --from=builder /workspace/freighter/integration/integration /app/integration
USER nonroot:nonroot
ENTRYPOINT ["/app/integration"]
