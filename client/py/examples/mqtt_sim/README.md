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

5. Read Sparkplug B tags of the edge node `Plant/Line1`.

   ```bash
   uv run python examples/mqtt_sim/sparkplug_read_task.py
   ```

6. Send Sparkplug B commands for the tag `setpoint`.

   ```bash
   uv run python examples/mqtt_sim/sparkplug_write_task.py
   ```

7. Run a Sparkplug B edge node on the Core.

   ```bash
   uv run python examples/mqtt_sim/sparkplug_edge_node.py
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

The plant also mirrors every Sparkplug B edge node that connects to it:

| Topic                                      | Payload                                 |
| ------------------------------------------ | --------------------------------------- |
| `plant/sparkplug/<group>/<node>/<tag>`     | Retained JSON `{"value": ...}`          |
| `plant/sparkplug/<group>/<node>/<tag>/set` | A JSON value, sent as a command (NCMD) |

## Sparkplug B tags of the mock plant

The plant has one edge node, `Plant/Line1`. It answers a rebirth request, and a command
for a tag sets the value of that tag.

| Device  | Tag           | Data type |
| ------- | ------------- | --------- |
|         | `temperature` | Double    |
|         | `count`       | Int64     |
|         | `running`     | Boolean   |
| `Pump1` | `speed`       | Float     |
| `Pump1` | `setpoint`    | Double    |
