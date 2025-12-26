#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to write data to an index channel and its corresponding
data channels in Synnax in a streaming fashion. Streaming data is ideal for live
applications, such as data acquisition from a sensor.
"""

import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="stream_write_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create two data channels that will be used to store our data values. We'll need to
# pass in the key of the time channel to these data channels when they are created
data_channel_1 = client.channels.create(
    name="stream_write_data_1",
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
data_channel_2 = client.channels.create(
    name="stream_write_data_2",
    index=time_channel.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# The rate at which we'll send samples to the cluster. sy.Loop  is a utility to help
# regulate the timing.
loop = sy.Loop(sy.Rate.HZ * 500)

client.channels.create(
    name="stream_write_avg",
    expression="return (stream_write_data_1 + f32(stream_write_data_2)) / 2",
    retrieve_if_name_exists=True,
)

client.channels.create(
    name="stream_write_avg_squared",
    expression="return stream_write_avg ^ 2",
    retrieve_if_name_exists=True,
)

# Open the writer as a context manager. Using a context manager is recommended as the
# context manager will automatically close the writer when we are done writing. We will
# write to both the time and data channels. To choose the channels to write to, you can
# use either the keys or the names of the channels (here, we're using the keys).
with client.open_writer(
    start,
    [time_channel.key, data_channel_1.key, data_channel_2.key],
) as writer:
    i = 0
    while loop.wait():
        # Write the data to the Synnax cluster using the writer.
        writer.write(
            {
                time_channel.key: sy.TimeStamp.now(),
                data_channel_1.key: np.sin(i / 10) * 25 + 12.5,
                data_channel_2.key: i % 2,
            }
        )
        i += 1
