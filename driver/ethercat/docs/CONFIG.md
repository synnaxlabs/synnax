# EtherCAT Driver Configuration Specification

This document defines the configuration structure for Synnax's EtherCAT driver,
including device models, task configurations, and channel mappings.

## Device Model

The EtherCAT driver uses a two-tier device hierarchy that mirrors EtherCAT's physical
architecture:

```
Rack
└── EtherCAT Network (device)
    ├── Slave Device (child device)
    ├── Slave Device (child device)
    └── Slave Device (child device)
```

### EtherCAT Network Device

Represents a single EtherCAT network interface. Analogous to an NI chassis - provides
timing and connectivity for child devices.

**Device Properties:**

```json
{
  "interface": "eth0",
  "rate": 1000
}
```

| Property    | Type   | Required | Description                                              |
| ----------- | ------ | -------- | -------------------------------------------------------- |
| `interface` | string | Yes      | Network interface name (e.g., "eth0", "enp2s0")          |
| `rate`      | float  | Yes      | Network cycle rate in Hz (e.g., 1000 = 1kHz, 100 = 10ms) |

**Relationships:**

- Parent: Rack
- Children: Slave devices (via Synnax ontology)

### Slave Device

Represents a single EtherCAT slave on the network. Identified by serial number for
portability across networks and positions.

**Device Properties:**

```json
{
  "serial": 12345678,
  "vendor_id": 3736871159,
  "product_code": 252,
  "revision": 1,
  "name": "32xDO Digital Output Module",
  "position": 2,
  "pdos": {
    "inputs": [
      {
        "name": "status_word",
        "index": 24576,
        "subindex": 1,
        "bit_length": 16,
        "data_type": "uint16"
      },
      {
        "name": "supply_voltage",
        "index": 24576,
        "subindex": 2,
        "bit_length": 32,
        "data_type": "float32"
      }
    ],
    "outputs": [
      {
        "name": "digital_outputs",
        "index": 28672,
        "subindex": 1,
        "bit_length": 32,
        "data_type": "uint32"
      }
    ]
  }
}
```

| Property       | Type   | Required | Description                                   |
| -------------- | ------ | -------- | --------------------------------------------- |
| `serial`       | uint32 | Yes      | Unique serial number from device EEPROM       |
| `vendor_id`    | uint32 | Yes      | EtherCAT vendor ID (assigned by ETG)          |
| `product_code` | uint32 | Yes      | Product code identifying device model         |
| `revision`     | uint32 | Yes      | Hardware/firmware revision                    |
| `name`         | string | Yes      | Human-readable device name from EEPROM        |
| `position`     | uint16 | Yes      | Current position on bus (0-based, may change) |
| `pdos`         | object | Yes      | Discovered PDO mappings (see below)           |

**PDO Entry Structure:**

| Field        | Type   | Description                                |
| ------------ | ------ | ------------------------------------------ |
| `name`       | string | PDO name (from discovery or ESI)           |
| `index`      | uint16 | CoE object dictionary index (e.g., 0x6000) |
| `subindex`   | uint8  | CoE object dictionary subindex             |
| `bit_length` | uint8  | Data size in bits                          |
| `data_type`  | string | Data type (see Data Type Mapping)          |

**Relationships:**

- Parent: EtherCAT Network device (via Synnax ontology)

**Device Status:**

- Online/offline state
- AL (Application Layer) state: INIT, PRE-OP, SAFE-OP, OP
- Error information (e.g., "Failed to transition to PRE-OP: AL status 38")

## Task Configuration

### Read Task

Reads input PDOs (TxPDO, slave→master) from one or more slaves on a network.

**Task Type:** `ethercat_read`

**Configuration:**

```json
{
  "device": "ethercat_network_device_key",
  "sample_rate": 100,
  "stream_rate": 20,
  "data_saving": true,
  "channels": [
    {
      "device": "slave_device_key",
      "type": "automatic",
      "pdo": "status_word",
      "channel": 12345,
      "enabled": true
    },
    {
      "device": "slave_device_key",
      "type": "manual",
      "index": 24576,
      "subindex": 2,
      "bit_length": 32,
      "data_type": "float32",
      "channel": 12346,
      "enabled": true
    }
  ]
}
```

**Task-Level Properties:**

| Property      | Type   | Required | Description                                       |
| ------------- | ------ | -------- | ------------------------------------------------- |
| `device`      | string | Yes      | Key of the EtherCAT Network device                |
| `sample_rate` | float  | Yes      | Samples per second (must be ≤ network cycle rate) |
| `stream_rate` | float  | Yes      | Frames per second sent to Synnax                  |
| `data_saving` | bool   | No       | Whether to persist data (default: true)           |
| `channels`    | array  | Yes      | Channel configurations (see below)                |

### Write Task

Writes output PDOs (RxPDO, master→slave) to one or more slaves on a network.

**Task Type:** `ethercat_write`

**Configuration:**

```json
{
  "device": "ethercat_network_device_key",
  "state_rate": 10,
  "data_saving": true,
  "channels": [
    {
      "device": "slave_device_key",
      "type": "automatic",
      "pdo": "digital_outputs",
      "channel": 12350,
      "state_channel": 12351,
      "enabled": true
    }
  ]
}
```

**Task-Level Properties:**

| Property      | Type   | Required | Description                             |
| ------------- | ------ | -------- | --------------------------------------- |
| `device`      | string | Yes      | Key of the EtherCAT Network device      |
| `state_rate`  | float  | Yes      | State feedback rate in Hz               |
| `data_saving` | bool   | No       | Whether to persist data (default: true) |
| `channels`    | array  | Yes      | Channel configurations (see below)      |

### Scan Task

Persistent background task that discovers and monitors EtherCAT networks and slaves.

**Task Type:** `ethercat_scan`

**Behavior:**

- One scan task per rack (created automatically)
- Runs continuously at low frequency
- Discovers all network interfaces with EtherCAT capability
- Creates/updates Network devices for each interface
- Creates/updates Slave devices as children
- Updates device status for all devices
- Detects slave movement (by serial number) and updates relationships

**Scan Modes:**

- Basic discovery: Enumerate slaves, read identification (always works if powered)
- Deep discovery: Transition to PRE-OP, read PDO mappings (best-effort per slave)

Failed slaves are still created as devices with basic info and error status.

## Channel Configuration

Channels map EtherCAT PDOs to Synnax channels. Two configuration types are supported.

### Automatic Channel (Discovery-Based)

References a PDO discovered during scan. PDO details (index, subindex, bit_length,
data_type) are looked up from the slave device properties.

```json
{
  "device": "slave_device_key",
  "type": "automatic",
  "pdo": "status_word",
  "channel": 12345,
  "enabled": true
}
```

| Property  | Type   | Required | Description                                |
| --------- | ------ | -------- | ------------------------------------------ |
| `device`  | string | Yes      | Key of the slave device                    |
| `type`    | string | Yes      | Must be `"automatic"`                      |
| `pdo`     | string | Yes      | PDO name (must exist in device properties) |
| `channel` | uint32 | Yes      | Synnax channel key                         |
| `enabled` | bool   | No       | Whether channel is active (default: true)  |

**For write tasks, additional property:**

| Property        | Type   | Required | Description                       |
| --------------- | ------ | -------- | --------------------------------- |
| `state_channel` | uint32 | No       | Synnax channel for state feedback |

### Manual Channel (User-Defined)

Defines PDO parameters inline. Use when:

- PDO wasn't discovered (scan failed or partial)
- User knows address from documentation
- PDO mapping differs from discovery

```json
{
  "device": "slave_device_key",
  "type": "manual",
  "index": 24576,
  "subindex": 1,
  "bit_length": 16,
  "data_type": "int16",
  "channel": 12345,
  "enabled": true
}
```

| Property     | Type   | Required | Description                               |
| ------------ | ------ | -------- | ----------------------------------------- |
| `device`     | string | Yes      | Key of the slave device                   |
| `type`       | string | Yes      | Must be `"manual"`                        |
| `index`      | uint16 | Yes      | CoE object dictionary index               |
| `subindex`   | uint8  | Yes      | CoE object dictionary subindex            |
| `bit_length` | uint8  | Yes      | Data size in bits                         |
| `data_type`  | string | Yes      | Data type (see Data Type Mapping)         |
| `channel`    | uint32 | Yes      | Synnax channel key                        |
| `enabled`    | bool   | No       | Whether channel is active (default: true) |

**For write tasks, additional property:**

| Property        | Type   | Required | Description                       |
| --------------- | ------ | -------- | --------------------------------- |
| `state_channel` | uint32 | No       | Synnax channel for state feedback |

## Data Type Mapping

| EtherCAT/PDO Type | Synnax Type | Notes                     |
| ----------------- | ----------- | ------------------------- |
| BOOL (1-bit)      | UINT8       | Single bit stored as byte |
| INT8              | INT8        |                           |
| UINT8             | UINT8       |                           |
| INT16             | INT16       |                           |
| UINT16            | UINT16      |                           |
| INT32             | INT32       |                           |
| UINT32            | UINT32      |                           |
| INT64             | INT64       |                           |
| UINT64            | UINT64      |                           |
| FLOAT32           | FLOAT32     |                           |
| FLOAT64           | FLOAT64     |                           |

**Data Type Strings:** `"bool"`, `"int8"`, `"uint8"`, `"int16"`, `"uint16"`, `"int32"`,
`"uint32"`, `"int64"`, `"uint64"`, `"float32"`, `"float64"`

## Synnax Channel Auto-Creation

When configuring tasks in Console, Synnax channels can be auto-created for selected
PDOs.

**Naming Convention:**

```
{user_prefix}_{pdo_name}
```

Example:

- User prefix: `test_stand_do`
- PDO name: `status_word`
- Result: `test_stand_do_status_word`

**Rules:**

- Underscores only for separation (no dots, dashes, spaces)
- PDO name is sanitized (lowercase, spaces→underscores)
- Channel data type matches PDO data type (per mapping table)

## Timing Model

### Network Cycle Rate

The `rate` on the Network device defines how often EtherCAT frames traverse the network
in Hz. All slaves on the network share this cycle rate.

### Task Sample Rate

The `sample_rate` on Read tasks defines how many samples per second are captured. Must
satisfy:

- `sample_rate ≤ rate` (cannot sample faster than network cycles)
- `rate % sample_rate == 0` (clean decimation factor)
- `sample_rate % stream_rate == 0` (integer samples per batch)

Example:

- Network rate: 100 Hz
- Task sample_rate: 50 → reads every 2nd cycle (decimation factor = 2)
- Task sample_rate: 100 → reads every cycle (decimation factor = 1)
- Task sample_rate: 200 → ERROR (exceeds network rate)

### Timestamps

Timestamps are calculated from cycle time with clock skew correction:

- First sample timestamped with wall clock
- Subsequent samples: `timestamp = first + (n × cycle_period)`
- Clock skew correction applied periodically to prevent drift

## Slave Identity and Movement

### Identity

Slaves are identified by **serial number** (unique per physical device), not by bus
position. This enables:

- Configuration portability when slaves move
- Tracking individual hardware units
- Replacement detection

### Automatic Movement Detection

The background scan task detects when slaves move:

1. Scan discovers slave by serial number
2. If serial found on different network or position, relationships updated
3. Channels stay associated with slave device
4. Driver resolves serial → current position at runtime

### Replacement Workflow

When a slave is replaced with an identical model (different serial):

1. Old device shows offline in device status
2. New device created for new serial
3. User reassigns channel configuration to new device

## Error Handling

### Scan Failures

If a slave fails during deep scan (e.g., can't transition to PRE-OP):

- Device is still created
- Basic identification info stored (serial, vendor, product, name, position)
- PDOs list may be empty or partial
- Error details in device status (e.g., "AL status 38: Invalid SM configuration")

### Runtime Failures

- Working counter mismatch → communication error reported
- Slave state change → device status updated
- Cycle overrun → logged, task continues

## Example: Complete Configuration

### Network Device

```json
{
  "key": "dev_ecat_network_001",
  "name": "Test Stand EtherCAT",
  "make": "EtherCAT",
  "model": "Network",
  "properties": {
    "interface": "eth0",
    "rate": 1000
  }
}
```

### Slave Device (child of network)

```json
{
  "key": "dev_ecat_slave_001",
  "name": "32xDO Module SN:1234",
  "make": "DEWESoft",
  "model": "32xDO",
  "properties": {
    "serial": 1234,
    "vendor_id": 3736871159,
    "product_code": 2,
    "revision": 1,
    "name": "32xDO",
    "position": 2,
    "pdos": {
      "inputs": [
        {
          "name": "status_word",
          "index": 24576,
          "subindex": 1,
          "bit_length": 16,
          "data_type": "uint16"
        },
        {
          "name": "supply_voltage",
          "index": 24576,
          "subindex": 2,
          "bit_length": 32,
          "data_type": "float32"
        }
      ],
      "outputs": [
        {
          "name": "digital_outputs",
          "index": 28672,
          "subindex": 1,
          "bit_length": 32,
          "data_type": "uint32"
        }
      ]
    }
  }
}
```

### Read Task

```json
{
  "type": "ethercat_read",
  "name": "Test Stand Inputs",
  "config": {
    "device": "dev_ecat_network_001",
    "sample_rate": 100,
    "stream_rate": 20,
    "data_saving": true,
    "channels": [
      {
        "device": "dev_ecat_slave_001",
        "type": "automatic",
        "pdo": "status_word",
        "channel": 100001,
        "enabled": true
      },
      {
        "device": "dev_ecat_slave_001",
        "type": "automatic",
        "pdo": "supply_voltage",
        "channel": 100002,
        "enabled": true
      }
    ]
  }
}
```

### Write Task

```json
{
  "type": "ethercat_write",
  "name": "Test Stand Outputs",
  "config": {
    "device": "dev_ecat_network_001",
    "state_rate": 10,
    "data_saving": true,
    "channels": [
      {
        "device": "dev_ecat_slave_001",
        "type": "automatic",
        "pdo": "digital_outputs",
        "channel": 100010,
        "state_channel": 100011,
        "enabled": true
      }
    ]
  }
}
```

## Future Considerations

Reserved for future implementation:

- **Scale/Offset transforms**: Linear scaling of raw values
- **ESI file support**: Parse vendor XML for richer PDO metadata
- **Distributed clocks**: Sub-microsecond synchronization across slaves
- **Watchdog configuration**: Customize timeout and retry behavior
- **Slave filtering**: Exclude problematic slaves by vendor/product
