#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Callable
from multiprocessing import Pool

import numpy as np

from synnax import framer
from synnax import Synnax
from synnax.channel.payload import ChannelParams, ChannelKey, ChannelName
from synnax.state import State, LatestState

# At the end of the day, everything can be characterized by:
# 1. A filter that takes in one or more channel values and determines
# whether a workflow should be executed.
# 2. A workflow that takes in some operational context and executes in
# a separate thread or process.


_InternalHandler = Callable[[State], Callable[[Synnax], None] | None]


class Registry:
    def on(
        self,
        channels: ChannelParams,
        filter_f: Callable[[LatestState], bool],
    ):
        ...


class Scheduler:
    __pool: Pool
    __streamer: framer.AsyncStreamer | None = None
    __handlers: list[_InternalHandler]
    __channels: set[ChannelKey | ChannelName]
    __client: Synnax
    __state: State

    def __init__(self, client: Synnax):
        self.__pool = Pool()
        self.__handlers = list()
        self.__channels = set()
        self.__client = client

    async def start(self):
        self.__streamer = await self.__client.new_async_streamer(
            list(self.__channels),
        )
        async for frame in self.__streamer:
            self.__state.update(frame)
            for handler in self.__handlers:
                res = handler(self.__state)
                if res is not None:
                    self.__pool.apply_async(res, (self.__client,))

    def stop(self):
        ...

    def add(
        self,
        channels: ChannelParams,
        handler: _InternalHandler,
    ):
        self.__handlers.append(handler)
        self.__channels.update(channels.keys)
