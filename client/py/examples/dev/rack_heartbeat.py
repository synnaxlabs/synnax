import json

import synnax as sy
from freighter import Payload

client = sy.Synnax()

r = client.hardware.retrieve_rack(names=["sy_node_1_rack"])[0]

print(r.key)

# task key is the first 32 bits of 64
task_key = 281479271677954

ch_idx = client.channels.create(
    name="time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

ch_data = client.channels.create(
    name="the.answer",
    data_type=sy.DataType.INT32,
    index=ch_idx.key,
    retrieve_if_name_exists=True,
)

client.hardware.create_task(
    [
        sy.Task(
            key=task_key,
            name="Task",
            type="opcuaReader",
            config=json.dumps(
                {
                    "connection": {
                        "endpoint": "opc.tcp://0.0.0.0:4840",
                    },
                    "rate": 1,
                    "channels": [{"ns": 1, "node": "the.answer", "key": ch_data.key}],
                }
            ),
        )
    ]
)

with client.new_streamer([ch_idx.key, ch_data.key]) as s:
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

with client.new_streamer(["the.answer"]) as s:
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
