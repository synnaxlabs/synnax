#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to export data from a Synnax cluster to a CSV file. We
will write data to a Synnax cluster, and then read the data back and export it to a CSV
file.
"""

import numpy as np

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="export_to_csv_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

# Create a data channel that will be used to store our data.
data_channel = client.channels.create(
    name="export_to_csv_data",
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

# Convert the data to a pandas DataFrame and export to a CSV file.
df = data.to_df().to_csv("exported_data.csv")
