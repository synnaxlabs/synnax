import numpy as np
import pytest
from freighter.encoder import JSON, Msgpack

import synnax
from synnax import Channel, telem
from synnax.segment.iterator import _Command, _Request


class TestNumpy:
    def test_basic_iteration(self, channel: Channel, client: synnax.Synnax):
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


class TestClientRead:
    def test_basic_read(self, channel: Channel, client: synnax.Synnax):
        w = client.data.new_writer([channel.key])
        # make an empty 1d numpy array
        data = np.random.rand(25).astype(np.float64)
        try:
            w.write(to=channel.key, data=data, start=0)
            w.write(to=channel.key, data=data, start=1 * telem.SECOND)
            w.write(to=channel.key, data=data, start=2 * telem.SECOND)
        finally:
            w.close()
        res_data = client.data.read(
            channel.key, telem.TimeRange(0, 2500 * telem.MILLISECOND)
        )
        assert res_data.shape == (62,)
        assert np.array_equal(res_data[0:25], data)
        assert np.array_equal(res_data[25:50], data)
        assert np.array_equal(res_data[50:62], data[0:12])
