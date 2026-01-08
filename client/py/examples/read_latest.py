#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to read the latest N samples from Synnax channels. We'll:
1. Create an index channel for timestamps and a data channel for values
2. Write some sample data to these channels
3. Read the latest N samples using an iterator
4. Print the results

This pattern is useful for real-time monitoring applications where you need to access
the most recent data points.
"""

import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
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

print(client.read_latest(["daq_time", "ox_pt_1"], 1))
