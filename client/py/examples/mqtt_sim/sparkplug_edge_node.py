#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
"""
This example runs a Sparkplug B edge node on the Core. It publishes two Synnax
channels as tags, and takes commands for one of them.

Before running this example:
1. Start the mock MQTT plant:
   uv run python examples/mqtt_sim/server.py

2. Connect the broker in Synnax:
   uv run python examples/mqtt_sim/connect_broker.py

3. Run this script:
   uv run python examples/mqtt_sim/sparkplug_edge_node.py

The plant mirrors the tags on plant/sparkplug/Synnax/Example/<tag>, and turns a value
published on that topic plus /set into a command for the tag.
"""

import math

import synnax as sy
from synnax import mqtt

client = sy.Synnax()

dev = client.devices.retrieve(name="MQTT Broker")

edge_time = client.channels.create(
    name="edge_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)
temperature = client.channels.create(
    name="edge_temperature",
    index=edge_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)
setpoint = client.channels.create(
    name="edge_setpoint",
    index=edge_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)
# A command for the setpoint tag is written to this channel.
setpoint_cmd_time = client.channels.create(
    name="edge_setpoint_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)
setpoint_cmd = client.channels.create(
    name="edge_setpoint_cmd",
    index=setpoint_cmd_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

tsk = mqtt.EdgeTask(
    name="MQTT Py - Sparkplug Edge Node",
    device=dev.key,
    group="Synnax",
    edge_node="Example",
    authority=200,
    tags=[
        mqtt.EdgeTag(
            name="temperature", channel=temperature.key, sparkplug_type="double"
        ),
        mqtt.EdgeTag(
            name="setpoint",
            channel=setpoint.key,
            sparkplug_type="double",
            command_channel=setpoint_cmd.key,
        ),
    ],
)

client.tasks.configure(tsk)
print("Task configured. Publishing Synnax/Example. Press Ctrl+C to stop.")

try:
    with tsk.run():
        with client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[edge_time.key, temperature.key, setpoint.key],
        ) as writer:
            with client.open_streamer([setpoint_cmd.key]) as streamer:
                t = 0.0
                current_setpoint = 50.0
                while True:
                    frame = streamer.read(timeout=0.5)
                    if frame is not None and setpoint_cmd.key in frame:
                        current_setpoint = frame[setpoint_cmd.key].to_numpy()[-1]
                    writer.write(
                        {
                            edge_time.key: sy.TimeStamp.now(),
                            temperature.key: 23.5 + 5 * math.sin(t),
                            setpoint.key: current_setpoint,
                        }
                    )
                    t += 0.5
                    print(
                        f"temperature={23.5 + 5 * math.sin(t):.2f} "
                        f"setpoint={current_setpoint:.1f}",
                        end="\r",
                        flush=True,
                    )
except KeyboardInterrupt:
    print("\nStopped.")
