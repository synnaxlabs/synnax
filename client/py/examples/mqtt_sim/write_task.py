#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example publishes a command to the mock MQTT plant each time a value is written
to a Synnax channel.

Before running this example:
1. Start the mock MQTT plant:
   uv run python examples/mqtt_sim/server.py

2. Connect the broker in Synnax:
   uv run python examples/mqtt_sim/connect_broker.py

3. Run this script:
   uv run python examples/mqtt_sim/write_task.py

Each command becomes this payload on plant/valve/set:
    {"position": 42.0, "source": "synnax", "id": "<uuid>"}
"""

import synnax as sy
from synnax import mqtt

client = sy.Synnax()

dev = client.devices.retrieve(name="MQTT Broker")

cmd_time = client.channels.create(
    name="mqtt_valve_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)
cmd = client.channels.create(
    name="mqtt_valve_cmd",
    index=cmd_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

tsk = mqtt.WriteTask(
    name="MQTT Py - Write Task",
    device=dev.key,
    targets=[
        mqtt.PlainWriteTarget(
            topic="plant/valve/set",
            qos="at_least_once",
            channel=mqtt.ChannelField(
                pointer="/position",
                json_type="number",
                channel=cmd.key,
                data_type="float64",
            ),
            fields=[
                mqtt.StaticWriteField(
                    pointer="/source", json_type="string", value="synnax"
                ),
                mqtt.GeneratedWriteField(pointer="/id", generator="uuid"),
            ],
        ),
    ],
)

client.tasks.configure(tsk)
print("Task configured. Sending a valve position each second. Ctrl+C to stop.")

try:
    with tsk.run():
        with client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[cmd_time.key, cmd.key],
        ) as writer:
            position = 0.0
            while True:
                writer.write({cmd_time.key: sy.TimeStamp.now(), cmd.key: position})
                print(f"position={position:.0f}", end="\r", flush=True)
                position = (position + 10) % 100
                sy.sleep(1)
except KeyboardInterrupt:
    print("\nStopped.")
