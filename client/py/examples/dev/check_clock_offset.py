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
import numpy as np
# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/python-client/get-started.
client = sy.Synnax()

# We can just specify the names of the channels we'd like to stream from. These channels
# were created by running the `stream_write.py`` script.
channels = ["T7_time"]

avg_clock_offset = sy.TimeSpan(0)
avg_delta = 0

# We will open the streamer with a context manager. The context manager will
# automatically close the streamer after we're done reading.
with client.open_streamer(channels) as streamer:
    # Loop through the frames in the streamer. Each iteration will block until a new
    # frame is available, then we'll print out the frame of data.
    count = 0
    while True:
        data = streamer.read()[channels[0]]
        offset = sy.TimeSpan(sy.TimeStamp.now() - sy.TimeStamp(data[-1]))
        avg_clock_offset += offset
        count += 1
        if count % 100 == 0:
            print(f"Average clock offset: {avg_clock_offset / count}")
        diff = sy.TimeSpan(data[-1] - data[-2])
        if diff > sy.TimeSpan.MICROSECOND * 550 or diff < sy.TimeSpan.MICROSECOND * 450:
            print(f"Diff: {diff}")
