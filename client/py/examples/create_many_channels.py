#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

NUM_CHANNELS = 1000

channels = list()
for i in range(NUM_CHANNELS):
    ch = sy.Channel(
        name=f"Channel {i}",
        data_type=sy.DataType.FLOAT32,
        rate=1*sy.Rate.HZ,
    )
    channels.append(ch)

client.channels.create(channels)
