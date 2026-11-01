#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from collections.abc import Callable

import numpy as np
import pydantic
import pytest

import synnax as sy
from x.strings import random_name
from x.telem import seconds_linspace


@pytest.mark.framer
@pytest.mark.iterator
class TestIterator:
    def test_basic_iterate(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        idx_ch, _ = indexed_pair
        d = seconds_linspace(1, count=50)
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key) as i:
            for f in i:
                assert np.array_equal(f[idx_ch.key], d)

    def test_valid(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should be valid after a read and invalid after a seek."""
        idx_ch, _ = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=5))
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key) as i:
            assert i.seek_first()
            assert not i.valid()
            assert i.next(sy.framer.AUTO_SPAN)
            assert i.valid()

    def test_auto_chunk(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        d = seconds_linspace(1, count=10)
        idx_ch, _ = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key, chunk_size=4) as i:
            assert i.seek_first()
            i.next(sy.framer.AUTO_SPAN)
            values = i.value[idx_ch.key].to_numpy().tolist()
            assert np.array_equal(values, seconds_linspace(1, count=4))

            i.next(sy.framer.AUTO_SPAN)
            values = i.value[idx_ch.key].to_numpy().tolist()
            assert np.array_equal(values, seconds_linspace(5, count=4))

            i.next(sy.framer.AUTO_SPAN)
            values = i.value[idx_ch.key].to_numpy().tolist()
            assert np.array_equal(values, seconds_linspace(9, count=2))

            assert not i.next(sy.framer.AUTO_SPAN)

    def test_auto_chunk_reverse(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        d = seconds_linspace(1, count=10)
        idx_ch, _ = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, d)
        with client.open_iterator(sy.TimeRange.MAX, idx_ch.key, chunk_size=4) as i:
            assert i.seek_last()
            assert i.prev(sy.framer.AUTO_SPAN)
            values = i.value[idx_ch.key].to_numpy().tolist()
            assert np.array_equal(values, seconds_linspace(7, count=4))

            assert i.prev(sy.framer.AUTO_SPAN)
            values = i.value[idx_ch.key].to_numpy().tolist()
            assert np.array_equal(values, seconds_linspace(3, count=4))

            assert i.prev(sy.framer.AUTO_SPAN)
            values = i.value[idx_ch.key].to_numpy().tolist()
            assert np.array_equal(values, seconds_linspace(1, count=2))

            assert not i.prev(sy.framer.AUTO_SPAN)

    def test_advanced_iterate(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        idx_ch, data_ch = indexed_pair
        for block in ([0, 1, 2, 3, 4, 5], [10, 11, 12, 13], [15, 16, 17, 18, 19]):
            seconds = np.array(block).astype(np.int64)
            client.write(
                sy.TimeStamp(block[0] * sy.TimeSpan.SECOND),
                {idx_ch.key: seconds * int(sy.TimeSpan.SECOND), data_ch.key: seconds},
            )
        with client.open_iterator(sy.TimeRange.MAX, data_ch.key) as i:
            assert i.seek_ge(sy.TimeStamp(16 * sy.TimeSpan.SECOND))
            assert i.next(4 * sy.TimeSpan.SECOND)
            values = i.value[data_ch.key].to_numpy().tolist()
            assert values == [16, 17, 18, 19]

            assert i.seek_last()
            assert i.prev(4 * sy.TimeSpan.SECOND)
            values = i.value[data_ch.key].to_numpy().tolist()
            assert values == [16, 17, 18, 19]

            assert i.prev(11 * sy.TimeSpan.SECOND)
            values = i.value[data_ch.key].to_numpy().tolist()
            assert values == [5, 10, 11, 12, 13, 15]

            assert i.seek_le(sy.TimeStamp(6 * sy.TimeSpan.SECOND))
            assert i.prev(3 * sy.TimeSpan.SECOND)
            values = i.value[data_ch.key].to_numpy().tolist()
            assert values == [3, 4, 5]

            assert not i.seek_le(sy.TimeStamp(-1))
            assert not i.seek_ge(sy.TimeStamp(20 * sy.TimeSpan.SECOND))

    def def_test_calculated_channel(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
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
    ) -> None:
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)
        latest = client.read_latest(data_ch.key, n=3)
        assert np.array_equal(latest, np.array([8, 9, 10]))

    def test_read_latest_single_name(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a MultiSeries for a single name"""
        idx_ch, data_ch = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=10))
        data_ch.write(sy.TimeSpan.SECOND * 1, np.arange(1, 11))
        latest = client.read_latest(data_ch.name, n=3)
        assert isinstance(latest, sy.MultiSeries)
        assert np.array_equal(latest, np.array([8, 9, 10]))

    def test_read_latest_one_key_list(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a Frame for a list with one key"""
        idx_ch, data_ch = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=10))
        data_ch.write(sy.TimeSpan.SECOND * 1, np.arange(1, 11))
        frame = client.read_latest([data_ch.key], n=3)
        assert isinstance(frame, sy.Frame)
        assert np.array_equal(frame[data_ch.key], np.array([8, 9, 10]))

    def test_read_latest_one_name_list(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should return a Frame for a list with one name"""
        idx_ch, data_ch = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=10))
        data_ch.write(sy.TimeSpan.SECOND * 1, np.arange(1, 11))
        frame = client.read_latest([data_ch.name], n=3)
        assert isinstance(frame, sy.Frame)
        assert np.array_equal(frame[data_ch.name], np.array([8, 9, 10]))

    def test_read_latest_frame(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)
        frame = client.read_latest([idx_ch.key, data_ch.key], n=3)
        assert np.array_equal(frame[data_ch.key], np.array([8, 9, 10]))
        assert np.array_equal(
            frame[idx_ch.key],
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
    ) -> None:
        """Test reading latest from an empty channel."""
        _idx_ch, data_ch = indexed_pair
        # Don't write any data
        result = client.read_latest(data_ch.key, n=5)
        assert len(result) == 0

    def test_read_latest_empty_channel_frame(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Test reading latest frame from empty channels."""
        idx_ch, data_ch = indexed_pair
        # Don't write any data
        frame = client.read_latest([idx_ch.key, data_ch.key], n=5)
        assert len(frame[data_ch.key]) == 0
        assert len(frame[idx_ch.key]) == 0

    def test_read_latest_single_sample(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Test reading latest when channel has only one sample."""
        idx_ch, data_ch = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=1))
        data_ch.write(sy.TimeSpan.SECOND * 1, [42.0])

        # Request more samples than available
        result = client.read_latest(data_ch.key, n=5)
        assert np.array_equal(result, np.array([42.0]))

        # Request exactly one sample
        result = client.read_latest(data_ch.key, n=1)
        assert np.array_equal(result, np.array([42.0]))

    def test_read_latest_n_zero(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Test reading latest with n=0."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # n=0 should return empty result
        result = client.read_latest(data_ch.key, n=0)
        assert len(result) == 0

    def test_read_latest_n_negative(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Test reading latest with negative n (should handle gracefully)."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # Negative n might be treated as 0 or raise an error
        try:
            result = client.read_latest(data_ch.key, n=-5)
            # If it doesn't raise, it should return empty or behave like n=0
            assert len(result) == 0
        except ValueError:
            # This is also acceptable behavior
            pass

    def test_read_latest_very_large_n(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Test reading latest with n much larger than available data."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=5)
        data = np.arange(1, 6)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # Request 1 million samples when only 5 exist
        result = client.read_latest(data_ch.key, n=1_000_000)
        assert np.array_equal(result, np.array([1, 2, 3, 4, 5]))

    def test_read_latest_multiple_channels_different_lengths(
        self, client: sy.Synnax
    ) -> None:
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
        time_data_long = seconds_linspace(1, count=20)

        # Write 20 samples to channel 1
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data_long)
        data_ch1.write(sy.TimeSpan.SECOND * 1, np.arange(1, 21))

        # Write only 5 samples to channel 2 (at the beginning of the time range)
        data_ch2.write(sy.TimeSpan.SECOND * 1, np.arange(100, 105))

        # Request latest 10 samples
        frame = client.read_latest([data_ch1.key, data_ch2.key], n=10)

        # Channel 1 should have the last 10 samples
        assert np.array_equal(frame[data_ch1.key], np.arange(11, 21))

        # Channel 2 should have only 5 samples (all it has)
        assert len(frame[data_ch2.key]) == 5
        assert np.array_equal(frame[data_ch2.key], np.arange(100, 105))

    def test_read_latest_sparse_data(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Test reading latest with gaps in the data."""
        idx_ch, data_ch = indexed_pair

        # Write data with gaps
        # First chunk: samples at t=1,2,3
        idx_ch.write(
            sy.TimeSpan.SECOND * 1,
            seconds_linspace(1, count=3),
        )
        data_ch.write(sy.TimeSpan.SECOND * 1, [1.0, 2.0, 3.0])

        # Second chunk: samples at t=10,11,12 (gap from 3 to 10)
        idx_ch.write(
            sy.TimeSpan.SECOND * 10,
            seconds_linspace(10, count=3),
        )
        data_ch.write(sy.TimeSpan.SECOND * 10, [10.0, 11.0, 12.0])

        # Read latest 4 samples - should get the last 4 available
        result = client.read_latest(data_ch.key, n=4)
        # Should get [3.0, 10.0, 11.0, 12.0] or just [10.0, 11.0, 12.0] depending on
        # implementation
        assert len(result) >= 3  # At minimum should get the last chunk
        assert 12.0 in result  # Should definitely include the latest value

    def test_read_latest_default_n(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Test reading latest with default n=1."""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=10)
        data = np.arange(1, 11)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        # Call without specifying n (should default to 1)
        result = client.read_latest(data_ch.key)
        assert np.array_equal(result, np.array([10]))

    def test_stride_factor_2(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should keep every second sample and report a multiple of 2"""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=8)
        data = np.array([1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0], dtype=np.float32)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        with client.open_iterator(
            sy.TimeRange.MAX,
            [idx_ch.key, data_ch.key],
            reduction=sy.StrideReduction(factor=2),
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            series = i.value[data_ch.key]
            assert np.array_equal(series.to_numpy(), np.array([1.0, 3.0, 5.0, 7.0]))
            assert series.series[0].alignment_multiple == 2

    def test_stride_factor_3(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should keep every third sample under a factor of 3"""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=9)
        data = np.array([1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0], dtype=np.float32)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        with client.open_iterator(
            sy.TimeRange.MAX,
            [idx_ch.key, data_ch.key],
            reduction=sy.StrideReduction(factor=3),
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            result = i.value[data_ch.key].to_numpy()
            assert np.array_equal(result, np.array([1.0, 4.0, 7.0]))

    @pytest.mark.parametrize("factor", [0, 1])
    def test_stride_factor_lte_1(
        self,
        factor: int,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should keep every sample under a factor of 0 or 1"""
        idx_ch, data_ch = indexed_pair
        time_data = seconds_linspace(1, count=4)
        data = np.array([1.0, 2.0, 3.0, 4.0], dtype=np.float32)
        idx_ch.write(sy.TimeSpan.SECOND * 1, time_data)
        data_ch.write(sy.TimeSpan.SECOND * 1, data)

        with client.open_iterator(
            sy.TimeRange.MAX, data_ch.key, reduction=sy.StrideReduction(factor=factor)
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            result = i.value[data_ch.key].to_numpy()
            assert np.array_equal(result, np.array([1.0, 2.0, 3.0, 4.0]))

    def test_stride_multiple_domains(self, client: sy.Synnax) -> None:
        """Should keep every factor-th sample of each domain"""
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
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=3))
        data_ch.write(
            sy.TimeSpan.SECOND * 1, np.array([1.0, 2.0, 3.0], dtype=np.float32)
        )

        # Second domain (gap)
        idx_ch.write(sy.TimeSpan.SECOND * 10, seconds_linspace(10, count=4))
        data_ch.write(
            sy.TimeSpan.SECOND * 10,
            np.array([10.0, 11.0, 12.0, 13.0], dtype=np.float32),
        )

        with client.open_iterator(
            sy.TimeRange.MAX,
            [idx_ch.key, data_ch.key],
            reduction=sy.StrideReduction(factor=2),
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            result = i.value[data_ch.key].to_numpy()
            assert np.array_equal(result, np.array([1.0, 3.0, 10.0, 12.0]))

    @pytest.mark.parametrize(
        ("aggregation", "point_limit", "expected", "multiple"),
        [
            ("min_max", 4, [1.0, 8.0, 2.0, 7.0], 2),
            ("average", 4, [2.5, 5.5, 3.5, 6.5], 2),
            ("decimate", 2, [4.0, 5.0], 4),
            ("min_max", 8, [4.0, 1.0, 3.0, 8.0, 5.0, 2.0, 6.0, 7.0], 1),
        ],
    )
    def test_aggregation(
        self,
        aggregation: sy.Aggregation,
        point_limit: int,
        expected: list[float],
        multiple: int,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should reduce each channel under the point limit with the aggregation."""
        idx_ch, data_ch = indexed_pair
        idx_ch.write(sy.TimeSpan.SECOND * 1, seconds_linspace(1, count=8))
        data_ch.write(
            sy.TimeSpan.SECOND * 1,
            np.array([4.0, 1.0, 3.0, 8.0, 5.0, 2.0, 6.0, 7.0], dtype=np.float32),
        )
        with client.open_iterator(
            sy.TimeRange.MAX,
            [idx_ch.key, data_ch.key],
            reduction=sy.LimitReduction(
                aggregation=aggregation, point_limit=point_limit
            ),
        ) as i:
            assert i.seek_first()
            assert i.next(sy.framer.AUTO_SPAN)
            data = i.value[data_ch.key].series[0]
            index = i.value[idx_ch.key].series[0]
            assert np.array_equal(data.to_numpy(), np.array(expected))
            assert data.alignment_multiple == multiple
            assert data.alignment == index.alignment
            assert not i.next(sy.framer.AUTO_SPAN)

    @pytest.mark.parametrize(
        ("make", "field"),
        [
            (lambda: sy.StrideReduction(factor=-1), "factor"),
            (lambda: sy.LimitReduction(point_limit=0), "point_limit"),
            (lambda: sy.LimitReduction(point_limit=-1), "point_limit"),
            (lambda: sy.LimitReduction(point_limit=2**32), "point_limit"),
        ],
        ids=["negative_factor", "zero_limit", "negative_limit", "limit_overflow"],
    )
    def test_reduction_out_of_range(
        self, make: Callable[[], sy.Reduction], field: str
    ) -> None:
        """Should reject a reduction the unsigned wire fields would reinterpret."""
        with pytest.raises(pydantic.ValidationError, match=field):
            make()
