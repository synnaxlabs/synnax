import synnax as sy
import numpy as np
import pandas as pd

client = sy.Synnax()

time_ch = client.channels.retrieve("Time")
data_ch = client.channels.retrieve("Data")

with client.new_writer(sy.TimeStamp.now(), [time_ch.key, data_ch.key]) as writer:
    for i in range(10000):
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
