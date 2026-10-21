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
from synnax.state import LatestState
from synnax.telem import SampleValue

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


@pytest.mark.signals
class TestRegistry:
    """Tests registering signal handlers and dispatching streamed values to them."""

    def test_on_returns_handler(
        self, fresh_client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should return the decorated handler unchanged."""

        def handler(_: LatestState) -> None: ...

        decorated = fresh_client.signals.on(virtual_channel.key, lambda _: True)
        assert decorated(handler) is handler

    @pytest.mark.asyncio
    async def test_process_by_key(
        self, fresh_client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        """Should call a handler registered by key with the latest value."""
        values: list[SampleValue] = []

        @fresh_client.signals.on(virtual_channel.key, lambda _: True)
        def handler(state: LatestState) -> None:
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

        @fresh_client.signals.on(virtual_channel.name, lambda _: True)
        def handler(state: LatestState) -> None:
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

        def above_five(state: LatestState) -> bool:
            value = state[key]
            return isinstance(value, float) and value > 5

        @fresh_client.signals.on(key, above_five)
        def above(state: LatestState) -> None:
            filtered.append(state[key])

        @fresh_client.signals.on(key, lambda _: True)
        def all_values(state: LatestState) -> None:
            every.append(state[key])

        async with processing(fresh_client):
            await write_until(fresh_client, virtual_channel, 1, lambda: len(every) > 0)
            await write_until(
                fresh_client, virtual_channel, 10, lambda: every[-1] == 10
            )
        assert set(filtered) == {10}
