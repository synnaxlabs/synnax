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
        assert all(data == d)


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

    @pytest.mark.asyncio
    async def test_write_persist_only_mode(
        self,
        channel: sy.Channel,
        client: sy.Synnax,
    ):
        """Should not stream written data"""
        with client.open_writer(0, channel.key, mode=sy.WriterMode.PERSIST_ONLY) as w:
            async with await client.open_async_streamer(channel.key) as s:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                with pytest.raises(TimeoutError):
                    async with asyncio.timeout(0.2):
                        await s.read()


@pytest.mark.framer
class TestStreamer:
    def test_basic_stream(self, channel: sy.Channel, client: sy.Synnax):
        """Should correctly stream data for a channel"""
        with client.open_streamer(channel.key) as s:
            with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read()
                all(frame[channel.key] == data)


@pytest.mark.framer
class TestAsyncStreamer:
    @pytest.mark.asyncio
    async def test_basic_stream(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.open_async_streamer(channel.key) as s:
                time.sleep(0.1)
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == data)
