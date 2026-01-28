#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from tests.channel import assert_eventually_channels_are_found


@pytest.mark.framer
@pytest.mark.streamer
class TestStreamer:
    def test_basic_stream_virtual(self, virtual_channel: sy.Channel, client: sy.Synnax):
        """Should correctly stream data for a virtual channel"""
        with client.open_streamer(virtual_channel.key) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                for i in range(10):
                    data = np.random.rand(10).astype(np.float64)
                    w.write(pd.DataFrame({virtual_channel.key: data}))
                    frame = s.read(timeout=1)
                    assert np.array_equal(frame[virtual_channel.key], data)

    def test_basic_stream_indexed_pair(
        self,
        indexed_pair: list[sy.Channel],
        client: sy.Synnax,
    ):
        idx, data = indexed_pair
        """Should correctly stream data from a virtual/index channel pair"""
        with client.open_streamer(channels=[idx.name, data.name]) as s:
            with client.open_writer(sy.TimeStamp.now(), channels=indexed_pair) as w:
                for i in range(10):
                    ts = sy.TimeStamp.now()
                    value = np.random.rand(1)
                    w.write(pd.DataFrame({idx.name: ts, data.name: value}))
                    frame = s.read(timeout=1)
                    assert frame[idx.name][0] == ts
                    assert frame[data.name][0] == value

    def test_basic_stream_multiple_writers(
        self,
        indexed_pair: list[sy.Channel],
        virtual_channel: sy.Channel,
        client: sy.Synnax,
    ):
        idx, data = indexed_pair
        start = sy.TimeStamp.now()
        with client.open_streamer(
            channels=[idx.name, data.name, virtual_channel.name]
        ) as s:
            with client.open_writer(start, channels=indexed_pair) as idx_writer:
                with client.open_writer(
                    start,
                    channels=virtual_channel,
                ) as virtual_writer:
                    for i in range(10):
                        ts = sy.TimeStamp.now()
                        value = np.random.rand(1)
                        v_value = np.random.rand(1)
                        idx_writer.write(pd.DataFrame({idx.name: ts, data.name: value}))
                        virtual_writer.write(
                            pd.DataFrame({virtual_channel.name: v_value})
                        )
                        for _ in range(2):
                            frame = s.read(timeout=1)
                            if len(frame.channels) == 1:
                                assert frame[virtual_channel.name] == v_value
                            else:
                                assert frame[idx.name][0] == ts
                                assert frame[data.name][0] == value

    def test_open_streamer_no_channels(self, client: sy.Synnax):
        """Should not throw an exception when a streamer is opened with no channels"""
        with client.open_streamer([]):
            pass

    def test_open_streamer_channel_not_found(self, client: sy.Synnax):
        """Should throw an exception when a streamer is opened with an unknown channel"""
        with pytest.raises(sy.NotFoundError):
            with client.open_streamer([123]):
                pass

    def test_open_streamer_channel_key_zero(self, client: sy.Synnax):
        """Should throw an exception when a streamer is opened with a channel key of
        zero"""
        with pytest.raises(sy.NotFoundError):
            with client.open_streamer([0, 0, 0]):
                pass

    def test_update_channels(self, virtual_channel: sy.Channel, client: sy.Synnax):
        """Should update the list of channels to stream"""
        with client.open_streamer([]) as s:
            s.update_channels([virtual_channel.key])
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = np.random.rand(1).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert np.array_equal(frame[virtual_channel.key], data)

    def test_update_channels_rapid(
        self,
        indexed_pair: list[sy.Channel],
        virtual_channel: sy.Channel,
        client: sy.Synnax,
    ):
        idx, data_ch = indexed_pair
        start = sy.TimeStamp.now()
        channel_names = [idx.name, data_ch.name, virtual_channel.name]
        curr_channels = [*channel_names]
        with client.open_streamer(channels=channel_names) as s:
            with client.open_writer(start, channels=channel_names) as writer:
                for i in range(300):
                    ts = sy.TimeStamp.now()
                    value = np.random.rand(1)
                    data = {
                        idx.name: ts,
                        data_ch.name: value,
                        virtual_channel.name: value,
                    }
                    writer.write(data)
                    if i % 50 == 0:
                        c = int((i % 150) / 50) + 1
                        curr_channels = channel_names[:c]
                        s.update_channels(curr_channels)
                    frm = s.read(timeout=1)
                    for channel in curr_channels:
                        d = frm.get(channel, None)
                        if len(d) > 0:
                            assert len(d) == 1
                            assert d[0] == data[channel]

    def test_timeout_seconds(self, client: sy.Synnax):
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=0.1)
            assert f is None
            assert abs(sy.TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_timeout_timespan(self, client: sy.Synnax):
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=100 * sy.TimeSpan.MILLISECOND)
            assert f is None
            assert abs(sy.TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_downsample(self, virtual_channel: sy.Channel, client: sy.Synnax):
        """Should correctly stream data for a channel"""
        with client.open_streamer(virtual_channel.key, 1) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert np.array_equal(frame[virtual_channel.key], data)
        with client.open_streamer(virtual_channel.key, 2) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0, 3.0, 5.0, 7.0, 9.0]
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert np.array_equal(frame[virtual_channel.key], expect)
        with client.open_streamer(virtual_channel.key, 10) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert np.array_equal(frame[virtual_channel.key], expect)
        with client.open_streamer(virtual_channel.key, 20) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert np.array_equal(frame[virtual_channel.key], expect)

    def test_downsample_negative(self, virtual_channel: sy.Channel, client: sy.Synnax):
        with pytest.raises(sy.ValidationError):
            with client.open_streamer(virtual_channel.key, -1) as s:
                ...

    @pytest.mark.multi_node
    def test_multi_node_stream_case_1(self):
        client = sy.Synnax(cache_channels=False)
        idx = client.channels.create(
            name="idx", data_type=sy.DataType.TIMESTAMP, is_index=True, leaseholder=2
        )
        data = client.channels.create(
            name="data", data_type=sy.DataType.FLOAT64, index=idx.key, leaseholder=2
        )
        assert_eventually_channels_are_found(client, [idx.key, data.key])
        with client.open_streamer(data.key) as s:
            with client.open_writer(sy.TimeStamp.now(), [idx.key, data.key]) as w:
                w.write({idx.key: [sy.TimeStamp.now()], data.key: [1]})
                f = s.read(timeout=1)
                assert f is not None
                assert f[data.key][0] == 1

    @pytest.mark.multi_node
    def test_multi_node_stream_case_2(self):
        node_1_client = sy.Synnax(
            host="localhost",
            port=9090,
            username="synnax",
            password="seldon",
            secure=False,
            cache_channels=False,
        )
        node_2_client = sy.Synnax(
            host="localhost",
            port=9091,
            username="synnax",
            password="seldon",
            secure=False,
            cache_channels=False,
        )
        idx = node_2_client.channels.create(
            name="idx", data_type=sy.DataType.TIMESTAMP, is_index=True, leaseholder=2
        )
        data = node_2_client.channels.create(
            name="data", data_type=sy.DataType.FLOAT64, index=idx.key, leaseholder=2
        )
        assert_eventually_channels_are_found(node_1_client, [idx.key, data.key])
        assert_eventually_channels_are_found(node_2_client, [idx.key, data.key])
        with node_1_client.open_streamer(data.key) as s:
            with node_2_client.open_writer(
                sy.TimeStamp.now(), [idx.key, data.key]
            ) as w:
                w.write({idx.key: [sy.TimeStamp.now()], data.key: [1]})
                f = s.read(timeout=1)
                assert f is not None
                assert f[data.key][0] == 1


@pytest.mark.framer
class TestAsyncStreamer:
    @pytest.mark.asyncio
    async def test_basic_stream(self, virtual_channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
            async with await client.open_async_streamer(virtual_channel.key, 1) as s:
                time.sleep(0.1)
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = await s.read()
                assert np.array_equal(frame[virtual_channel.key], data)
        with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
            async with await client.open_async_streamer(virtual_channel.key, 2) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0, 3.0, 5.0, 7.0, 9.0]
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = await s.read()
                assert np.array_equal(frame[virtual_channel.key], expect)
        with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
            async with await client.open_async_streamer(virtual_channel.key, 10) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = await s.read()
                assert np.array_equal(frame[virtual_channel.key], expect)
        with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
            async with await client.open_async_streamer(virtual_channel.key, 20) as s:
                time.sleep(0.1)
                data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = await s.read()
                assert np.array_equal(frame[virtual_channel.key], expect)

    @pytest.mark.asyncio
    async def test_downsample_negative(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ):
        with pytest.raises(sy.ValidationError):
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                async with await client.open_async_streamer(
                    virtual_channel.key, -1
                ) as s:
                    time.sleep(0.1)
                    data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                    w.write(pd.DataFrame({virtual_channel.key: data}))
                    frame = await s.read()
                    assert np.array_equal(frame[virtual_channel.key], data)
