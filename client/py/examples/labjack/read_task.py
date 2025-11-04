#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read data from a LabJack device.

Before running this example:
1. Connect your LabJack device (T4, T7, T7-Pro, T8, or Digit) to your computer

2. Register the device in Synnax:
   poetry run python examples/labjack/connect_device.py

3. Connect analog sensors to AIN0 and AIN1 (or update channel configuration below)

4. Run this script:
   poetry run python examples/labjack/read_task.py

This example reads from two analog input channels at 100 Hz.
"""

import synnax as sy
from synnax.hardware import labjack

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the LabJack device from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.hardware.devices.retrieve(name="My LabJack T7")

# Create an index channel that will be used to store the timestamps for the data.
labjack_time = client.channels.create(
    name="labjack_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create Synnax channels to store the analog input data.
ain0 = client.channels.create(
    name="labjack_ain0",
    index=labjack_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

ain1 = client.channels.create(
    name="labjack_ain1",
    index=labjack_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create the LabJack Read Task
# Reads two analog input channels at 100 Hz
tsk = labjack.ReadTask(
    name="LabJack Py - Read Task",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 100,  # Sample at 100 Hz
    stream_rate=sy.Rate.HZ * 25,   # Stream at 25 Hz (4 samples per stream)
    data_saving=True,
    channels=[
        # Analog input channel AIN0 with ±10V range
        labjack.AIChan(
            port="AIN0",
            channel=ain0.key,
            range=10.0,
            pos_chan=0,
            neg_chan=199,  # 199 = single-ended (GND)
        ),
        # Analog input channel AIN1 with ±10V range
        labjack.AIChan(
            port="AIN1",
            channel=ain1.key,
            range=10.0,
            pos_chan=1,
            neg_chan=199,  # 199 = single-ended (GND)
        ),
    ],
)

# Configure the task with Synnax
try:
    client.hardware.tasks.configure(tsk)
    print("✓ Task configured successfully")
except Exception as e:
    print(f"✗ Task configuration failed: {e}")
    exit(1)

print("=" * 70)
print("Starting LabJack Read Task")
print("=" * 70)
print("Reading analog inputs from AIN0 and AIN1...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<10} {'Time (s)':<10} {'AIN0 (V)':>12} {'AIN1 (V)':>12}")
print("-" * 70)

# Start the task and read data continuously
try:
    # Hide cursor for clean output
    print('\033[?25l', end='', flush=True)

    with tsk.run():
        with client.open_streamer(["labjack_ain0", "labjack_ain1"]) as streamer:
            sample_count = 0
            start_time = sy.TimeStamp.now()

            while True:
                frame = streamer.read()
                if frame:
                    # Print the latest values from both channels
                    if "labjack_ain0" in frame and len(frame["labjack_ain0"]) > 0:
                        val0 = frame["labjack_ain0"][-1]
                        val1 = frame["labjack_ain1"][-1]
                        elapsed = sy.TimeStamp.now().span(start_time).seconds

                        sample_count += 1
                        print(f"{sample_count:<10} {elapsed:<10.1f} {val0:>12.3f} {val1:>12.3f}", end='\r', flush=True)

# Output summary
except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Read task stopped by user")
    print(f"\nCollected {sample_count} samples")
    print("=" * 70)
finally:
    # Ensure cursor is always shown even if something goes wrong
    print('\033[?25h', end='', flush=True)
