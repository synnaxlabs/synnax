#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import contextlib
from collections.abc import Callable, Iterator

import pytest

import synnax as sy
from synnax.state import LatestState
from synnax.telem import SampleValue
from tests.proxy import Proxy
from x.strings import random_name

# Seconds to wait for a written sample to reach a handler.
TIMEOUT = 5
# Seconds between writes while the registry opens its streamer.
POLL = 0.05


@pytest.fixture
def signals_client(login_info: tuple[str, int, str, str]) -> Iterator[sy.Synnax]:
    """A client with its own signals registry, so handlers stay in one test."""
    host, port, username, password = login_info
    c = sy.Synnax(
        host=host, port=port, username=username, password=password, secure=False
    )
    yield c
    c.close()


def create_channel(client: sy.Synnax) -> sy.Channel:
    return client.channels.create(
        name=f"test_{random_name()}_virtual",
        virtual=True,
        data_type=sy.DataType.FLOAT64,
    )


async def write_until(
    client: sy.Synnax, ch: sy.Channel, value: float, done: Callable[[], bool]
) -> None:
    """Writes value to ch until done returns true. The registry opens its streamer
    in the background, so a single write can arrive before the streamer listens."""
    async with asyncio.timeout(TIMEOUT):
        while not done():
            with client.open_writer(sy.TimeStamp.now(), ch.key) as w:
                w.write({ch.key: [value]})
            await asyncio.sleep(POLL)


async def stop(task: asyncio.Task[None]) -> None:
    task.cancel()
    with contextlib.suppress(asyncio.CancelledError):
        await task


@pytest.mark.signals
class TestRegistryDispatch:
    """Tests which handlers the registry runs for a frame."""

    @pytest.mark.asyncio
    async def test_frame_without_handler_channels(
        self, signals_client: sy.Synnax
    ) -> None:
        """Should not run a filter on a frame without the handler's channels."""
        a, b = create_channel(signals_client), create_channel(signals_client)
        a_values: list[SampleValue] = []
        b_values: list[SampleValue] = []

        @signals_client.signals.on(a.key, lambda s: s[a.key] == 1)
        def on_a(state: LatestState) -> None:
            a_values.append(state[a.key])

        @signals_client.signals.on(b.key, lambda _: True)
        def on_b(state: LatestState) -> None:
            b_values.append(state[b.key])

        task = asyncio.create_task(signals_client.signals.process())
        try:
            await write_until(signals_client, b, 2, lambda: len(b_values) > 0)
            await write_until(signals_client, a, 1, lambda: len(a_values) > 0)
        finally:
            await stop(task)
        assert a_values[-1] == 1

    @pytest.mark.asyncio
    async def test_handler_channel_without_value(
        self, signals_client: sy.Synnax
    ) -> None:
        """Should not run a filter until every handler channel has a value."""
        a, b = create_channel(signals_client), create_channel(signals_client)
        a_values: list[SampleValue] = []
        sums: list[float] = []

        @signals_client.signals.on(a.key, lambda _: True)
        def on_a(state: LatestState) -> None:
            a_values.append(state[a.key])

        def sum_above_two(s: LatestState) -> bool:
            a_value, b_value = s[a.key], s[b.key]
            assert isinstance(a_value, float)
            assert isinstance(b_value, float)
            return a_value + b_value > 2

        @signals_client.signals.on([a.key, b.key], sum_above_two)
        def on_both(state: LatestState) -> None:
            a_value, b_value = state[a.key], state[b.key]
            assert isinstance(a_value, float)
            assert isinstance(b_value, float)
            sums.append(a_value + b_value)

        task = asyncio.create_task(signals_client.signals.process())
        try:
            await write_until(signals_client, a, 2, lambda: len(a_values) > 0)
            await write_until(signals_client, b, 3, lambda: len(sums) > 0)
        finally:
            await stop(task)
        assert sums[-1] == 5


@pytest.mark.signals
class TestRegistryStreamer:
    """Tests the lifecycle of the registry's streamer."""

    @pytest.mark.asyncio
    async def test_close_on_cancel(
        self, client: sy.Synnax, proxy: Proxy, proxied_client: sy.Synnax
    ) -> None:
        """Should close the streamer when process is cancelled."""
        ch = create_channel(client)
        values: list[SampleValue] = []

        @proxied_client.signals.on(ch.key, lambda _: True)
        def handler(state: LatestState) -> None:
            values.append(state[ch.key])

        task = asyncio.create_task(proxied_client.signals.process())
        try:
            await write_until(client, ch, 1, lambda: len(values) > 0)
            assert proxy.websockets == 1
        finally:
            await stop(task)
        async with asyncio.timeout(TIMEOUT):
            while proxy.websockets > 0:
                await asyncio.sleep(POLL)
