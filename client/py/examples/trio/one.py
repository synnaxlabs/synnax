#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import synnax as sy

client = sy.Synnax()

with client.control.acquire(
    name="Press Sequence",
    write_authorities=[sy.Authority.ABSOLUTE - 1],
    write=["press_en_cmd", "vent_en_cmd"],
    read=["pressure"],
) as auto:
    curr_target = 100
    auto["vent_en_cmd"] = False
    while True:
        auto["press_en_cmd"] = True
        if auto.wait_until(
            lambda c: c.pressure > curr_target or c.pressure < 1,
            timeout=10 * sy.TimeSpan.SECOND,
        ):
            curr_target += 100
        auto["press_en_cmd"] = False
        auto["vent_en_cmd"] = False
        time.sleep(5)
