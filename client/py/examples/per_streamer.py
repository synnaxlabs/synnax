import numpy as np

import synnax as sy

client = sy.Synnax()

time_ch = client.channels.create(
    name="Time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
)

data_ch = client.channels.create(
    name="Data",
    index=time_ch.key,
    data_type=sy.DataType.FLOAT32,
)

received_times = []
prev_received = sy.TimeStamp.now()
i = 0

with client.new_streamer([time_ch.key, data_ch.key]) as streamer:
    for frame in streamer:
        n = sy.TimeStamp.now()
        received_times.append(n - prev_received)
        prev_received = n
        i += 1

        if i == 10000:
            break

print(sy.Rate(sy.TimeSpan(np.average(np.array(received_times)))))
