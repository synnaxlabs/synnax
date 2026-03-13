# HTTP Driver Examples

This directory contains example scripts for working with HTTP/REST devices in Synnax.

## Prerequisites

1. **Software**:
   - Synnax server running
   - Synnax driver running
   - For testing: Mock HTTP server (included in `examples/http/server.py`)
2. **Authentication**: Logged in to Synnax CLI (`uv run sy login`)

## Quick Start Guide

**Important**: All commands in this guide should be run from the `client/py` directory.

Follow these scripts in order:

### 1. Start the Mock Server (Optional)

If you don't have a real HTTP endpoint, start the included mock server:

```bash
uv run python examples/http/server.py
```

This server simulates:

- **GET /health** — Simple health check returning `{"status": "ok"}`
- **GET /api/v1/data** — Sensor data (temperature, pressure, humidity) with sine waves
- **GET /api/v1/device** — Device state including writable fields
- **PUT /api/v1/setpoint** — Accept setpoint values
- **POST /api/v1/control** — Accept control commands (power, mode)
- **PATCH /api/v1/config** — Partial config updates

The server runs on `http://127.0.0.1:8081` by default. Use `--https` for TLS and
`--auth <type>` for authentication testing.

### 2. Connect Your HTTP Server

Register your HTTP server with Synnax:

```bash
uv run python examples/http/connect_server.py
```

This script will:

- Check if the server is already registered
- Register the server with the embedded Synnax rack
- Configure the health check endpoint

**Configuration**: Edit the constants at the top of `connect_server.py`:

- `DEVICE_NAME`: A friendly name for your HTTP server
- `HOST`: Host and port of the server (e.g., `127.0.0.1:8081`)

### 3. Read Data from HTTP Endpoints

Poll an HTTP endpoint for sensor data:

```bash
uv run python examples/http/read_task.py
```

This example:

- Creates channels for temperature, pressure, and humidity
- Configures a read task polling GET `/api/v1/data` at 1 Hz
- Uses JSON Pointers (`/temperature`, `/pressure`, `/humidity`) to extract fields
- Displays live values
- Press Ctrl+C to stop

### 4. Write Commands via HTTP

Send commands to an HTTP endpoint:

```bash
uv run python examples/http/write_task.py
```

This example:

- Creates a command channel for setpoint control
- Configures a write task sending PUT `/api/v1/setpoint`
- Writes setpoint values (20.0, 25.0, ..., 65.0) at 1 Hz for 10 seconds

## Device Configuration

### Connection

The HTTP device is configured with a host (stored as `device.location`) and a `secure`
flag that determines HTTP vs HTTPS:

```python
device = sy.http.Device(
    host="192.168.1.100:8080",
    secure=False,           # Use HTTP (True for HTTPS)
    timeout_ms=5000,        # Request timeout in milliseconds
    verify_ssl=True,        # Verify SSL certificates
)
```

### Authentication

Four authentication types are supported:

```python
# No authentication (default)
auth = {"type": "none"}

# Bearer token
auth = {"type": "bearer", "token": "my-token"}

# Basic auth
auth = {"type": "basic", "username": "admin", "password": "secret"}

# API key in header
auth = {"type": "api_key", "send_as": "header", "header": "X-API-Key", "key": "abc123"}

# API key in query parameter
auth = {"type": "api_key", "send_as": "query_param", "parameter": "api_key", "key": "abc123"}
```

### Health Check

Configure how the driver monitors device health:

```python
health_check = sy.http.HealthCheck(
    method="GET",
    path="/health",
    validate_response=True,
    response=sy.http.ExpectedResponse(
        pointer="/status",
        expected_value_type="string",
        expected_value="ok",
    ),
)
```

## Read Task Fields

### JSON Pointers

Fields use [JSON Pointer](https://datatracker.ietf.org/doc/html/rfc6901) syntax to
extract values from the response body:

```python
sy.http.ReadField(pointer="/temperature", channel=temp_ch.key)
sy.http.ReadField(pointer="/sensors/sensor_0", channel=sensor_ch.key)
```

### Timestamp Fields

For hardware-timed data, configure a timestamp field and set it as the endpoint's index:

```python
ts_field = sy.http.ReadField(
    pointer="/timestamp",
    channel=time_ch.key,
    data_type="timestamp",
    timestamp_format="unix_sec",  # iso8601, unix_sec, unix_ms, unix_us, unix_ns
)
endpoint = sy.http.ReadEndpoint(
    path="/api/data",
    index=ts_field.key,  # Use this field for timing
    fields=[ts_field, data_field],
)
```

### Enum Fields

Map string values to numbers:

```python
sy.http.ReadField(
    pointer="/status",
    channel=status_ch.key,
    enum_values={"OFF": 0, "ON": 1, "ERROR": 2},
)
```

## Write Task Fields

### Channel Field

Each write endpoint has one channel field that provides the dynamic value:

```python
sy.http.ChannelField(
    pointer="/value",       # Where to place the value in the request body
    json_type="number",     # JSON type: "number", "string", "boolean"
    channel=cmd_ch.key,
)
```

### Static Fields

Add fixed values to the request body:

```python
sy.http.StaticField(pointer="/source", json_type="string", value="python-client")
sy.http.StaticField(pointer="/priority", json_type="number", value=1)
```

### Generated Fields

Add auto-generated values (UUIDs or timestamps):

```python
sy.http.GeneratedField(pointer="/id", generator="uuid")
sy.http.GeneratedField(pointer="/ts", generator="timestamp", time_format="iso8601")
```

## Troubleshooting

### "Server not found"

- Ensure the HTTP server is running and accessible
- Check network connectivity: `curl http://<HOST>/health`
- Verify the correct host and port

### "Failed to connect server"

- Verify the Synnax driver is running
- Check that the host is correct
- For HTTPS: ensure `secure=True` and the certificate is valid (or set
  `verify_ssl=False`)

### "Task configuration failed"

- Ensure the device was connected successfully first
- Verify the endpoint paths are correct
- Check that JSON Pointers match the response structure
- Ensure channel data types are compatible

### Health check shows device as unhealthy

- Verify the health check path returns a 2xx status code
- If using response validation, confirm the response matches the expected value
- Check authentication configuration

## Additional Resources

- [JSON Pointer Specification (RFC 6901)](https://datatracker.ietf.org/doc/html/rfc6901)
- [Synnax HTTP Driver Documentation](https://docs.synnaxlabs.com/reference/driver/http/)

## Next Steps

After running these examples, you can:

- Create custom read/write tasks in the Synnax Console
- Build real-time dashboards with your HTTP endpoint data
- Integrate HTTP control into automated sequences
- Combine with other drivers (Modbus, OPC UA, etc.) for multi-protocol systems
