import json

import synnax as sy
from freighter import Payload

client = sy.Synnax()

r = client.hardware.retrieve_rack(names=["sy_node_1_rack"])[0]

with client.open_streamer("sy_task_set") as s:
    for frame in s:
        key = frame["sy_task_set"][0]
        task = client.hardware.retrieve_task(keys=[key])
        print(task)
