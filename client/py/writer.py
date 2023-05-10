from datetime import datetime

import numpy as np
import pandas as pd
import time

import synnax as sy

client = sy.Synnax()

N_CHANNELS = 100
RATE = 100

channels = []
for i in range(N_CHANNELS):
    channels.append(
        sy.Channel(
            name=f"my_chan_{i}",
            data_type=np.int64,
            rate=RATE * sy.Rate.HZ,
        )
    )
channels = client.channels.create(channels)
latencies = []

with client.new_writer(sy.TimeStamp.now(), [ch.key for ch in channels]) as w:
    d = {ch.key: [1] for ch in channels}
    df = pd.DataFrame.from_dict(d)
    for i in range(10000):
        start = datetime.now()
        if i % 100 == 0:
            print(f"Writing {i}")
        if not w.write(df):
            break
        end = datetime.now()
        latencies.append((end - start).total_seconds())
        time.sleep(1 / RATE)
