#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import (
    EOF,
    AsyncStream,
    AsyncStreamClient,
    ExceptionPayload,
    Payload,
    Stream,
    StreamClient,
)

from synnax.channel.payload import ChannelKeys, ChannelParams
from synnax.exceptions import UnexpectedError
from synnax.framer.adapter import ReadFrameAdapter
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import TimeSpan
from typing import overload


class _Request(Payload):
    keys: ChannelKeys
    downsample_factor: int


class _Response(Payload):
    frame: FramePayload
    error: ExceptionPayload | None


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

    For detailed documentation, see https://docs.synnaxlabs.com/reference/python-client/stream-data
    """

    _stream: Stream[_Request, _Response]
    _adapter: ReadFrameAdapter

    def __init__(
        self,
        client: StreamClient,
        adapter: ReadFrameAdapter,
        downsample_factor: int,
    ) -> None:
        self._stream = client.stream(_ENDPOINT, _Request, _Response)
        self._adapter = adapter
        self._downsample_factor = downsample_factor
        self.__open()

    def __open(self):
        self._stream.send(
            _Request(keys=self._adapter.keys, downsample_factor=self._downsample_factor)
        )
        _, exc = self._stream.receive()
        if exc is not None:
            raise exc

    @overload
    def read(self, timeout: float | int | TimeSpan) -> Frame | None:
        """Reads the next frame of telemetry from the streamer with a timeout. If no
        frame is received within the timeout, this method will return None. If an error
        occurs while reading the frame, an exception will be raised.

        :param timeout: The maximum amount of time to wait for a frame to be received
        before returning None. This can be a float or integer representing the number
        of seconds, or a synnax TimeSpan object.
        :return: The next frame of telemetry, or None if no frame is received within the
        timeout.
        """
        ...

    @overload
    def read(self) -> Frame:
        """Reads the next frame of telemetry from the streamer, blocking until a frame
        is received. If an error occurs while reading the frame, an exception will be
        raised.
        """
        ...

    def read(self, timeout: float | None = None) -> Frame | None:
        """Reads the next frame of telemetry from the streamer. If a timeout is provided,
        this method will wait for the specified amount of time for a frame to be received.
        If no frame is received within the timeout, this method will return None. If no
        timeout is provided, this method will block until a frame is received.

        If an error occurs while reading the frame, an exception will be raised.

        :param timeout: The maximum amount of time to wait for a frame to be received
        before returning None. If no timeout is provided, this method will block until a
        frame is received.
        :return: The next frame of telemetry, or None if no frame is received within the
        timeout.
        """
        try:
            res, exc = self._stream.receive(TimeSpan.to_seconds(timeout))
            if exc is not None:
                raise exc
            return self._adapter.adapt(Frame(res.frame))
        except TimeoutError:
            return None

    def update_channels(self, channels: ChannelParams):
        """Updates the list of channels to stream. This method will replace the current
        list of channels with the new list, not add to it.

        :param channels: The list of channels to stream.
        :raises NotFoundError: If any of the channels in the list are not found.
        """
        self._adapter.update(channels)
        self._stream.send(
            _Request(keys=self._adapter.keys, downsample_factor=self._downsample_factor)
        )

    def close(self, timeout: float | int | TimeSpan | None = None):
        """Closes the streamer and frees all network resources.

        :param timeout: The maximum amount of time to wait for the server to acknowledge
        the closure before raising a TimeoutError. This can be a float or integer
        representing the number of seconds, or a synnax TimeSpan object. If no timeout
        is provided, this method will block until the server acknowledges the closure.
        """
        exc = self._stream.close_send()
        if exc is not None:
            raise exc
        while True:
            r, exc = self._stream.receive(TimeSpan.to_seconds(timeout))
            if r is not None:
                continue
            if exc is None:
                raise UnexpectedError(
                    f"""Unexpected missing close acknowledgement from server.
                    Please report this issue to the Synnax team.
                    Response: {r}
                    """
                )
            elif not isinstance(exc, EOF):
                raise exc
            break

    def __iter__(self):
        """Returns an iterator object that can be used to iterate over the frames of
        telemetry as they are received. This is useful when you want to process each
        frame as it is received.
        """
        return self

    def __enter__(self):
        """Returns the streamer object when used as a context manager."""
        return self

    def __next__(self):
        """Reads the next frame of telemetry from the streamer."""
        return self.read()

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


class AsyncStreamer:
    """An asynchronous version of the Streamer class. This class is used to stream
    frames of telemetry in real-time from a Synnax cluster. It should not be constructed
    directly, and should instead be created using the client's `open_streamer` method.

    To open an async streamer, use the `open_async_streamer` method on the client and pass
    in the list of channels you'd like to stream. Once a new async streamer has been opened,
    you can call the `read` method to read the next frame of telemetry. Once done, call the
    `close` method to close the streamer and free all necessary resources. We recommend
    using the async streamer as an async context manager to ensure that it is closed properly.

    Async streamers also support the async iterator protocol, allowing you to iterate over
    the frames of telemetry as they are received. This is useful when you want to process
    each frame as it is received.
    """

    _stream: AsyncStream[_Request, _Response]
    _client: AsyncStreamClient
    _adapter: ReadFrameAdapter

    def __init__(
        self,
        client: AsyncStreamClient,
        adapter: ReadFrameAdapter,
        downsample_factor: int,
    ) -> None:
        self._client = client
        self._adapter = adapter
        self._downsample_factor = downsample_factor

    async def _open(self):
        self._stream = await self._client.stream(_ENDPOINT, _Request, _Response)
        await self._stream.send(
            _Request(
                keys=self._adapter.keys,
                downsample_factor=self._downsample_factor,
            )
        )
        _, exc = await self._stream.receive()
        if exc is not None:
            raise exc

    @property
    def received(self) -> bool:
        """Returns True if a frame has been received, False otherwise."""
        return self._stream.received()

    async def read(self) -> Frame:
        """Reads the next frame of telemetry from the streamer. If an error occurs while
        reading the frame, an exception will be raised.
        """
        res, exc = await self._stream.receive()
        if exc is not None:
            raise exc
        return self._adapter.adapt(Frame(res.frame))

    async def close_loop(self):
        """Closes the sending end of the streamer, requiring the caller to process all
        remaining frames and close acknowledgements by calling read. This method is
        useful for managing the lifecycle of a streamer within a separate event loop or
        thread.
        """
        await self._stream.close_send()

    async def close(self):
        """Close the streamer and free all network resources, waiting for the server to
        acknowledge the close request.
        """
        exc = await self._stream.close_send()
        if exc is not None:
            raise exc
        _, exc = await self._stream.receive()
        if exc is None:
            raise UnexpectedError(
                """Unexpected missing close acknowledgement from server.
                Please report this issue to the Synnax team."""
            )
        elif not isinstance(exc, EOF):
            raise exc

    async def __aenter__(self):
        """Returns the async streamer object when used as an async context manager."""
        return self

    def __aiter__(self):
        """Returns an async iterator object that can be used to iterate over the frames
        of telemetry as they are received. This is useful when you want to process each
        frame as it is received.
        """
        return self

    async def __anext__(self):
        """Reads the next frame of telemetry from the streamer."""
        try:
            return await self.read()
        except EOF:
            raise StopAsyncIteration

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Closes the streamer when used as an async context manager"""
        await self.close()
