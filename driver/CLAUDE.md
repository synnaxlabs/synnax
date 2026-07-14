C++ development rules: @../docs/claude/toolchains/cpp.md

# Driver System

C++ real-time hardware integration connecting industrial hardware (LabJack, NI, OPC UA,
Modbus TCP/IP) to Synnax.

## Architecture (4 layers)

**Rack → Task Management → Pipelines → Device Integrations**

1. **Rack** (`/driver/rack/`) — top-level orchestrator: cluster connection, 1Hz
   heartbeat, config loading (files/env/CLI), factory registration (`factories.cpp`),
   platform-aware compilation.
2. **Task Management** (`/driver/task/`) — `task::Factory` (plugin interface),
   `task::Task` (base), `task::Manager` (lifecycle + commands), `task::Context` (DI:
   Synnax client, state updates). Task types: read (acquisition), write (control), scan
   (discovery).
3. **Pipelines** (`/driver/pipeline/`) — generic streaming: **Acquisition** (Source →
   Writer → Synnax) and **Control** (Synnax → Streamer → Sink). Automatic retry on
   `freighter::UNREACHABLE`, breaker pattern (exponential backoff), thread management.
4. **Device integrations** — each implements
   `Factory → {Read|Write|Scan}Task → Source/Sink → Device API`.

Shared task bases in `/driver/task/common/`: `sample_clock.h` (hardware/software timed),
`read_task.h` (`common::Source`), `write_task.h` (`common::Sink`), `scan_task.h`,
`status.h`, `factory.h`.

## Integrations

- **LabJack** (`/driver/labjack/`) — LJM wrapper + device manager. ReadTask has two
  modes: `StreamSource` (high-performance LJM streaming) and `UnarySource`
  (thermocouples, with CJC config). Transform chain for scaling.
- **NI** (`/driver/ni/`) — DAQmx + NISysCfg wrappers. Analog/digital read/write +
  scanner. Scanner discovers CompactDAQ chassis/modules via link properties
  (`ConnectsToLinkName`/`ProvidesLinkName`) in a two-pass algorithm (collect devices +
  build link-name→chassis map, then resolve module parents; chassis sorted before
  modules). Error codes: `DEVICE_DISCONNECTED`, `REQUIRES_RESTART`,
  `APPLICATION_TOO_SLOW`. Windows/Linux/NI Linux RT.
- **Modbus** (`/driver/modbus/`) — vendored libmodbus. `device::Device` (RAII around
  `modbus_t*`), `device::Manager` (connection pooling via weak_ptr cache), byte/word
  swap config. Readers: `RegisterReader` (holding/input registers), `BitReader`
  (coils/discrete inputs). Channel types: `holding_register_input`, `register_input`,
  `coil_input`, `discrete_input`. **Excluded on NI Linux Real-Time.**
- **OPC UA** (`/driver/opc/`) — vendored open62541 + mbedtls. `util::ConnectionPool`
  shared clients; ReadTask modes for array vs scalar reads; NodeId-based writes;
  security policy support. Windows/Linux/macOS.

## Plugin Pattern

Every integration implements `task::Factory::configure_task` (return `{nullptr, false}`
for unrecognized types) and `configure_initial_tasks`; registered in
`rack/factories.cpp` gated by `config.integration_enabled(<name>)`.

## Device Hierarchy

Device create API takes optional `parent` ontology ID (e.g. `"device:SERIAL"`),
atomically creating the device + `ParentOf` relationship; defaults to the rack when
absent (`core/pkg/service/device/writer.go`). The common scan task
(`driver/common/scan_task.h`) tracks parent changes between cycles and re-creates the
device when the parent changes. Console decides node expandability via make-based
dispatch (`console/src/hardware/device/make.tsx`, e.g. NI `is_chassis`). Used today by
NI (cDAQ chassis + modules).

## Tests

Many driver tests (rack, http, task suites) connect to a core at `localhost:9090`. Check
for a running core and start one if missing per "Live-Core Tests" in
`docs/claude/testing.md`. The user runs bazel test invocations.

## Gotchas

- SDKs required: LabJack LJM, NI-DAQmx.
- Connection pooling: Modbus and OPC UA share connections; LabJack/NI don't.
- Minimize blocking in real-time acquisition loops.
- Platform differences via Bazel `select()`, not `#ifdef`.
- Each integration has specialized error types — wrap with context when propagating.
