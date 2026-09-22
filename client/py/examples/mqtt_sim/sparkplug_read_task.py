#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
"""
This example reads Sparkplug B tags from the mock MQTT plant.

Before running this example:
1. Start the mock MQTT plant:
   uv run python examples/mqtt_sim/server.py

2. Connect the broker in Synnax:
   uv run python examples/mqtt_sim/connect_broker.py

3. Run this script:
   uv run python examples/mqtt_sim/sparkplug_read_task.py

The plant has the edge node Plant/Line1 with the tag temperature, and the device Pump1
with the tag speed.
"""

import synnax as sy
from synnax import mqtt

client = sy.Synnax()

dev = client.devices.retrieve(name="MQTT Broker")


def create_tag_channel(name: str, data_type: sy.DataType) -> sy.Channel:
    """Creates a channel on an index of its own, because each Sparkplug B tag carries
    its own timestamp."""
    index = client.channels.create(
        name=f"{name}_time",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
        retrieve_if_name_exists=True,
    )
    return client.channels.create(
        name=name,
        index=index.key,
        data_type=data_type,
        retrieve_if_name_exists=True,
    )


temperature = create_tag_channel("sparkplug_temperature", sy.DataType.FLOAT64)
speed = create_tag_channel("sparkplug_speed", sy.DataType.FLOAT32)

tsk = mqtt.ReadTask(
    name="MQTT Py - Sparkplug Read Task",
    device=dev.key,
    entries=[
        mqtt.SparkplugReadEntry(
            group="Plant",
            edge_node="Line1",
            tag="temperature",
            channel=temperature.key,
            index=temperature.index,
            data_type="float64",
        ),
        # A device ID selects a tag of a device of the edge node.
        mqtt.SparkplugReadEntry(
            group="Plant",
            edge_node="Line1",
            device="Pump1",
            tag="speed",
            channel=speed.key,
            index=speed.index,
            data_type="float32",
        ),
    ],
)

client.tasks.configure(tsk)
print("Task configured. Reading Plant/Line1. Press Ctrl+C to stop.")

latest = {"sparkplug_temperature": 0.0, "sparkplug_speed": 0.0}
try:
    with tsk.run():
        with client.open_streamer(list(latest)) as streamer:
            while True:
                frame = streamer.read()
                if frame is None:
                    continue
                for name in latest:
                    if name in frame:
                        latest[name] = frame[name].to_numpy()[-1]
                print(
                    f"temperature={latest['sparkplug_temperature']:.2f} "
                    f"speed={latest['sparkplug_speed']:.1f}",
                    end="\r",
                    flush=True,
                )
except KeyboardInterrupt:
    print("\nStopped.")
