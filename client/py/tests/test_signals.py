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
from collections.abc import AsyncIterator, Callable, Iterator

import pytest

import synnax as sy
from synnax.state import Latest
from synnax.telem import SampleValue
from tests.proxy import Proxy
from x.strings import random_name

# Seconds to wait for a written sample to reach a handler.
TIMEOUT = 5
# Seconds between writes while the registry opens its streamer.
POLL = 0.05


@pytest.fixture
def fresh_client(login_info: tuple[str, int, str, str]) -> Iterator[sy.Synnax]:
    """A client with its own signals registry, so handlers stay in one test."""
    host, port, username, password = login_info
    c = sy.Synnax(
        host=host, port=port, username=username, password=password, secure=False
    )
    yield c
    c.close()


@contextlib.asynccontextmanager
async def processing(client: sy.Synnax) -> AsyncIterator[None]:
    task = asyncio.create_task(client.signals.process())
    try:
        yield
    finally:
        task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await task


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


def create_channel(client: sy.Synnax) -> sy.Channel:
    return client.channels.create(
        name=f"test_{random_name()}_virtual",
        virtual=True,
        data_type=sy.DataType.FLOAT64,
    )


@pytest.mark.signals
class TestRegistry:
    """Tests registering signal handlers and dispatching streamed values to them."""

    def test_on_returns_handler(
        self, fresh_client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should return the decorated handler unchanged."""

        def handler(_: Latest) -> None: ...

        decorated = fresh_client.signals.on(
            virtual_channel.key, filter_f=lambda _: True
        )
        assert decorated(handler) is handler

    @pytest.mark.asyncio
    async def test_process_by_key(
        self, fresh_client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should call a handler registered by key with the latest value."""
        values: list[SampleValue] = []

        @fresh_client.signals.on(virtual_channel.key, filter_f=lambda _: True)
        def handler(state: Latest) -> None:
            values.append(state[virtual_channel.key])

        async with processing(fresh_client):
            await write_until(fresh_client, virtual_channel, 7, lambda: len(values) > 0)
        assert values[-1] == 7

    @pytest.mark.asyncio
    async def test_process_by_name(
        self, fresh_client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should resolve a channel name and call the handler by name."""
        values: list[SampleValue] = []

        @fresh_client.signals.on(virtual_channel.name, filter_f=lambda _: True)
        def handler(state: Latest) -> None:
            values.append(state[virtual_channel.name])

        async with processing(fresh_client):
            await write_until(fresh_client, virtual_channel, 3, lambda: len(values) > 0)
        assert values[-1] == 3

    @pytest.mark.asyncio
    async def test_process_filter(
        self, fresh_client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should call a handler only when its filter passes."""
        key = virtual_channel.key
        filtered: list[SampleValue] = []
        every: list[SampleValue] = []

        def above_five(state: Latest) -> bool:
            value = state[key]
            return isinstance(value, float) and value > 5

        @fresh_client.signals.on(key, filter_f=above_five)
        def above(state: Latest) -> None:
            filtered.append(state[key])

        @fresh_client.signals.on(key, filter_f=lambda _: True)
        def all_values(state: Latest) -> None:
            every.append(state[key])

        async with processing(fresh_client):
            await write_until(fresh_client, virtual_channel, 1, lambda: len(every) > 0)
            await write_until(
                fresh_client, virtual_channel, 10, lambda: every[-1] == 10
            )
        assert set(filtered) == {10}


@pytest.mark.signals
class TestRegistryDispatch:
    """Tests which handlers the registry runs for a frame."""

    @pytest.mark.asyncio
    async def test_frame_without_handler_channels(
        self, fresh_client: sy.Synnax
    ) -> None:
        """Should not run a filter on a frame without the handler's channels."""
        a, b = create_channel(fresh_client), create_channel(fresh_client)
        a_values: list[SampleValue] = []
        b_values: list[SampleValue] = []

        @fresh_client.signals.on(a.key, filter_f=lambda s: s[a.key] == 1)
        def on_a(state: Latest) -> None:
            a_values.append(state[a.key])

        @fresh_client.signals.on(b.key, filter_f=lambda _: True)
        def on_b(state: Latest) -> None:
            b_values.append(state[b.key])

        async with processing(fresh_client):
            await write_until(fresh_client, b, 2, lambda: len(b_values) > 0)
            await write_until(fresh_client, a, 1, lambda: len(a_values) > 0)
        assert a_values[-1] == 1

    @pytest.mark.asyncio
    async def test_handler_channel_without_value(self, fresh_client: sy.Synnax) -> None:
        """Should not run a filter until every handler channel has a value."""
        a, b = create_channel(fresh_client), create_channel(fresh_client)
        a_values: list[SampleValue] = []
        sums: list[float] = []

        @fresh_client.signals.on(a.key, filter_f=lambda _: True)
        def on_a(state: Latest) -> None:
            a_values.append(state[a.key])

        def sum_above_two(s: Latest) -> bool:
            a_value, b_value = s[a.key], s[b.key]
            assert isinstance(a_value, float)
            assert isinstance(b_value, float)
            return a_value + b_value > 2

        @fresh_client.signals.on([a.key, b.key], filter_f=sum_above_two)
        def on_both(state: Latest) -> None:
            a_value, b_value = state[a.key], state[b.key]
            assert isinstance(a_value, float)
            assert isinstance(b_value, float)
            sums.append(a_value + b_value)

        async with processing(fresh_client):
            await write_until(fresh_client, a, 2, lambda: len(a_values) > 0)
            await write_until(fresh_client, b, 3, lambda: len(sums) > 0)
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

        @proxied_client.signals.on(ch.key, filter_f=lambda _: True)
        def handler(state: Latest) -> None:
            values.append(state[ch.key])

        async with processing(proxied_client):
            await write_until(client, ch, 1, lambda: len(values) > 0)
            assert proxy.websockets == 1
        async with asyncio.timeout(TIMEOUT):
            while proxy.websockets > 0:
                await asyncio.sleep(POLL)
