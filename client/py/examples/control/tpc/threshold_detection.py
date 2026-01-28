#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.control.tpc.common import DAQ_TIME, FUEL_PT_1

import synnax as sy

client = sy.Synnax()

with client.open_streamer([FUEL_PT_1, DAQ_TIME]) as streamer:
    above_threshold = None
    for frame in streamer:
        if frame[FUEL_PT_1] > 20 and not above_threshold:
            above_threshold = sy.TimeStamp(frame[DAQ_TIME][-1])
        elif frame[FUEL_PT_1] < 20 and above_threshold:
            client.ranges.create(
                name=f"Fuel Above Threshold - " + str(above_threshold)[11:19],
                time_range=sy.TimeRange(start=above_threshold, end=frame[DAQ_TIME][-1]),
                color="#BADA55",
            )
            above_threshold = None
