# Driver System

The Driver is a C++ real-time hardware integration system that connects industrial
hardware to the Synnax platform. It supports LabJack, National Instruments, OPC UA, and
Modbus devices.

See @docs/claude/toolchains/cpp.md for C++ toolchain details.

## Architecture (4 Layers)

```
Rack (Integration Point)
  ↓
Task Management
  ↓
Pipelines (Data Flow)
  ↓
Device Integrations (Hardware)
```

### Layer 1: Rack (`/driver/rack/`)

Top-level orchestrator:

- Connects to Synnax cluster
- Manages heartbeat (1Hz health reporting)
- Loads configuration from multiple sources
- Registers hardware integrations via factories
- Platform-aware (`#ifndef SYNNAX_NILINUXRT` for conditional compilation)

**Key Files:**

- `rack.h/cpp` - Main entry point
- `config.h/cpp` - Configuration loading (files, environment, CLI)
- `factories.cpp` - Hardware integration registration

### Layer 2: Task Management (`/driver/task/`)

Abstract task lifecycle:

- **task::Factory** - Plugin interface for creating device-specific tasks
- **task::Task** - Base interface for all executable tasks
- **task::Manager** - Coordinates task creation, execution, and commands
- **task::Context** - Dependency injection (Synnax client, state updates)

**Task Types:**

- Read tasks (acquisition)
- Write tasks (control)
- Scan tasks (device discovery)

### Layer 3: Pipelines (`/driver/pipeline/`)

Generic streaming infrastructure:

**Acquisition Pipeline** (hardware → Synnax):

```
Source → Writer → Synnax
```

**Control Pipeline** (Synnax → hardware):

```
Synnax → Streamer → Sink
```

**Features:**

- Automatic retry on `freighter::UNREACHABLE`
- Breaker pattern for exponential backoff
- Thread management
- Error handling

### Layer 4: Device Integrations

Each integration implements the same pattern:

```
Factory → {Read|Write|Scan}Task → Source/Sink → Device API
```

## Hardware Integrations

### LabJack (`/driver/labjack/`)

**Dependencies:**

- `ljm/` - LabJack Modbus wrapper
- `device/` - Device manager

**Tasks:**

- **ReadTask**: Two modes
  - `StreamSource` - LJM streaming for high-performance
  - `UnarySource` - For thermocouples
- **WriteTask**: Digital/analog output control
- **ScanTask**: Device discovery

**Special Features:**

- Thermocouple support with CJC configuration
- Transform chain for scaling
- Windows-specific compiler flags

### National Instruments (`/driver/ni/`)

**Dependencies:**

- `daqmx/` - NI-DAQmx wrapper
- `syscfg/` - System configuration
- `hardware/` - Hardware abstraction
- `channel/` - Channel management

**Tasks:**

- Analog/digital read
- Analog/digital write
- Scanner (with chassis/module hierarchy resolution)

**Chassis/Module Hierarchy:**

The NI scanner discovers CompactDAQ chassis and their installed modules using NISysCfg
link properties (`ConnectsToLinkName` / `ProvidesLinkName`). The scan uses a two-pass
algorithm: first collect all devices and build a link-name-to-chassis-key map, then
resolve each module's parent via that map. Chassis are sorted before modules to ensure
parents are created first. The driver keeps a local `parent_device` field on the C++
Device struct for change detection; when it changes, the common scan task
(`driver/task/common/scan_task.h`) re-sends the device create request with the
appropriate `parent` ontology ID.

**Error Handling:**

- Specialized error translation
- Error codes: `DEVICE_DISCONNECTED`, `REQUIRES_RESTART`, `APPLICATION_TOO_SLOW`

**Platform Support:**

- Windows, Linux, NI Linux Real-Time

### Modbus TCP/IP (`/driver/modbus/`)

**Dependencies:**

- `vendor/libmodbus` - External library

**Device Abstraction:**

- **device::Device**: Wraps `modbus_t*` with RAII
- **device::Manager**: Connection pooling with weak_ptr caching
- **device::ConnectionConfig**: TCP/IP settings with byte/word swapping

**Tasks:**

- **ReadTask**: Polymorphic `Reader` interface
  - `RegisterReader` - Holding/input registers
  - `BitReader` - Coils/discrete inputs
- **WriteTask**: Coil/register output
- **ScanTask**: Device discovery

**Channel Types:**

- `holding_register_input`
- `register_input`
- `coil_input`
- `discrete_input`

**Platform:**

- ❌ **Excluded on NI Linux Real-Time** via Bazel `select()`

### OPC UA (`/driver/opc/`)

**Dependencies:**

- `vendor/open62541` - OPC UA stack
- `vendor/mbedtls` - Cryptography

**Connection Management:**

- **util::ConnectionPool**: Shared client connections
- **util::ConnectionConfig**: Endpoint URL, security settings

**Tasks:**

- **ReadTask**: Two modes
  - `ArrayReadTaskSource` - Array data
  - `UnaryReadTaskSource` - Scalar reads
- **WriteTask**: NodeId-based writing
- **ScanTask**: Server/node discovery

**Features:**

- OPC UA NodeId parsing
- Array handling
- Security policy support

**Platform:**

- Windows, Linux, macOS

## Common Task Patterns (`/driver/task/common/`)

Shared across all integrations:

- **sample_clock.h**: Hardware-timed and software-timed sample clocks
- **read_task.h**: Generic read task base with `common::Source` interface
- **write_task.h**: Generic write task base with `common::Sink` interface
- **scan_task.h**: Device discovery task base
- **status.h**: Health reporting and error status
- **factory.h**: Factory pattern helpers

## Plugin Pattern

All integrations implement `task::Factory`:

```cpp
class Factory : public task::Factory {
    std::pair<std::unique_ptr<Task>, bool> configure_task(
        const std::shared_ptr<Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<Task>>>
    configure_initial_tasks(...) override;
};
```

### Registration in Rack

```cpp
// factories.cpp
void configure_opc(const rack::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(opc::INTEGRATION_NAME)) return;
    factories.push_back(std::make_unique<opc::Factory>());
}
```

## Cross-Platform Build Strategy

### Platform Detection

```python
config_setting(
    name = "nilinuxrt",
    values = {"define": "platform=nilinuxrt"},
)
```

### Conditional Compilation

**Source Files:**

```python
srcs = select({
    ":nilinuxrt": ["daemon_nilinuxrt.cpp"],
    "@platforms//os:linux": ["daemon_linux.cpp"],
    "//conditions:default": ["daemon_noop.cpp"],
})
```

**Dependencies:**

```python
deps = select({
    ":nilinuxrt": [],
    "//conditions:default": ["//driver/modbus"],
})
```

**Compiler Flags:**

```python
copts = select({
    "@platforms//os:windows": [
        "/DWIN32_LEAN_AND_MEAN",
        "/DNOMINMAX",
    ],
})
```

**Linker Options:**

```python
linkopts = select({
    "@platforms//os:windows": ["ws2_32.lib", "Iphlpapi.lib"],
})
```

## Device Hierarchy (Parent-Child Relationships)

Devices support parent-child relationships via the ontology. The device create API
accepts an optional `parent` ontology ID (e.g., `"device:SERIAL123"`) that atomically
creates both the device and the `ParentOf` relationship in a single transaction. When no
parent is provided, the device defaults to being parented under its rack.

**How it works:**

- The device create endpoint (`core/pkg/service/device/writer.go`) accepts an optional
  `parent ontology.ID`. If non-zero, the device is parented to that resource; otherwise
  it defaults to the device's rack.
- The C++ driver keeps a local `parent_device` field on the Device struct for change
  detection between scan cycles. This field is not stored on the server — instead, the
  driver maps it to the `parent` field on the create request (e.g.,
  `"device:" + parent_device`).
- The common scan task (`driver/common/scan_task.h`) detects local `parent_device`
  changes and triggers updates.
- The Console uses a make-based dispatch (`console/src/hardware/device/make.tsx`) to
  determine if a device node should be expandable (e.g., NI checks `is_chassis` in
  device properties).

**Currently used by:** NI integration (cDAQ chassis with swappable modules).

## Key Architectural Patterns

1. **Factory Pattern**: Each integration provides a factory for task creation
2. **Strategy Pattern**: Pluggable `Source`/`Sink` implementations
3. **Dependency Injection**: `task::Context` provides Synnax client and state
4. **RAII**: Device connections use smart pointers and destructors
5. **Error Hierarchy**: Specialized error codes per integration
6. **Breaker Pattern**: Automatic retry with exponential backoff
7. **Connection Pooling**: Shared device connections (Modbus, OPC UA)

## Common Gotchas

- **SDK Requirements**: LabJack LJM, NI-DAQmx must be installed
- **Modbus on NI Linux Real-Time**: Excluded via Bazel config
- **Platform-specific code**: Use Bazel `select()` mechanism
- **Connection pooling**: Modbus and OPC UA share connections, LabJack/NI don't
- **Blocking operations**: Minimize in real-time acquisition loops
- **Error codes**: Each integration has specialized error types

## Development Best Practices

- **Factory pattern**: Implement `task::Factory` for new integrations
- **Source/Sink interfaces**: Implement for pipeline compatibility
- **RAII**: Use smart pointers for device connections
- **Bazel select()**: Use for platform-specific code, not `#ifdef`
- **Error wrapping**: Add context when propagating errors
- **Thread safety**: Use mutexes for shared state in tasks
- **Connection reuse**: Pool connections when possible (Modbus, OPC UA pattern)
