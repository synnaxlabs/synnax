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

data = client.read(sy.TimeRange(1722464238568134144, 1722464246886652416), "press_pt_1")

print("Average", np.average(data))
