#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
"""
This example sends a Sparkplug B command to the mock MQTT plant each time a value is
written to a Synnax channel.

Before running this example:
1. Start the mock MQTT plant:
   uv run python examples/mqtt_sim/server.py

2. Connect the broker in Synnax:
   uv run python examples/mqtt_sim/connect_broker.py

3. Run this script:
   uv run python examples/mqtt_sim/sparkplug_write_task.py

Each command sets the tag setpoint of the device Pump1 of the edge node Plant/Line1.
"""

import synnax as sy
from synnax import mqtt

client = sy.Synnax()

dev = client.devices.retrieve(name="MQTT Broker")

cmd_time = client.channels.create(
    name="sparkplug_setpoint_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)
cmd = client.channels.create(
    name="sparkplug_setpoint_cmd",
    index=cmd_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

tsk = mqtt.WriteTask(
    name="MQTT Py - Sparkplug Write Task",
    device=dev.key,
    targets=[
        mqtt.SparkplugWriteTarget(
            group="Plant",
            edge_node="Line1",
            device="Pump1",
            tag="setpoint",
            channel=cmd.key,
            # The data type that the edge node declares for the tag.
            sparkplug_type="double",
        ),
    ],
)

client.tasks.configure(tsk)
print("Task configured. Sending a setpoint each second. Ctrl+C to stop.")

try:
    with tsk.run():
        with client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[cmd_time.key, cmd.key],
        ) as writer:
            setpoint = 0.0
            while True:
                writer.write({cmd_time.key: sy.TimeStamp.now(), cmd.key: setpoint})
                print(f"setpoint={setpoint:.0f}", end="\r", flush=True)
                setpoint = (setpoint + 100) % 2000
                sy.sleep(1)
except KeyboardInterrupt:
    print("\nStopped.")
