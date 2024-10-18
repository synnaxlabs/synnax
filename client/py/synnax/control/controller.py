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
from threading import Event, Lock
from typing import Any, Protocol, overload
from asyncio import create_task, Future

import numpy as np

from synnax.util.thread import AsyncThread
from synnax import framer, ValidationError
from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
    ChannelParams,
    ChannelPayload,
)
from synnax.channel.retrieve import ChannelRetriever, retrieve_required
from synnax.telem import CrudeTimeSpan, TimeSpan, TimeStamp
from synnax.telem.control import CrudeAuthority
from synnax.timing import sleep


class Processor(Protocol):
    def process(self, state: Controller) -> Any:
        ...


class WaitUntil(Processor):
    event: Event
    callback: Callable[[Controller], bool]
    exc: Exception | None
    reverse: bool

    def __init__(self, callback: Callable[[Controller], bool], reverse: bool = False):
        self.event = Event()
        self.callback = callback
        self.exc = None
        self.reverse = reverse

    def process(self, state: Controller) -> Any:
        try:
            res = self.callback(state)
            if self.reverse:
                res = not res
            if res:
                self.event.set()
        except Exception as e:
            self.exc = e
            self.event.set()
        return None


class RemainsTrueFor(Processor):
    event: Event
    callback: Callable[[Controller], bool]
    exc: Exception | None
    target: float
    actual: float = 0
    count: int = 0

    def __init__(
        self,
        callback: Callable[[Controller], bool],
        percentage: float,
    ):
        self.event = Event()
        self.callback = callback
        self.exc = None
        self.target = percentage

    def process(self, state: Controller) -> Any:
        try:
            v = self.callback(state)
            if v is False and self.target >= 1:
                self.event.set()
            else:
                self.actual = (self.actual * self.count + v) / (self.count + 1)
                self.count += 1
        except Exception as e:
            self.exc = e
            self.event.set()
        return None


class Controller:
    _writer_opt: framer.Writer | None = None
    _receiver_opt: _Receiver | None = None
    _idx_map: dict[ChannelKey, ChannelKey]
    _retriever: ChannelRetriever

    def __init__(
        self,
        name: str,
        write: ChannelParams | None,
        read: ChannelParams | None,
        frame_client: framer.Client,
        retriever: ChannelRetriever,
        write_authorities: CrudeAuthority | list[CrudeAuthority],
    ) -> None:
        self._retriever = retriever
        if write is not None and len(write) > 0:
            write_channels = retrieve_required(self._retriever, write)
            write_keys = [ch.index for ch in write_channels if ch.index != 0]
            write_keys.extend([ch.key for ch in write_channels])
            self._writer_opt = frame_client.open_writer(
                name=name,
                start=TimeStamp.now(),
                channels=write_keys,
                authorities=write_authorities,
            )
        if read is not None and len(read) > 0:
            self._receiver_opt = _Receiver(frame_client, read, retriever, self)
            self._receiver.start()
            self._receiver.startup_ack.wait()

    @property
    def _writer(self) -> framer.Writer:
        if self._writer_opt is None:
            raise ValidationError(
                """
            tried to command a channel but no channels were passed into the write
            argument when calling acquire()!
            """
            )
        return self._writer_opt

    @property
    def _receiver(self) -> _Receiver:
        if self._receiver_opt is None:
            raise ValidationError(
                """
            tried to read from a channel but no channels were passed into the read
            argument when calling acquire()!
            """
            )
        return self._receiver_opt

    @overload
    def set(self, ch: ChannelKey | ChannelName, value: int | float | bool):
        ...

    @overload
    def set(self, ch: dict[ChannelKey | ChannelName, int | float]):
        ...

    def set(
        self,
        channel: ChannelKey | ChannelName | dict[ChannelKey | ChannelName, int | float],
        value: int | float | None = None,
    ):
        """Sets the provided channel(s) to the provided value(s).

        :param channel: A single channel key or name, or a dictionary of channel keys and
        names to their corresponding values to set.
        :param value: The value to set the channel to. This parameter should not be
        provided if ch is a dictionary.

        Examples:
        >>> controller.set("my_channel", 42)
        >>> controller.set({
        ...     "channel_1": 42,
        ...     "channel_2": 3.14,
        ... })
        """
        if isinstance(channel, dict):
            values = list(channel.values())
            channels = retrieve_required(self._retriever, list(channel.keys()))
            now = TimeStamp.now()
            updated = {channels[i].key: values[i] for i in range(len(channels))}
            updated_idx = {
                channels[i].index: now
                for i in range(len(channels))
                if not channels[i].virtual
            }
            self._writer.write({**updated, **updated_idx})
            return
        ch = self._retriever.retrieve_one(channel)
        to_write = {ch.key: value}
        if not ch.virtual:
            to_write[ch.index] = TimeStamp.now()
        self._writer.write(to_write)

    @overload
    def set_authority(
        self,
        value: CrudeAuthority,
    ) -> bool:
        ...

    @overload
    def set_authority(
        self,
        value: dict[ChannelKey | ChannelName, CrudeAuthority],
    ) -> bool:
        ...

    @overload
    def set_authority(
        self,
        ch: ChannelKey | ChannelName,
        value: CrudeAuthority,
    ) -> bool:
        ...

    def set_authority(
        self,
        value: (
            dict[ChannelKey | ChannelName | ChannelPayload, CrudeAuthority]
            | ChannelKey
            | ChannelName
            | CrudeAuthority
        ),
        authority: CrudeAuthority | None = None,
    ) -> bool:
        if isinstance(value, dict):
            channels = retrieve_required(self._retriever, list(value.keys()))
            for ch in channels:
                value[ch.index] = value.get(ch.key, value.get(ch.name))
        elif authority is not None:
            ch = self._retriever.retrieve_one(value)
            value = {ch.key: authority, ch.index: authority}
        return self._writer.set_authority(value)

    def wait_until(
        self,
        cond: Callable[[Controller], bool],
        timeout: float | int | TimeSpan = None,
    ) -> bool:
        """Blocks the controller, calling the provided callback on every new sample
        received by the controller. Once the callback returns True, the method will
        return. If a timeout is provided, the method will return False if the timeout is
        reached before the callback returns True.

        CAVEAT: Do not call wait_until from within a callback that is being processed by
        the controller. This will cause a deadlock.

        :param cond: A callable that takes the controller as an argument and returns
        a boolean. The controller will execute this callback on every new sample
        received to the channels it is reading from. The controller will block until
        the callback returns True.
        :param timeout: An optional timeout in seconds. If the timeout is reached before
        the callback returns True, the method will return False.
        :returns: True if the callback returned True before the timeout,
        False otherwise.

        Examples:
        >>> controller.wait_until(lambda c: c["my_channel"] > 42)
        >>> controller.wait_until(lambda c: c["channel_1"] > 42 and c["channel_2"] < 3.14)
        """
        return self._internal_wait_until(cond, timeout)

    def wait_while(
        self,
        cond: Callable[[Controller], bool],
        timeout: CrudeTimeSpan = None,
    ) -> bool:
        """Blocks the controller, calling the provided callback on every new sample
        received. The controller will continue to block until the
        callback returns False. If a timeout is provided, the method will return False
        if the timeout is reached before the callback returns False.
        """
        return self._internal_wait_until(cond, timeout, reverse=True)

    def _internal_wait_until(
        self,
        cond: Callable[[Controller], bool],
        timeout: CrudeTimeSpan = None,
        reverse: bool = False,
    ):
        if not callable(cond):
            raise ValueError("First argument to wait_until must be a callable.")
        processor = WaitUntil(cond, reverse)
        try:
            self._receiver.add_processor(processor)
            timeout_seconds = (
                TimeSpan.from_seconds(timeout).seconds if timeout else None
            )
            ok = processor.event.wait(timeout=timeout_seconds)
        finally:
            self._receiver.remove_processor(processor)
        if processor.exc:
            raise processor.exc
        return ok

    def sleep(self, dur: float | int | TimeSpan, precise: bool = False):
        """Sleeps the controller for the provided duration.

        :param dur: The duration to sleep for. This can be a flot or int representing
        the number of seconds to sleep, or a synnax TimeSpan object.
        :param precise: If True, the controller will use a more precise sleep method
        that is more accurate for short durations, but consumes more CPU resources.
        If you're looking for millisecond level precision, set this value to True.
        Otherwise, we recommend leaving it as False.
        """
        sleep(dur, precise)

    def wait_until_defined(
        self,
        channels: ChannelKey | ChannelName | list[ChannelKey | ChannelName],
        timeout: CrudeTimeSpan = None,
    ) -> bool:
        """Blocks until the controller has received at least one value from all the
        provided channels. This is useful for ensuring that the controlled has reached
        a valid state before proceeding.

        :param channels: A single channel key or name, or a list of channel keys or
        names to wait for.
        :param timeout: An optional timeout in seconds. If the timeout is reached before
        the channels are defined, the method will return False.

        Examples:
        >>> controller.wait_until_defined("my_channel")
        >>> controller.wait_until_defined(["channel_1", "channel_2"])
        """
        res = retrieve_required(self._retriever, channels)
        return self.wait_until(lambda c: all(v.key in c.state for v in res), timeout)

    def remains_true_for(
        self,
        cond: Callable[[Controller], bool],
        duration: CrudeTimeSpan,
        percentage: float = 1,
    ) -> bool:
        """Blocks the controller and repeatedly checks that the provided callback
        returns True for the entire duration. Also accepts a decimal target percentage,
        which can be used to check that the callback returns True for a certain
        percentage of the duration.

        :param cond: A callable that takes the controller as an argument and returns
        a boolean. The controller will execute this callback on every new sample
        received to the channels it is reading from.
        :param duration: The duration in seconds to check that the callback returns True.
        :param percentage: The target percentage of the duration that the callback
        should return True. If this value is greater than 1, the controller will immediately
        unblock if the callback returns False. If this value is less than 1, the controller
        will return the result after the entire duration has elapsed.
        """
        if not callable(cond):
            raise ValueError("First argument to remains_true_for must be a callable.")
        processor = RemainsTrueFor(cond, percentage)
        try:
            self._receiver.processors.add(processor)
            timeout_seconds = TimeSpan(duration).seconds
            # If the event is set, this means the target percentage was >= 1 and the
            # callback returned False, so we just exit immediately.
            ok = not processor.event.wait(timeout=timeout_seconds)
        finally:
            self._receiver.processors.remove(processor)
        # If we executed the callback for the entire duration and the actual percentage
        # is still less than the target percentage, we return False.
        if ok:
            if processor.count == 0:
                return False
            ok = processor.actual >= processor.target
        if processor.exc:
            raise processor.exc
        return ok

    def release(self):
        """Release control and shuts down the controller. No further control operations
        can be performed after calling this method.
        """
        if self._writer_opt is not None:
            self._writer_opt.close()
        if self._receiver_opt is not None:
            self._receiver.stop()

    def __setitem__(
        self, ch: ChannelKey | ChannelName | ChannelPayload, value: int | float
    ):
        self.set(ch, value)

    @property
    def state(self) -> dict[ChannelKey, np.number]:
        """
        :returns: The current state of all channels passed to read_from in the acquire
        method. This is a dictionary of channel keys to their most recent values. It's
        important to note that this dictionary may not contain entries for all channels
        passed to read_from if no data has been received for them yet.
        """
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
        self, ch: ChannelKey | ChannelName, default: int | float = None
    ) -> int | float | None:
        """Gets the most recent value for the provided channel, and returns the default
        value if no value has been received yet.

        :param ch: The channel key or name to get the value for.
        :param default: The default value to return if no value has been received for the
        channel yet.

        Examples:
        >>> controller.get("my_channel")
        >>> controller.get("my_channel", 42)
        """
        ch = self._retriever.retrieve_one(ch)
        return self._receiver.state.get(ch.key, default)

    def __getitem__(self, item):
        ch = self._retriever.retrieve_one(item)
        try:
            return self._receiver.state[ch.key]
        except KeyError:
            raise KeyError(
                f"""
            Channel {ch} not found in controller state. This is for one of two reasons:

            1. The channel was not included in the read_from argument passed to
            client.control.acquire.

            2. No data has been received for the channel yet. If you'd like to block
            until a value exists for the channel in state, use the wait_until_defined
            method.
            """
            )

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
    processor_lock: Lock
    retriever: ChannelRetriever
    controller: Controller
    startup_ack: Event
    shutdown_future: Future

    def __init__(
        self,
        client: framer.Client,
        channels: ChannelParams,
        retriever: ChannelRetriever,
        controller: Controller,
    ):
        super().__init__()
        self.channels = retriever.retrieve(channels)
        self.client = client
        self.state = dict()
        self.controller = controller
        self.processor_lock = Lock()
        self.startup_ack = Event()
        self.processors = set()

    def add_processor(self, processor: Processor):
        with self.processor_lock:
            self.processors.add(processor)

    def remove_processor(self, processor: Processor):
        with self.processor_lock:
            self.processors.remove(processor)

    def _process(self):
        with self.processor_lock:
            for p in self.processors:
                p.process(self.controller)

    async def _listen_for_close(self):
        await self.shutdown_future
        await self.streamer.close_loop()

    async def run_async(self):
        self.streamer = await self.client.open_async_streamer(self.channels)
        self.shutdown_future = self.loop.create_future()
        self.loop.create_task(self._listen_for_close())
        self.startup_ack.set()
        async for frame in self.streamer:
            for i, key in enumerate(frame.channels):
                self.state[key] = frame.series[i][-1]
            self._process()

    def stop(self):
        self.loop.call_soon_threadsafe(self.shutdown_future.set_result, None)
        self.join()
