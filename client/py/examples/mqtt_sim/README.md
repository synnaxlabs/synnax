# MQTT driver examples

This directory holds example scripts for the Synnax MQTT integration.

## Prerequisites

- A running Synnax Core. MQTT tasks run on the driver inside the Core, so no separate
  Driver is necessary.
- A broker. `server.py` is a mock plant with its own small broker. For production, use a
  broker such as Mosquitto, EMQX, or HiveMQ.
- A login for the Synnax CLI (`uv run sy login`).

Run all commands from the `client/py` directory.

## Quick start

1. Start the mock plant. It listens on `127.0.0.1:1884`.

   ```bash
   uv run python examples/mqtt_sim/server.py
   ```

2. Register the broker as a device.

   ```bash
   uv run python examples/mqtt_sim/connect_broker.py
   ```

3. Read the `plant/sensors` topic.

   ```bash
   uv run python examples/mqtt_sim/read_task.py
   ```

4. Publish commands to `plant/valve/set`.

   ```bash
   uv run python examples/mqtt_sim/write_task.py
   ```

## Topics of the mock plant

| Topic                | Payload                                               |
| -------------------- | ----------------------------------------------------- |
| `plant/sensors`      | `{"temperature", "pressure", "count", "state"}`       |
| `plant/timed`        | `{"value", "timestamp"}`, timestamp in nanoseconds    |
| `plant/scalar`       | A bare number                                         |
| `plant/info`         | A retained message, published once                    |
| `plant/<name>/set`   | A command. The plant echoes it, retained, on `/state` |
| `plant/<name>/state` | The last command of `<name>`                          |
