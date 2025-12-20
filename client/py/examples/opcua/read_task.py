#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read data from the test OPC UA server (server_extended.py).

Before running this example:
1. Start the test server:
   poetry run python driver/opc/dev/server_extended.py

2. Connect the OPC UA server device in Synnax:
   - Endpoint: opc.tcp://127.0.0.1:4841/
   - Name the device "OPC UA Server" (or update line 27 below)

3. The server creates float variables (my_float_0, my_float_1, etc.) that continuously
   update with sine wave values. This example reads those values.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the OPC UA server from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="OPC UA Server")

# Create an index channel that will be used to store the timestamps for the data.
opcua_time = client.channels.create(
    name="opcua_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create Synnax channels to store the my_float_0 and my_float_1 node data from the server.
# These floats are continuously updated with sine wave values by server_extended.py.
my_float_0 = client.channels.create(
    name="my_float_0",
    index=opcua_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
my_float_1 = client.channels.create(
    name="my_float_1",
    index=opcua_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create the OPC UA Read Task
# Using node_name to reference the OPC UA variables directly by their names.
# This is more reliable than using node IDs which can change between server restarts.
tsk = sy.opcua.ReadTask(
    name="OPC UA Py - Read Task",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 10,  # Sample at 10 Hz
    stream_rate=sy.Rate.HZ * 10,  # Stream at 10 Hz
    data_saving=True,
    channels=[
        # Bind the Synnax channels to the OPC UA node IDs
        # These IDs correspond to my_float_0 and my_float_1 in server_extended.py
        sy.opcua.ReadChannel(
            channel=my_float_0.key,
            node_id="NS=2;I=8",  # my_float_0
            data_type="float32",
        ),
        sy.opcua.ReadChannel(
            channel=my_float_1.key,
            node_id="NS=2;I=9",  # my_float_1
            data_type="float32",
        ),
    ],
)

# Configure the task with Synnax
client.tasks.configure(tsk)

print("=" * 70)
print("Starting OPC UA Read Task")
print("=" * 70)
print("Reading sine wave data from server_extended.py...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<8} {'Timestamp':<12} {'my_float_0':>12} {'my_float_1':>12}")
print("-" * 70)

# Start the task and read data continuously
try:
    # Hide cursor for clean output
    print("\033[?25l", end="", flush=True)

    with tsk.run():
        with client.open_streamer(["my_float_0", "my_float_1"]) as streamer:
            sample_count = 0
            start_time = sy.TimeStamp.now()

            while True:
                frame = streamer.read()
                if frame:
                    # Print the latest values from each channel
                    if "my_float_0" in frame and len(frame["my_float_0"]) > 0:
                        val0 = frame["my_float_0"][-1]
                        val1 = frame["my_float_1"][-1]

                        elapsed = sy.TimeStamp.now().span(start_time).seconds

                        sample_count += 1
                        print(
                            f"{sample_count:<8} {elapsed:<12.1f} {val0:>12.4f} {val1:>12.4f}",
                            end="\r",
                            flush=True,
                        )

# Output summary
except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Read task stopped by user")
    print(f"\nCollected {sample_count} samples")
    print("The values are sine waves:")
    print("- my_float_0: sin(t) + 0      (oscillates around 0)")
    print("- my_float_1: sin(t) + 1      (oscillates around 1)")
    print("=" * 70)
finally:
    # Ensure cursor is always shown even if something goes wrong
    print("\033[?25h", end="", flush=True)
