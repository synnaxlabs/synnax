#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
import numpy as np
import synnax as sy

"""
This example sets up a simulated data acquisition system that writes data to two
channels at different rates (along with their indexes). This example should be run in
conjunction with the 'calculated_interpolation.py' example or the 'calculated_simple.py'
example to demonstrate how to calculate derived values from these channels with
different rates.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps for the first
# channel, operating at rate 1.
time_ch_1 = client.channels.create(
    name="time_ch_1",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create a second index channel that will be used to store our timestamps for the second
# channel, operating at rate 2.
time_ch_2 = client.channels.create(
    name="time_ch_2",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Data for the first channel, operating at rate 1.
data_ch_1 = client.channels.create(
    name="data_ch_1",
    index=time_ch_1.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Data for the second channel, operating at rate 2.
data_ch_2 = client.channels.create(
    name="data_ch_2",
    index=time_ch_2.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# Set a rough data rate of 20 Hz. This won't be exact because we're sleeping for a
# fixed amount of time, but it's close enough for demonstration purposes.
rough_rate = sy.Loop(sy.Rate.HZ * 30)


# Open the writer as a context manager. This will make sure the writer is properly
# closed when we're done writing. We'll write to both the time and data channels. In
# this example, we provide the keys of the channels we want to write to, but you can
# also provide the names and write that way.
start = sy.TimeStamp.now()
with client.open_writer(
    start,
    [time_ch_1.key, time_ch_2.key, data_ch_1.key, data_ch_2.key],
    enable_auto_commit=True,
) as writer:
    i = 0
    while rough_rate.wait():
        time = sy.TimeStamp.now()
        time_2 = time + sy.TimeSpan.MILLISECOND * 3
        # Generate data to write to the first channel.
        data_to_write = {
            time_ch_1.key: [np.int64(time), np.int64(time_2)],
            data_ch_1.key: [
                np.float32(np.sin(i / 10)),
                np.float32(np.sin((i + 1) / 10)),
            ],
        }

        # Only write to the second channel every third iteration, so its rate is 10Hz,
        # instead of 30Hz.
        if i % 3 == 0:
            # Generate timestamps at a different time to introduce intentional
            # misalignment.
            time = sy.TimeStamp.now()
            time_2 = time + sy.TimeSpan.MILLISECOND
            data_to_write[time_ch_2.key] = [np.int64(time), np.int64(time_2)]
            data_to_write[data_ch_2.key] = [np.sin(i / 100), np.sin((i + 1) / 100)]

        writer.write(data_to_write)
        i += 1

print(sy.TimeSpan.since(start))
