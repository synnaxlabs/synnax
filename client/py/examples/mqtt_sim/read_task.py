#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example reads a JSON topic from the mock MQTT plant.

Before running this example:
1. Start the mock MQTT plant:
   uv run python examples/mqtt_sim/server.py

2. Connect the broker in Synnax:
   uv run python examples/mqtt_sim/connect_broker.py

3. Run this script:
   uv run python examples/mqtt_sim/read_task.py

The plant publishes this payload on plant/sensors:
    {"temperature": 23.5, "pressure": 101.3, "count": 7, "state": "running"}
"""

import synnax as sy
from synnax import mqtt

client = sy.Synnax()

dev = client.devices.retrieve(name="MQTT Broker")

mqtt_time = client.channels.create(
    name="mqtt_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)
temperature = client.channels.create(
    name="mqtt_temperature",
    index=mqtt_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)
pressure = client.channels.create(
    name="mqtt_pressure",
    index=mqtt_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)
# An enum field turns a string in the payload into a number.
running = client.channels.create(
    name="mqtt_running",
    index=mqtt_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

tsk = mqtt.ReadTask(
    name="MQTT Py - Read Task",
    device=dev.key,
    entries=[
        mqtt.PlainReadEntry(
            topic="plant/sensors",
            fields=[
                mqtt.ReadField(
                    pointer="/temperature",
                    channel=temperature.key,
                    data_type="float64",
                    name="Temperature",
                ),
                mqtt.ReadField(
                    pointer="/pressure",
                    channel=pressure.key,
                    data_type="float64",
                    name="Pressure",
                ),
                mqtt.ReadField(
                    pointer="/state",
                    channel=running.key,
                    data_type="uint8",
                    name="Running",
                    enum_values=[
                        mqtt.EnumEntry(label="idle", value=0),
                        mqtt.EnumEntry(label="running", value=1),
                    ],
                ),
            ],
        ),
    ],
)

client.tasks.configure(tsk)
print("Task configured. Reading plant/sensors. Press Ctrl+C to stop.")

try:
    with tsk.run():
        with client.open_streamer(
            ["mqtt_temperature", "mqtt_pressure", "mqtt_running"]
        ) as streamer:
            while True:
                frame = streamer.read()
                if frame is not None and "mqtt_temperature" in frame:
                    print(
                        f"temperature={frame['mqtt_temperature'][-1]:.2f} "
                        f"pressure={frame['mqtt_pressure'][-1]:.2f} "
                        f"running={frame['mqtt_running'][-1]}",
                        end="\r",
                        flush=True,
                    )
except KeyboardInterrupt:
    print("\nStopped.")
