#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import numpy as np
import pytest

import synnax as sy
from synnax.util.random import random_name
from tests.telem import seconds_linspace


@pytest.mark.framer
@pytest.mark.iterator
class TestIterator:
    def test_basic_iterate(self, indexed_pair: list[sy.Channel], client: sy.Synnax):
        idx_ch, _ = indexed_pair
        d = seconds_linspace(1, 50)
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key) as i:
            for f in i:
                assert np.array_equal(f.get(idx_ch.key), d)

    def test_auto_chunk(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        d = seconds_linspace(1, 10)
        idx_ch, _ = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key, chunk_size=4) as i:
            assert i.seek_first()
            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(1, 4))

            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(5, 4))

            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(9, 2))

            assert not i.next(sy.framer.AUTO_SPAN)

    def test_auto_chunk_reverse(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        d = seconds_linspace(1, 10)
        idx_ch, _ = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key, chunk_size=4) as i:
            assert i.seek_last()
            assert i.prev(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(7, 4))

            assert i.prev(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(3, 4))

            assert i.prev(sy.framer.AUTO_SPAN)
            l = i.value.get(idx_ch.key).to_numpy().tolist()
            assert np.array_equal(l, seconds_linspace(1, 2))

            assert not i.prev(sy.framer.AUTO_SPAN)

    def test_advanced_iterate(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ):
        idx_ch, data_ch = indexed_pair
        idx_ch.write(
            0,
            np.array(
                [
                    0,
                    1 * sy.TimeSpan.SECOND,
                    2 * sy.TimeSpan.SECOND,
                    3 * sy.TimeSpan.SECOND,
                    4 * sy.TimeSpan.SECOND,
                    5 * sy.TimeSpan.SECOND,
                ]
            ).astype(np.int64),
        )
        data_ch.write(0, np.array([0, 1, 2, 3, 4, 5]).astype(np.int64))
        idx_ch.write(
            sy.TimeStamp(10 * sy.TimeSpan.SECOND),
            np.array(
                [
                    10 * sy.TimeSpan.SECOND,
                    11 * sy.TimeSpan.SECOND,
                    12 * sy.TimeSpan.SECOND,
                    13 * sy.TimeSpan.SECOND,
                ]
            ).astype(np.int64),
        )
        data_ch.write(
            sy.TimeStamp(10 * sy.TimeSpan.SECOND),
            np.array([10, 11, 12, 13]).astype(np.int64),
        )
        idx_ch.write(
            sy.TimeStamp(15 * sy.TimeSpan.SECOND),
            np.array(
                [
                    15 * sy.TimeSpan.SECOND,
                    16 * sy.TimeSpan.SECOND,
                    17 * sy.TimeSpan.SECOND,
                    18 * sy.TimeSpan.SECOND,
                    19 * sy.TimeSpan.SECOND,
                ]
            ).astype(np.int64),
        )
        data_ch.write(
            sy.TimeStamp(15 * sy.TimeSpan.SECOND),
            np.array([15, 16, 17, 18, 19]).astype(np.int64),
        )
        with client.open_iterator(sy.TimeRange.MAX, data_ch.key) as i:
            assert i.seek_ge(sy.TimeStamp(16 * sy.TimeSpan.SECOND))
            assert i.next(4 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [16, 17, 18, 19]

            assert i.seek_last()
            assert i.prev(4 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [16, 17, 18, 19]

            assert i.prev(11 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [5, 10, 11, 12, 13, 15]

            assert i.seek_le(sy.TimeStamp(6 * sy.TimeSpan.SECOND))
            assert i.prev(3 * sy.TimeSpan.SECOND)
            l = i.value.get(data_ch.key).to_numpy().tolist()
            assert l == [3, 4, 5]

            assert not i.seek_le(-1)
            assert not i.seek_ge(sy.TimeStamp(20 * sy.TimeSpan.SECOND))

    def def_test_calculated_channel(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ):
        idx_ch, data_ch = indexed_pair
        idx_ch.write(0, np.array([0, 1, 2, 3, 4, 5]).astype(np.int64))
        data_ch.write(0, np.array([0, 1, 2, 3, 4, 5]).astype(np.int64))
        calc = client.channels.create(
            name="calc",
            expression=f"return 2 * {data_ch.name}",
            data_type=data_ch.data_type,
        )
        data = calc.read(sy.TimeRange.MAX)
        assert np.array_equal(data, np.array([0, 2, 4, 6, 8, 10]))

    def test_read_latest(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)
        data = client.read_latest(data_ch.key, 3)
        assert np.array_equal(data, np.array([8, 9, 10]))

    def test_read_latest_frame(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)
        data = client.read_latest([idx_ch.key, data_ch.key], 3)
        assert np.array_equal(data.get(data_ch.key), np.array([8, 9, 10]))
        assert np.array_equal(
            data.get(idx_ch.key),
            np.array(
                [
                    8 * sy.TimeSpan.SECOND,
                    9 * sy.TimeSpan.SECOND,
                    10 * sy.TimeSpan.SECOND,
                ]
            ),
        )

    def test_read_latest_empty_channel(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test reading latest from an empty channel."""
        idx_ch, data_ch = indexed_pair
        # Don't write any data
        result = client.read_latest(data_ch.key, 5)
        assert len(result) == 0

    def test_read_latest_empty_channel_frame(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test reading latest frame from empty channels."""
        idx_ch, data_ch = indexed_pair
        # Don't write any data
        frame = client.read_latest([idx_ch.key, data_ch.key], 5)
        assert len(frame.get(data_ch.key)) == 0
        assert len(frame.get(idx_ch.key)) == 0

    def test_read_latest_single_sample(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test reading latest when channel has only one sample."""
        idx_ch, data_ch = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, [sy.TimeSpan.SECOND * 1])
        data_ch.write(sy.TimeSpan.SECOND * 1, [42.0])

        # Request more samples than available
        result = client.read_latest(data_ch.key, 5)
        assert np.array_equal(result, np.array([42.0]))

        # Request exactly one sample
        result = client.read_latest(data_ch.key, 1)
        assert np.array_equal(result, np.array([42.0]))

    def test_read_latest_n_zero(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ):
        """Test reading latest with n=0."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # n=0 should return empty result
        result = client.read_latest(data_ch.key, 0)
        assert len(result) == 0

    def test_read_latest_n_negative(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ):
        """Test reading latest with negative n (should handle gracefully)."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # Negative n might be treated as 0 or raise an error
        try:
            result = client.read_latest(data_ch.key, -5)
            # If it doesn't raise, it should return empty or behave like n=0
            assert len(result) == 0
        except ValueError:
            # This is also acceptable behavior
            pass

    def test_read_latest_very_large_n(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test reading latest with n much larger than available data."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 5)
        data = np.arange(1, 6)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # Request 1 million samples when only 5 exist
        result = client.read_latest(data_ch.key, 1_000_000)
        assert np.array_equal(result, np.array([1, 2, 3, 4, 5]))

    def test_read_latest_multiple_channels_different_lengths(self, client: sy.Synnax):
        """Test reading latest from multiple channels with different data lengths."""
        # Create channels with shared index
        idx_ch = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )

        data_ch1 = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )

        data_ch2 = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )

        # Write different amounts of data to each channel
        time_data_long = seconds_linspace(1, 20)

        # Write 20 samples to channel 1
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data_long)
        data_ch1.write(sy.TimeSpan.SECOND * 1, np.arange(1, 21))

        # Write only 5 samples to channel 2 (at the beginning of the time range)
        data_ch2.write(sy.TimeSpan.SECOND * 1, np.arange(100, 105))

        # Request latest 10 samples
        frame = client.read_latest([data_ch1.key, data_ch2.key], 10)

        # Channel 1 should have the last 10 samples
        assert np.array_equal(frame.get(data_ch1.key), np.arange(11, 21))

        # Channel 2 should have only 5 samples (all it has)
        assert len(frame.get(data_ch2.key)) == 5
        assert np.array_equal(frame.get(data_ch2.key), np.arange(100, 105))

    def test_read_latest_sparse_data(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test reading latest with gaps in the data."""
        idx_ch, data_ch = indexed_pair

        # Write data with gaps
        # First chunk: samples at t=1,2,3
        idx_ch.write(
            sy.TimeSpan.SECOND * 1,
            [sy.TimeSpan.SECOND * 1, sy.TimeSpan.SECOND * 2, sy.TimeSpan.SECOND * 3],
        )
        data_ch.write(sy.TimeSpan.SECOND * 1, [1.0, 2.0, 3.0])

        # Second chunk: samples at t=10,11,12 (gap from 3 to 10)
        idx_ch.write(
            sy.TimeSpan.SECOND * 10,
            [sy.TimeSpan.SECOND * 10, sy.TimeSpan.SECOND * 11, sy.TimeSpan.SECOND * 12],
        )
        data_ch.write(sy.TimeSpan.SECOND * 10, [10.0, 11.0, 12.0])

        # Read latest 4 samples - should get the last 4 available
        result = client.read_latest(data_ch.key, 4)
        # Should get [3.0, 10.0, 11.0, 12.0] or just [10.0, 11.0, 12.0] depending on implementation
        assert len(result) >= 3  # At minimum should get the last chunk
        assert 12.0 in result  # Should definitely include the latest value

    def test_read_latest_default_n(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test reading latest with default n=1."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # Call without specifying n (should default to 1)
        result = client.read_latest(data_ch.key)
        assert np.array_equal(result, np.array([10]))

    def test_downsample_factor_2(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test downsampling with factor of 2."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 8)
        data = np.array([1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0], dtype=np.float32)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        with client.open_iterator(
            sy.TimeRange.MAX, [idx_ch.key, data_ch.key], downsample_factor=2
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            # [1, 2, 3, 4, 5, 6, 7, 8] downsampled by 2 = [1, 3, 5, 7]
            result = i.value.get(data_ch.key).to_numpy()
            assert np.array_equal(result, np.array([1.0, 3.0, 5.0, 7.0]))

    def test_downsample_factor_3(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ):
        """Test downsampling with factor of 3."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 9)
        data = np.array([1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0], dtype=np.float32)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        with client.open_iterator(
            sy.TimeRange.MAX, [idx_ch.key, data_ch.key], downsample_factor=3
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            # [1, 2, 3, 4, 5, 6, 7, 8, 9] downsampled by 3 = [1, 4, 7]
            result = i.value.get(data_ch.key).to_numpy()
            assert np.array_equal(result, np.array([1.0, 4.0, 7.0]))

    @pytest.mark.parametrize("factor", [0, 1, -1])
    def test_no_downsample_when_factor_lte_1(
        self,
        factor: int,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ):
        """Test that downsampling does not occur when factor is 0, 1, or negative."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, 4)
        data = np.array([1.0, 2.0, 3.0, 4.0], dtype=np.float32)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        with client.open_iterator(
            sy.TimeRange.MAX, data_ch.key, downsample_factor=factor
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            result = i.value.get(data_ch.key).to_numpy()
            assert np.array_equal(result, np.array([1.0, 2.0, 3.0, 4.0]))

    def test_downsample_multiple_domains(self, client: sy.Synnax):
        """Test downsampling across multiple domains."""
        idx_ch = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        data_ch = client.channels.create(
            name=random_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )

        # First domain
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, 3))
        data_ch.write(
            sy.TimeSpan.SECOND * 1, np.array([1.0, 2.0, 3.0], dtype=np.float32)
        )

        # Second domain (gap)
        idx_ch.write(sy.TimeSpan.SECOND * 10, seconds_linspace(10, 4))
        data_ch.write(
            sy.TimeSpan.SECOND * 10,
            np.array([10.0, 11.0, 12.0, 13.0], dtype=np.float32),
        )

        with client.open_iterator(
            sy.TimeRange.MAX, [idx_ch.key, data_ch.key], downsample_factor=2
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            # Domain 0: [1, 2, 3] downsampled by 2 = [1, 3]
            # Domain 1: [10, 11, 12, 13] downsampled by 2 = [10, 12]
            result = i.value.get(data_ch.key).to_numpy()
            assert np.array_equal(result, np.array([1.0, 3.0, 10.0, 12.0]))
