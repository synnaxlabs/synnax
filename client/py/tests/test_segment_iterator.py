import numpy as np
import pytest

import synnax


class TestNumpyIterator:
    def test_basic_iteration(self, channel: synnax.Channel, client: synnax.Synnax):
        writer = client.data.new_writer([channel.key])
        try:
            data = np.random.rand(25).astype(np.float64)
            writer.write(to=channel.key, data=data, start=0)
            writer.write(to=channel.key, data=data, start=1 * synnax.SECOND)
            writer.write(to=channel.key, data=data, start=2 * synnax.SECOND)
        finally:
            writer.close()
        iterator = client.data.new_iterator([channel.key], tr=synnax.TIME_RANGE_MAX)
        try:
            assert iterator.first()
            assert iterator.value[channel.key].data.shape == (25,)
            c = 1
            while iterator.next():
                c += 1
                assert iterator.value[channel.key].data.shape == (25,)
            assert c == 3
        finally:
            iterator.close()


class TestClientRead:
    def test_basic_read(self, channel: synnax.Channel, client: synnax.Synnax):
        writer = client.data.new_writer([channel.key])
        data = np.random.rand(25).astype(np.float64)
        try:
            writer.write(to=channel.key, data=data, start=0)
            writer.write(to=channel.key, data=data, start=1 * synnax.SECOND)
            writer.write(to=channel.key, data=data, start=2 * synnax.SECOND)
        finally:
            writer.close()
        res_data = client.data.read(channel.key, 0, 2500 * synnax.MILLISECOND)
        assert res_data.shape == (62,)
        assert np.array_equal(res_data[0:25], data)
        assert np.array_equal(res_data[25:50], data)
        assert np.array_equal(res_data[50:62], data[0:12])

    def test_read_non_contiguous(self, channel: synnax.Channel, client: synnax.Synnax):
        channel.write(0, np.random.rand(25))
        channel.write(3 * synnax.SECOND, np.random.rand(25))
        with pytest.raises(synnax.ContiguityError):
            channel.read(0, 4 * synnax.SECOND)
