#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

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
