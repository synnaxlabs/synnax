#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to stream live data from a channel in Synnax.
Live-streaming is useful for real-time data processing and analysis, and is an
integral part of Synnax's control sequence and data streaming capabilities.

This example is meant to be used in conjunction with the stream_write.py example, and
assumes that example is running in a separate terminal.
"""

import synnax as sy

# We've logged in via the CLI, so there's no need to provide credentials here.
# See https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# We can just specify the names of the channels we'd like to stream from.
channels = [
    "stream_write_example_time",
    "stream_write_example_data_1",
    "stream_write_example_data_2",
]

count = 1000

# Open the streamer as a context manager. This will make sure the streamer is properly
# closed when we're done reading. We'll read from both the time and data channels. In
with client.open_streamer(channels) as s:
    # Loop through the frames in the streamer. Each iteration will block until a new
    # frame is available, then we'll just print it out.
    i = 0
    while i < count:
        s.read()
        i += 1
