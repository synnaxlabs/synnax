#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys

import synnax as sy

if len(sys.argv) < 2:
    print("Usage: python listen_to_channels.py <channel_name> [<channel_name> ...]")
    sys.exit(1)

channel_names = sys.argv[1:]
names_set = set(channel_names)
multiple_channels = len(channel_names) > 1

client = sy.Synnax()

with client.open_streamer(channel_names) as streamer:
    for frame in streamer:
        for channel in frame.channels:
            if channel not in names_set:
                continue
            for value in frame[channel]:
                if multiple_channels:
                    print(f"{channel}: {value}")
                else:
                    print(value)
