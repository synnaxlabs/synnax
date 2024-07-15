import json

import synnax as sy

client = sy.Synnax()

with client.open_streamer("sy_task_set") as s:
    for frame in s:
        t = client.hardware.retrieve_task(keys=[frame["sy_task_set"][0]])
        cfg = t[0].config
        json.dump(json.loads(cfg), open("task.json", "w"), indent=4)
