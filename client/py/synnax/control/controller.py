#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Callable
from threading import Event
from typing import Any, Protocol, overload
from asyncio import create_task

import numpy as np
from janus import Queue

from synnax.util.thread import AsyncThread
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


class Processor(Protocol):
    def process(self, state: Controller) -> Any:
        ...


class WaitUntil(Processor):
    event: Event
    callback: Callable[[Controller], bool]
    exc: Exception | None

    def __init__(self, callback: Callable[[Controller], bool]):
        self.event = Event()
        self.callback = callback
        self.exc = None

    def process(self, state: Controller) -> Any:
        try:
            if self.callback(state):
                self.event.set()
        except Exception as e:
            self.exc = e
            self.event.set()
        return None


class Controller:
    _writer: framer.Writer
    _idx_map: dict[ChannelKey, ChannelKey]
    _retriever: ChannelRetriever
    _receiver: _Receiver

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
        self._writer = frame_client.open_writer(
            name=name,
            start=TimeStamp.now(),
            channels=write_keys,
            authorities=write_authorities,
        )
        self._receiver = _Receiver(frame_client, read, retriever, self)
        self._retriever = retriever
        self._receiver.start()
        self._receiver.startup_ack.wait()

    def set(
        self,
        ch: ChannelKey | ChannelName | dict[ChannelKey | ChannelName, int | float],
        value: int | float | None = None,
    ):
        if isinstance(ch, dict):
            values = list(ch.values())
            channels = retrieve_required(self._retriever, list(ch.keys()))
            now = TimeStamp.now()
            updated = {channels[i].key: values[i] for i in range(len(channels))}
            updated_idx = {channels[i].index: now for i in range(len(channels))}
            self._writer.write({**updated, **updated_idx})
            return
        ch = retrieve_required(self._retriever, ch)[0]
        self._writer.write({ch.key: value, ch.index: TimeStamp.now()})

    def authorize(self, ch: ChannelKey | ChannelName, value: Authority):
        ch = retrieve_required(self._retriever, ch)[0]
        self._writer.set_authority({ch.key: value, ch.index: value})

    def wait_until(
        self,
        callback: Callable[[Controller], bool],
        timeout: CrudeTimeSpan = None,
    ) -> bool:
        if not callable(callback):
            raise ValueError("First argument to wait_until must be a callable.")
        processor = WaitUntil(callback)
        try:
            self._receiver.processors.add(processor)
            timeout_seconds = TimeSpan(timeout).seconds if timeout else None
            ok = processor.event.wait(timeout=timeout_seconds)
        finally:
            self._receiver.processors.remove(processor)
        if processor.exc:
            raise processor.exc
        return ok

    def wait_until_defined(
        self,
        channels: list[ChannelKey | ChannelName],
        timeout: CrudeTimeSpan = None,
    ) -> bool:
        channels = retrieve_required(self._retriever, channels)

        def f(c: Controller) -> bool:
            return all(v in c.state for v in channels)

        return self.wait_until(f, timeout)

    def release(self):
        self._writer.close()
        self._receiver.close()

    def __setitem__(
        self,
        ch: ChannelKey | ChannelName | ChannelPayload,
        value: int | float
    ):
        self.set(ch, value)

    @property
    def state(self) -> dict[ChannelKey, np.number]:
        return self._receiver.state

    def __setattr__(self, key, value):
        try:
            super().__setattr__(key, value)
        except AttributeError:
            self.set(key, value)

    @overload
    def get(self, ch: ChannelKey | ChannelName) -> int | float | None:
        ...

    @overload
    def get(self, ch: ChannelKey | ChannelName, default: int | float) -> int | float:
        ...

    def get(
        self,
        ch: ChannelKey | ChannelName,
        default: int | float = None
    ) -> int | float | None:
        ch = retrieve_required(self._retriever, ch)[0]
        return self._receiver.state.get(ch.key, default)

    def __getitem__(self, item):
        ch = retrieve_required(self._retriever, item)[0]
        try:
            return self._receiver.state[ch.key]
        except KeyError:
            raise KeyError(f"""
            Channel {ch} not found in controller state. This is for one of two reasons:

            1. The channel was not included in the read_from argument passed to
            client.control.acquire.

            2. No data has been received for the channel yet. If you'd like to block
            until a value exists for the channel in state, use the wait_until_defined
            method.
            """)

    def __getattr__(self, item):
        try:
            return super().__getattribute__(item)
        except AttributeError:
            return self[item]

    def __enter__(self) -> Controller:
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> None:
        self.release()


class _Receiver(AsyncThread):
    state: dict[ChannelKey, np.number]
    channels: ChannelParams
    client: framer.Client
    streamer: framer.AsyncStreamer
    processors: set[Processor]
    retriever: ChannelRetriever
    controller: Controller
    startup_ack: Event
    shutdown_ack: Event

    def __init__(
        self,
        client: framer.Client,
        channels: ChannelParams,
        retriever: ChannelRetriever,
        controller: Controller,
    ):
        self.channels = retriever.retrieve(channels)
        self.client = client
        self.state = dict()
        self.controller = controller
        self.startup_ack = Event()
        self.shutdown_ack = Event()
        self.processors = set()
        super().__init__()

    def __process(self):
        for p in self.processors:
            p.process(self.controller)

    async def __listen_for_close(self):
        await self.queue.async_q.get()
        await self.streamer.close_loop()

    async def run_async(self):
        self.queue = Queue(maxsize=1)
        self.streamer = await self.client.open_async_streamer(self.channels)
        self.startup_ack.set()
        create_task(self.__listen_for_close())

        async for frame in self.streamer:
            for i, key in enumerate(frame.channels):
                self.state[key] = frame.series[i][-1]
            self.__process()

        self.shutdown_ack.set()

    def close(self):
        self.queue.sync_q.put(None)
        self.shutdown_ack.wait()
