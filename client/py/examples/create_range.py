#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import numpy as np

client = sy.Synnax()

start = sy.TimeStamp.now()
end = start + 10 * sy.TimeSpan.SECOND
time_data = np.linspace(start, end, 1000)
data = np.sin(time_data - start)

t = client.channels.create(
    name="Time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)
d = client.channels.create(
    name="Data",
    data_type=sy.DataType.FLOAT64,
    index=t.key,
    retrieve_if_name_exists=True,
)
t.write(start, time_data)
d.write(start, data)

client.ranges.create(
    name="Example Range",
    time_range=sy.TimeRange(start, end),
)
