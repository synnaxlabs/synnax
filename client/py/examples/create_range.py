#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example shows how to create a named range in Synnax, which can be used to identify
and lookup specific periods of time in your data.

We'll write  data to an index and data channel, and then create a range that spans the
entire time range of the data. Then, we'll show how to read the data back using the
range.
"""

import matplotlib.pyplot as plt
import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Define the data.
start = sy.TimeStamp.now()
end = start + 10 * sy.TimeSpan.SECOND

time_data = np.linspace(start, end, 1000)
data = np.sin(time_data - start)

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="create_range_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

# Create a data channel that will be used to store our fake sensor data.
data_channel = client.channels.create(
    name="create_range_data",
    data_type=sy.DataType.FLOAT64,
    index=time_channel.key,
)

# Write the data to the Synnax cluster through the channels. Note that we need to write
# to the index channel first, otherwise our write will fail.
time_channel.write(start, time_data)
data_channel.write(start, data)

# Create a range that spans the start and end of the data.
example_range = client.ranges.create(
    name="create_range_range",
    time_range=sy.TimeRange(start, end),
)

# We can pull and plot the data from the range by just accessing the channel names as if
# they were attributes of the range itself.
plt.plot(
    # The elapsed_seconds method converts the timestamps to seconds since the start of
    # the range.
    sy.elapsed_seconds(example_range["create_range_time"]),
    example_range["create_range_data"],
)
plt.xlabel("Time")
plt.ylabel("Value")
plt.show()
