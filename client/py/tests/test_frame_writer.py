#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
from types import MappingProxyType

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from synnax.framer.frame import FramePayload
from tests.channel import RetrieveCounter
from x.strings import random_name
from x.telem import CrudeSeries, seconds_linspace


@pytest.mark.framer
@pytest.mark.writer
class TestWriter:
    def test_basic_write(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should write a small amount of data to Synnax"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=indexed_pair,
        ) as w:
            w.write(
                {
                    idx_ch.key: 1 * sy.TimeSpan.SECOND,
                    data_ch.key: 123.4,
                }
            )
            w.write(
                {
                    idx_ch.key: 2 * sy.TimeSpan.SECOND,
                    data_ch.key: 123.5,
                }
            )
            w.commit()

    def test_write_without_start(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should start the index at its first timestamp when the writer has no start"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(channels=indexed_pair) as w:
            w.write(
                {idx_ch.key: seconds_linspace(1, count=3), data_ch.key: [1.0, 2.0, 3.0]}
            )
            w.commit()
        frame = client.read(sy.TimeRange.MAX, [idx_ch.key, data_ch.key])
        assert frame[idx_ch.key].to_numpy()[0] == 1 * sy.TimeSpan.SECOND
        assert list(frame[data_ch.key]) == [1.0, 2.0, 3.0]

    def test_open_writer_without_channels(self, client: sy.Synnax) -> None:
        """Should fail to open a writer with no channels"""
        with pytest.raises(sy.ValidationError, match="open_writer requires channels"):
            client.open_writer()

    def test_write_by_name(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should write data by name to the Synnax cluster"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND, channels=indexed_pair
        ) as w:
            w.write(
                pd.DataFrame(
                    {
                        idx_ch.name: [1 * sy.TimeSpan.SECOND],
                        data_ch.name: 253.2,
                    }
                )
            )
            w.write(
                pd.DataFrame(
                    {
                        idx_ch.name: [2 * sy.TimeSpan.SECOND],
                        data_ch.name: 253.3,
                    }
                )
            )
            w.commit()

    def test_write_by_name_sends_no_retrieves(
        self, counted_client: tuple[sy.Synnax, RetrieveCounter]
    ) -> None:
        """Should resolve channel names once when the writer opens"""
        client, counter = counted_client
        v = random_name()
        idx = client.channels.create(
            name=f"test_{v}_time", is_index=True, data_type=sy.DataType.TIMESTAMP
        )
        data = client.channels.create(
            name=f"test_{v}_data", index=idx.key, data_type=sy.DataType.FLOAT64
        )
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND, channels=[idx.name, data.name]
        ) as w:
            counter.count = 0
            for i in range(1, 101):
                w.write({idx.name: i * sy.TimeSpan.SECOND, data.name: float(i)})
                w.set_authority({data.name: 200})
            assert counter.count == 0

    def test_write_frame_unknown_channel_name(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should throw a validation error when writing to an unknown channel"""
        with client.open_writer(start=sy.TimeStamp.now(), channels=indexed_pair) as w:
            data = np.random.rand(10).astype(np.float64)
            with pytest.raises(sy.ValidationError):
                w.write(pd.DataFrame({"missing": data}))

    def test_write_frame_unknown_channel_key(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should throw a validation error when writing an unknown frame by key"""
        with client.open_writer(start=sy.TimeStamp.now(), channels=indexed_pair) as w:
            data = np.random.rand(10).astype(np.float64)
            with pytest.raises(sy.ValidationError):
                w.write(pd.DataFrame({123: data}))

    def test_write_frame_idx_no_timestamps(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should throw a validation error on close when writing an indexed frame
        w/o the correct timing
        """
        [idx, data_ch] = indexed_pair
        with (
            pytest.raises(
                sy.ValidationError, match="received no data for index channel"
            ),
            client.open_writer(0, [idx.key, data_ch.key]) as w,
        ):
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({data_ch.key: data}))

    def test_auto_index_generates_timestamps_for_data_only_writer(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should generate timestamps for a writer opened with only a data channel
        when auto_index is true."""
        idx_ch, data_ch = indexed_pair
        before = sy.TimeStamp.now()
        with client.open_writer(
            start=before, channels=[data_ch.key], auto_index=True
        ) as w:
            data = np.random.rand(4).astype(np.float64)
            w.write({data_ch.key: data})
            w.commit()
        f = client.read(
            sy.TimeRange(before, sy.TimeStamp.now() + sy.TimeSpan.SECOND), idx_ch.key
        )
        assert len(f) == 4

    def test_auto_index_omitted_index_in_mixed_writer(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should auto-stamp the index when the caller opens with both index and data
        channels but omits the index series from a write."""
        idx_ch, data_ch = indexed_pair
        before = sy.TimeStamp.now()
        with client.open_writer(
            start=before, channels=[idx_ch.key, data_ch.key], auto_index=True
        ) as w:
            data = np.random.rand(3).astype(np.float64)
            w.write({data_ch.key: data})
            w.commit()
        f = client.read(
            sy.TimeRange(before, sy.TimeStamp.now() + sy.TimeSpan.SECOND), idx_ch.key
        )
        assert len(f) == 3

    def test_auto_index_mixes_user_provided_and_generated_in_one_writer(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should let a single writer alternate between user-provided and auto-stamped
        index series across separate write calls."""
        idx_ch, data_ch = indexed_pair
        user_stamps = seconds_linspace(200, count=2)
        before_auto = sy.TimeStamp.now()
        with client.open_writer(
            start=200 * sy.TimeSpan.SECOND,
            channels=[idx_ch.key, data_ch.key],
            auto_index=True,
        ) as w:
            w.write(
                {
                    idx_ch.key: user_stamps,
                    data_ch.key: np.random.rand(2).astype(np.float64),
                }
            )
            w.write({data_ch.key: np.random.rand(3).astype(np.float64)})
            w.commit()
        user_f = client.read(
            sy.TimeRange(200 * sy.TimeSpan.SECOND, 202 * sy.TimeSpan.SECOND), idx_ch.key
        )
        assert list(user_f) == user_stamps
        auto_f = client.read(
            sy.TimeRange(before_auto, sy.TimeStamp.now() + sy.TimeSpan.SECOND),
            idx_ch.key,
        )
        assert len(auto_f) == 3

    def test_auto_index_leaves_user_provided_index_untouched(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should leave user-provided index timestamps exactly as written when
        auto_index is enabled."""
        idx_ch, data_ch = indexed_pair
        expected = seconds_linspace(200, count=3)
        with client.open_writer(
            start=200 * sy.TimeSpan.SECOND,
            channels=[idx_ch.key, data_ch.key],
            auto_index=True,
        ) as w:
            w.write(
                {
                    idx_ch.key: expected,
                    data_ch.key: np.random.rand(3).astype(np.float64),
                }
            )
            w.commit()
        f = client.read(
            sy.TimeRange(200 * sy.TimeSpan.SECOND, 203 * sy.TimeSpan.SECOND), idx_ch.key
        )
        assert list(f) == expected

    def test_client_write_non_dict_mapping(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should write a non-dict mapping of channel names to series."""
        idx_ch, data_ch = indexed_pair
        client.write(
            1 * sy.TimeSpan.SECOND,
            MappingProxyType(
                {
                    idx_ch.name: seconds_linspace(1, count=3),
                    data_ch.name: np.array([1.0, 2.0, 3.0]),
                }
            ),
        )
        f = client.read(sy.TimeRange.MAX, data_ch.key)
        assert list(f) == [1.0, 2.0, 3.0]

    def test_client_write_channel_tuple(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should write a tuple of channels and a list of series."""
        idx_ch, data_ch = indexed_pair
        series: list[CrudeSeries] = [seconds_linspace(1, count=3), [1.0, 2.0, 3.0]]
        client.write(1 * sy.TimeSpan.SECOND, (idx_ch.key, data_ch.key), series)
        f = client.read(sy.TimeRange.MAX, data_ch.key)
        assert list(f) == [1.0, 2.0, 3.0]

    def test_client_write_frame_payload(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should write a frame payload."""
        idx_ch, data_ch = indexed_pair
        client.write(
            1 * sy.TimeSpan.SECOND,
            FramePayload(
                keys=[idx_ch.key, data_ch.key],
                series=[
                    sy.Series(seconds_linspace(1, count=3)),
                    sy.Series(np.array([1.0, 2.0, 3.0])),
                ],
            ),
        )
        f = client.read(sy.TimeRange.MAX, data_ch.key)
        assert list(f) == [1.0, 2.0, 3.0]

    def test_write_auto_commit(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should open an auto-committing writer to write data that persists after 1s"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=sy.TimeSpan.SECOND * 1, channels=indexed_pair
        ) as w:
            data = np.random.rand(3).astype(np.float64)
            w.write(
                pd.DataFrame(
                    {idx_ch.name: seconds_linspace(1, count=3), data_ch.key: data}
                )
            )
            w.write(
                pd.DataFrame(
                    {idx_ch.key: seconds_linspace(4, count=3), data_ch.name: data}
                )
            )

        f = client.read(sy.TimeRange.MAX, data_ch.key)
        assert len(f) == 6

    def test_write_bool_channel(self, client: sy.Synnax) -> None:
        """Should round-trip bool samples through a BOOLEAN channel, verifying
        bit-packed wire format and byte-packed persistence."""
        idx = client.channels.create(
            name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        bool_ch = client.channels.create(
            name=random_name(), data_type=sy.DataType.BOOLEAN, index=idx.key
        )
        samples = [True, False, True, True, False, False, False, True, True]
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND, channels=[idx, bool_ch]
        ) as w:
            w.write(
                pd.DataFrame(
                    {
                        idx.name: seconds_linspace(1, count=len(samples)),
                        bool_ch.name: samples,
                    }
                )
            )
            w.commit()

        f = client.read(sy.TimeRange.MAX, bool_ch.key)
        assert len(f) == len(samples)
        assert [bool(v) for v in list(f)] == samples

    def test_write_err_on_unauthorized(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should throw an error when a writer is opened to error on unauthorized"""
        start = sy.TimeStamp.now()
        w1 = client.open_writer(start=start, channels=indexed_pair, name="writer1")
        with (
            pytest.raises(sy.UnauthorizedError),
            client.open_writer(
                start=start,
                channels=indexed_pair,
                err_on_unauthorized=True,
                name="writer2",
            ),
        ):
            ...

        w1.close()

    def test_write_out_of_order_timestamps(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        """Should raise a validation error when writing timestamps that are out of
        order"""
        time_ch, data_ch = indexed_pair
        with pytest.raises(sy.ValidationError, match="commit timestamp"):
            with client.open_writer(
                start=sy.TimeSpan.SECOND,
                channels=[time_ch.key, data_ch.key],
            ) as w:
                for i in range(100):
                    time = sy.TimeSpan.SECOND * (101 - i)
                    w.write({time_ch.key: time, data_ch.key: [i]})

    def test_write_invalid_data_type(
        self,
        client: sy.Synnax,
        indexed_pair: tuple[sy.Channel, sy.Channel],
    ) -> None:
        """Should raise a validation error when writing a series with an invalid
        data type"""
        time_ch, data_ch = indexed_pair
        with pytest.raises(sy.ValidationError, match="type"):
            with client.open_writer(
                start=sy.TimeSpan.SECOND,
                channels=[time_ch.key, data_ch.key],
                strict=True,
            ) as w:
                w.write(
                    {
                        time_ch.key: [sy.TimeStamp.now()],
                        data_ch.key: sy.Series([1], data_type=sy.DataType.INT64),
                    }
                )

    @pytest.mark.asyncio
    async def test_write_persist_only_mode(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should not stream written data"""
        idx_ch, data_ch = indexed_pair
        with client.open_writer(
            start=sy.TimeSpan.SECOND * 1,
            channels=indexed_pair,
            mode=sy.WriterMode.PERSIST,
        ) as w:
            async with await client.open_async_streamer(indexed_pair) as s:
                data = np.random.rand(5).astype(np.float64)
                times = seconds_linspace(1, count=5)
                w.write(pd.DataFrame({idx_ch.key: times, data_ch.key: data}))
                with pytest.raises(TimeoutError):
                    async with asyncio.timeout(0.2):
                        await s.read()

    def test_write_persist_behind_live_writer(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should fill a gap before the session of a live writer"""
        idx_ch, data_ch = indexed_pair
        channels = [idx_ch.key, data_ch.key]
        start = 10 * sy.TimeSpan.SECOND
        with client.open_writer(start=start, channels=channels) as live:
            live.write(
                {idx_ch.key: seconds_linspace(10, count=2), data_ch.key: [10.0, 11.0]}
            )
            live.commit()
            client.write(
                1 * sy.TimeSpan.SECOND,
                {idx_ch.key: seconds_linspace(1, count=2), data_ch.key: [1.0, 2.0]},
            )
            live.write({idx_ch.key: seconds_linspace(12, count=1), data_ch.key: [12.0]})
            live.commit()
        assert list(client.read(sy.TimeRange.MAX, data_ch.key)) == [
            1.0,
            2.0,
            10.0,
            11.0,
            12.0,
        ]

    def test_write_persist_inside_live_writer(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        """Should fail to open a persist writer in the session of a live writer"""
        idx_ch, data_ch = indexed_pair
        channels = [idx_ch.key, data_ch.key]
        with client.open_writer(start=10 * sy.TimeSpan.SECOND, channels=channels):
            with pytest.raises(
                sy.UnauthorizedError, match="overlaps with a controlled region"
            ):
                client.open_writer(
                    start=15 * sy.TimeSpan.SECOND,
                    channels=channels,
                    mode=sy.WriterMode.PERSIST,
                )

    def test_write_persist_stream_regression(self, client: sy.Synnax) -> None:
        """Should work"""
        idx = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type="timestamp",
        )
        data = client.channels.create(
            name=random_name(),
            data_type="float64",
            index=idx.key,
        )
        # Write some data
        start = sy.TimeStamp.now()
        with client.open_writer(start, [idx.key, data.key]) as w:
            w.write({idx.key: [start], data.key: [1]})

        # Read the data
        next_start = start + 5 * sy.TimeSpan.MILLISECOND
        f = client.read(
            sy.TimeRange(start - 5 * sy.TimeSpan.MILLISECOND, next_start), data.key
        )
        assert len(f) == 1

        data_2 = client.channels.create(
            name=random_name(),
            data_type="float64",
            index=idx.key,
        )
        data_3 = client.channels.create(
            name=random_name(),
            data_type="float64",
            index=idx.key,
        )
        with client.open_writer(
            next_start, [idx.key, data.key, data_2.key, data_3.key]
        ) as w:
            w.write(
                {idx.key: [next_start], data.key: [1], data_2.key: [2], data_3.key: [3]}
            )

        tr = sy.TimeRange(
            start - 5 * sy.TimeSpan.MILLISECOND,
            next_start + 5 * sy.TimeSpan.MILLISECOND,
        )
        f = client.read(tr, data_2.key)
        assert len(f) == 1
        f2 = client.read(tr, [data.key, data_2.key, data_3.key])
        assert len(f2[data.key]) == 2
        assert len(f2[data_2.key]) == 1
        assert len(f2[data_3.key]) == 1

    def test_set_authority(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({data_ch.key: 255, idx_ch.key: 255})
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_data_channel_only(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        """A per-channel authority change naming only a data channel must also
        cover its index, since a data sample cannot be written without its
        timestamp."""
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({data_ch.key: 255})
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({data_ch.name: 255, idx_ch.name: 255})
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name_data_channel_only(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        """The by-name form of a per-channel authority change naming only a
        data channel must also cover its index."""
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({data_ch.name: 255})
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name_value_data_channel_only(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        """The (name, value) form of a per-channel authority change naming only
        a data channel must also cover its index."""
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(data_ch.name, authority=255)
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name_value(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(data_ch.name, authority=255)
            w1.set_authority(idx_ch.name, authority=255)
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_writer_overlap_err(
        self,
        client: sy.Synnax,
        indexed_pair: tuple[sy.Channel, sy.Channel],
    ) -> None:
        idx_ch, data_ch = indexed_pair
        start = sy.TimeSpan.SECOND * 30
        with client.open_writer(start=start, channels=indexed_pair) as w:
            w.write(
                {
                    idx_ch.key: seconds_linspace(30, count=10),
                    data_ch.key: np.random.rand(10).astype(np.float64),
                }
            )

        with (
            pytest.raises(sy.ValidationError),
            client.open_writer(
                start=start + sy.TimeSpan.SECOND * 3, channels=indexed_pair
            ),
        ):
            ...

    def test_set_authority_on_all_channels(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        start = sy.TimeSpan.SECOND * 1
        idx_ch, data_ch = indexed_pair
        w1 = client.open_writer(start=start, channels=indexed_pair, authorities=100)
        w2 = client.open_writer(start=start, channels=indexed_pair, authorities=200)
        try:
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(255)
            w1.write(
                pd.DataFrame(
                    {
                        idx_ch.key: seconds_linspace(1, count=10),
                        data_ch.key: np.random.rand(10).astype(np.float64),
                    }
                )
            )
            f = data_ch.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_writer_close_idempotency(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should allow the caller to call close() as many times as they want"""
        idx_ch, data_ch = indexed_pair
        w = client.open_writer(start=1 * sy.TimeSpan.SECOND, channels=indexed_pair)
        w.write(
            {
                idx_ch.key: 2 * sy.TimeSpan.SECOND,
                data_ch.key: 123.5,
            }
        )
        w.commit()
        w.close()
        w.close()
        w.close()

    def test_write_string_channel(self, client: sy.Synnax) -> None:
        """Should write and read persisted string data"""
        idx = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        data = client.channels.create(
            name=random_name(),
            index=idx.key,
            data_type=sy.DataType.STRING,
        )
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=[idx.key, data.key],
        ) as w:
            w.write(
                {
                    idx.key: seconds_linspace(1, count=2),
                    data.key: ["hello", "world"],
                }
            )
            w.commit()
        f = client.read(sy.TimeRange.MAX, data.key)
        assert list(f) == ["hello", "world"]

    def test_write_json_channel(self, client: sy.Synnax) -> None:
        """Should write and read persisted JSON data"""
        idx = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        data = client.channels.create(
            name=random_name(),
            index=idx.key,
            data_type=sy.DataType.JSON,
        )
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=[idx.key, data.key],
        ) as w:
            samples: list[dict[str, str | int]] = [{"key": "value"}, {"num": 42}]
            w.write(
                {
                    idx.key: seconds_linspace(1, count=2),
                    data.key: samples,
                }
            )
            w.commit()
        f = client.read(sy.TimeRange.MAX, data.key)
        assert len(f) == 2
        assert f[0] == {"key": "value"}
        assert f[1] == {"num": 42}

    def test_write_mixed_fixed_and_variable(self, client: sy.Synnax) -> None:
        """Should write fixed and variable channels sharing an index"""
        idx = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        float_ch = client.channels.create(
            name=random_name(),
            index=idx.key,
            data_type=sy.DataType.FLOAT64,
        )
        str_ch = client.channels.create(
            name=random_name(),
            index=idx.key,
            data_type=sy.DataType.STRING,
        )
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=[idx.key, float_ch.key, str_ch.key],
        ) as w:
            w.write(
                {
                    idx.key: seconds_linspace(1, count=3),
                    float_ch.key: np.array([1.1, 2.2, 3.3], dtype=np.float64),
                    str_ch.key: ["a", "b", "c"],
                }
            )
            w.commit()
        f = client.read(sy.TimeRange.MAX, [float_ch.key, str_ch.key])
        assert len(f[float_ch.key]) == 3
        assert list(f[str_ch.key]) == ["a", "b", "c"]

    def test_write_string_with_newlines(self, client: sy.Synnax) -> None:
        """Should persist strings containing newlines correctly"""
        idx = client.channels.create(
            name=random_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        data = client.channels.create(
            name=random_name(),
            index=idx.key,
            data_type=sy.DataType.STRING,
        )
        with client.open_writer(
            start=1 * sy.TimeSpan.SECOND,
            channels=[idx.key, data.key],
        ) as w:
            w.write(
                {
                    idx.key: seconds_linspace(1, count=2),
                    data.key: ["line1\nline2", "no newline"],
                }
            )
            w.commit()
        f = client.read(sy.TimeRange.MAX, data.key)
        assert list(f) == ["line1\nline2", "no newline"]
