#  Copyright 2022 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import pandas as pd
import pytest

import synnax


class TestNumpy:
    def test_basic_write(self, channel: synnax.Channel, client: synnax.Synnax):
        writer = client.data.new_writer(0, [channel.key])
        try:
            data = np.random.rand(10).astype(np.float64)
            writer.write(pd.DataFrame({channel.key: data}))
            writer.commit()
        finally:
            writer.close()
