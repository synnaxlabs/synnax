#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from alamos.dev import instrumentation

client = sy.Synnax(
    instrumentation=instrumentation(),
)

range = client.ranges.create(
    name="My Range with a Very Strange Name",
    time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
)
