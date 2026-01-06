#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to plot data from a Synnax cluster. We will write data to
a Synnax cluster, and then read the data back and plot it.
"""

import matplotlib.pyplot as plt
import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="plot_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

# Create a data channel that will be used to store our data.
data_channel = client.channels.create(
    name="plot_data",
    data_type=sy.DataType.FLOAT32,
    index=time_channel.key,
)


# We'll start our write at the current time. This timestamp should be the same as or
# before the first timestamp we write.
start = sy.TimeStamp.now()

# We'll end our write 1 minute later
end = start + sy.TimeSpan.MINUTE

# We'll write 1000 samples of data to Synnax.
SAMPLE_COUNT = 1000

# Generate linearly spaced timestamps
timestamps = np.linspace(start, end, SAMPLE_COUNT)

# Generate linearly spaced data with some noise
data = np.linspace(0, 100, SAMPLE_COUNT) + np.random.randn(SAMPLE_COUNT) * 5

# Write the data to the Synnax cluster through the channels.
client.write(
    start,
    {
        time_channel: timestamps,
        data_channel: data,
    },
)

# Create a time range for reading data.
time_range = sy.TimeRange(start, end)

# Read the data back from the Synnax cluster.
data = client.read(time_range, [time_channel.name, data_channel.name])

# For plotting, we want to convert the time to seconds since the start of the range. We
# will convert it to numpy, and then remove the start timestamp value from each sample,
# and then convert from nanoseconds to seconds.
time_data = data[time_channel.name].to_numpy()
time_data = time_data - time_data[0]
time_data = time_data / sy.TimeSpan.SECOND

# Plot the data
plt.plot(time_data, data[data_channel.name])
plt.xlabel(time_channel.name)
plt.ylabel(data_channel.name)
plt.title(f"Plot of {data_channel.name} vs {time_channel.name}")
plt.show()
