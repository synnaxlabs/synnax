#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from uuid import uuid4

import synnax as sy

client = sy.Synnax()

# Create a rack
rack = client.hardware.create_rack([sy.Rack(key=0, name="Rack 1")])

# Create a device
client.hardware.create_device(
    [
        sy.Device(
            key=str(uuid4()),
            rack=rack[0].key,
            name="PXI-6255",
            description="a new device",
            make="ni",
            model="PXI-6255",
            properties="{}",
        )
    ]
)

with client.new_streamer(["sy_task_set"]) as streamer:
    for f in streamer:
        print(list(f["sy_task_set"]))
        t = client.hardware.retrieve_task(None, list(f["sy_task_set"]))
        f = open("task.json", "w")
        f.write(json.dumps(json.loads(t[0].config), indent=4))
