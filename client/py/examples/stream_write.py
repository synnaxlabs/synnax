#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import time

import numpy as np
import pandas as pd

import synnax as sy

client = sy.Synnax()

time_ch = client.channels.create(
    name="Time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True
)

data_ch = client.channels.create(
    name="Data",
    index=time_ch.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True
)

print(
    f"""
    Time Channel Key: {time_ch.key}
    Data Channel Key: {data_ch.key}
"""
)


with client.new_writer(sy.TimeStamp.now(), [time_ch.key, data_ch.key]) as writer:
    i = 0
    while True:
        t = np.int64(sy.TimeStamp.now())
        d = np.float32(np.sin(i / 10) * 1)
        writer.write(
            pd.DataFrame(
                {
                    time_ch.key: [t],
                    data_ch.key: [d],
                }
            )
        )
        time.sleep(0.025)
        # print("Wrote", t, d)
        i += 1
        if i % 500 == 0:
            print("Commit", writer.commit())
