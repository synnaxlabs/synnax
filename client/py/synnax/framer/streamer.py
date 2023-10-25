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

from synnax.channel.payload import ChannelKeys
from synnax.exceptions import UnexpectedError
from synnax.framer.adapter import ReadFrameAdapter
from synnax.framer.frame import Frame, FramePayload
from synnax.telem import CrudeTimeStamp, TimeStamp


class _Request(Payload):
    start: TimeStamp
    keys: ChannelKeys


class _Response(Payload):
    frame: FramePayload
    error: ExceptionPayload | None


_ENDPOINT = "/frame/stream"


class Streamer:
    __stream: Stream[_Request, _Response]
    __adapter: ReadFrameAdapter
    from_: CrudeTimeStamp

    def __init__(
        self,
        client: StreamClient,
        adapter: ReadFrameAdapter,
        from_: CrudeTimeStamp | None = None,
    ) -> None:
        self.from_ = from_ or TimeStamp.now()
        self.__stream = client.stream(_ENDPOINT, _Request, _Response)
        self.__adapter = adapter
        self.__open()

    def __open(self):
        self.__stream.send(_Request(keys=self.__adapter.keys, start=self.from_))

    @property
    def received(self) -> bool:
        return self.__stream.received()

    def read(self) -> Frame:
        res, err = self.__stream.receive()
        if err is not None:
            raise err
        return self.__adapter.adapt(Frame(res.frame))

    def close(self):
        exc = self.__stream.close_send()
        if exc is not None:
            raise exc
        _, exc = self.__stream.receive()
        if exc is None:
            raise UnexpectedError(
                """Unexpected missing close acknowledgement from server.
                Please report this issue to the Synnax team."""
            )
        elif not isinstance(exc, EOF):
            raise exc

    def __iter__(self):
        return self

    def __enter__(self):
        return self

    def __next__(self):
        return self.read()

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


class AsyncStreamer:
    __stream: AsyncStream[_Request, _Response]
    __client: AsyncStreamClient
    __adapter: ReadFrameAdapter
    from_: CrudeTimeStamp

    def __init__(
        self,
        client: AsyncStreamClient,
        adapter: ReadFrameAdapter,
        from_: CrudeTimeStamp | None = None,
    ) -> None:
        self.from_ = from_ or TimeStamp.now()
        self.__client = client
        self.__adapter = adapter

    async def open(self):
        self.__stream = await self.__client.stream(_ENDPOINT, _Request, _Response)
        await self.__stream.send(_Request(keys=self.__adapter.keys, start=self.from_))

    @property
    def received(self) -> bool:
        return self.__stream.received()

    async def read(self) -> Frame:
        res, err = await self.__stream.receive()
        if err is not None:
            raise err
        return self.__adapter.adapt(Frame(res.frame))

    async def close_loop(self):
        await self.__stream.close_send()

    async def close(self):
        exc = await self.__stream.close_send()
        if exc is not None:
            raise exc
        _, exc = await self.__stream.receive()
        if exc is None:
            raise UnexpectedError(
                """Unexpected missing close acknowledgement from server.
                Please report this issue to the Synnax team."""
            )
        elif not isinstance(exc, EOF):
            raise exc

    async def __aenter__(self):
        return self

    def __aiter__(self):
        return self

    async def __anext__(self):
        try:
            return await self.read()
        except EOF:
            raise StopAsyncIteration

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()
