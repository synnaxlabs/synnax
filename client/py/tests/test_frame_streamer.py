#  Copyright 2025 Synnax Labs, Inc.
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
from synnax import TimeSpan


@pytest.mark.framer
@pytest.mark.streamer
class TestStreamer:
    def test_basic_stream(self, virtual_channel: sy.Channel, client: sy.Synnax):
        """Should correctly stream data for a channel"""
        with client.open_streamer(
            virtual_channel.key, use_experimental_codec=True
        ) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert np.array_equal(frame[virtual_channel.key], data)

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

    def test_timeout_seconds(self, client: sy.Synnax):
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=0.1)
            assert f is None
            assert abs(TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_timeout_timespan(self, client: sy.Synnax):
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=100 * TimeSpan.MILLISECOND)
            assert f is None
            assert abs(TimeSpan.since(start).seconds - 0.1) < 0.05

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

    def tesst_downsample_negative(self, virtual_channel: sy.Channel, client: sy.Synnax):
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
            with client.open_writer(
                sy.TimeStamp.now(), [idx.key, data.key], enable_auto_commit=True
            ) as w:
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
                sy.TimeStamp.now(), [idx.key, data.key], enable_auto_commit=True
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
