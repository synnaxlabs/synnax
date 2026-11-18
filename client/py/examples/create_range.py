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

We'll write data to an index and data channel, and then create a range that spans the
entire time range of the data. Then, we'll show how to read the data back using the
range.
"""

import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="create_range_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

# Create a data channel that will be used to store our fake sensor data.
data_channel = client.channels.create(
    name="create_range_data",
    data_type=sy.DataType.FLOAT64,
    index=time_channel.key,
    retrieve_if_name_exists=True,
)

# Define 10 seconds of data at 100 Hz. The end of the time range is exclusive, so it is
# one period after the last timestamp.
RATE = 100 * sy.Rate.HZ
SAMPLE_COUNT = 1000
# A second run reuses the same channels, so start one period after any data that an
# earlier run wrote past now. Writing over stored data fails.
start = sy.TimeStamp.now()
latest = client.read_latest(time_channel.key)
if len(latest) > 0:
    start = max(start, sy.TimeStamp(int(latest.to_numpy()[-1]) + int(RATE.period)))
time_data = int(start) + np.arange(SAMPLE_COUNT, dtype=np.int64) * int(RATE.period)
end = sy.TimeStamp(int(time_data[-1]) + int(RATE.period))
data = np.sin(sy.elapsed_seconds(time_data))

# Write the data to the Synnax cluster through the channels. Note that we need to write
# to the index channel first, otherwise our write will fail.
time_channel.write(start, time_data)
data_channel.write(start, data)

# Create a range that spans the start and end of the data.
example_range = client.ranges.create(
    name="create_range_range",
    time_range=sy.TimeRange(start, end),
)

# We can read the data from the range by accessing the channel names as if they were
# keys of the range itself.
seconds = sy.elapsed_seconds(example_range["create_range_time"].to_numpy())
values = example_range["create_range_data"].to_numpy()
print(f"Range {example_range.name} holds {len(values)} samples")
print(f"It spans {seconds[-1]:.2f} seconds")
print(f"Values range from {values.min():.2f} to {values.max():.2f}")
