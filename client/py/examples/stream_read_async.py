#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example demonstrates how to asynchronously stream live data from a channel in
Synnax. Live-streaming is useful for real-time data processing and analysis, and is an
integral part of Synnax's control sequence and data streaming capabilities.

This example requires the `stream_write.py` file to be running in a separate terminal.
"""

import asyncio

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# We can just specify the names of the channels we'd like to stream from. These channels
# were created by running the `stream_write.py`` script.
channels = ["stream_write_time", "stream_write_data_1", "stream_write_data_2"]


async def run():
    # We will open the streamer with a context manager. The context manager will
    # automatically close the streamer after we're done reading.
    async with await client.open_async_streamer(channels) as streamer:
        # Loop through the frames in the streamer. Each iteration will block until a new
        # frame is available, then we'll print out the frame of data.
        async for frame in streamer:
            print(frame)


# Run the async function
asyncio.run(run())
