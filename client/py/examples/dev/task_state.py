#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax(
    host="nuc",
    port=9090,
    secure=False,
    username="synnax",
    password="seldon",
)


with client.open_streamer(["sy_task_state"]) as s:
    for frame in s:
        print(frame["sy_task_state"][0])
