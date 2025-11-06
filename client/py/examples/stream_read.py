#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to stream live data from a channel in Synnax.
Live-streaming is useful for real-time data processing and analysis, and is an integral
part of Synnax's control sequence and data streaming capabilities.

This example requires the `stream_write.py` file to be running in a separate terminal.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# We can just specify the names of the channels we'd like to stream from. These channels
# were created by running the `stream_write.py`` script.
channels = ["stream_write_weird", "stream_write_avg", "stream_write_data_1"]

# We will open the streamer with a context manager. The context manager will
# automatically close the streamer after we're done reading.
with client.open_streamer(channels) as streamer:
    # Loop through the frames in the streamer. Each iteration will block until a new
    # frame is available, then we'll print out the frame of data.
    i = 0
    new_channels = []
    while True:
        i += 1
        if i % 500 == 0:
            n_i = (i % 2000) / 500
            if n_i == 0:
                continue
            new_channels = channels[:int(n_i)]
            streamer.update_channels(new_channels)
        try:
            print(streamer.read())
        except KeyboardInterrupt:
            break


