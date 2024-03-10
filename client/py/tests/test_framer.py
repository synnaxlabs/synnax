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
        with client.new_writer(0, channel.key) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()

    def test_write_by_name(self, channel: sy.Channel, client: sy.Synnax):
        with client.new_writer(0, channel.name) as w:
            data = np.random.rand(10).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()


@pytest.mark.framer
class TestStreamer:
    def test_basic_stream(self, channel: sy.Channel, client: sy.Synnax):
        with client.new_streamer(channel.key) as s:
            with client.new_writer(sy.TimeStamp.now(), channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = s.read()
                all(frame[channel.key] == data)


@pytest.mark.framer
class TestAsyncStreamer:
    @pytest.mark.asyncio
    async def test_basic_stream(self, channel: sy.Channel, client: sy.Synnax):
        with client.new_writer(sy.TimeStamp.now(), channel.key) as w:
            async with await client.new_async_streamer(channel.key) as s:
                time.sleep(0.1)
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({channel.key: data}))
                frame = await s.read()
                assert all(frame[channel.key] == data)
