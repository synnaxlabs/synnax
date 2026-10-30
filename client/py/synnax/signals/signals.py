#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass

from synnax import channel, framer
from synnax.state import Latest, State


@dataclass
class _Handler:
    channels: set[channel.Key]
    filter: Callable[[Latest], bool]
    f: Callable[[Latest], None]


class Registry:
    _handlers: list[_Handler]
    _channels: list[channel.Key]
    _frame_client: framer.Client
    _channel_retriever: channel.Retriever

    def __init__(self, *, frame_client: framer.Client, channels: channel.Retriever):
        self._handlers = []
        self._channels = []
        self._frame_client = frame_client
        self._channel_retriever = channels

    def on(
        self,
        channels: channel.Params,
        *,
        filter_f: Callable[[Latest], bool],
    ) -> Callable[[Callable[[Latest], None]], Callable[[Latest], None]]:
        """Registers the decorated function to run when filter_f passes.

        filter_f runs on each frame that holds one of the channels, once every channel
        has a value.

        :param channels: The channels the handler reads.
        :param filter_f: Returns true when the handler should run.
        :returns: A decorator that returns the function unchanged.
        """
        normal = channel.normalize_params(channels)
        if isinstance(normal, channel.NormalizedKeyResult):
            keys = set(normal.channels)
        else:
            keys = {ch.key for ch in self._channel_retriever.retrieve(channels)}
        self._channels.extend(keys)

        def decorator(
            f: Callable[[Latest], None],
        ) -> Callable[[Latest], None]:
            self._handlers.append(_Handler(channels=keys, filter=filter_f, f=f))
            return f

        return decorator

    async def process(self) -> None:
        """Streams the registered channels and runs handlers until cancelled. Closes the
        streamer on exit.
        """
        state = State(self._channel_retriever.retrieve(self._channels))
        async with await self._frame_client.open_async_streamer(
            self._channels
        ) as streamer:
            async for frame in streamer:
                state.update(frame)
                latest = Latest(state)
                for h in self._handlers:
                    if (
                        any(key in frame for key in h.channels)
                        and h.channels.issubset(state.value)
                        and h.filter(latest)
                    ):
                        h.f(latest)
