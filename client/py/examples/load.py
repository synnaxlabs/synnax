#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import time
import synnax as sy
import pandas as pd

client = sy.Synnax()

NUM_VALVES = 10
NUM_SENSORS = 100

valve_commands = []
valve_acks = []


sensor_idx = client.channels.create(
    name="Sensor Time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
)


sensors = []
command_to_res = {}

for i in range(NUM_VALVES):
    cmd_idx = client.channels.create(
        name=f"Valve Command Time {i}",
        is_index=True,
        data_type=sy.DataType.TIMESTAMP,
    )
    cmd = client.channels.create(
        name=f"Valve Command {i}",
        index=cmd_idx.key,
        data_type=sy.DataType.FLOAT32,
    )
    res = client.channels.create(
        name=f"Valve Response {i}",
        index=sensor_idx.key,
        data_type=sy.DataType.FLOAT32,
    )
    valve_commands.append(cmd)
    valve_acks.append(res)
    command_to_res[cmd.key] = res




for i in range(NUM_SENSORS):
    ch = client.channels.create(
        name=f"Sensor {i}",
        index=sensor_idx.key,
        data_type=sy.DataType.FLOAT32,
    )
    sensors.append(ch)

write_to = [*[s.key for s in sensors], *[v.key for v in valve_acks], sensor_idx.key]

rate = (sy.Rate.HZ * 5).period.seconds

valve_states = {
    v.key: False for v in valve_acks
}

i = 0


with client.new_streamer([a.key for a in valve_commands]) as streamer:
    with client.new_writer(sy.TimeStamp.now(), write_to) as writer:
        while True:
            time.sleep(rate)
            if streamer.received:
                f = streamer.read()
                for k in f.columns:
                    valve_states[command_to_res[k].key] = f[k][0] > 0.5

            data = {}

            i += 1

            for v in valve_acks:
                data[v.key] = [np.float32(valve_states[v.key])]
                # if np.random.random() > 0.9:
                #     valve_states[v.key] = not valve_states[v.key]
                #     data[v.key] = [np.float32(valve_states[v.key])]
            for s in sensors:
                data[s.key] = [np.float32(np.sin(i / 1000) * 25 + np.random.random())]

            data[sensor_idx.key] = [sy.TimeStamp.now()]

            writer.write(pd.DataFrame(data))
