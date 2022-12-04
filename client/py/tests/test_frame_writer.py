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

