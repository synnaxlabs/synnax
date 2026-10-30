#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import asyncio
import time
from typing import cast, overload

from pydantic import BaseModel

import synnax.channel.payload as channel
from freighter import (
    EOF,
    AsyncStream,
    AsyncStreamClient,
    Stream,
    Unreachable,
    WebsocketClient,
)
from freighter.transport import P
from freighter.websocket import Message
from synnax.framer.adapter import ReadFrameAdapter
from synnax.framer.codec import LOW_PERF_SPECIAL_CHAR, WSCodec
from synnax.framer.common import validate_downsample_factor
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import CrudeTimeSpan, TimeSpan

DEFAULT_KEEP_ALIVE = TimeSpan.SECOND * 5
# Missing this many keep-alive intervals in a row fails the pending read: one is normal
# jitter, three is a dead connection.
_KEEP_ALIVE_DEADLINE_FACTOR = 3


class _Request(BaseModel):
    keys: list[channel.Key]
    downsample_factor: int
    throttle_rate: float | None = None
    exclude_groups: list[int] = []
    keep_alive: TimeSpan = TimeSpan(0)


class _Response(BaseModel):
    frame: FramePayload
    keep_alive: bool = False


class _Liveness:
    """Tracks when a streamer must presume its connection dead."""

    def __init__(self, keep_alive: TimeSpan) -> None:
        self.keep_alive = keep_alive
        self.deadline = keep_alive * _KEEP_ALIVE_DEADLINE_FACTOR
        # Set once the Core proves keep-alive support by sending one, so the deadline
        # never arms against a Core that will not send them.
        self._armed = False
        self._last_response = 0.0
        self._grace: float | None = None

    def received(self, res: _Response) -> None:
        self._last_response = time.monotonic()
        self._grace = None
        if res.keep_alive and self.deadline > 0:
            self._armed = True

    def dead_at(self) -> float | None:
        """Returns the monotonic time at which the connection is presumed dead, or
        None if the deadline is not armed.
        """
        if not self._armed:
            return None
        dead = self._last_response + self.deadline.seconds
        now = time.monotonic()
        if dead > now:
            return dead
        # The caller held the thread or loop past the deadline, so responses may sit
        # unread in the socket. Give them one keep-alive interval, once.
        if self._grace is None:
            self._grace = now + self.keep_alive.seconds
        return self._grace

    def unreachable(self) -> Unreachable:
        return Unreachable(message=f"streamer received no response for {self.deadline}")


class _WSStreamerCodec(WSCodec):
    def encode(self, data: BaseModel) -> bytes:
        return self.lower_perf_codec.encode(data)

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        if data[0] == LOW_PERF_SPECIAL_CHAR:
            return self.lower_perf_codec.decode(data[1:], pld_t)
        frame = self.codec.decode(data, offset=1)
        return cast(P, Message(type="data", payload=_Response(frame=frame)))


_ENDPOINT = "/frame/stream"


class Streamer:
    """A streamer is used to stream frames of telemetry in real-time from a Synnax
    cluster. It should not be constructed directly, and should instead be created using
    the client's `open_streamer` method.

    To open a streamer, use the `open_streamer` method on the client and pass in the
    list of channels you'd like to stream. Once a new streamer has been opened, you
    can call the `read` method to read the next frame of telemetry. Once done, call
    the `close` method to close the streamer and free all necessary resources. We
    recommend using the streamer as a context manager to ensure that it is closed
    properly.

    Streamers also support the iterator protocol, allowing you to iterate over the
    frames of telemetry as they are received. This is useful when you want to process
    each frame as it is received.

    For detailed documentation, see
    https://docs.synnaxlabs.com/reference/client/stream-data
    """

    _stream: Stream[_Request, _Response]
    _adapter: ReadFrameAdapter

    def __init__(
        self,
        *,
        client: WebsocketClient,
        adapter: ReadFrameAdapter,
        downsample_factor: int = 1,
        throttle_rate: float = 0,
        exclude_groups: list[int] | None = None,
        keep_alive: CrudeTimeSpan = DEFAULT_KEEP_ALIVE,
    ) -> None:
        validate_downsample_factor(downsample_factor)
        keep_alive = TimeSpan.from_seconds(keep_alive)
        self._liveness = _Liveness(keep_alive)
        self._adapter = adapter
        client = client.with_codec(_WSStreamerCodec(self._adapter.codec))
        self._stream = client.stream(_ENDPOINT, _Request, _Response)
        self._downsample_factor = downsample_factor
        self._throttle_rate = throttle_rate
        self._exclude_groups = exclude_groups or []
        self._stream.send(
            _Request(
                keys=self._adapter.keys,
                downsample_factor=self._downsample_factor,
                throttle_rate=self._throttle_rate,
                exclude_groups=self._exclude_groups,
                keep_alive=keep_alive,
            )
        )
        # A keep-alive can beat the open ack onto the wire, so the ack is the first
        # non-keep-alive response.
        while self._receive(None).keep_alive:
            pass

    @overload
    def read(self, *, timeout: float | int | TimeSpan) -> Frame | None:
        """Reads the next frame of telemetry from the streamer with a timeout. If no
        frame is received within the timeout, this method will return None. If an error
        occurs while reading the frame, an exception will be raised.

        :param timeout: The maximum amount of time to wait for a frame to be received
        before returning None. This can be a float or integer representing the number
        of seconds, or a synnax TimeSpan object.
        :return: The next frame of telemetry, or None if no frame is received within the
        timeout.
        """

    @overload
    def read(self) -> Frame:
        """Reads the next frame of telemetry from the streamer, blocking until a frame
        is received. If an error occurs while reading the frame, an exception will be
        raised.
        """

    def read(self, *, timeout: float | None = None) -> Frame | None:
        """Reads the next frame of telemetry from the streamer. If a timeout is
        provided, this method will wait for the specified amount of time for a frame to
        be received. If no frame is received within the timeout, this method will return
        None. If no timeout is provided, this method will block until a frame is
        received.

        If an error occurs while reading the frame, an exception will be raised.

        :param timeout: The maximum amount of time to wait for a frame to be received
        before returning None. If no timeout is provided, this method will block until a
        frame is received.
        :return: The next frame of telemetry, or None if no frame is received within the
        timeout.
        :raises Unreachable: If keep-alives were flowing and then no response arrives,
        across reads, for the keep-alive deadline: the connection is presumed dead.
        """
        seconds = TimeSpan.to_seconds(timeout)
        end = None if seconds is None else time.monotonic() + seconds
        while True:
            dead = self._liveness.dead_at()
            deadline_bound = dead is not None and (end is None or dead <= end)
            expiry = dead if deadline_bound else end
            remaining = None if expiry is None else max(expiry - time.monotonic(), 0)
            try:
                res = self._receive(remaining)
            except TimeoutError as e:
                if deadline_bound:
                    raise self._liveness.unreachable() from e
                return None
            if not res.keep_alive:
                return self._adapter.adapt(Frame(res.frame))

    def _receive(self, timeout: float | None) -> _Response:
        res = self._stream.receive(timeout=timeout)
        self._liveness.received(res)
        return res

    def update_channels(self, channels: channel.Params) -> None:
        """Updates the list of channels to stream. This method will replace the current
        list of channels with the new list, not add to it.

        :param channels: The list of channels to stream.
        :raises NotFoundError: If any of the channels in the list are not found.
        """
        self._adapter.update(channels)
        self._stream.send(
            _Request(
                keys=self._adapter.keys,
                downsample_factor=self._downsample_factor,
                throttle_rate=self._throttle_rate,
                exclude_groups=self._exclude_groups,
            )
        )

    def close(self, *, timeout: float | int | TimeSpan | None = None) -> None:
        """Closes the streamer and frees all network resources.

        :param timeout: The maximum amount of time to wait for the server to acknowledge
        the closure before raising a TimeoutError. This can be a float or integer
        representing the number of seconds, or a synnax TimeSpan object. If no timeout
        is provided, this method will block until the server acknowledges the closure.
        """
        self._stream.close_send()
        while True:
            try:
                self._stream.receive(timeout=TimeSpan.to_seconds(timeout))
            except EOF:
                break

    def __iter__(self) -> Streamer:
        """Returns an iterator object that can be used to iterate over the frames of
        telemetry as they are received. This is useful when you want to process each
        frame as it is received.
        """
        return self

    def __enter__(self) -> Streamer:
        """Returns the streamer object when used as a context manager."""
        return self

    def __next__(self) -> Frame:
        """Reads the next frame of telemetry from the streamer."""
        return self.read()

    def __exit__(self, exc_type: object, exc_val: object, exc_tb: object) -> None:
        self.close()


class AsyncStreamer:
    """An asynchronous version of the Streamer class. This class is used to stream
    frames of telemetry in real-time from a Synnax cluster. It should not be constructed
    directly, and should instead be created using the client's `open_streamer` method.

    To open an async streamer, use the `open_async_streamer` method on the client and
    pass in the list of channels you'd like to stream. Once a new async streamer has
    been opened, you can call the `read` method to read the next frame of telemetry.
    Once done, call the `close` method to close the streamer and free all necessary
    resources. We recommend using the async streamer as an async context manager to
    ensure that it is closed properly.

    Async streamers also support the async iterator protocol, allowing you to iterate
    over the frames of telemetry as they are received. This is useful when you want to
    process each frame as it is received.
    """

    _stream: AsyncStream[_Request, _Response]
    _client: AsyncStreamClient
    _adapter: ReadFrameAdapter

    def __init__(
        self,
        *,
        client: AsyncStreamClient,
        adapter: ReadFrameAdapter,
        downsample_factor: int,
        throttle_rate: float = 0,
        exclude_groups: list[int] | None = None,
        keep_alive: CrudeTimeSpan = DEFAULT_KEEP_ALIVE,
    ) -> None:
        validate_downsample_factor(downsample_factor)
        self._client = client
        self._adapter = adapter
        self._downsample_factor = downsample_factor
        self._throttle_rate = throttle_rate
        self._exclude_groups = exclude_groups or []
        self._liveness = _Liveness(TimeSpan.from_seconds(keep_alive))

    async def _open(self) -> None:
        client = self._client.with_codec(_WSStreamerCodec(self._adapter.codec))
        self._stream = await client.stream(_ENDPOINT, _Request, _Response)
        await self._stream.send(
            _Request(
                keys=self._adapter.keys,
                downsample_factor=self._downsample_factor,
                throttle_rate=self._throttle_rate,
                exclude_groups=self._exclude_groups,
                keep_alive=self._liveness.keep_alive,
            )
        )
        # A keep-alive can beat the open ack onto the wire, so the ack is the first
        # non-keep-alive response.
        while (await self._receive()).keep_alive:
            pass

    async def read(self) -> Frame:
        """Reads the next frame of telemetry from the streamer. If an error occurs while
        reading the frame, an exception will be raised.

        :raises Unreachable: If keep-alives were flowing and then no response arrives,
        across reads, for the keep-alive deadline: the connection is presumed dead.
        """
        while True:
            res = await self._receive_with_deadline()
            if not res.keep_alive:
                return self._adapter.adapt(Frame(res.frame))

    async def _receive_with_deadline(self) -> _Response:
        dead = self._liveness.dead_at()
        if dead is None:
            return await self._receive()
        try:
            async with asyncio.timeout(max(dead - time.monotonic(), 0)):
                return await self._receive()
        except TimeoutError as e:
            raise self._liveness.unreachable() from e

    async def _receive(self) -> _Response:
        res = await self._stream.receive()
        self._liveness.received(res)
        return res

    async def close_loop(self) -> None:
        """Closes the sending end of the streamer, requiring the caller to process all
        remaining frames and close acknowledgments by calling read. This method is
        useful for managing the lifecycle of a streamer within a separate event loop or
        thread.
        """
        await self._stream.close_send()

    async def close(self) -> None:
        """Close the streamer and free all network resources, waiting for the server to
        acknowledge the close request.
        """
        await self._stream.close_send()
        while True:
            try:
                await self._stream.receive()
            except EOF:
                return

    async def __aenter__(self) -> AsyncStreamer:
        """Returns the async streamer object when used as an async context manager."""
        return self

    def __aiter__(self) -> AsyncStreamer:
        """Returns an async iterator object that can be used to iterate over the frames
        of telemetry as they are received. This is useful when you want to process each
        frame as it is received.
        """
        return self

    async def __anext__(self) -> Frame:
        """Reads the next frame of telemetry from the streamer."""
        try:
            return await self.read()
        except EOF:
            raise StopAsyncIteration from None

    async def __aexit__(
        self, exc_type: object, exc_val: object, exc_tb: object
    ) -> None:
        """Closes the streamer when used as an async context manager"""
        await self.close()
