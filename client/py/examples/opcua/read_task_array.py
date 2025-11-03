#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read array data from the test OPC UA server (server_extended.py).

Before running this example:
1. Start the test server:
   poetry run python driver/opc/dev/server_extended.py

2. Connect the OPC UA server device in Synnax:
   - Endpoint: opc.tcp://localhost:4841/
   - Name the device "OPC UA Server" (or update line 27 below)

3. The server creates array variables (my_array_0, my_array_1, etc.) that continuously
   update with arrays of sine wave values. This example reads those arrays in array mode.
"""

import synnax as sy
from synnax.hardware import opcua

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the OPC UA server from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.hardware.devices.retrieve(name="OPC UA Server")

# Create an index channel that will be used to store the timestamps for the data.
opcua_array_time = client.channels.create(
    name="opcua_array_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create Synnax channels to store the my_array_0 and my_array_1 node data from the server.
# These arrays are continuously updated with sine wave values by server_extended.py.
my_array_0 = client.channels.create(
    name="my_array_0",
    index=opcua_array_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
my_array_1 = client.channels.create(
    name="my_array_1",
    index=opcua_array_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create the OPC UA Read Task in array mode
# Array mode is more efficient for high-rate data collection when the OPC UA server
# provides data in array format with a consistent size.
# In this mode, each sample from the server contains an entire array of values.
tsk = opcua.ReadTask(
    name="OPC UA Py - Read Task (Array)",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 10,  # Sample at 10 Hz
    array_mode=True,  # Enable array mode
    array_size=5,  # Each array contains 5 float values (matches server ARRAY_SIZE)
    data_saving=True,
    channels=[
        # Bind the Synnax channels to the OPC UA node IDs
        # These IDs correspond to my_array_0 and my_array_1 in server_extended.py
        opcua.ReadChannel(
            channel=my_array_0.key,
            node_id="NS=2;I=2",  # my_array_0
            data_type="float32"
        ),
        opcua.ReadChannel(
            channel=my_array_1.key,
            node_id="NS=2;I=3",  # my_array_1
            data_type="float32"
        ),
    ],
)

# Configure the task with Synnax
client.hardware.tasks.configure(tsk)

print("=" * 80)
print("\nStarting OPC UA Array Read Task")
print("=" * 80)
print("Reading array sine wave data from server_extended.py...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<8} {'Timestamp':<12} {'my_array_0 (5 values)':^35} {'my_array_1 (5 values)':^20}")
print("-" * 80)

# Start the task and read data continuously
try:
    # Hide cursor for clean output
    print('\033[?25l', end='', flush=True)

    with tsk.run():
        with client.open_streamer(["my_array_0", "my_array_1"]) as streamer:
            sample_count = 0
            start_time = sy.TimeStamp.now()

            while True:
                frame = streamer.read()
                if frame:
                    # Print the latest array values from each channel
                    if "my_array_0" in frame and len(frame["my_array_0"]) > 0:

                        elapsed = sy.TimeStamp.now().span(start_time).seconds
                        sample_count += 1

                        # Format arrays for display by iterating directly over MultiSeries
                        arr0_str = "[" + ",".join(f"{v:5.2f}" for v in frame["my_array_0"]) + "]"
                        arr1_str = "[" + ",".join(f"{v:5.2f}" for v in frame["my_array_1"]) + "]"

                        print(f"{sample_count:<8} {elapsed:<12.1f} {arr0_str:^35} {arr1_str:^20}", end='\r', flush=True)

# Output summary
except KeyboardInterrupt:
    print("\n" + "-" * 80)
    print("✓ Array read task stopped by user")
    print(f"\nCollected {sample_count} array samples (each containing 5 values)")
    print("The arrays contain sine wave values:")
    print("- my_array_0: [sin(t)+0]")
    print("- my_array_1: [sin(t)+1]")
    print("=" * 80)
finally:
    # Ensure cursor is always shown even if something goes wrong
    print('\033[?25h', end='', flush=True)
