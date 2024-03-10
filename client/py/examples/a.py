#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax(
    host="141.212.23.215",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False
)

tr = sy.TimeRange(1707070831214829300, 1707070848461897000)

print(tr)

data = client.read(tr, "gse_ai_time")

print(tr.span)

print(len(data), data.time_range.span)
