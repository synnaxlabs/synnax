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
and then read the data back from the cluster and print a summary of it.
"""

import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="basic_read_write_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

# Create a data channel that will be used to store our data.
data_channel = client.channels.create(
    name="basic_read_write_data",
    # We need to specify the index channel that will be used to store the timestamps for
    # this data channel.
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# We'll write 100 seconds of data at 1 kHz.
RATE = 1000 * sy.Rate.HZ
SAMPLE_COUNT = 100_000

# We'll start our write at the current time. A second run reuses the same channels, so
# if an earlier run wrote data that ends after now, start one period after that data
# instead. Writing over stored data fails.
start = sy.TimeStamp.now()
latest = client.read_latest(time_channel.key)
if len(latest) > 0:
    start = max(start, sy.TimeStamp(int(latest.to_numpy()[-1]) + int(RATE.period)))

# Space the int64 timestamps one period apart. The end of the time range is exclusive,
# so it is one period after the last timestamp.
stamps = int(start) + np.arange(SAMPLE_COUNT, dtype=np.int64) * int(RATE.period)
end = sy.TimeStamp(int(stamps[-1]) + int(RATE.period))

# Generate a sine wave with some noise as our data
data = (
    np.sin(np.linspace(1, 10, SAMPLE_COUNT)) * 20 + np.random.randn(SAMPLE_COUNT)
).astype(np.float32)

# Write the data to the Synnax cluster through the channels. Note that we need to write
# to the index channel first, otherwise our write will fail.
time_channel.write(start, stamps)
data_channel.write(start, data)

# Read the data back from the Synnax cluster. The order doesn't matter here.
time_range = sy.TimeRange(start, end)
retrieved_time = time_channel.read(time_range)
retrieved_data = data_channel.read(time_range).to_numpy()

print(f"Read {len(retrieved_data)} samples from {time_range}")
print(f"Min: {retrieved_data.min():.2f}")
print(f"Max: {retrieved_data.max():.2f}")
print(f"Mean: {retrieved_data.mean():.2f}")
