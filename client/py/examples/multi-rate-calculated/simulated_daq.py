#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example sets up a simulated data acquisition system (DAQ) that writes data to two
channels at different rates (along with their indexes). This example should be run in
conjunction with the `interpolation.py` example or the `simple_average.py` example to
demonstrate how to calculate derived values from these channels with different rates.
"""

import random
import time

import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps for the first
# channel.
time_ch_1 = client.channels.create(
    name="time_ch_1", is_index=True, retrieve_if_name_exists=True
)

# Create a second index channel that will be used to store our timestamps for the second
# channel.
time_ch_2 = client.channels.create(
    name="time_ch_2", is_index=True, retrieve_if_name_exists=True
)

# Data for the first channel.
data_ch_1 = client.channels.create(
    name="data_ch_1",
    index=time_ch_1.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# Data for the second channel.
data_ch_2 = client.channels.create(
    name="data_ch_2",
    index=time_ch_2.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# Set a data rate of 30 Hz. This won't be an exact loop and could drift over long
# periods of time, but it works well for a demonstration.
rough_rate = sy.Loop(sy.Rate.HZ * 30)

# Open the writer as a context manager. Using a context manager is recommended as the
# context manager will automatically close the writer when we are done writing. We will
# write to both the time and data channels. To choose the channels to write to, you can
# use either the keys or the names of the channels (here, we're using the keys).
with client.open_writer(
    start,
    [time_ch_1.key, time_ch_2.key, data_ch_1.key, data_ch_2.key],
) as writer:
    i = 0
    while rough_rate.wait():
        time = sy.TimeStamp.now()
        time_2 = time + sy.TimeSpan.MICROSECOND * 3
        # Generate data to write to the first channel.
        data_to_write = {
            time_ch_1.key: [time, time_2],
            data_ch_1.key: [np.sin(i / 10), np.sin((i + 1) / 10)],
        }

        # Only write to the second channel every third iteration, so its rate is 10 Hz
        # instead of 30 Hz.
        if i % 3 == 0:
            # Generate timestamps at a random time that is off by between -5 and +5
            # nanoseconds using random.randint
            second_time = time + sy.TimeSpan.NANOSECOND * random.randint(-5, 5)
            second_time_2 = (
                second_time
                + sy.TimeSpan.MICROSECOND * 3
                + sy.TimeSpan.NANOSECOND * random.randint(-5, 5)
            )
            data_to_write[time_ch_2.key] = [second_time, second_time_2]
            data_to_write[data_ch_2.key] = [np.sin(i / 100), np.sin((i + 1) / 100)]

        writer.write(data_to_write)
        i += 1
