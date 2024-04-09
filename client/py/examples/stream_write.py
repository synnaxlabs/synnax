#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


"""
This example demonstrates how to write data to an index channel and its corresponding
data channel in Synnax in a streaming fashion. Streaming data is ideal for live
applications (such as data acquisition from a sensor) or for very large datasets that
cannot be written all at once.
"""

import time
import numpy as np
import synnax as sy

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/python-client/get-started for more information.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_ch = client.channels.create(
    name="stream_write_example_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create two data channels that will be used to store our data values. We'll write to
data_ch_1 = client.channels.create(
    name="stream_write_example_data_1",
    index=time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
data_ch_2 = client.channels.create(
    name="stream_write_example_data_2",
    index=time_ch.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# Set a rough data rate of 20 Hz. This won't be exact because we're sleeping for a
# fixed amount of time, but it's close enough for demonstration purposes.
rough_rate = sy.Rate.HZ * 20

# Make the writer commit every 500 samples. This will make the data available for
# historical reads every 500 samples.
commit_interval = 500

# Open the writer as a context manager. This will make sure the writer is properly
# closed when we're done writing. We'll write to both the time and data channels. In
# this example, we provide the keys of the channels we want to write to, but you can
# also provide the names and write that way.
with client.open_writer(start, [time_ch.key, data_ch_1.key, data_ch_2.key]) as writer:
    i = 0
    while True:
        # Generate our timestamp and data value
        timestamp = np.int64(sy.TimeStamp.now())
        data_1 = np.float32(np.sin(i / 10))
        data_2 = i % 2

        # Write the data to the writer
        writer.write(
            {
                time_ch.key: timestamp,
                data_ch_1.key: data_1,
                data_ch_2.key: data_2,
            }
        )

        time.sleep(rough_rate.period.seconds)

        i += 1

        if i % 60 == 0:
            print(f"Writing sample {i} at {sy.TimeStamp.now()}")

        if i % 500 == 0:
            print(f"Committing at {sy.TimeStamp.now()}")
            # Commit the writer. This method will return false if the commit fails i.e.
            # we've made an invalid write or someone has already written to this region.
            if not writer.commit():
                break
