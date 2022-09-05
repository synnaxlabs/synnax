import numpy as np

import arya
from arya import Channel, telem


class TestNumpy:
    def test_basic_iteration(self, channel: Channel, client: arya.Client):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(25).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
            writer.write(to=channel.key, data=data, start=1 * telem.SECOND)
            writer.write(to=channel.key, data=data, start=2 * telem.SECOND)
        finally:
            writer.close()
        iter = client.data.new_iterator([channel.key], tr=telem.TIME_RANGE_MAX)
        try:
            assert iter.first()
            assert iter.value[channel.key].data.shape == (25,)
            c = 1
            while iter.next():
                c += 1
                assert iter.value[channel.key].data.shape == (25,)
            assert c == 3
        finally:
            iter.close()
