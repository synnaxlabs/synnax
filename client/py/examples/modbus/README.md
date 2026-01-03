# Modbus TCP Examples

This directory contains example scripts for working with Modbus TCP devices in Synnax.

## Prerequisites

1. **Hardware**: A Modbus TCP server or device accessible over the network
2. **Software**:
   - Synnax server running
   - Synnax driver running
   - For testing: Modbus TCP test server (included in `examples/modbus/server.py`)
3. **Authentication**: Logged in to Synnax CLI (`uv run sy login`)

## Quick Start Guide

**Important**: All commands in this guide should be run from the `client/py` directory.

Follow these scripts in order:

### 1. Start the Test Server (Optional)

If you don't have a real Modbus device, start the included test server:

```bash
uv run python examples/modbus/server.py
```

This server simulates:

- **Input registers** (addresses 0-1): Sine wave data (0-255)
- **Holding registers** (addresses 2-3): Writable values
- **Coils** (addresses 0-1): Digital outputs (writable)
- **Discrete inputs** (addresses 0-1): Digital inputs (read-only)

The server runs on `localhost:5020` by default.

### 2. Connect Your Modbus Server

Register your Modbus TCP server with Synnax:

```bash
uv run python examples/modbus/connect_server.py
```

This script will:

- Check if the server is already registered
- Register the server with the embedded Synnax rack
- Set up the server configuration

**Configuration**: Edit the constants at the top of `connect_server.py` to match your
server:

- `DEVICE_NAME`: A friendly name for your Modbus server
- `HOST`: IP address or hostname of the Modbus server
- `PORT`: Modbus TCP port (typically 502 for production, 5020 for test server)

### 3. Read Data from Modbus Registers

Read data from input registers:

```bash
uv run python examples/modbus/read_task.py
```

This example:

- Creates channels for timestamps and two input registers (addresses 0, 1)
- Configures a read task sampling at 10 Hz, streaming at 10 Hz
- Displays live values from the registers
- Press Ctrl+C to stop

**What you'll see**: Real-time values from input registers 0 and 1 (sine wave data from
test server).

### 4. Write Commands to Modbus

Send commands to coils and holding registers:

```bash
uv run python examples/modbus/write_task.py
```

This example:

- Controls two coils (addresses 0-1) with alternating ON/OFF patterns
- Writes to two holding registers (addresses 2-3) with cycling values
- Runs for 10 seconds with 1 second intervals
- Tracks command channels

**What you'll see**: Commands sent to both digital (coils) and analog (holding
registers) outputs.

### 5. Test Connection (Diagnostic)

Verify your Modbus setup:

```bash
uv run python examples/modbus/check_connection.py
```

This diagnostic script checks:

- Device registration
- Task status
- Channel creation
- Configuration errors

### 6. Delete Server (Cleanup)

When finished, remove the server registration:

```bash
uv run python examples/modbus/delete_server.py
```

This will remove the server and all associated tasks from Synnax.

## Modbus Register Types

### Read-Only Registers

- **Input Registers** (16-bit): Read-only analog values
  - Function code: 0x04 (Read Input Registers)
  - Typical use: Sensor readings, measurements

- **Discrete Inputs** (1-bit): Read-only digital values
  - Function code: 0x02 (Read Discrete Inputs)
  - Typical use: Switch states, digital sensors

### Read-Write Registers

- **Holding Registers** (16-bit): Read-write analog values
  - Function code: 0x03 (Read), 0x06/0x10 (Write)
  - Typical use: Setpoints, configuration values, analog outputs

- **Coils** (1-bit): Read-write digital values
  - Function code: 0x01 (Read), 0x05/0x0F (Write)
  - Typical use: Digital outputs, relays, control signals

## Channel Types

### Read Channels

- **InputRegisterChan**: Input register (16-bit read-only)
  - Configurable data type (uint8, uint16, int16, uint32, int32, float32)
  - Address: Modbus register address
  - Use for read-only analog sensors

- **HoldingRegisterInputChan**: Holding register input (16-bit read-write)
  - Same data types as input registers
  - Address: Modbus register address
  - Use for reading setpoints or analog outputs

- **DiscreteInputChan**: Discrete input (1-bit read-only)
  - Data type: uint8 (0 or 1)
  - Address: Modbus coil/discrete input address
  - Use for digital sensor states

- **CoilInputChan**: Coil input (1-bit read-write)
  - Data type: uint8 (0 or 1)
  - Address: Modbus coil address
  - Use for reading digital output states

### Write Channels

- **HoldingRegisterOutputChan**: Holding register output (16-bit)
  - Configurable data type (uint8, uint16, int16, uint32, int32, float32)
  - Address: Modbus register address
  - Use for analog control signals

- **CoilOutputChan**: Coil output (1-bit)
  - Data type: uint8 (0 or 1)
  - Address: Modbus coil address
  - Use for digital control (relays, solenoids, etc.)

## Byte and Word Swapping

Modbus supports different byte orderings. Configure in `device_props`:

```python
modbus.device_props(
    host=HOST,
    port=PORT,
    swap_bytes=False,  # Swap bytes within 16-bit words
    swap_words=False,  # Swap word order for 32-bit values
)
```

Common configurations:

- **Big-endian**: `swap_bytes=False, swap_words=False`
- **Little-endian**: `swap_bytes=True, swap_words=True`
- **Mid-big-endian**: `swap_bytes=False, swap_words=True`
- **Mid-little-endian**: `swap_bytes=True, swap_words=False`

## Sample Rates

- **Typical rates**: 1-100 Hz for most Modbus devices
- **Fast devices**: Up to 1 kHz for high-speed Modbus
- **Slow devices**: 0.1-1 Hz for slow-updating sensors

**Stream Rate**: Can match sample rate or be lower to buffer samples. For example, 10 Hz
sampling with 10 Hz streaming sends every sample immediately.

## Troubleshooting

### "Server not found"

- Ensure the Modbus server/device is powered on and connected to the network
- Check network connectivity: `ping <HOST>`
- Verify the correct IP address and port
- Check firewall settings

### "Failed to connect server"

- Verify the Synnax driver is running
- Check that the host and port are correct
- Ensure no other software is using the same Modbus connection
- Verify network routing and firewall rules

### "Task configuration failed"

- Ensure the server was connected successfully first
- Verify register addresses match your device's Modbus map
- Check that data types are appropriate for the register size
- Ensure sample rates are within device limits

### Register values seem wrong

- Verify byte/word swapping configuration matches your device
- Check the Modbus address mapping (some devices use 0-based, others 1-based)
- Confirm data type matches the register format (uint16 vs int16 vs float32)
- Review device documentation for register scaling/units

### Connection timeouts

- Reduce sample rate (try 1 Hz first)
- Check network latency: `ping <HOST>`
- Verify the device supports the requested Modbus function codes
- Some devices have connection limits - check concurrent connections

## Additional Resources

- [Modbus Protocol Specification](https://modbus.org/specs.php)
- [Synnax Modbus Driver Documentation](https://docs.synnaxlabs.com/reference/driver/modbus/)
- [libmodbus Documentation](https://libmodbus.org/)

## Next Steps

After running these examples, you can:

- Create custom tasks in the Synnax Console
- Build real-time dashboards with your Modbus data
- Integrate Modbus control into automated sequences
- Export data for analysis in Python, MATLAB, or other tools
