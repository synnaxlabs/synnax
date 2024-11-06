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
    host="localhost",
    port=9090,
    secure=False,
    username="synnax",
    password="seldon",
)


with client.open_streamer(["sy_task_set", "sy_task_state"]) as s:
    print("STREAMING")
    for frame in s:
        if "sy_task_set" in frame:
            print(client.hardware.tasks.retrieve(frame["sy_task_set"][0]).to_payload())
        elif "sy_task_state" in frame:
            print(frame["sy_task_state"][0])
