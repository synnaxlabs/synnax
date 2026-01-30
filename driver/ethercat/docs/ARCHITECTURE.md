# EtherCAT Driver Architecture

This document describes the architectural considerations and design decisions for
Synnax's EtherCAT driver, informed by hands-on testing with DEWESoft IOLITE R8 hardware.

## EtherCAT Fundamentals

### How EtherCAT Differs from Other Protocols

Unlike Modbus or OPC UA which use request/response patterns, EtherCAT is **cyclic**:

- A single Ethernet frame travels through all slaves in a daisy chain
- Each slave reads/writes its portion of the frame as it passes through
- The master sends frames at a fixed cycle rate (typically 1-10ms)
- All I/O updates happen synchronously in each cycle

This means:

- **One master per network interface** - shared by all tasks
- **No per-device connections** - all slaves share the cyclic frame
- **Deterministic timing** - data updates at predictable intervals

### EtherCAT State Machine

Slaves transition through states:

```
INIT → PRE_OP → SAFE_OP → OPERATIONAL
```

- **INIT**: Basic communication, no mailbox
- **PRE_OP**: Mailbox (SDO) communication available, no process data
- **SAFE_OP**: Process data exchange, outputs forced to safe state
- **OPERATIONAL**: Full process data exchange, outputs active

Transitions can fail with AL (Application Layer) status codes indicating the reason.

### Process Data Objects (PDOs)

PDOs are the cyclic data exchanged each cycle:

- **TxPDO** (Transmit): Slave → Master (inputs from Synnax's perspective)
- **RxPDO** (Receive): Master → Slave (outputs from Synnax's perspective)

PDOs are identified by:

- Slave position on the bus
- Index (e.g., 0x6000 for inputs, 0x7000 for outputs)
- Subindex (specific data item within the index)
- Bit length

## Lessons from Hardware Testing

### Lesson 1: PDO Mapping is Runtime-Determined

**Problem**: We initially tried to calculate PDO offsets at registration time.

**Reality**: The EtherCAT master library (SOEM) determines the actual IOmap layout
during `ecx_config_map_group()`. The layout depends on:

- Which slaves are present on the bus
- Their physical order
- Their group assignments
- Their individual PDO configurations

**Example from IOLITE R8**:

```
Our calculated offsets:    Inputs @ 0, 4, 8, 12
Actual SOEM layout:        Inputs @ 122, 152, 182, 212
                          (after 18 bytes of outputs + 104 bytes from other slaves)
```

**Implication**: Cannot pre-calculate offsets. Must query actual slave data locations
after master activation.

### Lesson 2: Device-Specific Quirks Are Common

**Problem**: DEWESoft 6xSTG strain gauge modules failed to reach SAFE_OP with AL status
code 38 (Invalid SM OUT configuration).

**Likely causes**:

- Default PDO mapping doesn't match device expectations
- May need CoE SDO configuration before state transition
- Firmware version differences

**Workaround**: Exclude problematic slaves by assigning them to a different group.

**Implication**: Need mechanisms for:

- Device-specific configuration profiles
- Slave exclusion/filtering
- Graceful degradation when some slaves fail

### Lesson 3: Input Data Structure Varies by Device

**Problem**: We assumed digital output modules would have input data that mirrors their
output state.

**Reality**: The 32xDO modules have:

- 4 bytes of output data (the 32 digital outputs)
- 30 bytes of input data (status, diagnostics, supply voltage, etc.)

The input structure is device-specific and documented in ESI files or device manuals.

**Implication**: Channel configuration must explicitly specify which PDO to read/write.
Cannot assume input/output symmetry.

### Lesson 4: Working Counter Validates Communication

The Working Counter (WKC) indicates how many slaves processed the frame:

- Each slave increments WKC when it successfully reads/writes its data
- Expected WKC = (output slaves × 2) + input slaves
- WKC mismatch indicates communication failure or slave dropout

This is useful for detecting:

- Cable disconnections
- Slave failures
- Configuration mismatches

## Architecture Design

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Console/Config                            │
│  - Scan and discover slaves                                     │
│  - Map Synnax channels to PDO addresses                         │
│  - Configure slave inclusion/exclusion                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Driver Tasks                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  ScanTask    │  │  ReadTask    │  │  WriteTask   │          │
│  │  (discovery) │  │  (TxPDO)     │  │  (RxPDO)     │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│              CyclicEngine (one per network interface)            │
│  - Runs cyclic exchange thread at configured rate               │
│  - Reference counting: activates on first task, stops on last   │
│  - Thread-safe input snapshot for readers                       │
│  - Thread-safe output staging for writers                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Master Interface                            │
│  - Abstract interface for EtherCAT master operations            │
│  - Implementations: SOEMMaster, IgHMaster (future), MockMaster  │
└─────────────────────────────────────────────────────────────────┘
```

### Master Interface

The abstract Master interface allows swapping implementations:

```cpp
class Master {
    // Lifecycle
    virtual Error initialize() = 0;      // Scan bus, enumerate slaves
    virtual Error activate() = 0;        // Transition to OPERATIONAL
    virtual void deactivate() = 0;       // Return to INIT

    // Cyclic operations (called from cycle thread)
    virtual Error receive() = 0;         // Receive process data
    virtual Error send() = 0;            // Send process data

    // Slave information
    virtual std::vector<SlaveInfo> slaves() const = 0;
    virtual SlaveDataInfo slave_data(uint16_t position) const = 0;
};
```

Key design decision: **expose actual IOmap offsets after activation**, not before.

### CyclicEngine

Coordinates between the master and Synnax tasks:

```cpp
class CyclicEngine {
    // PDO registration (before activation)
    std::pair<size_t, Error> register_input_pdo(const PDOEntry& entry);
    std::pair<size_t, Error> register_output_pdo(const PDOEntry& entry);

    // Task lifecycle (reference counted)
    Error add_task();      // First call activates master
    void remove_task();    // Last call deactivates master

    // Data access (thread-safe)
    Error wait_for_inputs(std::vector<uint8_t>& buffer, std::atomic<bool>& breaker);
    void write_output(size_t offset, const void* data, size_t length);
};
```

**Critical insight**: The offsets returned by `register_*_pdo()` are for tracking
purposes only. After `add_task()` activates the master, the engine must resolve PDO
entries to actual IOmap offsets using `master->slave_data()`.

### Channel Configuration

Channels reference PDOs by address, not offset:

```json
{
  "name": "motor_drive_torque",
  "slave_position": 4,
  "pdo_index": "0x6000",
  "pdo_subindex": 1,
  "bit_offset": 0,
  "bit_length": 16,
  "data_type": "int16",
  "direction": "input",
  "scale": 0.01,
  "offset": 0
}
```

At activation time, the driver:

1. Looks up the slave at the specified position
2. Finds the PDO in the slave's mapped data
3. Calculates the actual byte offset in the IOmap
4. Uses that offset for cyclic reads/writes

### Slave Filtering

Allow configuration to handle problematic slaves:

```json
{
  "interface": "eth0",
  "cycle_time_ms": 10,
  "slaves": {
    "exclude": [{ "vendor": "0xDEBE50F7", "product": "0x000000FC" }],
    "optional": [{ "position": 1 }]
  }
}
```

- **exclude**: Don't attempt to activate these slaves (assign to group 1)
- **optional**: Don't fail if these slaves can't reach OPERATIONAL

## Implementation Status

| Component        | Status         | Notes                          |
| ---------------- | -------------- | ------------------------------ |
| Master interface | ✅ Complete    | Abstract interface defined     |
| SOEMMaster       | ⚠️ Partial     | Works, needs offset exposure   |
| MockMaster       | ✅ Complete    | For unit testing               |
| CyclicEngine     | ⚠️ Partial     | Needs actual offset resolution |
| ScanTask         | 🔲 Not started | Return slave info for config   |
| ReadTask         | 🔲 Not started | Needs channel→offset mapping   |
| WriteTask        | 🔲 Not started | Needs channel→offset mapping   |
| Factory          | 🔲 Not started | Task creation and routing      |

## Future Considerations

### ESI File Support

EtherCAT Slave Information (ESI) files are XML documents describing:

- Supported PDOs and their structure
- CoE object dictionary
- State machine requirements
- Timing parameters

Parsing ESI files would enable:

- Auto-discovery of available PDOs
- Validation of channel configurations
- Better error messages

### IgH EtherCAT Master

SOEM is suitable for most applications, but IgH EtherCAT Master offers:

- Kernel-space operation for lower latency
- Better real-time performance on Linux
- Distributed clocks support

The Master interface is designed to support both backends.

### Hot-Plug Support

Currently assumes static slave configuration. Future work:

- Detect slave connect/disconnect
- Graceful degradation when slaves drop out
- Re-initialization when slaves return

### Distributed Clocks

EtherCAT supports synchronized clocks across all slaves for:

- Coordinated motion control
- Synchronized sampling
- Sub-microsecond timing accuracy

Not currently implemented but the architecture allows for it.

## Testing

### Hardware Testing

Tested with DEWESoft IOLITE R8:

- 7 slaves: 1 gateway, 2 strain gauge modules, 4 digital output modules
- Successfully reached OPERATIONAL with 5/7 slaves
- Cyclic exchange at 10ms (100 Hz) verified
- Working counter validation confirmed

### Unit Testing

MockMaster enables testing without hardware:

- CyclicEngine lifecycle tests
- PDO registration tests
- Thread safety tests
- Error handling tests

## References

- [EtherCAT Technology Group](https://www.ethercat.org/)
- [SOEM (Simple Open EtherCAT Master)](https://github.com/OpenEtherCATsociety/SOEM)
- [IgH EtherCAT Master](https://etherlab.org/en/ethercat/)
- [ETG.1000 EtherCAT Specification](https://www.ethercat.org/en/downloads.html)
