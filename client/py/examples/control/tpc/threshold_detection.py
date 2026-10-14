#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.control.tpc.common import DAQ_TIME, FUEL_PT_1

client = sy.Synnax()

with client.open_streamer([FUEL_PT_1, DAQ_TIME]) as streamer:
    above_threshold: sy.TimeStamp | None = None
    for frame in streamer:
        pressure = frame[FUEL_PT_1].to_numpy()[-1]
        time = sy.TimeStamp(frame[DAQ_TIME].to_numpy()[-1])
        if pressure > 20 and not above_threshold:
            above_threshold = time
        elif pressure < 20 and above_threshold:
            client.ranges.create(
                name="Fuel Above Threshold - " + str(above_threshold)[11:19],
                time_range=sy.TimeRange(start=above_threshold, end=time),
                color="#BADA55",
            )
            above_threshold = None
