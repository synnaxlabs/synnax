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

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Retrieve the USB-6289 device from Synnax.
dev = client.hardware.devices.retrieve(model="USB-6289")

# Create an index channel that will be used to store the timestamps
# for the digital read data.
ai_time = client.channels.create(
    name="di_time",
    is_index=True,
    retrieve_if_name_exists=True,
    data_type=sy.DataType.TIMESTAMP,
)
# Create two synnax channels that will be used to store the input data. Notice
# how these channels aren't specifically bound to the device. You'll do that in a
# later step when you create the Digital Read Task.
di_0 = client.channels.create(
    name="di_0",
    # Pass in the index key here to associate the channel with the index channel.
    index=ai_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)
di_1 = client.channels.create(
    name="di_1",
    # Pass in the index key here to associate the channel with the index channel.
    index=ai_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# Instantiate the task. A task is a background process that can be used to acquire data
# from, or write commands to a device. Tasks are the primary method for interacting with
# Synnax hardware devices.
tsk = ni.DigitalReadTask(
    # A name to find and monitor the task via the Synnax Console.
    name="Basic Digital Read",
    # The key of the device to execute the task on.
    device=dev.key,
    # The rate at which the task will sample data from the device.
    sample_rate=sy.Rate.HZ * 50,
    # The rate at which data will be streamed from the device into Synnax. Since we're
    # sampling at 50 hz and streaming at 25Hz, we'll get 2 samples at a time.
    stream_rate=sy.Rate.HZ * 25,
    # Whether to save data acquired by the task to disk. If set to True, the data will
    # be streamed into Synnax for real-time consumption but not saved to disk.
    data_saving=True,
    # The list of physical channels we'd like to acquire data from.
    channels=[
        ni.DIChan(channel=di_0.key, port=0, line=0),
        ni.DIChan(channel=di_1.key, port=0, line=1),
    ],
)

# Create the task in Synnax and wait for the driver to validate that the
# configuration is correct.
client.hardware.tasks.configure(tsk)

# Stream 100 reads, which will accumulate a total of 200 samples per channel over
# a period of 4 seconds.
total_reads = 100

frame = sy.Frame()

# Start the task under a context manager, which ensures the task gets stopped
# when the block exits. If you want to stop the task manually, you can call
# tsk.start();
# ...your code
# tsk.stop()
# We recommend wrapped your code in a try/finally block to ensure the task is
# stopped in case of an exception.
with tsk.start():
    # Open a streamer on the analog input channels.
    with client.open_streamer(["di_0", "di_1"]) as streamer:
        while total_reads > 0:
            frame.append(streamer.read())
            total_reads -= 1

client.hardware.tasks.delete(tsk.key)

# Save the data to a CSV file.
frame.to_df().to_csv("digital_read_result.csv")
