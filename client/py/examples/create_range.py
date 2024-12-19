#  Copyright 2023 Synnax Labs, Inc.
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

We'll write some data to an index and data channel, and then create a range that spans
the entire time range of the data. Then we'll show how to read the data back using the
range.
"""


import numpy as np
import matplotlib.pyplot as plt
import synnax as sy

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()


# Define the data.
start = sy.TimeStamp.now()
end = start + 10 * sy.TimeSpan.SECOND

time_data = np.linspace(start, end, 1000)
data = np.sin(time_data - start)

# Create an index channel that will be used to store our timestamps.
time_ch = client.channels.create(
    name="create_range_example_time", data_type=sy.DataType.TIMESTAMP, is_index=True
)

# Create a data channel that will be used to store our fake sensor data.
data_ch = client.channels.create(
    name="create_range_example_data", data_type=sy.DataType.FLOAT64, index=time_ch.key
)

# Write the data to the channels.
time_ch.write(start, time_data)
data_ch.write(start, data)

# Create a range that spans the start and end of the data.
example_range = client.ranges.create(
    name="example_range", time_range=sy.TimeRange(start, end)
)

# We can pull and plot the data from the range by just accessing the channel names as
# if they were attributes of the range itself. We'll use the elapsed_seconds method to
# convert the timestamps to seconds since the start of the range.
plt.plot(
    sy.elapsed_seconds(example_range.create_range_example_time),
    example_range.create_range_example_data,
)
plt.xlabel("Time")
plt.ylabel("Value")
plt.show()
