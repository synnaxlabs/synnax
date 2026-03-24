#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to write commands to a mock HTTP server using
the HTTP driver.

Before running this example:
1. Start the mock HTTP server:
   uv run python examples/http_sim/server.py

2. Connect the HTTP device in Synnax:
   uv run python examples/http_sim/connect_server.py

3. Run this script:
   uv run python examples/http_sim/write_task.py

The mock server accepts PUT /api/v1/setpoint with body:
    {"value": 42.0}
"""

import time

import synnax as sy
from synnax import http

client = sy.Synnax()

# Retrieve the HTTP device
dev = client.devices.retrieve(name="HTTP Server")

# Create an index channel for command timestamps
http_cmd_time = client.channels.create(
    name="http_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create a command channel for setpoint control
setpoint_cmd = client.channels.create(
    name="http_setpoint_cmd",
    index=http_cmd_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

# Create the HTTP Write Task
tsk = http.WriteTask(
    name="HTTP Py - Write Task",
    device=dev.key,
    endpoints=[
        http.WriteEndpoint(
            method="PUT",
            path="/api/v1/setpoint",
            channel=http.ChannelField(
                pointer="/value",
                json_type="number",
                channel=setpoint_cmd.key,
                name="Setpoint",
                data_type="float64",
            ),
        ),
    ],
)

# Configure the task with Synnax
client.tasks.configure(tsk)
print("✓ Task configured successfully")

print("=" * 70)
print("Starting HTTP Write Task")
print("=" * 70)
print("Sending PUT /api/v1/setpoint commands...")
print("Writing 10 setpoint values at 1 Hz (10 seconds)\n")

print(f"{'Cycle':<8} {'Setpoint':>12}")
print("-" * 70)

with tsk.run():
    with client.open_writer(
        start=sy.TimeStamp.now(),
        channels=[http_cmd_time.key, setpoint_cmd.key],
        enable_auto_commit=True,
    ) as writer:
        for i in range(10):
            setpoint_val = 20.0 + i * 5.0
            print(f"{i + 1:<8} {setpoint_val:>12.1f}")
            writer.write(
                {
                    http_cmd_time.key: sy.TimeStamp.now(),
                    setpoint_cmd.key: setpoint_val,
                }
            )
            writer.commit()
            time.sleep(1)

print("-" * 70)
print("✓ Write task completed successfully!")
print("\nCommands sent:")
print("- Setpoint values: 20.0, 25.0, 30.0, ..., 65.0")
print("=" * 70)
