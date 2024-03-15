#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import uuid
from asyncio import create_task, events, tasks
from collections.abc import Callable
from threading import Event, Thread
from typing import Any, Protocol

import numpy as np
from janus import Queue

from synnax import framer
from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
    ChannelParams,
    ChannelPayload,
)
from synnax.channel.retrieve import ChannelRetriever, retrieve_required
from synnax.telem import CrudeTimeSpan, TimeSpan, TimeStamp
from synnax.telem.control import Authority, CrudeAuthority


class State:
    value: dict[ChannelKey, np.number]
    __retriever: ChannelRetriever

    def __init__(self, retrieve: ChannelRetriever):
        self.__retriever = retrieve
        self.value = dict()

    def update(self, value: framer.Frame):
        for i, key in enumerate(value.columns):
            self.value[key] = value.series[i][0]

    def __getattr__(self, ch: ChannelKey | ChannelName | ChannelPayload):
        ch = retrieve_required(self.__retriever, ch)[0]
        return self.value[ch.key]

    def __getitem__(self, ch: ChannelKey | ChannelName | ChannelPayload):
        return self.__getattr__(ch)


class Processor(Protocol):
    def process(self, state: Controller) -> Any:
        ...


class WaitUntil(Processor):
    event: Event
    callback: Callable[[Controller], bool]

    def __init__(self, callback: Callable[[Controller], bool]):
        self.event = Event()
        self.callback = callback

    def process(self, state: Controller) -> Any:
        if self.callback(state):
            self.event.set()
        return None


class Controller:
    writer: framer.Writer
    idx_map: dict[ChannelKey, ChannelKey]
    retriever: ChannelRetriever
    receiver: _Receiver

    def __init__(
        self,
        name: str,
        write: ChannelParams,
        read: ChannelParams,
        frame_client: framer.Client,
        retriever: ChannelRetriever,
        write_authorities: CrudeAuthority | list[CrudeAuthority],
    ) -> None:
        write_channels = retrieve_required(retriever, write)
        write_keys = [ch.index for ch in write_channels if ch.index != 0]
        write_keys.extend([ch.key for ch in write_channels])
        self.writer = frame_client.new_writer(
            name=name,
            start=TimeStamp.now(),
            channels=write_keys,
            authorities=write_authorities,
        )
        self.receiver = _Receiver(frame_client, read, retriever, self)
        self.retriever = retriever
        self.receiver.start()
        self.receiver.bootup_ack.wait()

    def set(
        self,
        ch: ChannelKey | ChannelName | dict[ChannelKey | ChannelName, int | float],
        value: int | float | None = None,
    ):
        if isinstance(ch, dict):
            values = list(ch.values())
            channels = retrieve_required(self.retriever, list(ch.keys()))
            now = TimeStamp.now()
            updated = {channels[i].key: values[i] for i in range(len(channels))}
            updated_idx = {channels[i].index: now for i in range(len(channels))}
            self.writer.write({**updated, **updated_idx})
            return
        ch = retrieve_required(self.retriever, ch)[0]
        self.writer.write({ch.key: value, ch.index: TimeStamp.now()})

    def authorize(self, ch: ChannelKey | ChannelName, value: Authority):
        ch = retrieve_required(self.retriever, ch)[0]
        self.writer.set_authority({ch.key: value, ch.index: value})

    def wait_until(
        self,
        callback: Callable[[Controller], bool],
        timeout: CrudeTimeSpan = None,
    ) -> bool:
        processor = WaitUntil(callback)
        key = uuid.uuid4()
        try:
            self.receiver.processors[key] = processor
            ok = processor.event.wait(
                timeout=TimeSpan(timeout).seconds if timeout else None
            )
        finally:
            del self.receiver.processors[key]
        return ok

    def release(self):
        self.writer.close()
        self.receiver.close()

    def __setitem__(
        self, ch: ChannelKey | ChannelName | ChannelPayload, value: int | float
    ):
        self.set(ch, value)

    def __setattr__(self, key, value):
        try:
            super().__setattr__(key, value)
        except AttributeError:
            self.set(key, value)

    def get(self, ch: ChannelKey | ChannelName) -> int | float:
        ch = retrieve_required(self.retriever, ch)[0]
        return self.receiver.state[ch.key]

    def __getitem__(self, item):
        return self.get(item)

    def __getattr__(self, item):
        try:
            return super().__getattr__(item)
        except AttributeError:
            return self.get(item)

    def __enter__(self) -> Controller:
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
    processors: dict[uuid.UUID, Processor]
    retriever: ChannelRetriever
    controller: Controller
    bootup_ack: Event

    def __init__(
        self,
        client: framer.Client,
        channels: ChannelParams,
        retriever: ChannelRetriever,
        controller: Controller,
    ):
        self.channels = retriever.retrieve(channels)
        self.client = client
        self.state = State(retriever)
        self.controller = controller
        self.bootup_ack = Event()
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
            processor.process(self.controller)

    async def __listen_for_close(self):
        await self.queue.async_q.get()
        await self.streamer.close_loop()

    async def __run(self):
        self.queue = Queue(maxsize=1)
        self.streamer = await self.client.new_async_streamer(self.channels)
        self.bootup_ack.set()
        create_task(self.__listen_for_close())

        async for frame in self.streamer:
            self.state.update(frame)
            self.__process()

    def close(self):
        self.queue.sync_q.put(None)
