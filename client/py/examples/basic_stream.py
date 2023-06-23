#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import matplotlib.pyplot as plt
import synnax as sy
import pandas as pd


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

print(f"""
    Time Channel: {time_ch}
    Data Channel: {data_ch}
""")

# plt.ion()
plt.title("Streaming")
accumulated_time = []
accumulated_data = []
with client.new_writer(sy.TimeStamp.now(), [time_ch.key, data_ch.key]) as writer:
    with client.new_streamer([time_ch.key, data_ch.key]) as streamer:
        for i in range(50000):
            time = np.int64(sy.TimeStamp.now())
            data = np.float32(np.sin(i / 5))
            writer.write(
                pd.DataFrame(
                    {
                        time_ch.key: [time],
                        data_ch.key: [data],
                    }
                )
            )
            frame = streamer.read()
            accumulated_time.extend(frame[time_ch.key].to_datetime())
            accumulated_data.extend(frame[data_ch.key].to_list())
            # plt.plot(accumulated_time, accumulated_data, "r-")
            # plt.pause(0.1)
    writer.commit()

plt.ioff()
plt.close()

plt.title("Historic")

now = sy.TimeStamp.now()
time_data = time_ch.read(now - 11 * sy.TimeSpan.SECOND, now)
data_data = data_ch.read(now - 11 * sy.TimeSpan.SECOND, now)

plt.plot(time_data.to_datetime(), data_data.to_list(), "b-")

plt.show()
