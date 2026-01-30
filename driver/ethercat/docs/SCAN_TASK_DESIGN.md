# EtherCAT Scan Task Design

This document specifies the design for the EtherCAT scan task, which discovers EtherCAT
networks and slaves, creates Synnax devices, and monitors their health.

## Table of Contents

1. [Overview](#overview)
2. [Device Model](#device-model)
3. [Device Keys](#device-keys)
4. [Identity and Matching](#identity-and-matching)
5. [Scanning Behavior](#scanning-behavior)
6. [Coordination with CyclicEngine](#coordination-with-cyclicengine)
7. [Status vs Properties](#status-vs-properties)
8. [Error Handling](#error-handling)
9. [Commands](#commands)
10. [Component Responsibilities](#component-responsibilities)
11. [Implementation Phases](#implementation-phases)

---

## Overview

The scan task automatically discovers EtherCAT networks and slaves, creating Synnax
devices that represent them. It runs continuously in the background, monitoring health
and detecting topology changes.

**Key principles:**

- Automatic network discovery (no manual interface configuration required)
- Global slave identity (slaves maintain identity across racks and interfaces)
- Non-intrusive scanning (doesn't interfere with active cyclic operations)
- Graceful degradation (partial failures don't block overall discovery)

---

## Device Model

### Network Device

Represents an Ethernet interface with EtherCAT slaves attached.

| Field      | Description                                        |
| ---------- | -------------------------------------------------- |
| `key`      | `ethercat_{interface}` (e.g., `ethercat_en7`)      |
| `name`     | Human-readable name (e.g., "EtherCAT Network en7") |
| `make`     | `"ethercat"`                                       |
| `model`    | `"network"`                                        |
| `location` | Interface name                                     |

**Properties (JSON):**

```json
{
  "interface": "en7",
  "mac_address": "00:1A:2B:3C:4D:5E",
  "slave_count": 5
}
```

**Status examples:**

| Condition                   | Variant | Message                                       |
| --------------------------- | ------- | --------------------------------------------- |
| All slaves operational      | SUCCESS | "Network operational (5 slaves)"              |
| Some slaves degraded        | WARNING | "Network degraded (3/5 slaves operational)"   |
| No slaves detected          | WARNING | "No slaves detected"                          |
| Interface permission denied | ERROR   | "Failed to open interface: permission denied" |

### Slave Device

Represents an individual EtherCAT slave on a network.

| Field      | Description                              |
| ---------- | ---------------------------------------- |
| `key`      | See [Device Keys](#device-keys)          |
| `name`     | Device name from EEPROM (e.g., "EL3004") |
| `make`     | `"ethercat"`                             |
| `model`    | Product code as string                   |
| `location` | Network interface                        |

**Properties (JSON):**

```json
{
  "vendor_id": 3739017463,
  "product_code": 252,
  "revision": 1,
  "serial": 12345,
  "name": "EL3004",
  "network": "en7",
  "position": 3,
  "pdo_discovery_complete": true,
  "input_pdos": [
    {
      "name": "Status",
      "index": 6144,
      "subindex": 1,
      "bit_length": 16,
      "data_type": "uint16"
    }
  ],
  "output_pdos": [
    {
      "name": "Control",
      "index": 7168,
      "subindex": 1,
      "bit_length": 16,
      "data_type": "uint16"
    }
  ]
}
```

**Status examples:**

| Condition                               | Variant | Message                                          |
| --------------------------------------- | ------- | ------------------------------------------------ |
| Slave operational                       | SUCCESS | "Slave operational"                              |
| Slave operational, PDO discovery failed | WARNING | "Operational, PDO discovery failed: SDO timeout" |
| Slave stuck in PRE-OP                   | WARNING | "Slave in PRE_OP: AL status 0x26"                |
| Slave disconnected                      | WARNING | "Slave disconnected"                             |
| Serial anomaly                          | WARNING | "Serial mismatch: vendor/product changed"        |
| Potential duplicate                     | WARNING | "Potential duplicate - review recommended"       |

---

## Device Keys

Device keys must be deterministic and stable to enable matching across scan cycles.

### Network Device Key

```
ethercat_{interface}
```

Example: `ethercat_en7`

### Slave Device Key

**With serial number (serial != 0):**

```
ethercat_{vendor_id}_{product_code}_{serial}
```

Example: `ethercat_3739017463_252_12345`

This key is globally stable - the same slave will have the same key regardless of which
network or position it's on.

**Without serial number (serial == 0):**

```
ethercat_{network}_{vendor_id}_{product_code}_{position}
```

Example: `ethercat_en7_3739017463_252_3`

This key is position-bound. If the slave moves to a different position, it will be
treated as a new device. This is unavoidable without a serial number.

---

## Identity and Matching

### Global Identity

Slaves with serial numbers maintain their identity globally across:

- Different positions on the same network
- Different networks on the same rack
- Different racks entirely

When a slave moves, the scan task detects it by serial number and updates the device's
`network` and `position` properties.

### Matching Rules

When a slave is discovered, the scan task determines whether it matches an existing
device:

| Condition                                                    | Action                                       |
| ------------------------------------------------------------ | -------------------------------------------- |
| Serial != 0, key exists in cluster                           | **Auto-match** - update properties           |
| Serial != 0, key does not exist                              | **Create new** device                        |
| Serial == 0, key exists in cluster                           | **Auto-match** - same position, same device  |
| Serial == 0, key does not exist                              | **Create new** device                        |
| Serial match but vendor/product mismatch                     | **Create new** + flag as anomaly             |
| Same vendor/product/network, different position, serial == 0 | **Create new** + flag as potential duplicate |

### User Deduplication

When the scan task cannot confidently match a device (serial == 0 cases), it creates a
new device and flags it for review. The Console provides UI for users to manually merge
duplicate devices.

### Reconnection Handling

When a previously disconnected slave reappears:

| Scenario                        | Behavior                                  |
| ------------------------------- | ----------------------------------------- |
| Same position                   | Auto-match, status → healthy              |
| Different position, serial != 0 | Auto-match by key, update position        |
| Different position, serial == 0 | Create new device, old stays disconnected |

---

## Scanning Behavior

### Automatic Network Discovery

On startup and each scan cycle, the scan task:

1. Enumerates all network interfaces via SOEM (`ec_find_adapters`)
2. For each interface not currently active (no CyclicEngine), probes for slaves
3. For active interfaces, queries cached slave info from CyclicEngine via Factory
4. Creates/updates network and slave devices as needed

### Scan Rate

- **Default:** 5 seconds between scan cycles
- Consistent with other driver scan tasks (Modbus, OPC UA, NI, LabJack)

### Scan Depth

**Basic discovery (INIT state):**

- Always performed
- Reads EEPROM/SII: vendor_id, product_code, revision, serial, name
- Fast (~100-500ms per network)

**Deep discovery (PRE-OP state):**

- Performed on first discovery of a slave
- Transitions slave to PRE-OP, reads PDO mappings via SDO (0x1C12, 0x1C13)
- Slower (~1-5s depending on slave count)
- May fail on some slaves (state transition issues)

**Retry logic:**

- Failed PDO discovery: retry every 10th scan cycle
- On-demand: `refresh_pdos` command forces deep discovery

### State Management

After probing:

- Return all slaves to INIT state (safest, minimal footprint)
- One retry on state transition failure, then continue
- Capture AL status code in device status for debugging

---

## Coordination with CyclicEngine

### Single Master Per Interface

EtherCAT is a single-master protocol. Only one master can control a network at a time.
The scan task and CyclicEngine cannot both independently access the same interface.

### Factory as Coordinator

The Factory manages shared resources and coordinates access:

```cpp
// ScanTask queries Factory
auto [slaves, is_active] = factory->get_network_info("en7");

if (is_active) {
    // Use cached slave info from CyclicEngine
    // No direct interface access
} else {
    // Probe interface directly via Scanner
}
```

### Active Network Behavior

When CyclicEngine is running cyclic I/O on a network:

- Scan task reads cached slave info only (no blocking)
- No automatic SDO reads (would disrupt timing)
- `refresh_pdos` command pauses cyclic, performs SDO reads, resumes
- Topology changes detected via working counter mismatch

### Lifecycle Transitions

**IDLE → ACTIVE:**

- User creates read/write task
- Factory creates CyclicEngine, takes over interface
- Scan task stops direct probing, uses cached info

**ACTIVE → IDLE:**

- User stops all tasks on network
- CyclicEngine shuts down
- Scan task resumes direct probing on next cycle

---

## Status vs Properties

### Properties (Stable)

Stored in `device.properties` JSON. Updated infrequently.

| Field                    | Description                 |
| ------------------------ | --------------------------- |
| `vendor_id`              | EtherCAT vendor ID          |
| `product_code`           | EtherCAT product code       |
| `revision`               | Hardware/firmware revision  |
| `serial`                 | Serial number (may be 0)    |
| `name`                   | Device name from EEPROM     |
| `network`                | Interface name (last known) |
| `position`               | Bus position (last known)   |
| `input_pdos`             | Array of input PDO info     |
| `output_pdos`            | Array of output PDO info    |
| `pdo_discovery_complete` | Boolean flag                |

### Status (Transient)

Updated every scan cycle. Reflects current state.

| Field               | Description                                  |
| ------------------- | -------------------------------------------- |
| `variant`           | SUCCESS, WARNING, or ERROR                   |
| `message`           | Human-readable status message                |
| `al_state`          | Current AL state (INIT, PRE_OP, SAFE_OP, OP) |
| `al_status_code`    | AL status code if error                      |
| `al_status_message` | Human-readable AL status                     |
| `time`              | Timestamp of last update                     |

**Key principle:** Properties store stable configuration. Status stores transient state.
Avoid updating properties for transient conditions.

---

## Error Handling

### Interface Probe Failures

| Condition                            | Behavior                                 |
| ------------------------------------ | ---------------------------------------- |
| Permission denied on ALL interfaces  | ERROR status: "Insufficient privileges"  |
| Permission denied on SOME interfaces | Report working, log warning about failed |
| No slaves found                      | Silently skip (no device created)        |
| Probe timeout                        | Skip, retry next cycle                   |

### Slave Discovery Failures

| Condition                  | Behavior                                           |
| -------------------------- | -------------------------------------------------- |
| State transition fails     | One retry, then continue with basic info           |
| SDO read timeout           | Log warning, mark `pdo_discovery_complete: false`  |
| Slave stuck in error state | Create device with WARNING status, include AL code |

### Partial Success

The scan task continues even when individual operations fail:

- One slave fails → other slaves still discovered
- One interface fails → other interfaces still probed
- PDO discovery fails → basic device info still captured

---

## Commands

| Command          | Description              | Parameters                                     |
| ---------------- | ------------------------ | ---------------------------------------------- |
| `scan`           | Force immediate scan     | None                                           |
| `stop`           | Pause scanning           | None                                           |
| `start`          | Resume scanning          | None                                           |
| `refresh_pdos`   | Re-run deep discovery    | `{slave_key: "..."}` or `{network_key: "..."}` |
| `test_interface` | Probe specific interface | `{interface: "en7"}`                           |
| `identify_slave` | Blink LED (best-effort)  | `{slave_key: "..."}`                           |

### refresh_pdos Behavior

- If `slave_key` provided: refresh single slave
- If `network_key` provided: refresh all slaves on network
- If network is active (CyclicEngine running): pauses cyclic, performs SDO reads,
  resumes

### identify_slave Behavior

- Best-effort implementation (requires slave hardware support)
- Uses CoE object for LED control if available
- Returns success/failure status

---

## Component Responsibilities

### Scanner (Abstract Interface)

_"I know how to talk to EtherCAT hardware"_

```cpp
namespace ethercat {

class Scanner {
public:
    virtual ~Scanner() = default;

    // Enumerate available network interfaces
    virtual std::vector<InterfaceInfo> enumerate_interfaces() = 0;

    // Probe interface for slaves (basic discovery)
    virtual std::pair<std::vector<SlaveInfo>, xerrors::Error>
    probe_interface(const std::string &interface) = 0;

    // Deep discovery for a slave (PDO mappings via SDO)
    virtual std::pair<std::vector<PDOInfo>, xerrors::Error>
    discover_pdos(const std::string &interface, uint16_t position) = 0;

    // For active networks: get cached slave info
    virtual std::pair<std::vector<SlaveInfo>, xerrors::Error>
    get_active_slaves(const std::string &interface) = 0;

    // Check if interface is currently active
    virtual bool is_interface_active(const std::string &interface) = 0;
};

} // namespace ethercat
```

**Responsibilities:**

- Enumerate network interfaces
- Open/close interfaces for probing
- Discover slaves (read SII/EEPROM)
- State transitions for deep discovery
- SDO reads for PDO discovery

**Not responsible for:**

- Device creation (ScanTask)
- Matching/deduplication (ScanTask)
- Status propagation (ScanTask)
- Cyclic I/O (CyclicEngine)

### ScanTask (Orchestrator)

_"I manage device lifecycle and talk to the cluster"_

**Responsibilities:**

- Call Scanner to discover hardware
- Track known devices (internal state map)
- Match scanned slaves to existing devices
- Create/update Synnax devices via ClusterAPI
- Propagate device statuses
- Handle commands

**Not responsible for:**

- Direct SOEM/IgH calls (Scanner)
- Cyclic I/O (CyclicEngine)
- Master lifecycle (Factory)

### CyclicEngine (Real-time I/O)

_"I run the cyclic loop and exchange process data"_

**Responsibilities:**

- Own Master instance for active network
- Run cycle thread
- Provide cached slave info for queries
- Manage input/output data exchange

**Not responsible for:**

- Initial discovery (Scanner)
- Device creation (ScanTask)
- Status propagation (ScanTask)

### Factory (Coordinator)

_"I create tasks and manage shared resources"_

**Responsibilities:**

- Create ScanTask (via `configure_initial_tasks`)
- Create/cache CyclicEngines per interface
- Answer "is interface active?" queries
- Route ScanTask to Scanner vs CyclicEngine

**Not responsible for:**

- Running scan logic (ScanTask)
- Running cyclic I/O (CyclicEngine)

---

## Implementation Phases

### Phase 1: MVP

- [ ] Abstract `Scanner` interface
- [ ] SOEM implementation (`SOEMScanner`)
- [ ] Basic discovery (INIT state only)
- [ ] Network device creation
- [ ] Slave device creation with basic properties
- [ ] Key-based matching
- [ ] Status propagation (network + slave)
- [ ] Coordination with Factory for active networks
- [ ] Basic commands: `scan`, `stop`, `start`

### Phase 2: Deep Discovery

- [ ] Deep discovery (PRE-OP, PDO mappings)
- [ ] `refresh_pdos` command
- [ ] `identify_slave` command (best-effort)
- [ ] `MockScanner` for unit testing

### Phase 3: Future

- [ ] IgH EtherCAT Master support
- [ ] User-driven deduplication UI (Console side)
- [ ] Hot-connect support
- [ ] Distributed clocks
