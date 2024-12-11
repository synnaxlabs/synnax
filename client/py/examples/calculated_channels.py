#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

"""
This example demonstrates how to calculate derived values from a set of channels and
write them to a new set of channels in Synnax in a streaming fashion. These channels,
typically referred to as "calculated" or "derived" channels, are useful for storing
values that are calculated from other channels.

For this example to run, you'll need to run the "stream_write.py" file also contained
in this directory BEFORE you run this script.
"""

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# We create a separate index channel to store the timestamps for the calculated values.
# These will store the same timestamps as the raw time channel, but will be used to
# index the calculated values.
derived_time_ch = client.channels.create(
    name="derived_time", is_index=True, retrieve_if_name_exists=True
)

# We'll store the squared value of "stream_write_example_data_1" in this channel.
squared_example_data_1 = client.channels.create(
    name="squared_example_data_1",
    index=derived_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

# We'll store the average of "stream_write_example_data_1" and "stream_write_example_data_2"
# in this channel.
average_example_data_1 = client.channels.create(
    name="average_example_data_1",
    index=derived_time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

with client.open_writer(
    start=sy.TimeStamp.now(),
    channels=["derived_time", "squared_example_data_1", "average_example_data_1"],
    enable_auto_commit=True,
) as writer:
    with client.open_streamer(
        [
            "stream_write_example_time",
            "stream_write_example_data_1",
            "stream_write_example_data_2",
        ]
    ) as s:
        for fr in s:
            time = fr["stream_write_example_time"]
            # Square
            squared = fr["stream_write_example_data_1"] ** 2
            # Average
            avg = (
                fr["stream_write_example_data_1"] + fr["stream_write_example_data_2"]
            ) / 2
            writer.write(
                {
                    # Write back the same timestamps as the raw data, so they align
                    # correctly.
                    "derived_time": time,
                    "squared_example_data_1": squared,
                    "average_example_data_1": avg,
                }
            )
