#  Copyright 2026 Synnax Labs, Inc.
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

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

CHANNEL_COUNT = 100

# Create an index channel that will be used to store our timestamps.
time_channel = client.channels.create(
    name="create_channels_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

# Create data channels to store our data. Since we did not call client.channels.create
# here, the channels are not actually created in the Synnax cluster yet. We will do that
# in the next step.
data_channels = [
    sy.Channel(
        name=f"create_channels_data_{i}",
        data_type=sy.DataType.FLOAT64,
        index=time_channel.key,
    )
    for i in range(CHANNEL_COUNT)
]

# Notice how we reassign the result of the create call to the data_channels variable.
# This means that all of the channels will have the correct key given to the channel by
# the server.
data_channels = client.channels.create(data_channels)
