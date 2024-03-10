#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import matplotlib.pyplot as plt
from test_complete import test_complete


@test_complete()
def process(s: sy.Range):
    plt.plot(sy.elapsed_seconds(s.gse_ai_time), s.gse_ai_0)
    plt.savefig("test.png")
