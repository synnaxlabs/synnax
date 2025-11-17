#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import numpy as np

import synnax as sy

client = sy.Synnax()

idx = client.channels.create(
    name="idx", is_index=True, data_type="timestamp", retrieve_if_name_exists=True
)

data = client.channels.create(
    name="data", index=idx.key, data_type="float32", retrieve_if_name_exists=True
)

count = 100
with client.open_writer(sy.TimeStamp.now(), [idx.key, data.key]) as writer:
    for i in range(count):
        time.sleep(0.1)
        print(np.round(i / count * count))
        writer.write(
            {
                idx.key: sy.TimeStamp.now(),
                data.key: np.sin(i),
            }
        )


time.sleep(1)

data_2 = client.channels.create(
    name="data_2", index=idx.key, data_type="float32", retrieve_if_name_exists=True
)

data_3 = client.channels.create(
    name="data_3", index=idx.key, data_type="float32", retrieve_if_name_exists=True
)

with client.open_writer(
    sy.TimeStamp.now(),
    [data.key, data_2.key, data_3.key, idx.key],
) as writer:
    for i in range(50000):
        time.sleep(0.1)
        writer.write(
            {
                idx.key: sy.TimeStamp.now(),
                data.key: np.sin(i),
                data_2.key: np.sin(i * 2),
                data_3.key: np.sin(i * 3),
            }
        )
