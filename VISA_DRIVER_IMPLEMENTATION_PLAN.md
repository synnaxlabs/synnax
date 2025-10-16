# VISA Driver Implementation Plan

**Status:** Planning
**Started:** 2025-01-XX
**Target Completion:** TBD

---

## Overview

Implement a VISA (Virtual Instrument Software Architecture) driver for Synnax that
enables communication with test and measurement instruments over GPIB, USB, TCPIP, and
Serial interfaces using the industry-standard VISA API.

**Key Capabilities:**
- SCPI command-based instrument control
- Multiple transport protocols (GPIB, USB, TCPIP, Serial)
- Flexible response parsing (float, int, string, arrays, binary blocks)
- Session pooling for shared instrument connections
- Device discovery via resource enumeration

---

## Architecture Summary

```
C++ Driver (Backend)                    TypeScript Console (Frontend)
├── device/                             ├── device/
│   ├── Session (RAII wrapper)          │   ├── Connect.tsx (connection form)
│   └── Manager (connection pool)       │   ├── Configure.tsx (wizard)
├── task/                               │   └── types.ts (Zod schemas)
│   ├── ReadTask                        ├── task/
│   ├── WriteTask                       │   ├── Read.tsx (read config UI)
│   └── ScanTask                        │   ├── Write.tsx (write config UI)
└── Factory (integration)               │   ├── Scan.tsx (scanner UI)
                                        │   └── types.ts (task schemas)
                                        └── external.ts (registration)
```

---

## Phase 1: C++ Driver Backend

### 1.1 Project Setup & Dependencies

**Files to Create:**
- [ ] `/driver/visa/BUILD.bazel` - Bazel build configuration
- [ ] `/driver/visa/visa.h` - Main header with integration constants

**Dependencies:**
- [ ] NI-VISA library (platform-specific)
  - Windows: `C:\Program Files\IVI Foundation\VISA\Win64\Include\visa.h`
  - Linux: `/usr/local/vxipnp/linux/include/visa.h`
  - macOS: `/Library/Frameworks/VISA.framework/Headers/visa.h`
- [ ] Add Bazel `select()` for platform-specific include paths and linker flags

**Bazel Configuration:**
```python
cc_library(
    name = "visa",
    srcs = [...],
    deps = [
        "//driver/errors",
        "//driver/visa/device",
        "//driver/visa/util",
        "//driver/pipeline",
        "//driver/task",
        "//driver/task/common",
    ],
    copts = select({
        "@platforms//os:windows": [
            "-I\"C:/Program Files/IVI Foundation/VISA/Win64/Include\"",
        ],
        "//conditions:default": ["-I/usr/local/vxipnp/linux/include"],
    }),
    linkopts = select({
        "@platforms//os:windows": ["visa64.lib"],
        "//conditions:default": ["-lvisa"],
    }),
)
```

**Open Questions:**
- [ ] Do we need to support NI Linux RT? (Probably yes, add conditional compilation)
- [ ] What minimum VISA version should we target? (VISA 5.0+)

---

### 1.2 Device Session Management

**Files to Create:**
- [ ] `/driver/visa/device/device.h` - Session wrapper & manager
- [ ] `/driver/visa/device/device.cpp` - Implementation
- [ ] `/driver/visa/device/BUILD.bazel` - Build config

**Implementation Checklist:**

**Session Class (RAII wrapper):**
- [ ] Constructor: Store `ViSession`
- [ ] Destructor: Call `viClose(session)`
- [ ] Delete copy constructor/assignment (move-only)
- [ ] `read()` method - wraps `viRead()`
- [ ] `write()` method - wraps `viWrite()`
- [ ] `query()` method - combined write + read
- [ ] `set_timeout()` - wraps `viSetAttribute(VI_ATTR_TMO_VALUE)`
- [ ] `set_term_char()` - wraps `viSetAttribute(VI_ATTR_TERMCHAR)`
- [ ] Error handling with `parse_visa_error()`

**ConnectionConfig Struct:**
- [ ] `std::string resource_name` (e.g., "TCPIP0::192.168.1.100::INSTR")
- [ ] `uint32_t timeout_ms` (default: 5000)
- [ ] `char term_char` (default: '\n')
- [ ] `bool term_char_enabled` (default: true)
- [ ] Constructor from `xjson::Parser`
- [ ] `to_json()` method

**Manager Class (Connection Pool):**
- [ ] `std::unordered_map<std::string, std::weak_ptr<Session>> sessions`
- [ ] `ViSession resource_manager` (opened in constructor via `viOpenDefaultRM()`)
- [ ] `acquire(ConnectionConfig)` - get or create session
  - Check cache for existing session
  - If expired, create new via `viOpen()`
  - Configure timeout and term char
  - Return `shared_ptr<Session>`
- [ ] Destructor: `viClose(resource_manager)`

**Error Handling:**
- [ ] `/driver/visa/errors.h`
- [ ] Define `CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("visa")`
- [ ] Define `TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("visa")`
- [ ] `parse_visa_error(ViStatus)` function
  - Map `VI_ERROR_TMO`, `VI_ERROR_CONN_LOST` → TEMPORARY_ERROR
  - Map other errors → CRITICAL_ERROR
  - Use `viStatusDesc()` for error messages

**Reference Files:**
- Pattern: `driver/modbus/device/device.h:159` (Manager class)
- Pattern: `driver/opc/util/conn_pool.h` (Connection pooling)

---

### 1.3 Channel Configuration

**Files to Create:**
- [ ] `/driver/visa/channels.h` - Channel type definitions

**Channel Types:**

**BaseChannel:**
- [ ] `synnax::ChannelKey synnax_key`
- [ ] `synnax::Channel ch`
- [ ] `std::string scpi_command`
- [ ] Constructor from `xjson::Parser`

**InputChannel (extends BaseChannel):**
- [ ] `ResponseFormat format` enum
- [ ] `telem::DataType data_type`
- [ ] `std::string delimiter` (for arrays, default: ",")
- [ ] `size_t array_length` (0 = variable length)

**ResponseFormat enum:**
- [ ] `FLOAT` - Parse single float
- [ ] `INTEGER` - Parse single int
- [ ] `STRING` - Return as string
- [ ] `FLOAT_ARRAY` - Parse comma-separated floats
- [ ] `BINARY_BLOCK` - IEEE 488.2 definite length block
- [ ] `BOOLEAN` - Parse 0/1 or ON/OFF

**OutputChannel (extends BaseChannel):**
- [ ] `std::string command_template` (e.g., "SOUR:VOLT:DC {value}")

**Utilities:**
- [ ] `/driver/visa/util/parse.h` - Response parsing utilities
- [ ] `/driver/visa/util/parse.cpp` - Implementations
  - `parse_float(const std::string&)`
  - `parse_int(const std::string&)`
  - `parse_float_array(const std::string&, const std::string& delimiter)`
  - `parse_binary_header(const std::string&)` - Parse `#<digit><length>` header
  - `parse_response(const std::string&, const InputChannel&)` - Dispatcher

---

### 1.4 Read Task Implementation

**Files to Create:**
- [ ] `/driver/visa/read_task.h` - ReadTask types and source

**ReadTaskConfig:**
- [ ] Extend `common::BaseReadTaskConfig`
- [ ] `std::string device_key`
- [ ] `device::ConnectionConfig conn`
- [ ] `std::vector<InputChannel> channels`
- [ ] `std::set<synnax::ChannelKey> indexes`
- [ ] `size_t samples_per_chan`
- [ ] `static parse()` method from `synnax::Task`
- [ ] `writer_config()` method
- [ ] `data_channels()` method

**ReadTaskSource (implements `common::Source`):**
- [ ] Constructor: Store session, config, create sample clock
- [ ] `read(breaker::Breaker&, synnax::Frame&)` override
  - Loop `samples_per_chan` times
  - For each channel:
    - Send SCPI query via `session->query()`
    - Parse response with `util::parse_response()`
    - Write to frame series
  - Add timestamps to index channels
  - Use `sample_clock.wait()` for timing
- [ ] `writer_config()` override
- [ ] `channels()` override

**Testing:**
- [ ] `/driver/visa/read_task_test.cpp`
- [ ] Mock VISA session for testing
- [ ] Test float parsing
- [ ] Test array parsing
- [ ] Test binary block parsing
- [ ] Test error handling

**Reference Files:**
- Pattern: `driver/modbus/read_task.h:321` (ReadTaskSource)
- Pattern: `driver/task/common/read_task.h:82` (Source interface)

---

### 1.5 Write Task Implementation

**Files to Create:**
- [ ] `/driver/visa/write_task.h` - WriteTask types and sink

**WriteTaskConfig:**
- [ ] `std::string device_key`
- [ ] `device::ConnectionConfig conn`
- [ ] `std::vector<OutputChannel> channels`
- [ ] `static parse()` method

**WriteTaskSink (implements `common::Sink`):**
- [ ] Constructor: Store session, config
- [ ] `write(const synnax::Frame&)` override
  - For each channel in frame:
    - Get value from series
    - Format command using template (replace `{value}`)
    - Send via `session->write()`
  - Return error on first failure

**Testing:**
- [ ] `/driver/visa/write_task_test.cpp`
- [ ] Test command formatting
- [ ] Test multi-channel writes
- [ ] Test error handling

**Reference Files:**
- Pattern: `driver/task/common/write_task.h` (Sink interface)

---

### 1.6 Scan Task Implementation

**Files to Create:**
- [ ] `/driver/visa/scan_task.h` - Scanner implementation
- [ ] `/driver/visa/scan_task.cpp`

**ScanTask:**
- [ ] Constructor: Store context, task, device manager
- [ ] `exec(task::Command&)` override
  - Handle "scan" command
  - Handle "test_connection" command (for UI validation)
- [ ] `stop(bool)` override
- [ ] `name()` override
- [ ] Private `scan()` method:
  - Use `viFindRsrc()` with pattern "?*::INSTR"
  - Iterate with `viFindNext()`
  - For each resource:
    - Attempt `viOpen()` and `viClose()`
    - Send `*IDN?` query to get instrument info
    - Report discovered devices via state channel

**Testing:**
- [ ] `/driver/visa/scan_task_test.cpp`
- [ ] Test resource enumeration
- [ ] Test connection validation

**Reference Files:**
- Pattern: `driver/modbus/scan_task.h` (Scanner structure)

---

### 1.7 Factory & Registration

**Files to Create:**
- [ ] `/driver/visa/factory.cpp` - Factory implementation

**Factory Class:**
- [ ] Constructor: Initialize `std::shared_ptr<device::Manager> devices`
- [ ] `configure_task()` override:
  - Check if `task.type.find("visa") == 0`
  - Route to `configure_read()`, `configure_write()`, or `configure_scan()`
  - Use `common::handle_config_err()` for error handling
- [ ] `configure_initial_tasks()` override:
  - Use `common::configure_initial_factory_tasks()` to create scanner
- [ ] `name()` override: return "visa"

**Helper Functions:**
- [ ] `configure_read()` - Parse config, acquire device, create ReadTask
- [ ] `configure_write()` - Parse config, acquire device, create WriteTask
- [ ] `configure_scan()` - Create ScanTask

**Registration in Rack:**
- [ ] Modify `/driver/rack/rack.h:57` - Add `visa::INTEGRATION_NAME` to defaults
- [ ] Modify `/driver/rack/factories.cpp`:
  - Add `#include "driver/visa/visa.h"`
  - Add `configure_visa()` function
  - Call in `new_factory()`

**Reference Files:**
- Pattern: `driver/modbus/factory.cpp:71` (Factory implementation)
- Pattern: `driver/rack/factories.cpp:36` (Integration registration)

---

### 1.8 Testing Strategy

**Unit Tests:**
- [ ] Device session management (mock VISA calls)
- [ ] Response parsing (all formats)
- [ ] Read task configuration parsing
- [ ] Write task command formatting
- [ ] Error handling (timeout, connection loss, invalid responses)

**Integration Tests:**
- [ ] Full read/write cycle with simulated instrument
- [ ] Scanner discovery
- [ ] Session pooling (multiple tasks sharing device)
- [ ] Pipeline integration (acquisition/control)

**Hardware Tests (Manual):**
- [ ] Test with real instruments:
  - [ ] Keysight DMM (TCPIP)
  - [ ] Oscilloscope (USB)
  - [ ] Power supply (GPIB - if available)
  - [ ] Serial instrument (ASRL)

**Performance Tests:**
- [ ] High-frequency sampling (1kHz+)
- [ ] Large binary block transfers (oscilloscope waveforms)
- [ ] Multi-instrument concurrent access

---

## Phase 2: Console Frontend

### 2.1 Device Configuration

**Files to Create:**
- [ ] `/console/src/hardware/visa/device/types.ts`
- [ ] `/console/src/hardware/visa/device/Connect.tsx`
- [ ] `/console/src/hardware/visa/device/Configure.tsx`
- [ ] `/console/src/hardware/visa/device/external.ts`
- [ ] `/console/src/hardware/visa/device/BUILD.bazel` (if needed)

**types.ts:**
- [ ] Import Zod, migrate, device from client
- [ ] Define `connectionConfigZ` schema
  - `resourceName: z.string().min(1)`
  - `timeoutMs: z.number().int().positive().default(5000)`
  - `termChar: z.string().optional()`
  - `termCharEnabled: z.boolean().default(true)`
- [ ] Define `propertiesV0Z` schema
  - `connection: connectionConfigZ`
  - `read: { index: number, channels: Record<string, number> }`
  - `write: { channels: Record<string, number> }`
- [ ] Define `VISA_TYPE = "visa"`
- [ ] Define make/model as literals ("Generic", "VISA")
- [ ] Export `Device` type

**Connect.tsx:**
- [ ] Use `Device.createForm<Properties, Make, Model>()`
- [ ] Form fields:
  - Name (text)
  - Location (rack selector)
  - Resource Name (text with examples placeholder)
  - Timeout (numeric)
  - Term Char Enabled (toggle)
  - Term Char (text, conditional)
- [ ] "Test Connection" button
  - Uses `useMutation` to call scanner's `test_connection` command
  - Shows loading/success/error status
  - 10s timeout
- [ ] Device identifier field

**Configure.tsx:**
- [ ] Use `Common.Device.Configure` wrapper
- [ ] Two-step wizard (name → identifier)

**Reference Files:**
- Pattern: `console/src/hardware/modbus/device/Connect.tsx` (Connection form)
- Pattern: `console/src/hardware/modbus/device/types.ts` (Zod schemas)

---

### 2.2 Task Configuration Types

**Files to Create:**
- [ ] `/console/src/hardware/visa/task/types.ts`

**Schemas:**
- [ ] `responseFormatZ` - enum of response formats
- [ ] `inputChannelZ` - Input channel schema
  - `key: z.string()`
  - `channel: channel.keyZ`
  - `enabled: z.boolean().default(true)`
  - `scpiCommand: z.string().min(1)`
  - `format: responseFormatZ`
  - `dataType: z.string()`
  - `delimiter: z.string().default(",")`
  - `arrayLength: z.number().int().default(0)`
- [ ] `outputChannelZ` - Output channel schema
  - `key: z.string()`
  - `channel: channel.keyZ`
  - `scpiCommand: z.string().min(1)` (command template)
- [ ] `readConfigZ` - Extends `Common.Task.baseReadConfigZ`
  - `channels: z.array(inputChannelZ)`
  - `sampleRate: z.number().positive().max(10000)`
  - `streamRate: z.number().positive().max(10000)`
  - Add `.check(Common.Task.validateStreamRate)`
  - Add `.check(Common.Task.validateChannels)`
- [ ] `writeConfigZ` - Extends `Common.Task.baseConfigZ`
  - `channels: z.array(outputChannelZ)`
- [ ] `scanConfigZ` - Minimal config
  - `rackKey: z.number()`

**Constants:**
- [ ] `READ_TYPE = "visa_read"`
- [ ] `WRITE_TYPE = "visa_write"`
- [ ] `SCAN_TYPE = "visa_scan"`

**Helpers:**
- [ ] `readMapKey(ch: InputChannel)` - Generate unique key for channel map
- [ ] `writeMapKey(ch: OutputChannel)` - Same for output
- [ ] `channelName(device, channel)` - Generate Synnax channel name from SCPI

**Reference Files:**
- Pattern: `console/src/hardware/modbus/task/types.ts` (Task schemas)
- Pattern: `console/src/hardware/common/task/types.ts` (Base configs)

---

### 2.3 Read Task Configuration UI

**Files to Create:**
- [ ] `/console/src/hardware/visa/task/Read.tsx`

**Components:**

**ChannelListItem:**
- [ ] SCPI command text input
- [ ] Response format dropdown (Float, Integer, String, Float Array, etc.)
- [ ] Data type selector (conditional on format)
- [ ] Delimiter input (conditional - arrays only)
- [ ] Array length input (conditional - arrays only)
- [ ] Synnax channel name display
- [ ] Enable/disable button

**ChannelList:**
- [ ] Use `Common.Task.ChannelList` wrapper
- [ ] Pass custom `ChannelListItem`
- [ ] Empty state message

**Form:**
- [ ] Sample rate field
- [ ] Stream rate field
- [ ] Data saving toggle
- [ ] Channels list
- [ ] "Add Channel" button

**onConfigure callback:**
- [ ] Retrieve device from Synnax
- [ ] Check/create index channel
- [ ] Identify new channels to create
- [ ] Create channels in Synnax
- [ ] Update device properties with channel keys
- [ ] Map channel keys back to config
- [ ] Return `[config, rackKey]`

**getInitialValues:**
- [ ] Set defaults for new task
- [ ] Load existing task config

**Export:**
- [ ] Use `Common.Task.wrapForm()` to create renderer
- [ ] Export as `Read` component

**Reference Files:**
- Pattern: `console/src/hardware/modbus/task/Read.tsx` (Read task UI)
- Pattern: `console/src/hardware/ni/task/AnalogRead.tsx` (Complex channel list)

---

### 2.4 Write Task Configuration UI

**Files to Create:**
- [ ] `/console/src/hardware/visa/task/Write.tsx`

**Components:**

**ChannelListItem:**
- [ ] SCPI command template input (e.g., "SOUR:VOLT:DC {value}")
- [ ] Synnax channel selector (control source)
- [ ] Preview of formatted command

**Form:**
- [ ] Channels list
- [ ] "Add Channel" button

**onConfigure:**
- [ ] Similar to read, but for output channels
- [ ] Validate command templates contain `{value}` placeholder

**Reference Files:**
- Pattern: `console/src/hardware/modbus/task/Write.tsx`

---

### 2.5 Scan Task UI

**Files to Create:**
- [ ] `/console/src/hardware/visa/task/Scan.tsx`

**Components:**
- [ ] Auto-start toggle
- [ ] "Scan Now" button
- [ ] Results display (discovered instruments)
- [ ] Status indicator

**Reference Files:**
- Pattern: `console/src/hardware/modbus/task/Scan.tsx`

---

### 2.6 Registration & Export

**Files to Create:**
- [ ] `/console/src/hardware/visa/task/external.ts`
- [ ] `/console/src/hardware/visa/device/external.ts`
- [ ] `/console/src/hardware/visa/external.ts`
- [ ] `/console/src/hardware/visa/palette.tsx`

**task/external.ts:**
- [ ] Export `LAYOUTS` mapping task types to components
- [ ] Export `SELECTABLES` for command palette
- [ ] Export `EXTRACTORS` for task data extraction

**device/external.ts:**
- [ ] Export device types and forms

**external.ts:**
- [ ] Re-export all task and device exports

**palette.tsx:**
- [ ] Define palette commands for creating tasks
- [ ] Icons and labels

**Top-level Registration:**
- [ ] Modify `/console/src/hardware/task/external.ts`:
  - Import VISA layouts
  - Add to `LAYOUTS` object
  - Add to `SELECTABLES` array
- [ ] Modify `/console/src/hardware/device/external.ts`:
  - Import VISA device forms
  - Add to device registry

**Reference Files:**
- Pattern: `console/src/hardware/modbus/task/external.ts`
- Pattern: `console/src/hardware/task/external.ts` (Top-level registration)

---

### 2.7 UI/UX Polish

**Enhancements:**
- [ ] Add SCPI command autocomplete/suggestions
- [ ] Add instrument library (common commands for popular instruments)
- [ ] Add resource name validator with examples
- [ ] Add tooltip help text for response formats
- [ ] Add SCPI syntax highlighting in text inputs
- [ ] Add command preview/test feature
- [ ] Add "Import from CSV" for bulk channel setup

**Documentation:**
- [ ] Add user guide for VISA driver
- [ ] Add SCPI command reference
- [ ] Add troubleshooting guide (common errors)

---

## Phase 3: Documentation & Examples

### 3.1 User Documentation

**Files to Create:**
- [ ] `/docs/drivers/visa/overview.md` - High-level overview
- [ ] `/docs/drivers/visa/getting-started.md` - Quick start guide
- [ ] `/docs/drivers/visa/configuration.md` - Configuration reference
- [ ] `/docs/drivers/visa/scpi-commands.md` - SCPI command guide
- [ ] `/docs/drivers/visa/troubleshooting.md` - Common issues

**Content:**
- [ ] Supported instruments and interfaces
- [ ] Resource string syntax examples
- [ ] Response format guide with examples
- [ ] Step-by-step setup tutorials
- [ ] Example configurations for popular instruments

---

### 3.2 Example Configurations

**Files to Create:**
- [ ] `/examples/visa/keysight-34461a-dmm.json` - Keysight DMM config
- [ ] `/examples/visa/tektronix-mso64-scope.json` - Oscilloscope config
- [ ] `/examples/visa/rs-nrp-power-meter.json` - Power meter config
- [ ] `/examples/visa/gpib-legacy-instrument.json` - GPIB example

**Each example should include:**
- Device configuration (connection details)
- Read task (measurement channels)
- Write task (control channels, if applicable)
- Comments explaining SCPI commands

---

### 3.3 Developer Documentation

**Files to Create:**
- [ ] `/driver/visa/README.md` - C++ driver architecture
- [ ] `/console/src/hardware/visa/README.md` - UI component guide

**Content:**
- [ ] Architecture diagram
- [ ] Adding new response formats
- [ ] Testing guidelines
- [ ] Contributing guide

---

## Phase 4: Testing & Validation

### 4.1 Automated Testing

**C++ Tests:**
- [ ] Run all unit tests: `bazel test //driver/visa/...`
- [ ] Verify test coverage >80%
- [ ] Add tests for edge cases:
  - Invalid SCPI syntax
  - Timeout handling
  - Malformed responses
  - Session cleanup on errors

**TypeScript Tests:**
- [ ] Add Vitest tests for:
  - Schema validation (Zod)
  - Form state management
  - Channel configuration logic
  - onConfigure callback

**Integration Tests:**
- [ ] Add to `/integration/` directory
- [ ] Test full read/write cycle
- [ ] Test multi-task scenarios
- [ ] Test scanner discovery

---

### 4.2 Manual Testing

**Hardware Validation:**
- [ ] Set up test instruments
- [ ] Test TCPIP connection
- [ ] Test USB connection
- [ ] Test GPIB connection (if available)
- [ ] Test Serial connection

**Functional Tests:**
- [ ] Create device from Console
- [ ] Configure read task with multiple channels
- [ ] Start/stop acquisition
- [ ] Verify data in Synnax
- [ ] Test write task (if instrument supports)
- [ ] Test scanner
- [ ] Test connection error handling
- [ ] Test parse error handling

**Performance Tests:**
- [ ] 1 Hz sampling - verify timing accuracy
- [ ] 10 Hz sampling
- [ ] 100 Hz sampling
- [ ] 1 kHz sampling (if supported by instrument)
- [ ] Large binary block transfer (oscilloscope waveform)

**Stress Tests:**
- [ ] Run for 24 hours continuously
- [ ] Multiple concurrent tasks on same device
- [ ] Rapid start/stop cycles
- [ ] Connection loss recovery

---

### 4.3 Beta Testing

**Internal Testing:**
- [ ] Deploy to internal test cluster
- [ ] Set up common instruments (DMM, scope, power supply)
- [ ] Run for 1 week
- [ ] Collect feedback

**External Beta:**
- [ ] Select 3-5 beta users
- [ ] Provide documentation and support
- [ ] Collect usage data and feedback
- [ ] Iterate based on feedback

---

## Phase 5: Release & Maintenance

### 5.1 Release Preparation

**Code Review:**
- [ ] Full code review by team
- [ ] Address all TODOs and FIXMEs
- [ ] Ensure consistent code style
- [ ] Verify all tests passing

**Documentation Review:**
- [ ] User docs complete and accurate
- [ ] Developer docs complete
- [ ] Examples tested and working
- [ ] Changelog updated

**Platform Testing:**
- [ ] Windows build and test
- [ ] Linux build and test
- [ ] macOS build and test
- [ ] NI Linux RT build and test (if supported)

---

### 5.2 Release

**Version:**
- [ ] Determine version number (e.g., v0.47.0)
- [ ] Tag release in git
- [ ] Update release notes

**Deployment:**
- [ ] Merge to main branch
- [ ] Build release artifacts
- [ ] Update documentation site
- [ ] Announce release (blog post, Discord, etc.)

---

### 5.3 Post-Release

**Monitoring:**
- [ ] Monitor error reports
- [ ] Track usage metrics
- [ ] Collect user feedback

**Maintenance:**
- [ ] Address critical bugs within 48 hours
- [ ] Triage feature requests
- [ ] Plan future improvements

---

## Open Questions & Decisions

### Technical Decisions

**Q1: Should we support async operations (viReadAsync, etc.)?**
- **Decision:** No, start with synchronous only for simplicity. Add async later if needed.

**Q2: Should we support VISA events (SRQ, trigger, etc.)?**
- **Decision:** Not in v1. Add in future version if users request.

**Q3: Should we cache instrument identification (*IDN?)?**
- **Decision:** Yes, cache in device properties to avoid repeated queries.

**Q4: What's the maximum supported sample rate?**
- **Decision:** 10 kHz hard limit in UI, but actual limit depends on instrument and network.

**Q5: Should we support raw binary transfers (not just IEEE 488.2 blocks)?**
- **Decision:** Not in v1. Binary block format covers most use cases.

### UI/UX Decisions

**Q6: Should we provide a SCPI command builder/wizard?**
- **Decision:** Not in v1. Add instrument library with preset commands first.

**Q7: Should we support importing/exporting task configurations?**
- **Decision:** Yes, JSON export/import for easy sharing.

**Q8: Should we validate SCPI commands in real-time?**
- **Decision:** Basic syntax validation only (regex). Can't validate semantics without instrument.

### Platform Support

**Q9: Should we support NI Linux RT?**
- **Decision:** Yes, but conditionally compile (like Modbus). Requires NI-VISA RT.

**Q10: Should we support other VISA implementations (Keysight, R&S, Tektronix)?**
- **Decision:** Yes, VISA is a standard. Should work with all compliant implementations.

---

## Dependencies & Blockers

### External Dependencies

- [ ] NI-VISA installed on development machines
- [ ] Access to test instruments for validation
- [ ] VISA documentation and specs

### Internal Dependencies

- [ ] No blocking dependencies on other Synnax features
- [ ] Requires current driver architecture (already in place)

### Resources

- [ ] Estimated developer time: 2-3 weeks for C++ driver
- [ ] Estimated developer time: 1-2 weeks for Console UI
- [ ] Testing time: 1 week
- [ ] Total: 4-6 weeks

---

## Success Metrics

### Functional Metrics

- [ ] Can connect to all major instrument types (GPIB, USB, TCPIP, Serial)
- [ ] Can read/write at 1 kHz sustained
- [ ] Zero memory leaks (valgrind clean)
- [ ] Zero goroutine leaks
- [ ] Test coverage >80%

### User Metrics

- [ ] <5 min setup time for new instrument
- [ ] <10 steps to configure basic read task
- [ ] Clear error messages for common issues
- [ ] Positive feedback from beta testers

### Performance Metrics

- [ ] Latency: <10ms per query (TCPIP)
- [ ] Throughput: 1000+ samples/sec (small payloads)
- [ ] Binary block: 10 MB/sec transfer rate
- [ ] Memory: <50 MB per task
- [ ] CPU: <5% per task (1 Hz sampling)

---

## Risk Assessment

### High Risk

1. **VISA library compatibility issues**
   - Different vendors may have subtle API differences
   - **Mitigation:** Test with multiple VISA implementations early

2. **Response parsing complexity**
   - SCPI responses are highly variable
   - **Mitigation:** Start with common formats, add edge cases iteratively

### Medium Risk

3. **Performance with high-frequency sampling**
   - Network latency may limit sample rates
   - **Mitigation:** Document realistic limits, optimize query batching

4. **Session management bugs**
   - Shared sessions could cause conflicts
   - **Mitigation:** Thorough testing of connection pool, mutex protection

### Low Risk

5. **UI complexity overwhelming users**
   - Too many configuration options
   - **Mitigation:** Progressive disclosure, good defaults, examples

---

## Timeline (Tentative)

| Phase | Duration | Start | End |
|-------|----------|-------|-----|
| Phase 1: C++ Driver | 2-3 weeks | TBD | TBD |
| Phase 2: Console UI | 1-2 weeks | TBD | TBD |
| Phase 3: Documentation | 3-5 days | TBD | TBD |
| Phase 4: Testing | 1 week | TBD | TBD |
| Phase 5: Release | 2-3 days | TBD | TBD |
| **Total** | **4-6 weeks** | TBD | TBD |

---

## Next Steps

1. **Review this plan** with team - get feedback and alignment
2. **Set up development environment** - install NI-VISA, test with instrument
3. **Create initial file structure** - stub out all files with TODOs
4. **Start with device session management** - foundational piece
5. **Implement read task** - core functionality
6. **Build UI** - enable testing and iteration
7. **Test, iterate, polish** - refine based on real usage

---

## Notes & Updates

_Use this section to track important decisions, changes to the plan, and learnings as
implementation progresses._

**2025-01-XX:** Plan created. Ready to start implementation.
