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
import matplotlib.pyplot as plt

client = sy.Synnax()

with client.new_streamer("sy_range_set") as s:
    for r in s:
        r = client.ranges.retrieve(r.series[0][0])
        t = r.read("Time (hs)")
        d = r.read("ec.pressure[12] (hs)")
        print(t.__array__(), d.__array__())
        plt.plot(t, d, "r-")
        plt.show()
