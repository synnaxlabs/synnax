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
from functools import wraps

from synnax import channel, framer
from synnax.state import LatestState, State

_InternalHandler = Callable[[State], Callable[[LatestState], None] | None]


class Registry:
    _handlers: list[_InternalHandler]
    _channels: list[channel.Key]
    _frame_client: framer.Client
    _channel_retriever: channel.Retriever

    def __init__(self, frame_client: framer.Client, channels: channel.Retriever):
        self._handlers = list()
        self._channels = list()
        self._frame_client = frame_client
        self._channel_retriever = channels

    def on(
        self,
        channels: channel.Params,
        filter_f: Callable[[LatestState], bool],
    ) -> Callable[[Callable[[LatestState], None]], Callable[[LatestState], None]]:
        normal = channel.normalize_params(channels)
        if normal.variant == "keys":
            self._channels.extend(normal.channels)
        else:
            resolved = self._channel_retriever.retrieve(channels)
            self._channels.extend(ch.key for ch in resolved)

        def decorator(
            f: Callable[[LatestState], None],
        ) -> Callable[[LatestState], None]:
            @wraps(f)
            def wrapper(state: State) -> Callable[[LatestState], None] | None:
                if filter_f(LatestState(state)):
                    return f
                return None

            self._handlers.append(wrapper)
            return f

        return decorator

    async def process(self) -> None:
        await Scheduler(
            channels=self._channels,
            handlers=self._handlers,
            frame_client=self._frame_client,
            channel_retriever=self._channel_retriever,
        ).start()


class Scheduler:
    _streamer: framer.AsyncStreamer | None = None
    _handlers: list[_InternalHandler]
    _channels: list[channel.Key]
    _state: State
    _frame_client: framer.Client
    _channel_retriever: channel.Retriever

    def __init__(
        self,
        channels: list[channel.Key],
        handlers: list[_InternalHandler],
        frame_client: framer.Client,
        channel_retriever: channel.Retriever,
    ):
        self._frame_client = frame_client
        self._channels = channels
        self._handlers = handlers
        self._channel_retriever = channel_retriever
        self._state = State(channel_retriever)

    async def start(self):
        self._streamer = await self._frame_client.open_async_streamer(self._channels)
        async for frame in self._streamer:
            self._state.update(frame)
            for handler in self._handlers:
                res = handler(self._state)
                if res is not None:
                    res(LatestState(self._state))

    def stop(self): ...
