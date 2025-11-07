#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to write commands to a LabJack device.

Before running this example:
1. Connect your LabJack device (T4, T7, T7-Pro, T8, or Digit) to your computer

2. Register the device in Synnax:
   poetry run python examples/labjack/connect_device.py

3. Connect LEDs or other output devices to DAC0 and FIO4 (or update configuration below)

4. Run this script:
   poetry run python examples/labjack/write_task.py

This example controls an analog output (DAC0) and a digital output (FIO4).
"""

import time

import synnax as sy
from synnax.hardware import labjack

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Retrieve the LabJack device from Synnax
# Update this with the name you gave the device in the Synnax Console
dev = client.hardware.devices.retrieve(name="My LabJack T7")

# Create an index channel for command timestamps
labjack_cmd_time = client.channels.create(
    name="labjack_cmd_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create an index channel for state feedback timestamps
labjack_state_time = client.channels.create(
    name="labjack_state_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create command and state channels for analog output (DAC0)
dac0_cmd = client.channels.create(
    name="dac0_cmd",
    index=labjack_cmd_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

dac0_state = client.channels.create(
    name="dac0_state",
    index=labjack_state_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Create command and state channels for digital output (FIO4)
fio4_cmd = client.channels.create(
    name="fio4_cmd",
    index=labjack_cmd_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

fio4_state = client.channels.create(
    name="fio4_state",
    index=labjack_state_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Create the LabJack Write Task
# Controls an analog output (DAC0) and a digital output (FIO4)
tsk = labjack.WriteTask(
    name="LabJack Py - Write Task",
    device=dev.key,
    state_rate=sy.Rate.HZ * 20,  # Update state at 20 Hz
    data_saving=True,
    channels=[
        # Analog output (DAC0) - voltage control
        labjack.OutputChan(
            type="AO",
            port="DAC0",
            cmd_channel=dac0_cmd.key,
            state_channel=dac0_state.key,
        ),
        # Digital output (FIO4) - binary control
        labjack.OutputChan(
            type="DO",
            port="FIO4",
            cmd_channel=fio4_cmd.key,
            state_channel=fio4_state.key,
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
print("Starting LabJack Write Task")
print("=" * 70)
print("Sending commands to LabJack device...")
print("Writing 10 command sequences at 1 Hz (10 seconds)\n")

print(f"{'Cycle':<8} {'DAC0 (V)':>12} {'FIO4 State':>12}")
print("-" * 70)

# Start the write task
try:
    with tsk.run():
        # Open a writer to send commands
        with client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[
                labjack_cmd_time.key,
                dac0_cmd.key,
                fio4_cmd.key,
            ],
            enable_auto_commit=True,
        ) as writer:
            # Write test commands
            for i in range(10):
                # Analog output - sine wave from 0V to 5V
                import math

                dac0_val = 2.5 + 2.5 * math.sin(2 * math.pi * i / 10)

                # Digital output - alternating ON/OFF
                fio4_val = i % 2

                print(f"{i+1:<8} {dac0_val:>12.2f} {'HIGH' if fio4_val else 'LOW':>12}")

                # Write all commands with timestamp
                writer.write(
                    {
                        labjack_cmd_time.key: sy.TimeStamp.now(),
                        dac0_cmd.key: dac0_val,
                        fio4_cmd.key: fio4_val,
                    }
                )
                writer.commit()
                time.sleep(1)

    print("-" * 70)
    print("✓ Write task completed successfully!")
    print("\nCommands sent:")
    print("- DAC0: Sine wave from 0V to 5V")
    print("- FIO4: Alternated between LOW and HIGH")
    print("=" * 70)

except KeyboardInterrupt:
    print("\n" + "-" * 70)
    print("✓ Write task stopped by user")
    print("=" * 70)
except Exception as e:
    print(f"\n✗ Error: {e}")
    print("=" * 70)
