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

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)

try:
    with client.control.acquire(
        name="Press Sequence",
        write_authorities=[sy.Authority.ABSOLUTE - 1],
        write=["gse_doc_0", "gse_doc_1"],
        read=["gse_ai_0"],
    ) as auto:
        curr_target = 100
        start = sy.TimeStamp.now()
        auto["gse_doc_1"] = False
        for i in range(5):
            auto["gse_doc_0"] = True
            if auto.wait_until(
                lambda c: c.gse_ai_0 > curr_target,
                timeout=10 * sy.TimeSpan.SECOND,
            ):
                curr_target += 100
                auto["gse_doc_0"] = False
                auto["gse_doc_1"] = False
                time.sleep(5)

        end = sy.TimeStamp.now()
        client.ranges.create(
            name=f"Auto Pressure Sequence {end}",
            time_range=sy.TimeRange(
                start=start,
                end=sy.TimeStamp.now(),
            ),
        )
finally:
    time.sleep(100)
    print("Auto Done")
