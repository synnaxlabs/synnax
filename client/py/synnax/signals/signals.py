#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from functools import wraps
from multiprocessing import Pool
from typing import Callable

from synnax import framer
from synnax.channel.payload import ChannelKey, ChannelName, ChannelParams
from synnax.channel.retrieve import ChannelRetriever
from synnax.state import LatestState, State

_InternalHandler = Callable[[State], Callable[[LatestState], None] | None]


class Registry:
    __handlers: list[_InternalHandler]
    __channels: set[ChannelKey | ChannelName]
    __frame_client: framer.Client
    __channel_retriever: ChannelRetriever

    def __init__(self, frame_client: framer.Client, channels: ChannelRetriever):
        self.__handlers = list()
        self.__channels = set()
        self.__frame_client = frame_client
        self.__channel_retriever = channels

    def on(
        self,
        channels: ChannelParams,
        filter_f: Callable[[LatestState], bool],
    ) -> Callable[[Callable[[LatestState], None]], Callable[[], None] | None]:
        self.__channels.update(channels)

        def decorator(f: Callable[[LatestState], None]) -> None:
            @wraps(f)
            def wrapper(state: State) -> Callable[[], None] | None:
                if filter_f(LatestState(state)):
                    return f
                return None

            self.__handlers.append(wrapper)
            return wrapper

        return decorator

    async def process(self) -> None:
        await Scheduler(
            channels=self.__channels,
            handlers=self.__handlers,
            frame_client=self.__frame_client,
            channel_retriever=self.__channel_retriever,
        ).start()


class Scheduler:
    __pool: Pool
    __streamer: framer.AsyncStreamer | None = None
    __handlers: list[_InternalHandler]
    __channels: ChannelParams
    __state: State
    __frame_client: framer.Client
    __channel_retriever: ChannelRetriever

    def __init__(
        self,
        channels: ChannelParams,
        handlers: list[_InternalHandler],
        frame_client: framer.Client,
        channel_retriever: ChannelRetriever,
    ):
        self.__frame_client = frame_client
        self.__channels = channels
        self.__handlers = handlers
        self.__channel_retriever = channel_retriever
        self.__state = State(channel_retriever)

    async def start(self):
        self.__streamer = await self.__frame_client.open_async_streamer(
            list(self.__channels)
        )
        async for frame in self.__streamer:
            self.__state.update(frame)
            for handler in self.__handlers:
                res = handler(self.__state)
                if res is not None:
                    res(LatestState(self.__state))

    def stop(self):
        ...
