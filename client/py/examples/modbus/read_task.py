#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read data from the test Modbus TCP server (server.py).

Before running this example:
1. Start the test server:
   poetry run python driver/modbus/dev/server.py

2. Connect the Modbus device in Synnax:
   - Host: localhost (127.0.0.1)
   - Port: 5020
   - Name the device "Modbus Server" (or update line 27 below)

3. The server creates simulated sensor data with sine waves and digital patterns.
   This example reads from two input registers.
"""

import synnax as sy
from synnax import modbus

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Retrieve the Modbus device from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="Modbus Server")

# Create an index channel that will be used to store the timestamps for the data.
modbus_time = client.channels.create(
    name="modbus_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create Synnax channels to store the Modbus input register data.
# The server provides sine wave data (0-255) at input register addresses 0 and 1.
input_reg_0 = client.channels.create(
    name="input_register_0",
    index=modbus_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

input_reg_1 = client.channels.create(
    name="input_register_1",
    index=modbus_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Create the Modbus Read Task
# Reads two input registers with sine wave data from the test server.
tsk = sy.modbus.ReadTask(
    name="Modbus Py - Read Task",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 10,  # Sample at 10 Hz
    stream_rate=sy.Rate.HZ * 10,  # Stream at 10 Hz
    data_saving=True,
    channels=[
        # Input register (16-bit R-only) at address 0
        sy.modbus.InputRegisterChan(
            channel=input_reg_0.key, address=0, data_type="uint8"
        ),
        # Input register (16-bit R-only) at address 1
        sy.modbus.InputRegisterChan(
            channel=input_reg_1.key, address=1, data_type="uint8"
        ),
    ],
)

# Configure the task with Synnax
try:
    client.tasks.configure(tsk)
    print("✓ Task configured successfully")
except Exception as e:
    print(f"✗ Task configuration failed: {e}")
    exit(1)

print("=" * 70)
print("Starting Modbus Read Task")
print("=" * 70)
print("Reading sine wave data from server.py...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<10} {'Time (s)':<10} {'Reg 0':>10} {'Reg 1':>10}")
print("-" * 70)

# Start the task and read data continuously
try:
    # Hide cursor for clean output
    print("\033[?25l", end="", flush=True)

    with tsk.run():
        with client.open_streamer(["input_register_0", "input_register_1"]) as streamer:
            sample_count = 0
            start_time = sy.TimeStamp.now()

            while True:
                frame = streamer.read()
                if frame:
                    # Print the latest values from both channels
                    if (
                        "input_register_0" in frame
                        and len(frame["input_register_0"]) > 0
                    ):
                        val0 = frame["input_register_0"][-1]
                        val1 = frame["input_register_1"][-1]
                        elapsed = sy.TimeStamp.now().span(start_time).seconds

                        sample_count += 1
                        print(
                            f"{sample_count:<10} {elapsed:<10.1f} {val0:>10} {val1:>10}",
                            end="\r",
                            flush=True,
                        )

# Output summary
except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Read task stopped by user")
    print(f"\nCollected {sample_count} samples")
    print(
        "The server provides sine wave data (0-255) at input register addresses 0 and 1."
    )
    print("=" * 70)
finally:
    # Ensure cursor is always shown even if something goes wrong
    print("\033[?25h", end="", flush=True)
