# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in
this repository.

## Common Development Commands

### Building the Project

- `pnpm build` - Build all packages using Turbo
- `pnpm build:console` - Build only the Console application
- `pnpm build:pluto` - Build only the Pluto component library
- `pnpm build:client` - Build only the client libraries
- `pnpm check-types` - Type check all TypeScript packages
- `pnpm check-types:console` - Type check only Console

### Development & Testing

- `pnpm dev:console` - Start Console in development mode (Tauri)
- `pnpm dev:console-vite` - Start Console Vite dev server only
- `pnpm dev:pluto` - Start Pluto development server
- `pnpm test` - Run all tests across packages
- `pnpm test:console` - Run Console tests
- `pnpm test:pluto` - Run Pluto tests
- `pnpm watch` - Watch mode for all packages

### Code Quality

- `pnpm lint` - Lint all packages with ESLint
- `pnpm fix` - Auto-fix linting issues across packages
- `pnpm lint:console` - Lint only Console package
- `pnpm fix:console` - Fix linting issues in Console

### Integration Tests

The repository includes comprehensive integration testing:

- Integration tests are located in `/integration/` directory
- Tests cover read, write, streaming, and delete operations across all client languages
- Run with configurable parameters for stress testing and performance measurement
- Tests validate the entire system from Python/TypeScript/C++ clients through Synnax
  server to Cesium storage

### Go Development

- Individual Go modules: `/core/`, `/aspen/`, `/cesium/`, `/freighter/go/`
- Use standard `go test`, `go build` commands within each module
- Integration tests available via scripts in `/integration/`

## High-Level Architecture

### Core Philosophy

Synnax is a **horizontally-scalable observability and control platform** built around a
strict **4-layer architecture** with unidirectional dependencies. The system prioritizes
**real-time performance** and **distributed reliability** for hardware telemetry
systems.

### Key Components

#### 1. Synnax Server (Go) - `/core/`

The core time-series engine with 4-layer architecture:

- **Storage Layer**: Cesium (time-series) + Pebble (key-value) for disk persistence
- **Distribution Layer**: Aspen-based clustering with gossip protocols for horizontal
  scaling
- **Service Layer**: Business logic for channels, users, authentication, ranges
- **Interface Layer**: HTTP/WebSocket APIs via Freighter transport abstraction

#### 2. Cesium - `/cesium/` (Go)

High-performance embedded time-series database:

- **Columnar storage** with automatic compression and domain-based indexing
- **Transactional writers** with conflict detection for data integrity
- **Memory-efficient iterators** for streaming large datasets
- Optimized for high-frequency (>1kHz) sensor data with microsecond precision

#### 3. Aspen - `/aspen/` (Go)

Distributed key-value store and cluster management:

- **SI gossip protocol** for cluster membership and failure detection
- **SIR gossip protocol** for eventually consistent metadata synchronization
- **Dynamic node joining** with distributed counter assignment
- Read-optimized design suitable for metadata that's frequently accessed

#### 4. Freighter - `/freighter/` (Multi-language)

Protocol-agnostic transport layer:

- **Unary and streaming** communication patterns
- Support for **gRPC, HTTP, WebSockets** with middleware architecture
- **Cross-language consistency** across Go, TypeScript, Python, C++
- Error handling and authentication standardization

#### 5. Pluto - `/pluto/` (TypeScript/React)

High-performance visualization component library:

- **Aether framework** with worker-thread rendering to maintain 60fps
- **GPU-accelerated** visualization for real-time telemetry streams
- **Modular components** with incremental update patterns
- Generic architecture independent of Synnax-specific APIs

#### 6. Console - `/console/` (Tauri + React + TypeScript)

Cross-platform desktop application:

- **Tauri (Rust) + React** for native performance with web UI
- **Redux Toolkit** for complex state management with persistence
- **Drag-and-drop layout system** for custom interface building
- **Embedded Synnax server** capability for standalone deployments

#### 7. Driver - `/driver/` (C++)

Real-time hardware integration system:

- **Task-based architecture** with separate acquisition and control pipelines
- **Device abstraction** supporting LabJack, National Instruments, OPC UA
- **Real-time OS support** including NI Linux RT
- **Heartbeat mechanism** with 1Hz health reporting to cluster

### Data Flow Patterns

#### Telemetry Ingestion

```
Hardware → Driver (C++) → Synnax Server (Go) → Cesium Storage → Distribution → Clients
```

#### Control Commands

```
Client → Synnax Server → Validation → Distribution → Driver → Hardware (with feedback)
```

#### Cluster Communication

```
Node A ←→ Aspen Gossip ←→ Node B (metadata sync)
       ↓                    ↓
   Cesium Storage    Cesium Storage (time-series data routing)
```

### Build System & Monorepo

- **PNPM workspaces** with catalog dependencies for TypeScript packages
- **Turbo** for build orchestration and caching across packages
- **Poetry** for Python package management
- **Go modules** with local workspace replacements
- **Bazel** for C++ components with complex dependencies

### Key Architectural Patterns

- **Dependency Injection** throughout Go services for testability
- **Interface Segregation** with clear layer boundaries and abstractions
- **Command Pattern** for control operations with validation pipelines
- **Observer Pattern** for real-time event propagation
- **Strategy Pattern** for pluggable transports and storage backends

### Testing Strategy

- **Unit tests** per component using language-specific frameworks (Vitest,
  Ginkgo/Gomega, pytest)
- **Integration tests** in `/integration/` covering full system with configurable
  parameters
- **Performance benchmarking** built into integration test framework
- **Cross-language validation** ensuring API consistency across client libraries

### Development Guidelines

- **Strict layering** - dependencies only flow downward in the 4-layer server
  architecture
- **Protocol agnostic** - use Freighter abstractions rather than direct HTTP/gRPC
- **Real-time focus** - optimize for low-latency, high-frequency data streams
- **Multi-language consistency** - maintain API parity across Go, TypeScript, Python,
  C++
- **Horizontal scalability** - design with distributed deployment in mind
- **Safety-critical reliability** - prefer availability over consistency for metadata,
  strong consistency for telemetry
- **Absolute imports** - always prefer absolute imports over relative imports in
  TypeScript projects (e.g., `@/components/Button` instead of
  `../../../components/Button`)
- **Vitest for testing** - always use Vitest APIs in TypeScript test files (e.g.,
  `import { describe, it, expect } from "vitest"` instead of Jest or other testing
  frameworks)
- **Dependency injection & composition** - prefer dependency injection and composition
  over singletons, mocking, and inheritance across all languages (Go interfaces,
  TypeScript composition, etc.)
- **Cross-platform C++** - always consider cross-platform compatibility when writing C++
  code (Windows, macOS, Linux, NI Linux RT); avoid platform-specific APIs unless
  absolutely necessary

### Common Gotchas

- Console has both Tauri (dev) and Vite-only (dev-vite) development modes
- Cesium requires careful handling of overlapping time ranges to prevent write conflicts
- Aspen's eventual consistency means metadata updates may take up to 1 second to
  propagate
- Driver components require specific hardware SDKs (LabJack LJM, NI DAQmx) for
  compilation
- Pluto components use worker threads - ensure proper serialization for data passing
- Integration tests require running Synnax server instances - check for port conflicts

### Performance Considerations

- Cesium is optimized for columnar reads - structure queries to take advantage of this
- Pluto uses incremental rendering - avoid full component re-renders on data updates
- Freighter connection pooling - reuse clients rather than creating new instances
- Aspen gossip intervals are configurable but default to 1Hz for cluster coordination
- Driver tasks should minimize blocking operations in real-time acquisition loops

### Code Style Guidelines

- All lines in our codebase are 88 characters

### Claude.md Self-Editing

- **Sparing edits** - When provided with useful context from humans that would benefit
  future interactions, make minimal edits to this file to preserve that context
- Only add information that is genuinely useful for future development work
- Keep additions concise and relevant to the codebase or development process
