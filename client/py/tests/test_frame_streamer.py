#  Copyright 2026 Synnax Labs, Inc.
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
from freighter import Unreachable
from tests.channel import assert_eventually_channels_are_found
from tests.proxy import Proxy

# Seconds to wait for a written sample to come back through the streamer.
READ_TIMEOUT = 5
# The Core's minimum accepted keep-alive interval.
KEEP_ALIVE = 2 * sy.TimeSpan.SECOND
# Three keep-alive intervals: how long a read may stay silent once armed.
DEADLINE = 3 * KEEP_ALIVE
# Seconds for one keep-alive interval plus slack, so the streamer has armed.
ARMED = 2.5


def write(client: sy.Synnax, ch: sy.Channel, data: list[float]) -> None:
    with client.open_writer(sy.TimeStamp.now(), ch.key) as w:
        w.write({ch.key: data})


def read_while_writing(
    client: sy.Synnax, s: sy.Streamer, ch: sy.Channel
) -> sy.Frame | None:
    """Writes until the streamer reads a frame. The Core does not acknowledge an
    updated channel set, so a write can reach the relay before the update does."""
    with client.open_writer(sy.TimeStamp.now(), ch.key) as w:
        deadline = time.monotonic() + READ_TIMEOUT
        while time.monotonic() < deadline:
            w.write({ch.key: [1.0]})
            if (frame := s.read(timeout=0.05)) is not None:
                return frame
    return None


@pytest.mark.framer
@pytest.mark.streamer
class TestStreamer:
    def test_basic_stream_virtual(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should correctly stream data for a virtual channel"""
        with client.open_streamer(virtual_channel.key) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                for i in range(10):
                    data = np.random.rand(10).astype(np.float64)
                    w.write(pd.DataFrame({virtual_channel.key: data}))
                    frame = s.read(timeout=READ_TIMEOUT)
                    assert frame is not None, f"read {i} timed out"
                    assert np.array_equal(frame[virtual_channel.key], data)

    def test_basic_stream_indexed_pair(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        client: sy.Synnax,
    ) -> None:
        idx, data = indexed_pair
        """Should correctly stream data from a virtual/index channel pair"""
        with client.open_streamer(channels=[idx.name, data.name]) as s:
            with client.open_writer(sy.TimeStamp.now(), channels=indexed_pair) as w:
                for i in range(10):
                    ts = sy.TimeStamp.now()
                    value = np.random.rand(1)
                    w.write(pd.DataFrame({idx.name: ts, data.name: value}))
                    frame = s.read(timeout=READ_TIMEOUT)
                    assert frame is not None, f"read {i} timed out"
                    assert frame[idx.name][0] == ts
                    assert frame[data.name][0] == value

    def test_basic_stream_multiple_writers(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        virtual_channel: sy.Channel,
        client: sy.Synnax,
    ) -> None:
        idx, data = indexed_pair
        start = sy.TimeStamp.now()
        with (
            client.open_streamer(
                channels=[idx.name, data.name, virtual_channel.name]
            ) as s,
            client.open_writer(start, channels=indexed_pair) as idx_writer,
        ):
            with client.open_writer(
                start,
                channels=virtual_channel,
            ) as virtual_writer:
                for i in range(10):
                    ts = sy.TimeStamp.now()
                    value = np.random.rand(1)
                    v_value = np.random.rand(1)
                    idx_writer.write(pd.DataFrame({idx.name: ts, data.name: value}))
                    virtual_writer.write(pd.DataFrame({virtual_channel.name: v_value}))
                    for _ in range(2):
                        frame = s.read(timeout=READ_TIMEOUT)
                        assert frame is not None, f"read {i} timed out"
                        if len(frame.channels) == 1:
                            assert frame[virtual_channel.name] == v_value
                        else:
                            assert frame[idx.name][0] == ts
                            assert frame[data.name][0] == value

    def test_exclude_groups(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should filter out frames from a writer whose group matches exclude_groups"""
        group_id = 42
        with client.open_streamer(virtual_channel.key, exclude_groups=[group_id]) as s:
            with client.open_writer(
                sy.TimeStamp.now(),
                virtual_channel.key,
                group=group_id,
            ) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=0.5)
                assert frame is None

    def test_exclude_groups_non_matching(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should deliver frames from a writer whose group does not match
        exclude_groups"""
        with client.open_streamer(virtual_channel.key, exclude_groups=[99]) as s:
            with client.open_writer(
                sy.TimeStamp.now(),
                virtual_channel.key,
                group=42,
            ) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], data)

    def test_exclude_groups_zero_group_always_delivered(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should deliver frames from a writer with group=0 even if 0 is in
        exclude_groups"""
        with client.open_streamer(virtual_channel.key, exclude_groups=[0]) as s:
            with client.open_writer(
                sy.TimeStamp.now(),
                virtual_channel.key,
            ) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], data)

    def test_open_streamer_no_channels(self, client: sy.Synnax) -> None:
        """Should not throw an exception when a streamer is opened with no channels"""
        with client.open_streamer([]):
            pass

    def test_open_streamer_channel_not_found(self, client: sy.Synnax) -> None:
        """Should throw an exception when a streamer is opened with an unknown
        channel
        """
        with pytest.raises(sy.NotFoundError), client.open_streamer([123]):
            pass

    def test_open_streamer_channel_key_zero(self, client: sy.Synnax) -> None:
        """Should throw an exception when a streamer is opened with a channel key of
        zero"""
        with pytest.raises(sy.NotFoundError), client.open_streamer([0, 0, 0]):
            pass

    def test_update_channels(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should update the list of channels to stream"""
        with client.open_streamer([]) as s:
            s.update_channels([virtual_channel.key])
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = np.random.rand(1).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], data)

    def test_update_same_channels(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should read frames after setting the channels the streamer already has"""
        with client.open_streamer(virtual_channel.key) as s:
            s.update_channels([virtual_channel.key])
            frame = read_while_writing(client, s, virtual_channel)
            assert frame is not None
            assert np.array_equal(frame[virtual_channel.key], [1.0])

    def test_update_channels_after_none(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should read frames of the channels set after an empty set of channels"""
        with client.open_streamer(virtual_channel.key) as s:
            s.update_channels([])
            s.update_channels([virtual_channel.key])
            frame = read_while_writing(client, s, virtual_channel)
            assert frame is not None
            assert np.array_equal(frame[virtual_channel.key], [1.0])

    def test_update_channels_rapid(
        self,
        indexed_pair: tuple[sy.Channel, sy.Channel],
        virtual_channel: sy.Channel,
        client: sy.Synnax,
    ) -> None:
        idx, data_ch = indexed_pair
        start = sy.TimeStamp.now()
        channel_names = [idx.name, data_ch.name, virtual_channel.name]
        curr_channels = [*channel_names]
        with client.open_streamer(channels=channel_names) as s:
            with client.open_writer(start, channels=channel_names) as writer:
                for i in range(300):
                    ts = sy.TimeStamp.now()
                    value = np.random.rand(1)
                    writer.write(
                        {
                            idx.name: ts,
                            data_ch.name: value,
                            virtual_channel.name: value,
                        }
                    )
                    if i % 50 == 0:
                        c = int((i % 150) / 50) + 1
                        curr_channels = channel_names[:c]
                        s.update_channels(curr_channels)
                    frm = s.read(timeout=1)
                    assert frm is not None
                    for channel in curr_channels:
                        d = frm[channel]
                        if len(d) > 0:
                            assert len(d) == 1
                            assert d[0] == (ts if channel == idx.name else value)

    def test_timeout_seconds(self, client: sy.Synnax) -> None:
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=0.1)
            assert f is None
            assert abs(sy.TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_timeout_timespan(self, client: sy.Synnax) -> None:
        """Should return None after the specified timeout is exceeded"""
        with client.open_streamer([]) as s:
            start = sy.TimeStamp.now()
            f = s.read(timeout=100 * sy.TimeSpan.MILLISECOND)
            assert f is None
            assert abs(sy.TimeSpan.since(start).seconds - 0.1) < 0.05

    def test_downsample(self, virtual_channel: sy.Channel, client: sy.Synnax) -> None:
        """Should correctly stream data for a channel"""
        with client.open_streamer(virtual_channel.key, downsample_factor=1) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                data = np.random.rand(10).astype(np.float64)
                w.write(pd.DataFrame({virtual_channel.key: data}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], data)
        with client.open_streamer(virtual_channel.key, downsample_factor=2) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                samples = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0, 3.0, 5.0, 7.0, 9.0]
                w.write(pd.DataFrame({virtual_channel.key: samples}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], expect)
        with client.open_streamer(virtual_channel.key, downsample_factor=10) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                samples = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({virtual_channel.key: samples}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], expect)
        with client.open_streamer(virtual_channel.key, downsample_factor=20) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                samples = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
                expect = [1.0]
                w.write(pd.DataFrame({virtual_channel.key: samples}))
                frame = s.read(timeout=1)
                assert frame is not None
                assert np.array_equal(frame[virtual_channel.key], expect)

    @staticmethod
    def _count_frames(
        streamer: sy.Streamer, writer: sy.Writer, key: sy.channel.Key, n: int
    ) -> int:
        expected = np.arange(n, dtype=np.float64)
        for value in expected:
            writer.write(key, value)
        received: list[float] = []
        frames = 0
        while len(received) < n:
            frame = streamer.read(timeout=READ_TIMEOUT)
            assert frame is not None, f"timed out after {len(received)} values"
            received.extend(frame[key].to_numpy())
            frames += 1
        assert np.array_equal(received, expected)
        return frames

    def test_throttle(self, virtual_channel: sy.Channel, client: sy.Synnax) -> None:
        """Should merge the frames written in one throttle period into one frame"""
        n = 10
        with client.open_streamer(virtual_channel.key, throttle_rate=1) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                frames = self._count_frames(s, w, virtual_channel.key, n)
        # A tick between two writes can split them across two frames.
        assert frames <= 2

    def test_no_throttle(self, virtual_channel: sy.Channel, client: sy.Synnax) -> None:
        """Should send each written frame on its own without a throttle rate"""
        n = 10
        with client.open_streamer(virtual_channel.key) as s:
            with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
                assert self._count_frames(s, w, virtual_channel.key, n) == n

    def test_downsample_negative(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        with pytest.raises(sy.ValidationError):
            with client.open_streamer(virtual_channel.key, downsample_factor=-1):
                ...

    @pytest.mark.multi_node
    def test_multi_node_stream_case_1(self) -> None:
        client = sy.Synnax()
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
    def test_multi_node_stream_case_2(self) -> None:
        node_1_client = sy.Synnax(
            host="localhost",
            port=9090,
            username="synnax",
            password="seldon",
            secure=False,
        )
        node_2_client = sy.Synnax(
            host="localhost",
            port=9091,
            username="synnax",
            password="seldon",
            secure=False,
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
    @pytest.mark.parametrize(
        ("downsample_factor", "expected"),
        [
            (1, [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]),
            (2, [1.0, 3.0, 5.0, 7.0, 9.0]),
            (10, [1.0]),
            (20, [1.0]),
        ],
    )
    async def test_basic_stream(
        self,
        downsample_factor: int,
        expected: list[float],
        virtual_channel: sy.Channel,
        client: sy.Synnax,
    ) -> None:
        """Should downsample a streamed frame by the given factor."""
        data = [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0]
        with client.open_writer(sy.TimeStamp.now(), virtual_channel.key) as w:
            async with await client.open_async_streamer(
                virtual_channel.key, downsample_factor=downsample_factor
            ) as s:
                w.write(pd.DataFrame({virtual_channel.key: data}))
                async with asyncio.timeout(READ_TIMEOUT):
                    frame = await s.read()
                assert np.array_equal(frame[virtual_channel.key], expected)

    @pytest.mark.asyncio
    async def test_downsample_negative(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should reject a negative downsample factor."""
        with pytest.raises(sy.ValidationError):
            await client.open_async_streamer(virtual_channel.key, downsample_factor=-1)


@pytest.mark.framer
@pytest.mark.streamer
class TestStreamerKeepAlive:
    def test_keep_alives_not_surfaced(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should skip queued keep-alives and return the next frame"""
        with client.open_streamer(virtual_channel.key, keep_alive=KEEP_ALIVE) as s:
            time.sleep(KEEP_ALIVE.seconds * 2.5)
            write(client, virtual_channel, [1, 2, 3])
            frame = s.read(timeout=READ_TIMEOUT)
            assert frame is not None
            assert np.array_equal(frame[virtual_channel.key], [1, 2, 3])

    def test_idle_read_past_deadline(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should return None from an idle read whose timeout exceeds the deadline"""
        with client.open_streamer(virtual_channel.key, keep_alive=KEEP_ALIVE) as s:
            assert s.read(timeout=DEADLINE + KEEP_ALIVE) is None

    def test_interval_below_minimum(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should reject an interval below the Core's minimum"""
        with pytest.raises(
            sy.ValidationError, match="must be greater than or equal to 2s"
        ):
            client.open_streamer(virtual_channel.key, keep_alive=1)

    def test_silent_read_unreachable(
        self, virtual_channel: sy.Channel, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should raise Unreachable when a read stays silent past the deadline"""
        s = proxied_client.open_streamer(virtual_channel.key, keep_alive=KEEP_ALIVE)
        time.sleep(ARMED)
        assert proxy.blackhole() > 0
        start = sy.TimeStamp.now()
        with pytest.raises(
            Unreachable, match=f"streamer received no response for {DEADLINE}"
        ):
            s.read()
        assert sy.TimeSpan.since(start) >= DEADLINE

    def test_silent_read_user_timeout(
        self, virtual_channel: sy.Channel, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should return None when the read timeout elapses before the deadline"""
        s = proxied_client.open_streamer(virtual_channel.key, keep_alive=KEEP_ALIVE)
        time.sleep(ARMED)
        assert proxy.blackhole() > 0
        assert s.read(timeout=1) is None

    def test_polled_read_unreachable(
        self, virtual_channel: sy.Channel, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should raise Unreachable to a caller polling with a short timeout"""
        s = proxied_client.open_streamer(virtual_channel.key, keep_alive=KEEP_ALIVE)
        time.sleep(ARMED)
        assert proxy.blackhole() > 0
        start = sy.TimeStamp.now()
        with pytest.raises(Unreachable):
            while sy.TimeSpan.since(start) < DEADLINE + KEEP_ALIVE:
                assert s.read(timeout=1) is None

    def test_early_keep_alive_arms(
        self,
        virtual_channel: sy.Channel,
        early_proxy: Proxy,
        early_proxied_client: sy.Synnax,
    ) -> None:
        """Should arm the deadline on a keep-alive that precedes the open ack"""
        s = early_proxied_client.open_streamer(
            virtual_channel.key, keep_alive=KEEP_ALIVE
        )
        assert early_proxy.blackhole() > 0
        with pytest.raises(Unreachable):
            s.read(timeout=DEADLINE + KEEP_ALIVE)

    def test_disabled(
        self, virtual_channel: sy.Channel, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should never arm the deadline when keep-alive is disabled"""
        s = proxied_client.open_streamer(virtual_channel.key, keep_alive=0)
        time.sleep(0.25)
        assert proxy.blackhole() > 0
        assert s.read(timeout=1) is None


@pytest.mark.framer
class TestAsyncStreamerKeepAlive:
    @pytest.mark.asyncio
    async def test_keep_alives_not_surfaced(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should skip queued keep-alives and return the next frame"""
        async with await client.open_async_streamer(
            virtual_channel.key, keep_alive=KEEP_ALIVE
        ) as s:
            await asyncio.sleep(KEEP_ALIVE.seconds * 2.5)
            write(client, virtual_channel, [1, 2, 3])
            async with asyncio.timeout(READ_TIMEOUT):
                frame = await s.read()
            assert np.array_equal(frame[virtual_channel.key], [1, 2, 3])

    @pytest.mark.asyncio
    async def test_blocked_loop_past_deadline(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should read keep-alives that arrived while the loop was blocked"""
        async with await client.open_async_streamer(
            virtual_channel.key, keep_alive=KEEP_ALIVE
        ) as s:
            await asyncio.sleep(ARMED)
            write(client, virtual_channel, [1, 2, 3])
            async with asyncio.timeout(READ_TIMEOUT):
                await s.read()
            time.sleep((DEADLINE + KEEP_ALIVE).seconds)
            write(client, virtual_channel, [4, 5, 6])
            async with asyncio.timeout(READ_TIMEOUT):
                frame = await s.read()
            assert np.array_equal(frame[virtual_channel.key], [4, 5, 6])

    @pytest.mark.asyncio
    async def test_silent_read_unreachable(
        self, virtual_channel: sy.Channel, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should raise Unreachable when a read stays silent past the deadline"""
        s = await proxied_client.open_async_streamer(
            virtual_channel.key, keep_alive=KEEP_ALIVE
        )
        await asyncio.sleep(ARMED)
        assert proxy.blackhole() > 0
        start = sy.TimeStamp.now()
        with pytest.raises(
            Unreachable, match=f"streamer received no response for {DEADLINE}"
        ):
            await s.read()
        assert sy.TimeSpan.since(start) >= DEADLINE

    @pytest.mark.asyncio
    async def test_polled_read_unreachable(
        self, virtual_channel: sy.Channel, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should raise Unreachable to a caller polling with a short timeout"""
        s = await proxied_client.open_async_streamer(
            virtual_channel.key, keep_alive=KEEP_ALIVE
        )
        await asyncio.sleep(ARMED)
        assert proxy.blackhole() > 0
        start = sy.TimeStamp.now()
        with pytest.raises(Unreachable):
            while sy.TimeSpan.since(start) < DEADLINE + KEEP_ALIVE:
                with pytest.raises(TimeoutError):
                    async with asyncio.timeout(1):
                        await s.read()

    @pytest.mark.asyncio
    async def test_early_keep_alive_arms(
        self,
        virtual_channel: sy.Channel,
        early_proxy: Proxy,
        early_proxied_client: sy.Synnax,
    ) -> None:
        """Should arm the deadline on a keep-alive that precedes the open ack"""
        s = await early_proxied_client.open_async_streamer(
            virtual_channel.key, keep_alive=KEEP_ALIVE
        )
        assert early_proxy.blackhole() > 0
        with pytest.raises(Unreachable):
            async with asyncio.timeout((DEADLINE + KEEP_ALIVE).seconds):
                await s.read()


@pytest.mark.framer
@pytest.mark.streamer
class TestAsyncStreamerClose:
    @pytest.mark.asyncio
    async def test_close_with_unread_frames(
        self, virtual_channel: sy.Channel, client: sy.Synnax
    ) -> None:
        """Should close while frames the caller has not read are still queued."""
        s = await client.open_async_streamer(virtual_channel.key)
        write(client, virtual_channel, [1])
        write(client, virtual_channel, [2])
        async with asyncio.timeout(READ_TIMEOUT):
            await s.read()
        await asyncio.sleep(0.1)
        await s.close()
