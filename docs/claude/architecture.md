# Synnax Architecture

Synnax is a horizontally-scalable observability and control platform for hardware
telemetry, optimized for real-time performance and distributed reliability.

## Components

- **Server** (`/core/`, Go) — strict 4-layer architecture, dependencies flow downward
  only. See `core/CLAUDE.md` for the deep dive.
  - **Storage** — engine management on a single node: Cesium for telemetry frames,
    Pebble KV for metadata. No cluster awareness.
  - **Distribution** — turns per-node storage into one cluster-transparent data plane:
    node membership via Aspen, channels with node-aware keys, framer (reads/writes
    routed to the nodes that lease the data), the ontology resource graph, and
    cluster-wide search.
  - **Service** — business logic on distributed primitives: auth, access control, users,
    ranges, hardware (task/rack/device), arc automations, workspace items (schematic,
    lineplot, log, table).
  - **API** — transport-agnostic client interface: request/response types, validation,
    freighter-compatible service definitions.
  - Alongside the stack: **transport** (concrete gRPC/HTTP bindings), **server**
    (multiplexes both protocols on one port), **security** (TLS/cert provider).
- **Cesium** (`/cesium/`, Go) — embedded time-series DB: columnar storage, domain-based
  indexing, transactional writers with conflict detection, streaming iterators. Built
  for >1kHz sensor data at microsecond precision.
- **Aspen** (`/aspen/`, Go) — distributed KV + cluster management: SI gossip
  (membership/failure detection), SIR gossip (eventually consistent metadata),
  read-optimized.
- **Freighter** (`/freighter/`, multi-language) — protocol-agnostic transport: unary +
  streaming over gRPC/HTTP/WebSockets, middleware, consistent across Go/TS/Python/C++.
- **Pluto** (`/pluto/`, TS/React) — GPU-accelerated viz components; Aether framework
  renders in a worker thread to hold 60fps; incremental updates.
- **Console** (`/console/`, Tauri + React) — desktop app; Redux Toolkit + Drift
  multi-window state sync; mosaic drag-and-drop layouts. See `console/CLAUDE.md`.
- **Driver** (`/driver/`, C++) — task-based hardware integration (LabJack, NI, OPC UA,
  Modbus). See `driver/CLAUDE.md`.
- **Arc** (`/arc/`, Go) — DSL for control systems: reactive event-driven stages,
  channel-based communication, value variables (literal stateful cell, channel read,
  channel read/write), compiles to WebAssembly (runtime provides host functions). Parser
  → Analyzer → Compiler + LSP.
- **Alamos** (`/alamos/`, multi-language) — instrumentation: OpenTelemetry
  traces/metrics/logs with cross-service context propagation.

## Dependency Graph

Every language stacks the same way, low to high: `x` (utilities) → `alamos`
(instrumentation) → `freighter` (transport) → client → application. Higher depends on
lower, never the reverse.

- **Go**: `x` and `alamos` are mutually dependent modules (packages stay acyclic) →
  `freighter` → `aspen`, `arc` → `core`. `cesium` uses only `x` + `alamos` (no
  transport). `oracle` uses only `x` + `alamos`.
- **TS**: `x` → `alamos` → `freighter` → `client` → `pluto` → `console`. `drift` depends
  only on `x`; `x/media` is a leaf; `arc` is consumed by `pluto` and `console`.
- **Python**: `x` → `alamos` → `freighter` → `client` (synnax) → `integration`.
- **C++**: `x` → `freighter` → `client` → `driver`.

## What Belongs Where

Put code in the lowest package that can hold it without gaining a forbidden dependency.

- `x/*` — generic utilities with no Synnax concepts: telemetry primitives, errors, data
  structures, config. If it mentions channels or clusters, it doesn't belong.
- `alamos/*` — instrumentation only: traces, metrics, logs.
- `freighter/*` — transport abstraction: unary/stream interfaces, middleware. No
  business logic.
- `cesium` — single-node time-series storage. No networking, no cluster awareness.
- `aspen` — cluster membership + distributed KV. No time-series data.
- `arc` — the Arc language toolchain: parser, analyzer, compiler, LSP, runtime.
- `oracle` — schema codegen CLI. Generators only, no runtime code.
- `core` — the server: composes cesium + aspen under the 4-layer architecture (see
  `core/CLAUDE.md`).
- `client/*` — cluster API clients, feature parity across languages. No UI, no hardware.
- `drift` — multi-window Redux sync. Knows Tauri + Redux, nothing about Synnax.
- `pluto` — reusable Synnax-aware React/viz components. Arrangement-blind: no mosaic, no
  layout registries — that's console.
- `console` — the desktop app: composition + arrangement (see `console/CLAUDE.md`).
- `driver` — hardware integration (see `driver/CLAUDE.md`).
- `integration` — cross-component integration tests + the tc conductor.

## Data Flow

- Ingestion: Hardware → Driver → Server → Cesium → Distribution → Clients.
- Control: Client → Server (validation) → Distribution → Driver → Hardware.
- Cluster: nodes sync metadata via Aspen gossip; time-series data routed between Cesium
  stores.

## Development Guidelines

- **Protocol agnostic** — use Freighter abstractions, never direct HTTP/gRPC.
- **Multi-language API parity** across Go, TS, Python, C++.
- **Real-time focus** — low latency, high frequency; design for horizontal scale.
- Availability over consistency for metadata; strong consistency for telemetry.

## Gotchas & Performance

- **Cesium**: overlapping time ranges cause write conflicts; structure queries for
  columnar reads.
- **Aspen**: eventual consistency — metadata updates may take up to 1s to propagate
  (gossip defaults to 1Hz).
- **Pluto**: incremental rendering — avoid full re-renders on data updates.
- **Freighter**: reuse pooled clients, don't construct new ones per call.
- **Driver**: minimize blocking operations in real-time acquisition loops.
- **Integration tests**: need running server instances — watch for port conflicts.
