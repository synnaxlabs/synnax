#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read boolean data from the test OPC UA server (server_extended.py).

Before running this example:
1. Start the test server:
   poetry run python driver/opc/dev/server_extended.py

2. Connect the OPC UA server device in Synnax:
   - Endpoint: opc.tcp://127.0.0.1:4841/
   - Name the device "OPC UA Server" (or update line 27 below)

3. The server creates boolean variables (my_bool_0, my_bool_1, etc.) that continuously
   update with sequential square wave patterns. This example reads those values.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Retrieve the OPC UA server from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="OPC UA Server")

# Create an index channel that will be used to store the timestamps for the data.
opcua_bool_time = client.channels.create(
    name="opcua_bool_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create Synnax channels to store the my_bool_0 and my_bool_1 node data from the server.
# These booleans are continuously updated with sequential square wave patterns by server_extended.py.
my_bool_0 = client.channels.create(
    name="my_bool_0",
    index=opcua_bool_time.key,
    data_type=sy.DataType.UINT8,  # Boolean data is stored as UINT8 (0 or 1)
    retrieve_if_name_exists=True,
)
my_bool_1 = client.channels.create(
    name="my_bool_1",
    index=opcua_bool_time.key,
    data_type=sy.DataType.UINT8,  # Boolean data is stored as UINT8 (0 or 1)
    retrieve_if_name_exists=True,
)

# Create the OPC UA Read Task
# Using node IDs to reference the OPC UA boolean variables.
# The server creates booleans with sequential square wave patterns offset by 0.2 seconds.
tsk = sy.opcua.ReadTask(
    name="OPC UA Py - Read Task (Boolean)",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 10,  # Sample at 10 Hz
    stream_rate=sy.Rate.HZ * 10,  # Stream at 10 Hz
    data_saving=True,
    channels=[
        # Bind the Synnax channels to the OPC UA node IDs
        # These IDs correspond to my_bool_0 and my_bool_1 in server_extended.py
        sy.opcua.ReadChannel(
            channel=my_bool_0.key, node_id="NS=2;I=13", data_type="bool"  # my_bool_0
        ),
        sy.opcua.ReadChannel(
            channel=my_bool_1.key, node_id="NS=2;I=14", data_type="bool"  # my_bool_1
        ),
    ],
)

# Configure the task with Synnax
client.tasks.configure(tsk)

print("=" * 70)
print("Starting OPC UA Boolean Read Task")
print("=" * 70)
print("Reading square wave boolean data from server_extended.py...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<8} {'Timestamp':<12} {'my_bool_0':>12} {'my_bool_1':>12}")
print("-" * 70)

# Start the task and read data continuously
try:
    # Hide cursor for clean output
    print("\033[?25l", end="", flush=True)

    with tsk.run():
        with client.open_streamer(["my_bool_0", "my_bool_1"]) as streamer:
            sample_count = 0
            start_time = sy.TimeStamp.now()

            while True:
                frame = streamer.read()
                if frame:
                    # Print the latest values from each channel
                    if "my_bool_0" in frame and len(frame["my_bool_0"]) > 0:
                        val0 = frame["my_bool_0"][-1]
                        val1 = frame["my_bool_1"][-1]

                        elapsed = sy.TimeStamp.now().span(start_time).seconds

                        sample_count += 1
                        # Convert uint8 to True/False for display
                        bool0 = "True " if val0 else "False"
                        bool1 = "True " if val1 else "False"
                        print(
                            f"{sample_count:<8} {elapsed:<12.1f} {bool0:>12} {bool1:>12}",
                            end="\r",
                            flush=True,
                        )

# Output summary
except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Boolean read task stopped by user")
    print(f"\nCollected {sample_count} samples")
    print("The values are sequential square waves:")
    print("- my_bool_0: toggles every second starting at t=0")
    print("- my_bool_1: toggles every second starting at t=0.2")
    print("=" * 70)
finally:
    # Ensure cursor is always shown even if something goes wrong
    print("\033[?25h", end="", flush=True)
