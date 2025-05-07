#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import sys

if len(sys.argv) < 2:
    print("Usage: python listen_to_channels.py <channel_name> [<channel_name> ...]")
    sys.exit(1)

names = sys.argv[1:]
multiple_channels = len(names) > 1

client = sy.Synnax()

with client.open_streamer(names) as s:
    for frame in s:
        for channel_name in names:
            if channel_name in frame:
                for v in frame[channel_name]:
                    if multiple_channels:
                        print(f"{channel_name}: {v}")
                    else:
                        print(v)
