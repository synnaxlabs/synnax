#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates the basics of reading and writing data from an index and data
channel in Synnax. We'll write data to an index and data channel in the Synnax cluster,
and then read the data back from the cluster and plot it.
"""

import matplotlib.pyplot as plt
import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="read_latest_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
)

# Create a data channel that will be used to store our data.
data_channel = client.channels.create(
    name="read_latest_data",
    # We need to specify the index channel that will be used to store the timestamps for
    # this data channel.
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
)

SAMPLE_COUNT = int(100)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# We'll end our write 100 seconds later
end = start + 2 * sy.TimeSpan.SECOND

# Generate linearly spaced int64 timestamps
stamps = np.linspace(start, end, SAMPLE_COUNT)

# Generate a sine wave with some noise as our data
data = np.sin(np.linspace(1, 10, SAMPLE_COUNT)) * 20 + np.random.randn(SAMPLE_COUNT)

# Write the data to the Synnax cluster through the channels. Note that we need to write
# to the index channel first, otherwise our write will fail.
time_channel.write(start, stamps)
data_channel.write(start, data)


def read_latest_n(n: int) -> sy.Frame:
    """
    Reads the latest n samples from time_channel and data_channel.

    Args:
        n: The number of samples to read.

    Returns:
        A frame containing the latest n samples from time_channel and data_channel
    """
    with client.open_iterator(
        # Open a time range starting at the start of our write (could also be
        # sy.TimeStamp.MIN) and ends at the end of time.
        tr=sy.TimeRange(start, sy.TimeStamp.MAX),
        # We'll read from both channels.
        channels=[time_channel, data_channel],
        # Set the chunk size to n.
        chunk_size=n,
    ) as i:
        # Seek to the last sample in the iterator (i.e. the most recent sample).
        i.seek_last()
        # Iterate backwards using sy.AUTO_SPAN, which will iterate by the chunk_size
        # provided.
        if not i.prev(sy.AUTO_SPAN):
            raise ValueError("No samples were found in either channel.")
        # Return the current iterator value.
        return i.value


print(read_latest_n(10))
