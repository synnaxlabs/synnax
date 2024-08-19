#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

with client.open_streamer(["fuel_pt_1", "daq_time"]) as s:
    above_threshold = None
    for value in s:
        if value["fuel_pt_1"] > 20 and not above_threshold:
            above_threshold = sy.TimeStamp(value["daq_time"][-1])
        elif value["fuel_pt_1"] < 20 and above_threshold:
            client.ranges.create(
                name=f"Fuel Above Threshold - " + str(above_threshold)[11:19],
                time_range=sy.TimeRange(
                    start=above_threshold, end=value["daq_time"][-1]
                ),
                color="#BADA55",
            )
            above_threshold = None
