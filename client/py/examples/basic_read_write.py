#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


"""
This example demonstrates the basics of reading and writing data from an index and data
channel in Synnax. We'll write a linearly increasing line of data to a data channel and
read it back to plot it.
"""

import matplotlib.pyplot as plt
import numpy as np

import synnax as sy


# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="basic_read_write_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
)

# Create a data channel that will be used to store our data.
data_channel = client.channels.create(
    name="basic_read_write_data",
    # We need to specify the index channel that will be used to store the timestamps
    # for this data channel.
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
)

N_SAMPLES = int(6e6)

# We'll start our write at the current time. This timestamp should be the same as or
# just before the first timestamp we write.
start = sy.TimeStamp.now()

# We'll end our write 100 seconds later
end = start + 100 * sy.TimeSpan.SECOND

# Generate linearly space int64 timestamps
stamps = np.linspace(start, end, N_SAMPLES, dtype=np.int64)

# Generate a line from 1 to 10
data = np.sin(np.linspace(1, 10, N_SAMPLES, dtype=np.float32)) * 20 + np.random.randn(
    N_SAMPLES
).astype(np.float32)

# Write the data to the channel. Note that we need to write the timestamps first,
# otherwise writing the data will fail.
time_channel.write(start, stamps)
data_channel.write(start, data)

# Define the time range to read the data back from.
time_range = sy.TimeRange(start, end)

# Read the data back. The order doesn't matter here.
res_stamps = time_channel.read(time_range)
res_data = data_channel.read(time_range)

# Plot the data
plt.plot(res_stamps, res_data)
plt.show()
