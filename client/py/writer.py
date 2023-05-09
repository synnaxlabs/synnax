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

with client.new_writer(sy.TimeStamp.now(), "my_chan") as w:
    with client.stream(sy.TimeStamp.now(), "my_chan") as r:
        for i in range(100000):
            df = pd.DataFrame.from_dict({"my_chan": [i]})
            start = datetime.now()
            if not w.write(df):
                break
            v = r.read()
            print(f"write {i} in {(datetime.now() - start)} {v}")
            time.sleep(0.01)
