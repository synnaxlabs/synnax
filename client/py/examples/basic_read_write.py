#  Copyright 2026 Synnax Labs, Inc.
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
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="basic_read_write_time", is_index=True, data_type=sy.DataType.TIMESTAMP
)

# Create a data channel that will be used to store our data.
data_channel = client.channels.create(
    name="basic_read_write_data",
    # We need to specify the index channel that will be used to store the timestamps for
    # this data channel.
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
)

SAMPLE_COUNT = int(6e6)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# We'll end our write 100 seconds later
end = start + 100 * sy.TimeSpan.SECOND

# Generate linearly spaced int64 timestamps
stamps = np.linspace(start, end, SAMPLE_COUNT)

# Generate a sine wave with some noise as our data
data = np.sin(np.linspace(1, 10, SAMPLE_COUNT)) * 20 + np.random.randn(SAMPLE_COUNT)

# Write the data to the Synnax cluster through the channels. Note that we need to write
# to the index channel first, otherwise our write will fail.
time_channel.write(start, stamps)
data_channel.write(start, data)

# Create a time range for reading data.
time_range = sy.TimeRange(start, end)

# Read the data back from the Synnax cluster. The order doesn't matter here.
retrieved_time = time_channel.read(time_range)
retrieved_data = data_channel.read(time_range)

# Plot the data
plt.plot(retrieved_time, retrieved_data)
plt.show()
