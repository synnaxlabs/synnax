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

import synnax
import matplotlib.pyplot as plt


client = synnax.Synnax(
    host="localhost", port=9090, username="synnax", password="seldon"
)

ch = client.channel.retrieve(name="gse.pressure[7] (psi)")
tCH = client.channel.retrieve(name="Time")

data = ch.read(0, synnax.TIME_STAMP_MAX)
t_data = tCH.read(0, synnax.TIME_STAMP_MAX)

plt.plot(t_data, data)
plt.show()
