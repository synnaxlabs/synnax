#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

burst_time = client.channels.create(
    name="burst_time", data_type=sy.DataType.TIMESTAMP, is_index=True
)

t = client.channels.create(
    name="burst_t",
    data_type=sy.DataType.FLOAT32,
    index=burst_time.key,
)

p = client.channels.create(
    name="burst_p",
    data_type=sy.DataType.FLOAT32,
    index=burst_time.key,
    retrieve_if_name_exists=True,
)

burst_range = client.ranges.create(
    name="burst_test",
    time_range=sy.TimeRange(
        start=sy.TimeStamp.now(), end=sy.TimeStamp.now() + 1 * sy.TimeSpan.HOUR
    ),
)

temperatures = [55, 55.1, 55.2, 55.3]
pressures = [100, 100.1, 100.7, 102.2]
