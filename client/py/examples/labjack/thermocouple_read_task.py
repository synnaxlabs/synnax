#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read thermocouple data from a LabJack device.

Before running this example:
1. Connect a LabJack T7 or T7-Pro to your computer

2. Register the device in Synnax:
   poetry run python examples/labjack/connect_device.py

3. Connect thermocouples to AIN0 and AIN2 (or update channel configuration below)

4. Run this script:
   poetry run python examples/labjack/thermocouple_read_task.py

This example reads from two K-type thermocouples with device CJC at 10 Hz.
Note: Thermocouple streaming is slower than standard analog inputs due to the
      required cold junction compensation calculations.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the LabJack device from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.devices.retrieve(name="My LabJack T7")

# Create an index channel that will be used to store the timestamps for the data.
labjack_tc_time = client.channels.create(
    name="labjack_tc_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create Synnax channels to store the thermocouple temperature data.
tc0 = client.channels.create(
    name="thermocouple_0",
    index=labjack_tc_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

tc1 = client.channels.create(
    name="thermocouple_1",
    index=labjack_tc_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create the LabJack Read Task with Thermocouples
# Reads two K-type thermocouples at 10 Hz with device cold junction compensation
tsk = sy.labjack.ReadTask(
    name="LabJack Py - Thermocouple Task",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 10,  # Sample at 10 Hz (max for thermocouples)
    stream_rate=sy.Rate.HZ * 10,  # Stream at 10 Hz
    data_saving=True,
    channels=[
        # K-type thermocouple on AIN0 with device CJC in Celsius
        sy.labjack.ThermocoupleChan(
            port="AIN0",
            channel=tc0.key,
            thermocouple_type="K",
            cjc_source="TEMPERATURE_DEVICE_K",  # Use device internal temp sensor
            cjc_slope=1.0,  # Device temp sensor
            cjc_offset=0.0,  # Device temp sensor
            units="C",  # Celsius
            pos_chan=0,
            neg_chan=199,  # 199 = single-ended (GND)
        ),
        # K-type thermocouple on AIN2 with device CJC in Fahrenheit
        sy.labjack.ThermocoupleChan(
            port="AIN2",
            channel=tc1.key,
            thermocouple_type="K",
            cjc_source="TEMPERATURE_DEVICE_K",  # Use device internal temp sensor
            cjc_slope=1.0,  # Device temp sensor
            cjc_offset=0.0,  # Device temp sensor
            units="F",  # Fahrenheit
            pos_chan=2,
            neg_chan=199,  # 199 = single-ended (GND)
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
print("Starting LabJack Thermocouple Read Task")
print("=" * 70)
print("Reading K-type thermocouples from AIN0 and AIN2...")
print("Running continuously - Press Ctrl+C to stop\n")

print(f"{'Sample':<10} {'Time (s)':<10} {'TC0 (°C)':>12} {'TC1 (°F)':>12}")
print("-" * 70)

# Start the task and read data continuously
try:
    # Hide cursor for clean output
    print("\033[?25l", end="", flush=True)

    with tsk.run():
        with client.open_streamer(["thermocouple_0", "thermocouple_1"]) as streamer:
            sample_count = 0
            start_time = sy.TimeStamp.now()

            while True:
                frame = streamer.read()
                if frame:
                    # Print the latest values from both channels
                    if "thermocouple_0" in frame and len(frame["thermocouple_0"]) > 0:
                        temp_c = frame["thermocouple_0"][-1]
                        temp_f = frame["thermocouple_1"][-1]
                        elapsed = sy.TimeStamp.now().span(start_time).seconds

                        sample_count += 1
                        print(
                            f"{sample_count:<10} {elapsed:<10.1f} {temp_c:>12.1f} {temp_f:>12.1f}",
                            end="\r",
                            flush=True,
                        )

# Output summary
except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Thermocouple read task stopped by user")
    print(f"\nCollected {sample_count} samples")
    print("\nNote: Thermocouple data is acquired using unary reads (not streaming)")
    print("      due to LJM limitations with extended features.")
    print("=" * 70)
finally:
    # Ensure cursor is always shown even if something goes wrong
    print("\033[?25h", end="", flush=True)
