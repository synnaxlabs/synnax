#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

import synnax as sy

client = sy.Synnax()


# We need to create the index first, that way we can provide the index key to our data
# channels.
time_index = client.channels.create(
    name="time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
)

sensor_one = sy.Channel(
    name="sensor_one",
    data_type=np.float32,  # You can use numpy to define data types
    index=time_index.key,
)

sensor_two = sy.Channel(
    name="sensor_two",
    data_type="float32",  # Or you can use strings
    index=time_index.key,
)

sensor_three = sy.Channel(
    name="sensor_three",
    data_type=sy.DataType.FLOAT32,  # Or you can use Synnax data types
    index=time_index.key,
)

data_channels = client.channels.create(
    [
        sensor_one,
        sensor_two,
        sensor_three,
    ]
)
