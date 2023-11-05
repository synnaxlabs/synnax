#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import synnax as sy
from synnax.telem.control import Authority

client = sy.Synnax()

with client.control.acquire(
    name="Abort Sequence",
    write_authorities=[0],
    write=[
        "press_en_cmd",
        "press_en_cmd_time",
        "vent_en_cmd",
        "vent_en_cmd_time",
    ],
    read=["pressure"],
) as auto:
    auto.wait_until(lambda c: c.pressure > 1000)
    auto.authorize("press_en_cmd", Authority.ABSOLUTE)
    auto.authorize("vent_en_cmd", Authority.ABSOLUTE)
    auto["press_en_cmd"] = False
    auto["vent_en_cmd"] = True
    time.sleep(1000)
