#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import datetime

import numpy as np

import synnax as sy

client = sy.Synnax()

N_CHANNELS = 100
RATE = 100

channels = [f"my_chan_{i}" for i in range(N_CHANNELS)]

times = []

with client.stream(sy.TimeStamp.now(), channels) as r:
    i = 0
    start = datetime.now()
    for v in r:
        i += 1
        end = datetime.now()
        if i % 100 == 0:
            print(i)
        if i == 8000:
            break
        times.append(end - start)
        start = end

# print mean and the 25% and 75% quantiles

print(f"Mean: {np.mean(times)}")
print(f"25%: {np.quantile(times, 0.25)}")
print(f"75%: {np.quantile(times, 0.75)}")
