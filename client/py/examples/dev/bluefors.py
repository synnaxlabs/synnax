#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import time

client = sy.Synnax()


def poll_bluefors() -> dict[str, int]:
    ...


N_VALVES = 1

idx = client.channels.create(
    name="fridge_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
)

for i in range(N_VALVES):
    client.channels.create(
        name=f"fridge_vlv_{i}_state",
        index=idx.key,
        data_type=sy.DataType.UINT8,
    )

rate = (sy.Rate.HZ * 1).period.seconds

with client.open_writer(sy.TimeStamp.now(),
                        [f"fridge_vlv_{i}_state" for i in range(N_VALVES)]) as w:
    while True:
        time.sleep(rate)
        data = poll_bluefors()
        w.write(data)
