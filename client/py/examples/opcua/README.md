# OPC UA Examples

This directory contains example scripts for working with OPC UA servers in Synnax.

## Prerequisites

1. **Hardware/Software**: An OPC UA server accessible over the network
2. **Software**:
   - Synnax server running
   - Synnax driver running
   - For testing: OPC UA test server (included in `/driver/opc/dev/server_extended.py`)
3. **Authentication**: Logged in to Synnax CLI (`poetry run sy login`)

## Quick Start Guide

Follow these scripts in order:

### 1. Start the Test Server (Optional)

If you don't have a real OPC UA server, start the included test server:

```bash
poetry run python driver/opc/dev/server_extended.py
```

This server simulates:
- **Float variables** (my_float_0, my_float_1): Sine wave data
- **Array variables** (my_array_0, my_array_1): Arrays of sine wave values (5 elements each)
- **Boolean variables** (my_bool_0, my_bool_1): Square wave patterns
- **Command variables** (command_0, command_1, command_2): Writable float values

The server runs on `opc.tcp://localhost:4841/` by default and prints node IDs on startup.

### 2. Connect Your OPC UA Server

Register your OPC UA server with Synnax:

```bash
poetry run python examples/opcua/connect_opc_server.py
```

This script will:
- Check if the server is already registered
- Register the server with the embedded Synnax rack
- Set up the server configuration

**Configuration**: Edit the constants at the top of `connect_opc_server.py` to match your server:
- `DEVICE_NAME`: A friendly name for your OPC UA server
- `ENDPOINT`: OPC UA endpoint URL (e.g., `opc.tcp://localhost:4841/`)

### 3. Read Float Data from OPC UA Nodes

Read scalar float values from the server:

```bash
poetry run python examples/opcua/read_task.py
```

This example:
- Creates channels for timestamps and two float variables (my_float_0, my_float_1)
- Configures a read task sampling at 10 Hz, streaming at 10 Hz
- Displays live values from the OPC UA nodes
- Press Ctrl+C to stop

**What you'll see**: Real-time sine wave values from my_float_0 and my_float_1.

**Node IDs**: The example uses node IDs like `NS=2;I=8` to identify OPC UA variables. These IDs are printed by `server_extended.py` on startup.

### 4. Read Array Data from OPC UA Nodes

Read array data in high-performance array mode:

```bash
poetry run python examples/opcua/read_task_array.py
```

This example:
- Creates channels for two array variables (my_array_0, my_array_1)
- Configures a read task in **array mode** with `array_size=5`
- Each sample contains an entire array of 5 float values
- Displays live array values
- Press Ctrl+C to stop

**What you'll see**: Arrays of 5 sine wave values updated at 10 Hz.

**Array mode**: More efficient for high-rate data when the OPC UA server provides data in array format with consistent size.

### 5. Read Boolean Data from OPC UA Nodes

Read boolean (digital) values from the server:

```bash
poetry run python examples/opcua/read_task_boolean.py
```

This example:
- Creates channels for two boolean variables (my_bool_0, my_bool_1)
- Configures a read task sampling at 10 Hz
- Displays live True/False states
- Press Ctrl+C to stop

**What you'll see**: Boolean values toggling in a square wave pattern.

**Note**: Boolean data is stored as UINT8 in Synnax (0 for False, 1 for True).

### 6. Write Commands to OPC UA Nodes

Send commands to writable OPC UA nodes:

```bash
poetry run python examples/opcua/write_task.py
```

This example:
- Controls three command variables (command_0, command_1, command_2)
- Writes float values at 1 Hz for 10 seconds
- Tracks command channels

**What you'll see**: Commands sent to the OPC UA server. Check the server terminal to verify values were received.

**Important**: Update the node IDs in `write_task.py` with the actual values printed by your server.

### 7. Delete Server (Cleanup)

When finished, remove the server registration:

```bash
poetry run python examples/opcua/delete_opc_server.py
```

This will remove the server and all associated tasks from Synnax.

## OPC UA Node Identification

### Node IDs

OPC UA uses Node IDs to identify variables. Common formats:

- **Numeric**: `NS=2;I=8` (namespace 2, integer identifier 8)
- **String**: `NS=2;S=MyVariable` (namespace 2, string identifier)
- **GUID**: `NS=2;G=<uuid>` (namespace 2, GUID identifier)
- **Opaque**: `NS=2;B=<base64>` (namespace 2, byte string)

**Important**: Node IDs can change between server restarts. Use browsing/discovery to find current IDs.

### Node Classes

- **Variable**: Data values (most common for I/O)
- **Object**: Organizational structure
- **Method**: Callable functions
- **ObjectType**: Type definitions
- **VariableType**: Variable type definitions

## Channel Types

### Read Channels

- **ReadChannel**: Generic read channel for any OPC UA variable
  - `channel`: Synnax channel key
  - `node_id`: OPC UA node identifier (e.g., `"NS=2;I=8"`)
  - `data_type`: Synnax data type (e.g., `"float32"`, `"uint16"`, `"bool"`)
  - Use for any readable OPC UA variable

### Write Channels

- **WriteChannel**: Generic write channel for any writable OPC UA variable
  - `cmd_channel`: Synnax command channel key
  - `node_id`: OPC UA node identifier
  - Use for any writable OPC UA variable

## Data Types

OPC UA supports many data types. Common mappings to Synnax:

| OPC UA Type | Synnax Type | Notes |
|-------------|-------------|-------|
| Float | `float32` | 32-bit floating point |
| Double | `float64` | 64-bit floating point |
| Int16 | `int16` | 16-bit signed integer |
| Int32 | `int32` | 32-bit signed integer |
| Int64 | `int64` | 64-bit signed integer |
| UInt16 | `uint16` | 16-bit unsigned integer |
| UInt32 | `uint32` | 32-bit unsigned integer |
| UInt64 | `uint64` | 64-bit unsigned integer |
| Boolean | `uint8` | Stored as 0 or 1 |
| Byte | `uint8` | 8-bit unsigned integer |

## Array Mode vs. Unary Mode

### Unary Mode (Default)

- Reads scalar (single) values from OPC UA nodes
- Each sample is one value per channel
- Best for: Individual sensors, setpoints, status values
- Example: `ReadTask(sample_rate=10*sy.Rate.HZ, ...)`

### Array Mode

- Reads entire arrays from OPC UA nodes
- Each sample contains multiple values (fixed array size)
- Best for: High-speed data acquisition, waveforms, batched readings
- Example: `ReadTask(sample_rate=10*sy.Rate.HZ, array_mode=True, array_size=5, ...)`

**Performance**: Array mode is more efficient for high-rate data when the server provides arrays.

## Security

OPC UA supports various security policies:

- **None**: No encryption (insecure, for testing only)
- **Basic128Rsa15**: Deprecated, avoid using
- **Basic256**: Moderate security
- **Basic256Sha256**: Recommended for most applications
- **Aes128-Sha256-RsaOaep**: High security
- **Aes256-Sha256-RsaPss**: Highest security

**Note**: The current examples use `SecurityPolicy.None` for simplicity. For production deployments, configure security in `device_props()`.

## Sample Rates

- **Typical rates**: 1-100 Hz for most OPC UA servers
- **Fast servers**: Up to 1 kHz for high-performance servers
- **Slow servers**: 0.1-1 Hz for slow-updating values

**Stream Rate**: Can match sample rate or be lower. For example, 10 Hz sampling with 10 Hz streaming sends every sample immediately.

**Subscription vs. Polling**: OPC UA typically uses subscriptions with server-side sampling, which is more efficient than polling.

## Troubleshooting

### "Device not found"
- Ensure the OPC UA server is running
- Check network connectivity: `ping <hostname>`
- Verify the endpoint URL is correct
- Check that the server is listening on the correct port

### "Failed to connect device"
- Verify the Synnax driver is running
- Check the endpoint URL format: `opc.tcp://hostname:port/path`
- Ensure no firewall is blocking the OPC UA port (default: 4840 or 4841)
- Verify security policy compatibility

### "Task configuration failed"
- Ensure the server was connected successfully first
- Verify node IDs are correct (check server startup output)
- Check that data types match the node's actual type
- Ensure sample rates are within server limits

### Node IDs are invalid
- Node IDs can change between server restarts
- Run the server and check its startup output for current node IDs
- Use browsing tools (e.g., UAExpert, UaExplorer) to find node IDs
- Consider using string-based node IDs if available (more stable)

### Array reads fail
- Verify the node actually contains an array
- Check `array_size` matches the server's array length
- Ensure the node's data type supports arrays
- Some servers require specific subscription settings for arrays

### Write commands not working
- Verify the node is writable (check AccessLevel attribute)
- Ensure the command channel data type matches the node type
- Check server logs for write errors
- Verify security settings allow writes

### Connection timeouts
- Reduce sample rate (try 1 Hz first)
- Check network latency: `ping <hostname>`
- Verify the server can handle the requested subscription rate
- Some servers have connection limits - check concurrent sessions

## Additional Resources

- [OPC Foundation](https://opcfoundation.org/)
- [OPC UA Specification](https://reference.opcfoundation.org/)
- [Synnax OPC UA Driver Documentation](https://docs.synnaxlabs.com/reference/driver/opc-ua/)
- [open62541 Documentation](https://www.open62541.org/)

## Next Steps

After running these examples, you can:
- Create custom tasks in the Synnax Console
- Build real-time dashboards with your OPC UA data
- Integrate OPC UA control into automated sequences
- Export data for analysis in Python, MATLAB, or other tools
- Browse your OPC UA server's full namespace to discover more variables
