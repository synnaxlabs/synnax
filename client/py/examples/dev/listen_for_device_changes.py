#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import synnax as sy

client = sy.Synnax()

with client.open_streamer("sy_task_set") as s:
    for frame in s:
        t = client.tasks.retrieve(keys=[frame["sy_task_set"][0]])
        cfg = t[0].config
        json.dump(json.loads(cfg), open("task.json", "w"), indent=4)
