#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

"""
This example demonstrates how to configure and start a Counter Read Task on a National
Instruments USB-6289 device to measure frequency and count edges.

To run this example, you'll need to have your Synnax cluster properly configured to
detect National Instruments devices: https://docs.synnaxlabs.com/reference/driver/ni/get-started

You'll also need to have either a Counter Input module or create a simulated device
via the NI-MAX software.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/client/quick-start for more information.
client = sy.Synnax()

dev = client.devices.retrieve(model="NI 9326")

# Create an index channel that will be used to store the timestamps
# for the counter read data.
ci_time = client.channels.create(
    name="ci_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create synnax channels that will be used to store the counter input data.
# We'll create one for frequency measurement and one for edge counting.
ci_0_freq = client.channels.create(
    name="ci_0_freq",
    # Pass in the index key here to associate the channel with the index channel.
    index=ci_time.key,
    data_type=sy.DataType.FLOAT64,
    retrieve_if_name_exists=True,
)

ci_1_count = client.channels.create(
    name="ci_1_count",
    # Pass in the index key here to associate the channel with the index channel.
    index=ci_time.key,
    data_type=sy.DataType.UINT32,
    retrieve_if_name_exists=True,
)

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or write commands to a device. Tasks are the primary method for interacting with
# hardware in Synnax.
tsk = sy.ni.CounterReadTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Counter Read",
    # The key of the device to execute the task on.
    device=dev.key,
    # The rate at which the task will sample data from the device.
    sample_rate=sy.Rate.HZ * 100,
    # The rate at which data will be streamed from the device into Synnax.
    # Since we're sampling at 100 Hz and streaming at 25 Hz, we'll get 4 samples at a
    # time.
    # It's generally best to keep the stream rate under 100 Hz.
    stream_rate=sy.Rate.HZ * 25,
    # Whether to save data acquired by the task to disk. If set to False, the data
    # will be streamed into Synnax for real-time consumption but not saved to disk.
    data_saving=True,
    # The list of counter channels we'd like to acquire data from.
    channels=[
        sy.ni.CIFrequencyChan(
            # The key of the Synnax channel we're acquiring data for.
            channel=ci_0_freq.key,
            # The counter port on the device (ctr0, ctr1, etc.)
            port=0,
            # Minimum expected frequency value in Hz
            min_val=1,
            # Maximum expected frequency value in Hz
            max_val=10000,
            # Units of measurement (Hz, Seconds, or Ticks)
            units="Hz",
            # Edge to count on (Rising or Falling)
            edge="Rising",
            # Measurement method (LowFreq1Ctr, HighFreq2Ctr, or DynAvg)
            meas_method="LowFreq1Ctr",
            # Measurement time in seconds
            meas_time=0.001,
            # Divisor for internal timebase
            divisor=4,
        ),
        sy.ni.CIEdgeCountChan(
            # The key of the Synnax channel we're acquiring data for.
            channel=ci_1_count.key,
            # The counter port on the device
            port=1,
            # Edge to count on (Rising or Falling)
            active_edge="Rising",
            # Count direction (CountUp, CountDown, or ExtControlled)
            count_direction="CountUp",
            # Initial count value
            initial_count=0,
        ),
    ],
)

# This will create the task in Synnax and wait for the driver to validate that the
# configuration is correct.
client.tasks.configure(tsk)

# Stream 100 reads, which will accumulate a total of 400 samples
# for each channel over a period of 4 seconds.
total_reads = 100

# Create a synnax frame to accumulate data.
frame = sy.Frame()

# Start the task under a context manager, which ensures the task gets stopped
# when the block exits.
with tsk.run():
    # Open a streamer on the counter input channels.
    with client.open_streamer(["ci_0_freq", "ci_1_count"]) as streamer:
        for i in range(total_reads):
            frame.append(streamer.read())

# Clean up by deleting the task
client.tasks.delete(tsk.key)

# Save the data to a CSV file.
frame.to_df().to_csv("counter_read_result.csv")
