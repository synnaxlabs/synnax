#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
A simple example that creates a large number of channels that are indexed by a single
timestamp channel.
"""

import synnax as sy

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

NUM_CHANNELS = 1000

# We need to create the time channel first, so it has a key assigned.
time_channel = client.channels.create(
    name="create_many_channels_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

# Define our data channels with the correct index key set.
data_channels = [
    sy.Channel(
        name=f"create_many_channels_data_{i}",
        data_type=sy.DataType.FLOAT64,
        index=time_channel.key,
    )
    for i in range(NUM_CHANNELS)
]

# Notice how we reassign the result of the create call to the channels variable, this
# ensures that our data channels have the correct keys assigned to them.
data_channels = client.channels.create(data_channels)
