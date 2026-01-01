#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to configure and use a write task to send commands
to an OPC UA server.

Before running this example:
1. Start the OPC UA test server:
   poetry run python driver/opc/dev/server_extended.py

2. Connect the OPC UA server device in Synnax:
   See: https://docs.synnaxlabs.com/reference/driver/opc-ua/connect-server
   Use endpoint: opc.tcp://127.0.0.1:4841/

3. The server will print the node IDs for the command variables on startup.
   Update the node_id values below with the actual IDs from your server output.
"""

import time

import synnax as sy
from synnax import opcua

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the OPC UA server from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="OPC UA Server")

# Create an index channel for the command channels
opcua_cmd_time = client.channels.create(
    name="opcua_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create command channels that will be used to send control values to the OPC UA nodes.
# These are regular Synnax channels - the write task will listen to them and forward
# the values to the OPC UA server.
cmd_channel_0 = client.channels.create(
    name="opcua_cmd_0",
    index=opcua_cmd_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
cmd_channel_1 = client.channels.create(
    name="opcua_cmd_1",
    index=opcua_cmd_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
cmd_channel_2 = client.channels.create(
    name="opcua_cmd_2",
    index=opcua_cmd_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create and configure the OPC UA write task
# NOTE: Update these node IDs with the actual values from your server output!
# The server prints them on startup like: "command_0: ns=2;i=42"
tsk = opcua.WriteTask(
    name="OPC UA Write Task Example",
    device=dev.key,
    auto_start=False,  # We'll start it manually for this example
    channels=[
        # Map Synnax command channels to OPC UA node IDs
        # Replace these node_id values with the actual IDs from your server
        opcua.WriteChannel(cmd_channel=cmd_channel_0.key, node_id="NS=2;I=18"),
        opcua.WriteChannel(cmd_channel=cmd_channel_1.key, node_id="NS=2;I=19"),
        opcua.WriteChannel(cmd_channel=cmd_channel_2.key, node_id="NS=2;I=20"),
    ],
)

# Configure the task with Synnax
client.tasks.configure(tsk)

print("=" * 70)
print("Starting OPC UA Write Task")
print("=" * 70)
print("Sending commands to server_extended.py...")
print("Writing 10 commands at 1 Hz (10 seconds)")

print(f"{'Cycle':<8} {'command_0':>12} {'command_1':>12} {'command_2':>12}")
print("-" * 70)

# Start the write task
with tsk.run():
    # Open a writer to send commands to the OPC UA server
    # IMPORTANT: Must include the index channel (opcua_cmd_time) along with data channels
    with client.open_writer(
        start=sy.TimeStamp.now(),
        channels=[
            opcua_cmd_time.key,
            cmd_channel_0.key,
            cmd_channel_1.key,
            cmd_channel_2.key,
        ],
        enable_auto_commit=True,
    ) as writer:
        # Write some test values to the OPC UA nodes
        for i in range(10):
            val0 = float(i)
            val1 = float(i * 2)
            val2 = float(i * 3)

            print(f"{i+1:<8} {val0:>12.1f} {val1:>12.1f} {val2:>12.1f}")

            # Write timestamp to index channel along with command values
            writer.write(
                {
                    opcua_cmd_time.key: sy.TimeStamp.now(),
                    cmd_channel_0.key: val0,
                    cmd_channel_1.key: val1,
                    cmd_channel_2.key: val2,
                }
            )
            writer.commit()
            sy.sleep(1)

print("-" * 70)
print("✓ Write task completed successfully!")
print("Check your server terminal to verify the commands were received:")
print("You should see timestamps and values for each command variable")
