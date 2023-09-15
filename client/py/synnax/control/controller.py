#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations
from collections.abc import Callable, Iterable, Mapping

import numpy as np
from threading import Thread, Event
from typing import Any, Protocol
from asyncio import events, create_task, tasks
from janus import Queue

from synnax import framer
from synnax.channel import ChannelKey, ChannelName, ChannelRetriever, ChannelParams, ChannelPayload
from synnax.telem import TimeStamp

class State:
    value: dict[ChannelKey, np.number]
    __retriever: ChannelRetriever

    def __init__(self, retrieve: ChannelRetriever):
        self.__retriever = retrieve
        self.value = {}

    def update(self, value: framer.Frame):
        for i, key in enumerate(value.columns):
            self.value[key]  = value.series[i][0]

    def __getattr__(self, ch: ChannelKey | ChannelName | ChannelPayload):
        ch = self.__retriever.retrieve(ch)[0]
        return self.value[ch.key]

    def __getitem__(self, ch: ChannelKey | ChannelName | ChannelPayload):
        return self.__getattr__(ch)

class Processor(Protocol):
    def process(self, frame: State) -> Any:
        ...

class WaitUntil(Processor):
    event: Event
    callback: Callable[[State], bool]

    def __init__(self, callback: Callable[[State], bool]):
        self.event = Event()
        self.callback = callback
    
    def process(self, frame: State) -> Any:
        if self.callback(frame):
            self.event.set()
        return None



class Controller:
    writer: framer.Writer
    idx_map: dict[ChannelKey, ChannelKey]
    retriever: ChannelRetriever
    receiver: _Receiver;


    def __init__(
            self,
            write: ChannelParams,
            read: ChannelParams,
            framer: framer.Client,
            retriever: ChannelRetriever
    ) -> None:
        self.writer = framer.new_writer(TimeStamp.now(), write)
        self.receiver = _Receiver(framer, read, retriever)
        self.retriever = retriever
        self.receiver.start()
        

    def set(self, ch: ChannelKey | ChannelName, value: int | float):
        ch = self.retriever.retrieve(ch)[0]
        self.writer.write({ch.key: value, ch.index: TimeStamp.now()})

    def wait_until(self, callback: Callable[[State], bool]):
        processor = WaitUntil(callback)
        self.receiver.processors[processor] = processor
        processor.event.wait()

    def release(self):
        self.writer.close()
        self.receiver.close()

    def __setitem__(self, ch: ChannelKey | ChannelName | ChannelPayload, value: int | float):
        self.set(ch, value)

    def __enter__(self) -> Client:
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> None:
        self.release()

def cancel_all_tasks(loop):
    to_cancel = tasks.all_tasks(loop)
    if not to_cancel:
        return

    for task in to_cancel:
        task.cancel()

    loop.run_until_complete(tasks.gather(*to_cancel, return_exceptions=True))

    for task in to_cancel:
        if task.cancelled():
            continue
        if task.exception() is not None:
            loop.call_exception_handler(
                {
                    "message": "unhandled exception during asyncio.run() shutdown",
                    "exception": task.exception(),
                    "task": task,
                }
            )


class _Receiver(Thread):
    state: State
    channels: ChannelParams
    client: framer.Client
    streamer: framer.AsyncStreamer
    processors: dict[ChannelKey, Callable[[framer.Frame], Any]]
    retriever: ChannelRetriever

    def __init__(
        self, 
        client: framer.Client,
        channels: ChannelParams,
        retriever: ChannelRetriever,
    ):
        self.channels = retriever.retrieve(channels)
        self.client = client
        self.state = State(retriever)
        self.processors = {}
        super().__init__()

    def run(self):
        loop = events.new_event_loop()
        try: 
            events.set_event_loop(loop)

            loop.run_until_complete(self.__run())
        finally:
            try:
                cancel_all_tasks(loop)
                loop.run_until_complete(loop.shutdown_asyncgens())
                loop.run_until_complete(loop.shutdown_default_executor())
            finally:
                events.set_event_loop(None)
                loop.close()

    def __process(self):
        for processor in self.processors.values():
            try: 
                processor.process(self.state)
            except Exception as e:
                # print(e)
                pass

    async def __listen_for_close(self):
        await self.queue.async_q.get()
        await self.streamer.close_loop()

    async def __run(self):
        self.queue = Queue(maxsize=1)
        self.streamer = await self.client.new_async_streamer(self.channels)
        create_task(self.__listen_for_close())

        async for frame in self.streamer:
            self.state.update(frame)
            self.__process()

    def close(self):
        self.queue.sync_q.put(None)



            
