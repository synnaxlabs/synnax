#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to calculate derived values from a set of channels and
write them to a new set of channels in Synnax in a streaming fashion. These channels,
typically referred to as "calculated" or "derived" channels, are useful for storing
values that are calculated from other channels.

For this example to run, you'll need to run the `stream_write.py` script before you run
this script.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# We create a separate index channel to store the timestamps for the calculated values.
# These will store the same timestamps as the raw time channel, but will be used to
# index the calculated values.
time_channel = client.channels.create(
    name="calculated_channels_time",
    is_index=True,
    retrieve_if_name_exists=True,
)

# We'll store the squared value of "stream_write.data_1" in this channel.
squared_data_channel = client.channels.create(
    name="calculated_channels_squared_data",
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# We'll store the average of "stream_write.data_1" and "stream_write.data_2" in this
# channel.
averaged_data_channel = client.channels.create(
    name="calculated_channels_averaged_data",
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

with client.open_writer(
    start=sy.TimeStamp.now(),
    channels=[
        "calculated_channels_time",
        "calculated_channels_squared_data",
        "calculated_channels_averaged_data",
    ],
) as writer:
    with client.open_streamer(
        ["stream_write_time", "stream_write_data_1", "stream_write_data_2"],
    ) as streamer:
        for frame in streamer:
            time = frame["stream_write_time"]
            # Square
            squared = frame["stream_write_data_1"] ** 2
            # Average
            avg = (frame["stream_write_data_1"] + frame["stream_write_data_2"]) / 2
            writer.write(
                {
                    # Write back the same timestamps as raw data so they align
                    # correctly.
                    "calculated_channels_time": time,
                    "calculated_channels_squared_data": squared,
                    "calculated_channels_averaged_data": avg,
                }
            )
