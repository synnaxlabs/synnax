#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

ch = client.channels.retrieve("sphinx")

with client.open_streamer([ch.key]) as streamer:
    for frame in streamer:
        if ch.key in frame:
            print(frame[ch.key])
