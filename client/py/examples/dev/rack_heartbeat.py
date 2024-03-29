import json

import synnax as sy
from freighter import Payload

client = sy.Synnax()

r = client.hardware.retrieve_rack(names=["sy_node_1_rack"])[0]

print(r.key)

# task key is the first 32 bits of 64
task_key = 281479271677954

client.hardware.create_task([sy.Task(
        key=task_key,
        name="Task",
        type="opcuaScanner"
)])



with client.new_streamer(["node1"]) as s:
    # with client.new_writer(sy.TimeStamp.now(), ["sy_task_cmd"]) as w:
    #     w.write({
    #         "sy_task_cmd": [{
    #             "task": 281479271677954,
    #             "type": "scan",
    #             "args": {
    #                 "endpoint": "opc.tcp://0.0.0.0:4840/freeopcua/server/",
    #             }
    #         }]
    #     })
    for frame in s:
        print(frame)
