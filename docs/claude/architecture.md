# Synnax Architecture

## Core Philosophy

Synnax is a **horizontally-scalable observability and control platform** built around a
strict **4-layer architecture** with unidirectional dependencies. The system prioritizes
**real-time performance** and **distributed reliability** for hardware telemetry
systems.

## Key Components

### 1. Synnax Server (Go) - `/core/`

The core time-series engine with 4-layer architecture:

- **Storage Layer**: Cesium (time-series) + Pebble (key-value) for disk persistence
- **Distribution Layer**: Aspen-based clustering with gossip protocols for horizontal
  scaling
- **Service Layer**: Business logic for channels, users, authentication, ranges
- **Interface Layer**: HTTP/WebSocket APIs via Freighter transport abstraction

### 2. Cesium - `/cesium/` (Go)

High-performance embedded time-series database:

- **Columnar storage** with automatic compression and domain-based indexing
- **Transactional writers** with conflict detection for data integrity
- **Memory-efficient iterators** for streaming large datasets
- Optimized for high-frequency (>1kHz) sensor data with microsecond precision

### 3. Aspen - `/aspen/` (Go)

Distributed key-value store and cluster management:

- **SI gossip protocol** for cluster membership and failure detection
- **SIR gossip protocol** for eventually consistent metadata synchronization
- **Dynamic node joining** with distributed counter assignment
- Read-optimized design suitable for metadata that's frequently accessed

### 4. Freighter - `/freighter/` (Multi-language)

Protocol-agnostic transport layer:

- **Unary and streaming** communication patterns
- Support for **gRPC, HTTP, WebSockets** with middleware architecture
- **Cross-language consistency** across Go, TypeScript, Python, C++
- Error handling and authentication standardization

### 5. Pluto - `/pluto/` (TypeScript/React)

High-performance visualization component library:

- **Aether framework** with worker-thread rendering to maintain 60fps
- **GPU-accelerated** visualization for real-time telemetry streams
- **Modular components** with incremental update patterns
- Generic architecture independent of Synnax-specific APIs

### 6. Console - `/console/` (Tauri + React + TypeScript)

Cross-platform desktop application (see @docs/claude/components/console.md for details):

- **Tauri (Rust) + React** for native performance with web UI
- **Redux Toolkit** for complex state management with persistence
- **Drift** for multi-window state synchronization via Tauri IPC
- **Drag-and-drop layout system** for custom interface building (mosaic pattern)
- **Embedded Synnax server** capability for standalone deployments
- Pre-rendering optimization for instant window creation

### 7. Driver - `/driver/` (C++)

Real-time hardware integration system (see @docs/claude/components/driver.md for
details):

- **Task-based architecture** with separate acquisition and control pipelines
- **Plugin architecture** via Factory pattern for hardware integrations
- **Device abstraction** supporting LabJack, National Instruments, OPC UA, Modbus TCP/IP
- **Pipeline infrastructure**: Acquisition (hardware → Synnax) and Control (Synnax →
  hardware)
- **Connection pooling** for shared device connections (Modbus, OPC UA)
- **Cross-platform support**: Windows, macOS, Linux, NI Linux RT (Modbus excluded on NI
  Linux RT)
- **Bazel build system** with `select()` for platform-specific compilation
- **Heartbeat mechanism** with 1Hz health reporting to cluster

### 8. Arc - `/arc/` (Go)

Domain-specific programming language for control systems:

- **Reactive execution model** with event-driven stages
- **Channel-based communication** between tasks (unbounded FIFO queues)
- **Stateful variables** that persist across reactive executions
- **WebAssembly compilation target** for sandboxed execution
- **Type system**: Primitives, series (arrays), channels, timestamps/timespans
- **Parser → Analyzer → Compiler** pipeline with LSP support
- Designed for hardware automation sequences and telemetry processing

### 9. Alamos - `/alamos/` (Multi-language)

Distributed instrumentation framework:

- **OpenTelemetry integration** for traces, metrics, and logs
- **Context propagation** across service boundaries
- **Cross-language support**: Go, TypeScript, Python
- Used throughout Synnax for observability and debugging

## Data Flow Patterns

### Telemetry Ingestion

```
Hardware → Driver (C++) → Synnax Server (Go) → Cesium Storage → Distribution → Clients
```

### Control Commands

```
Client → Synnax Server → Validation → Distribution → Driver → Hardware (with feedback)
```

### Cluster Communication

```
Node A ←→ Aspen Gossip ←→ Node B (metadata sync)
       ↓                    ↓
   Cesium Storage    Cesium Storage (time-series data routing)
```

## Architectural Patterns

- **Dependency Injection** throughout Go services for testability (Context objects,
  interface parameters)
- **Interface Segregation** with clear layer boundaries and abstractions (Source/Sink
  interfaces)
- **Factory Pattern** for creating hardware integrations and tasks (task::Factory in
  C++, service constructors in Go)
- **Command Pattern** for control operations with validation pipelines
- **Observer Pattern** for real-time event propagation (channel writes trigger stage
  execution)
- **Strategy Pattern** for pluggable transports and storage backends (Freighter
  middleware, FS backends)
- **RAII Pattern** in C++ for resource management (device connections, file handles)
- **Middleware Pattern** in Redux and Freighter for cross-cutting concerns

## Development Guidelines

- **Strict layering** - dependencies only flow downward in the 4-layer server
  architecture
- **Protocol agnostic** - use Freighter abstractions rather than direct HTTP/gRPC
- **Real-time focus** - optimize for low-latency, high-frequency data streams
- **Multi-language consistency** - maintain API parity across Go, TypeScript, Python,
  C++
- **Horizontal scalability** - design with distributed deployment in mind
- **Safety-critical reliability** - prefer availability over consistency for metadata,
  strong consistency for telemetry
- **Dependency injection & composition** - prefer dependency injection and composition
  over singletons, mocking, and inheritance across all languages (Go interfaces,
  TypeScript composition, etc.)

## Common Gotchas

- **Cesium**: Requires careful handling of overlapping time ranges to prevent write
  conflicts
- **Aspen**: Eventual consistency means metadata updates may take up to 1 second to
  propagate
- **Arc**: Compiles to WebAssembly; runtime provides host functions for channel/series
  operations
- **Integration tests**: Require running Synnax server instances - check for port
  conflicts

## Performance Considerations

- Cesium is optimized for columnar reads - structure queries to take advantage of this
- Pluto uses incremental rendering - avoid full component re-renders on data updates
- Freighter connection pooling - reuse clients rather than creating new instances
- Aspen gossip intervals are configurable but default to 1Hz for cluster coordination
- Driver tasks should minimize blocking operations in real-time acquisition loops
