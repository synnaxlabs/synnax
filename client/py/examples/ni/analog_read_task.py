#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware import ni

"""
This examples demonstrates how to configure and start an Analog Read Task on a National
Instruments USB-6289 device.

To run this example, you'll need to have your Synnax cluster properly configured to
detect National Instruments devices: https://docs.synnaxlabs.com/reference/driver/ni/get-started

You'll also need to have either a physical USB-6289 device or create a simulated device
via the NI-MAX software.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

dev = client.hardware.devices.retrieve(model="USB-6289")

# Create an index channel that will be used to store the timestamps
# for the analog read data.
ai_time = client.channels.create(
    name="ai_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create two synnax channels that will be used to store the input data. Notice
# how these channels aren't specifically bound to the device. You'll do that in a
# later step when you create the Analog Read Task.
ai_0 = client.channels.create(
    name="ai_0",
    # Pass in the index key here to associate the channel with the index channel.
    index=ai_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
ai_1 = client.channels.create(
    name="ai_1",
    # Pass in the index key here to associate the channel with the index channel.
    index=ai_time.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or write commands to a device. Tasks are the primary method for interacting with
# hardware in Synnax.
tsk = ni.AnalogReadTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Analog Read",
    # The rate at which the task will sample data from the device.
    sample_rate=sy.Rate.HZ * 100,
    # The rate at which data will be streamed from the device into Synnax.
    # Since we're sampling at 100hz and streaming at 25hz, we'll get 4 samples at a
    # time.
    # It's generally best to keep the stream rate under 100Hz.
    stream_rate=sy.Rate.HZ * 25,
    # Whether to save data acquired by the task to disk. If set to False, the data
    # will be streamed into Synnax for real-time consumption but not saved to disk.
    data_saving=True,
    # The list of physical channels we'd like to acquire data from.
    channels=[
        ni.AIVoltageChan(
            # The key of the Synnax channel we're acquiring data for.
            channel=ai_0.key,
            # The key of the device on which the channel is located.
            device=dev.key,
            # The port on the device the channel is connected to.
            port=0,
            # A custom scale to apply to the data. This is optional, but can be useful
            # for converting raw data into meaningful units.
            custom_scale=ni.LinScale(
                slope=2e4,
                y_intercept=50,
                pre_scaled_units="Volts",
                scaled_units="Volts",
            ),
        ),
        ni.AIVoltageChan(
            channel=ai_1.key,
            device=dev.key,
            port=1,
            custom_scale=ni.MapScale(
                pre_scaled_min=0,
                pre_scaled_max=10,
                scaled_min=0,
                scaled_max=200,
                pre_scaled_units="Volts",
                scaled_units="Degrees",
            ),
        ),
    ],
)

# This will create the task in Synnax and wait for the driver to validate that the
# configuration is correct.
client.hardware.tasks.configure(tsk)

# Stream 100 reads, which will accumulate a total of 400 samples
# for each channel over a period of 4 seconds.
total_reads = 100

# Create a synnax frame to accumulate data.
frame = sy.Frame()

# Start the task under a context manager, which ensures the task gets stopped
# when the block exits.
with tsk.run():
    # Open a streamer on the analog input channels.
    with client.open_streamer(["ai_0", "ai_1"]) as streamer:
        for i in range(total_reads):
            frame.append(streamer.read())

# Save the data to a CSV file.
frame.to_df().to_csv("analog_read_result.csv")
