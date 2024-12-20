#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import time

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from synnax import TimeSpan, TimeRange, TimeStamp, UnauthorizedError


@pytest.mark.framer
class TestChannelWriteRead:
    def test_write_read(self, client: sy.Synnax):
        """Should create a channel and write then read from it"""
        channel = client.channels.create(
            sy.Channel(
                name="test",
                rate=1 * sy.Rate.HZ,
                data_type=sy.DataType.INT64,
            )
        )
        d = np.array([1, 2, 3, 4, 5], dtype=np.int64)
        start = 1 * sy.TimeSpan.SECOND
        channel.write(start, d)
        data = channel.read(start, (start + len(d)) * sy.TimeSpan.SECOND)
        assert data.time_range.start == start
        assert len(d) == len(data)
        assert data.time_range.end == start + (len(d) - 1) * sy.TimeSpan.SECOND + 1
        assert np.array_equal(data, d)


@pytest.mark.framer
@pytest.mark.iterator
class TestIterator:
    def test_basic_iterate(self, channel: sy.Channel, client: sy.Synnax):
        d = np.array([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]).astype(np.float64)
        channel.write(0, d)
        with client.open_iterator(sy.TimeRange.MAX, channel.key) as i:
            for f in i:
                assert np.array_equal(f.get(channel.key), d)

    def test_auto_chunk(self, channel: sy.Channel, client: sy.Synnax):
        d = np.array([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]).astype(np.float64)
        channel.write(0, d)
        with client.open_iterator(sy.TimeRange.MAX, channel.key, chunk_size=4) as i:
            assert i.seek_first()
            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(channel.key).to_numpy().tolist()
            assert l == [0, 1, 2, 3]

            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(channel.key).to_numpy().tolist()
            assert l == [4, 5, 6, 7]

            i.next(sy.framer.AUTO_SPAN)
            l = i.value.get(channel.key).to_numpy().tolist()
            assert l == [8, 9]

            assert not i.next(sy.framer.AUTO_SPAN)

    def test_advanced_iterate(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ):
        [idx_ch, data_ch] = indexed_pair
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
            TimeStamp(10 * sy.TimeSpan.SECOND),
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
            TimeStamp(10 * sy.TimeSpan.SECOND),
            np.array([10, 11, 12, 13]).astype(np.int64),
        )
        idx_ch.write(
            TimeStamp(15 * sy.TimeSpan.SECOND),
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
            TimeStamp(15 * sy.TimeSpan.SECOND),
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


@pytest.mark.framer
@pytest.mark.writer
class TestWriter:
    def test_basic_write(self, channel: sy.Channel, client: sy.Synnax):
        """Should write data to the Synnax database"""
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()

    def test_write_by_name(self, channel: sy.Channel, client: sy.Synnax):
        """Should write data by name to the Synnax cluster"""
        with client.open_writer(0, channel.name) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.name: data}))
            w.commit()

    def test_write_frame_unknown_channel_name(
        self,
        channel: sy.Channel,
        client: sy.Synnax,
    ):
        """Should throw a validation error when writing to an unknown channel"""
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(10).astype(np.float64)
            with pytest.raises(sy.ValidationError):
                w.write(pd.DataFrame({"missing": data}))

    def test_write_frame_unknown_channel_key(
        self,
        channel: sy.Channel,
        client: sy.Synnax,
    ):
        """Should throw a validation error when writing an unknown frame by key"""
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(10).astype(np.float64)
            with pytest.raises(sy.ValidationError):
                w.write(pd.DataFrame({123: data}))

    def test_write_frame_idx_no_timestamps(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ):
        """Should throw a validation error on close when writing an indexed frame
        w/o the correct timing
        """
        [idx, data_ch] = indexed_pair
        with pytest.raises(sy.ValidationError):
            with client.open_writer(0, [idx.key, data_ch.key]) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({data_ch.key: data}))

    def test_write_auto_commit(self, channel: sy.Channel, client: sy.synnax):
        """Should open an auto-committing writer to write data that persists after 1s"""
        with client.open_writer(0, channel.key, enable_auto_commit=True) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.write(pd.DataFrame({channel.key: data}))
            assert w.error() is None

        f = client.read(TimeRange(0, TimeStamp(1 * TimeSpan.SECOND)), channel.key)
        assert f.__len__() == 20

    def test_write_auto_commit_always_persist(
        self, channel: sy.Channel, client: sy.Synnax
    ):
        """Should open an auto-committing writer to write data to Synnax."""
        with client.open_writer(
            0,
            channel.key,
            enable_auto_commit=True,
            auto_index_persist_interval=sy.framer.writer.ALWAYS_INDEX_PERSIST_ON_AUTO_COMMIT,
        ) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.write(pd.DataFrame({channel.key: data}))
            assert w.error() is None

        f = client.read(TimeRange(0, TimeStamp(1 * TimeSpan.SECOND)), channel.key)
        assert f.__len__() == 20

    def test_write_auto_commit_set_persist(
        self, channel: sy.Channel, client: sy.Synnax
    ):
        """Should open an auto-committing-and-persisting writer to write data."""
        with client.open_writer(
            0,
            channel.key,
            enable_auto_commit=True,
            auto_index_persist_interval=50 * TimeSpan.MILLISECOND,
        ) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.write(pd.DataFrame({channel.key: data}))

        f = client.read(TimeRange(0, TimeStamp(1 * TimeSpan.SECOND)), channel.key)
        assert f.__len__() == 20

    def test_write_err_on_unauthorized(self, channel: sy.Channel, client: sy.Synnax):
        """Should throw an error when a writer is opened to error on unauthorized"""
        w1 = client.open_writer(0, channel.key, name="writer1")

        with pytest.raises(sy.UnauthorizedError):
            with client.open_writer(
                0, channel.key, err_on_unauthorized=True, name="writer2"
            ) as w2:
                data = np.random.rand(10).astype(np.float64)

        assert w1.close() is None

    @pytest.mark.asyncio
    async def test_write_persist_only_mode(
        self,
        channel: sy.Channel,
        client: sy.Synnax,
    ):
        """Should not stream written data"""
        with client.open_writer(0, channel.key, mode=sy.WriterMode.PERSIST) as w:
            async with await client.open_async_streamer(channel.key) as s:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                with pytest.raises(TimeoutError):
                    async with asyncio.timeout(0.2):
                        await s.read()

    def test_write_persist_stream_regression(self, client: sy.Synnax):
        """Should work"""
        idx = client.channels.create(
            name="idx",
            is_index=True,
            data_type="timestamp",
        )
        data = client.channels.create(
            name="data",
            data_type="float64",
            index=idx.key,
        )
        # Write some data
        start = sy.TimeStamp.now()
        with client.open_writer(
            start, [idx.key, data.key], enable_auto_commit=True
        ) as w:
            w.write({idx.key: [start], data.key: [1]})

        # Read the data
        next_start = start + 5 * sy.TimeSpan.MILLISECOND
        f = client.read(
            TimeRange(start - 5 * sy.TimeSpan.MILLISECOND, next_start), data.key
        )
        assert len(f) == 1

        data_2 = client.channels.create(
            name="data_2",
            data_type="float64",
            index=idx.key,
        )
        data_3 = client.channels.create(
            name="data_3",
            data_type="float64",
            index=idx.key,
        )
        with client.open_writer(
            next_start,
            [idx.key, data.key, data_2.key, data_3.key],
            enable_auto_commit=True,
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

    def test_set_authority(self, client: sy.Synnax, channel: sy.channel):
        w1 = client.open_writer(0, channel.key, 100, enable_auto_commit=True)
        w2 = client.open_writer(0, channel.key, 200, enable_auto_commit=True)
        try:
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({channel.key: 255})
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name(self, client: sy.Synnax, channel: sy.channel):
        w1 = client.open_writer(0, channel.key, 100, enable_auto_commit=True)
        w2 = client.open_writer(0, channel.key, 200, enable_auto_commit=True)
        try:
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority({channel.name: 255})
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_by_name_value(self, client: sy.Synnax, channel: sy.channel):
        w1 = client.open_writer(0, channel.key, 100, enable_auto_commit=True)
        w2 = client.open_writer(0, channel.key, 200, enable_auto_commit=True)
        try:
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(channel.name, 255)
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()

    def test_set_authority_on_all_channels(
        self, client: sy.Synnax, channel: sy.channel
    ):
        w1 = client.open_writer(0, channel.key, 100, enable_auto_commit=True)
        w2 = client.open_writer(0, channel.key, 200, enable_auto_commit=True)
        try:
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 0
            w1.set_authority(255)
            w1.write(pd.DataFrame({channel.key: np.random.rand(10).astype(np.float64)}))
            f = channel.read(sy.TimeRange.MAX)
            assert len(f) == 10
        finally:
            w1.close()
            w2.close()


@pytest.mark.framer
class TestStreamer:
    def test_basic_stream(self, channel: sy.Channel, client: sy.Synnax):
        """Should correctly stream data for a channel"""
        with client.open_streamer(channel.key) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                all(frame[channel.key] == data)

    def test_open_streamer_no_channels(self, client: sy.Synnax):
        """Should not throw an exception when a streamer is opened with no channels"""
        with client.open_streamer([]):
            pass

    @pytest.mark.focus
    def test_open_streamer_channel_not_found(self, client: sy.Synnax):
        """Should throw an exception when a streamer is opened with an unknown channel"""
        with pytest.raises(sy.NotFoundError):
            with client.open_streamer([123]):
                pass

    def test_update_channels(self, channel: sy.Channel, client: sy.Synnax):
        """Should update the list of channels to stream"""
        with client.open_streamer([]) as s:
            s.update_channels([channel.key])
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = np.random.rand(1).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                all(frame[channel.key] == data)

    def test_timeout_seconds(self, channel: sy.Channel, client: sy.Synnax):
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=0.1)
            assert f is None
            assert abs(TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_timeout_timespan(self, channel: sy.Channel, client: sy.Synnax):
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=100 * TimeSpan.MILLISECOND)
            assert f is None
            assert abs(TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_downsample(self, channel: sy.Channel, client: sy.Synnax):
        """Should correctly stream data for a channel"""
        with client.open_streamer(channel.key, 1) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                assert all(frame[channel.key] == data)
        with client.open_streamer(channel.key, 2) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0, 3.0, 5.0, 7.0, 9.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                assert all(frame[channel.key] == expect)
        with client.open_streamer(channel.key, 10) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                assert all(frame[channel.key] == expect)
        with client.open_streamer(channel.key, 20) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                assert all(frame[channel.key] == expect)
        with client.open_streamer(channel.key, -1) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read(timeout=1)
                assert all(frame[channel.key] == data)


@pytest.mark.framer
class TestAsyncStreamer:
    @pytest.mark.asyncio
    async def test_basic_stream(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.open_async_streamer(channel.key, 1) as s:
                time.sleep(0.1)
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == data)
        with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.open_async_streamer(channel.key, 2) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0, 3.0, 5.0, 7.0, 9.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == expect)
        with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.open_async_streamer(channel.key, 10) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == expect)
        with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.open_async_streamer(channel.key, 20) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == expect)
        with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.open_async_streamer(channel.key, -1) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == data)


@pytest.mark.framer
class TestDeleter:
    def test_basic_delete(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(51).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()

        client.delete(
            [channel.key],
            TimeRange(0, TimeStamp(1 * TimeSpan.SECOND)),
        )

        data = channel.read(TimeRange.MAX)
        assert data.to_numpy().size == 26
        assert data.time_range == TimeRange(
            TimeStamp(1 * TimeSpan.SECOND), TimeStamp(2 * TimeSpan.SECOND) + 1
        )

    def test_delete_by_name(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(51).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()

        client.delete(
            [channel.name],
            TimeRange(0, TimeStamp(1 * TimeSpan.SECOND)),
        )

        data = channel.read(TimeRange.MAX)
        assert data.size == 26 * 8
        assert data.time_range == TimeRange(
            TimeStamp(1 * TimeSpan.SECOND), TimeStamp(2 * TimeSpan.SECOND) + 1
        )

    def test_delete_channel_not_found_name(
        self, channel: sy.Channel, client: sy.Synnax
    ):
        client.write(0, channel.key, np.random.rand(50).astype(np.float64))
        with pytest.raises(sy.NotFoundError):
            client.delete([channel.name, "kaka"], TimeRange.MAX)

        data = channel.read(TimeRange.MAX)
        assert data.size == 50 * 8

    def test_delete_channel_not_found_key(self, channel: sy.Channel, client: sy.Synnax):
        client.write(0, channel.key, np.random.rand(50).astype(np.float64))
        with pytest.raises(sy.NotFoundError):
            client.delete([channel.key, 23423], TimeRange.MAX)

        data = channel.read(TimeRange.MAX)
        assert data.size == 50 * 8

    def test_delete_with_writer(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(0, channel.key):
            with pytest.raises(UnauthorizedError):
                client.delete(
                    [channel.key],
                    TimeRange(
                        TimeStamp(1 * TimeSpan.SECOND).range(
                            TimeStamp(2 * TimeSpan.SECOND)
                        )
                    ),
                )

    def test_delete_index_alone(self, client: sy.Synnax):
        ch1 = client.channels.create(
            sy.Channel(name="index", data_type=sy.DataType.TIMESTAMP, is_index=True)
        )

        ch2 = sy.Channel(
            name="data",
            data_type=sy.DataType.FLOAT32,
            index=ch1.key,
        )

        ch2 = client.channels.create(ch2)
        timestamps = [
            sy.TimeStamp(0),
            sy.TimeStamp(1 * TimeSpan.SECOND),
            sy.TimeStamp(2 * TimeSpan.SECOND),
            sy.TimeStamp(3 * TimeSpan.SECOND),
            sy.TimeStamp(4 * TimeSpan.SECOND),
        ]
        ch1.write(0, np.array(timestamps))
        ch2.write(0, np.array([0, 1, 2, 3, 4]))

        with pytest.raises(Exception):
            client.delete(
                [ch1.key],
                TimeRange(
                    TimeStamp(1 * TimeSpan.SECOND).range(TimeStamp(2 * TimeSpan.SECOND))
                ),
            )
