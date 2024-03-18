#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import matplotlib.pyplot as plt
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

data_ch_2 = client.channels.create(
    name="Data 2",
    index=time_ch.key,
    data_type=sy.DataType.FLOAT32,
)

N_SAMPLES = int(5000)
start = sy.TimeStamp.now()
stamps = np.linspace(
    int(start), int(start + 100 * sy.TimeSpan.SECOND), N_SAMPLES, dtype=np.int64
)
data = np.linspace(1, 10, N_SAMPLES, dtype=np.float32)

data_2 = np.linspace(5, 30, N_SAMPLES, dtype=np.float32)

r = sy.TimeRange.MAX
time_ch.write(start, stamps)
data_ch.write(start, data)
data_ch_2.write(start, data_2)

print(
    f"""
Time channel: {time_ch.key}
Data channel: {data_ch.key}
"""
)

res_stamps = time_ch.read(r)
res_data = data_ch.read(r)

client.ranges.create(
    name="Time Range",
    time_range=sy.TimeRange(start, start + 100 * sy.TimeSpan.SECOND),
)

plt.plot(res_stamps, res_data)
plt.show()
