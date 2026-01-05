#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to  calculate the average of two sensor channels that are
being sampled at different rates. This example uses a naive method that simply grabs and
uses the latest value from each channel. This approach is simple, computationally
inexpensive, and works well when both channels are operating at consistent rates. Good
examples of this are a sensor operating at 100Hz and another at 50Hz.

This example must be run in conjunction with the `simulated_daq.py` file in this
directory. Run the `simulated_daq.py` file first, and then run this file.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# We create a separate index channel to store the timestamps for the calculated values.
# These will store the same timestamps as the raw time channel, but will be used to
# index the calculated values.
time_channel = client.channels.create(
    name="simple_average_time", is_index=True, retrieve_if_name_exists=True
)

# We'll store the average in this channel
data_channel = client.channels.create(
    name="simple_average_data",
    index=time_channel.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

TO_READ = ["time_ch_1", "time_ch_2", "data_ch_1", "data_ch_2"]

# Create a dictionary to store the latest values of each channel.
current_values = dict()

with client.open_writer(
    start=sy.TimeStamp.now(),
    channels=["simple_average_time", "simple_average_data"],
) as writer:
    with client.open_streamer(TO_READ) as streamer:
        for frame in streamer:
            time = frame["time_ch_1"][-1]

            # Store the latest values in state.
            for k, _ in frame.items():
                current_values[k] = frame[k][-1]

            # If we don't have values for all channels, skip this iteration.
            if len(current_values.items()) < 4:
                continue

            # Caluclate and write the average.
            avg = (current_values["data_ch_1"] + current_values["data_ch_2"]) / 2
            writer.write({"simple_average_time": time, "simple_average_data": avg})
