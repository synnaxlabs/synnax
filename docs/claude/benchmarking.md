# Allocation Benchmarking

Benchmark memory allocations by profiling the Synnax server under the Python integration
test suite with pprof.

## Prerequisites

- Go toolchain
- Python client with `uv` (`client/py/`)
- Two branches to compare (e.g., `main` vs feature branch)

## 1. Build Server Binaries

Build a binary for each branch you want to compare:

```bash
# From the branch under test
cd core && go build -o /tmp/synnax-<label> ./cmd/main.go
```

Example comparing `rc` vs a feature branch:

```bash
git checkout rc
cd core && go build -o /tmp/synnax-rc ./cmd/main.go

git checkout feature-branch
cd core && go build -o /tmp/synnax-feature ./cmd/main.go
```

## 2. Start Server with Profiling

Start the server with `--debug` to enable the pprof endpoint:

```bash
/tmp/synnax-<label> start \
    -d /tmp/synnax-<label>-data \
    --listen localhost:9090 \
    --debug \
    --insecure \
    --no-driver
```

- `--debug` enables pprof at `http://localhost:9090/debug/pprof/`
- `--insecure` skips TLS (required for local profiling)
- `--no-driver` disables the hardware driver (not needed for allocation profiling)
- `-d` sets a fresh data directory per run

Verify pprof is running:

```bash
curl -s http://localhost:9090/debug/pprof/ | head -5
```

## 3. Run Test Suite

In a separate terminal, run the Python integration tests to generate load:

```bash
cd client/py && uv run pytest
```

This exercises the full server path: channels, ranges, framing, ontology, etc.

## 4. Capture Allocation Profile

After the test suite completes (while the server is still running):

```bash
go tool pprof -text -cum http://localhost:9090/debug/pprof/allocs
```

Useful pprof commands:

```bash
# Top allocators by cumulative count
go tool pprof -text -cum -alloc_objects http://localhost:9090/debug/pprof/allocs

# Top allocators by cumulative bytes
go tool pprof -text -cum -alloc_space http://localhost:9090/debug/pprof/allocs

# Interactive mode
go tool pprof http://localhost:9090/debug/pprof/allocs
# Then: top20, web, list <function>, etc.

# Save profile to file for later analysis
curl -o /tmp/allocs-<label>.pb.gz http://localhost:9090/debug/pprof/allocs
go tool pprof -text -cum /tmp/allocs-<label>.pb.gz
```

## 5. Compare Results

Key metrics to compare between branches:

- **Total alloc objects** — number of heap allocations
- **Total alloc space** — total bytes allocated
- **Top cumulative allocators** — which functions dominate

Example comparison format:

| Metric | Branch A | Branch B |
|---|---|---|
| Total alloc objects | 25,303,927 | 137,779,404 |
| Total alloc space | 1,903.76 MB | 8,897.19 MB |
| Tests passed | 1,652 | 1,857 |

## 6. Cleanup

Stop the server (`Ctrl+C` or type `stop` in the server terminal), then remove data:

```bash
rm -rf /tmp/synnax-<label>-data
```

## Tips

- Always use a **fresh data directory** per run to avoid migration noise
- Run the **same test suite** on both branches for apples-to-apples comparison
- The test suite count may differ between branches — note this in results
- For CPU profiling, use `/debug/pprof/profile?seconds=30` instead of `/allocs`
