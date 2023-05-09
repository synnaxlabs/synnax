from datetime import datetime

import numpy as np
import pandas as pd
import time

import synnax as sy

client = sy.Synnax()

ch = client.channels.create(
    name="my_chan",
    data_type=np.int64,
    rate=1 * sy.Rate.HZ,
)

latencies = []

with client.new_writer(sy.TimeStamp.now(), "my_chan") as w:
    with client.stream(sy.TimeStamp.now(), "my_chan") as r:
        for i in range(10000):
            df = pd.DataFrame.from_dict({"my_chan": [i]})
            start = datetime.now()
            if not w.write(df):
                break
            v = r.read()
            end = datetime.now()
            latencies.append((end - start).total_seconds())
            time.sleep(0.01)

        print(f"Mean latency: {np.mean(latencies)}")
