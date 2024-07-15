#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl
from typing import Any, Generic, Literal, Type, MutableMapping
from warnings import warn

from pydantic import BaseModel
from websockets.client import WebSocketClientProtocol, connect
from websockets.exceptions import ConnectionClosedOK
from websockets.sync.client import (
    ClientConnection as SyncClientProtocol,
    connect as sync_connect,
)

from freighter.context import Context
from freighter.encoder import EncoderDecoder
from freighter.exceptions import EOF, ExceptionPayload, StreamClosed, decode_exception
from freighter.stream import AsyncStream, AsyncStreamClient, StreamClient, Stream
from freighter.transport import RQ, RS, AsyncMiddlewareCollector, P, MiddlewareCollector
from freighter.url import URL


class _Message(Generic[P], BaseModel):
    type: Literal["data", "close"]
    payload: P | None
    error: ExceptionPayload | None


def _new_res_msg_t(res_t: Type[RS]) -> Type[_Message[RS]]:
    class _ResMsg(_Message[RS]):
        payload: res_t | None

    return _ResMsg


class AsyncWebsocketStream(AsyncStream[RQ, RS]):
    """An implementation of AsyncStream that is backed by a websocket."""

    __encoder: EncoderDecoder
    __internal: WebSocketClientProtocol
    __server_closed: Exception | None
    __send_closed: bool
    __res_msg_t: Type[_Message[RS]]

    def __init__(
        self,
        encoder: EncoderDecoder,
        ws: WebSocketClientProtocol,
        res_t: Type[RS],
    ):
        self.__encoder = encoder
        self.__internal = ws
        self.__send_closed = False
        self.__server_closed = None
        self.__res_msg_t = _new_res_msg_t(res_t)

    async def receive(
        self,
        timeout: float | None = None,
    ) -> tuple[RS | None, Exception | None]:
        """Implements the AsyncStream protocol."""
        if timeout is not None:
            warn("Timeout is not supported for async websockets", stacklevel=2)
        if self.__server_closed is not None:
            return None, self.__server_closed

        data = await self.__internal.recv()
        assert isinstance(data, bytes)
        msg = self.__encoder.decode(data, self.__res_msg_t)

        if msg.type == "close":
            await self.__close_server(msg.error)
            return None, self.__server_closed

        return msg.payload, None

    async def send(self, payload: RQ) -> Exception | None:
        """Implements the AsyncStream protocol."""
        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        if self.__server_closed is not None:
            return EOF()

        if self.__send_closed:
            raise StreamClosed

        msg = _Message(type="data", payload=payload, error=None)
        encoded = self.__encoder.encode(msg)

        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        try:
            await self.__internal.send(encoded)
        except ConnectionClosedOK:
            return EOF()
        return None

    async def close_send(self) -> Exception | None:
        """Implements the AsyncStream protocol."""
        if self.__send_closed or self.__server_closed is not None:
            return None

        msg = _Message(type="close", payload=None, error=None)
        try:
            await self.__internal.send(self.__encoder.encode(msg))
        finally:
            self.__send_closed = True
        return None

    async def __close_server(self, exc_pld: ExceptionPayload | None):
        if self.__server_closed is not None:
            return
        try:
            assert exc_pld is not None
            self.__server_closed = decode_exception(exc_pld)
        finally:
            await self.__internal.close()


DEFAULT_MAX_SIZE = 2 ** 20


class SyncWebsocketStream(Stream[RQ, RS]):
    __encoder: EncoderDecoder
    __internal: SyncClientProtocol
    __server_closed: Exception | None
    __send_closed: bool
    __res_msg_t: Type[_Message[RS]]

    def __init__(
        self,
        encoder: EncoderDecoder,
        ws: SyncClientProtocol,
        res_t: Type[RS],
    ):
        self.__encoder = encoder
        self.__internal = ws
        self.__send_closed = False
        self.__server_closed = None
        self.__res_msg_t = _new_res_msg_t(res_t)

    def receive(
        self,
        timeout: float | None = None
    ) -> tuple[RS | None, Exception | None]:
        if self.__server_closed is not None:
            return None, self.__server_closed

        data = self.__internal.recv(timeout)
        assert isinstance(data, bytes)
        msg = self.__encoder.decode(data, self.__res_msg_t)

        if msg.type == "close":
            self.__close_server(msg.error)
            return None, self.__server_closed

        return msg.payload, None

    def send(self, payload: RQ) -> Exception | None:
        if self.__server_closed is not None:
            return EOF()

        if self.__send_closed:
            raise StreamClosed

        msg = _Message(type="data", payload=payload, error=None)
        encoded = self.__encoder.encode(msg)

        try:
            self.__internal.send(encoded)
        except ConnectionClosedOK:
            return EOF()
        return None

    def close_send(self) -> Exception | None:
        if self.__send_closed or self.__server_closed is not None:
            return None

        msg = _Message(type="close", payload=None, error=None)
        try:
            self.__internal.send(self.__encoder.encode(msg))
        finally:
            self.__send_closed = True
        return None

    def __close_server(self, exc_pld: ExceptionPayload | None):
        if self.__server_closed is not None:
            return
        try:
            assert exc_pld is not None
            self.__server_closed = decode_exception(exc_pld)
        finally:
            self.__internal.close()


class _Base:
    _endpoint: URL
    _encoder: EncoderDecoder
    _max_message_size: int
    _secure: bool = False
    _kwargs: dict[str, Any]

    def __init__(
        self,
        encoder: EncoderDecoder,
        base_url: URL,
        max_message_size: int = DEFAULT_MAX_SIZE,
        secure: bool = False,
        **kwargs,
    ) -> None:
        self._encoder = encoder
        self._secure = secure
        self._endpoint = base_url.replace(protocol="ws" if not secure else "wss")
        self._max_message_size = max_message_size
        self._kwargs = kwargs
        if self._secure and "ssl" not in self._kwargs:
            self._kwargs["ssl"] = ssl._create_unverified_context()

    def additional_headers(self, others: MutableMapping[str, str]) -> dict[str, str]:
        return {"Content-Type": self._encoder.content_type(), **others}


class AsyncWebsocketClient(_Base, AsyncMiddlewareCollector, AsyncStreamClient):
    """An implementation of AsyncStreamClient that is backed by a websocket"""

    def __init__(self, **kwargs) -> None:
        """
        :param encoder: The encoder to use for this client.
        :param base_url: A base url to use as a prefix for all requests.
        :param max_message_size: The maximum size of a message to receive. Defaults to
        DEFAULT_MAX_SIZE.
        :param secure: Whether to use TLS encryption on the connection or not.
        """
        AsyncMiddlewareCollector.__init__(self)
        _Base.__init__(self, **kwargs)

    def __(self) -> AsyncStreamClient:
        return self

    async def stream(
        self,
        target: str,
        _req_t: Type[RQ],
        res_t: Type[RS],
    ) -> AsyncStream[RQ, RS]:
        """Implements the AsyncStreamClient protocol."""

        socket: AsyncWebsocketStream[RQ, RS] | None = None

        async def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            nonlocal socket
            out_ctx = Context(target, "websocket", "client")
            try:
                ws = await connect(
                    self._endpoint.child(target).stringify(),
                    extra_headers=self.additional_headers(ctx.params),
                    max_size=self._max_message_size,
                    **self._kwargs,
                )
                socket = AsyncWebsocketStream[RQ, RS](self._encoder, ws, res_t)
            except Exception as e:
                return out_ctx, e
            return out_ctx, None

        _, exc = await self.exec(Context(target, "websocket", "client"), finalizer)
        if exc is not None:
            raise exc

        assert socket is not None
        return socket


class WebsocketClient(_Base, MiddlewareCollector, StreamClient):
    def __init__(self, **kwargs) -> None:
        MiddlewareCollector.__init__(self)
        _Base.__init__(self, **kwargs)

    def __(self) -> StreamClient:
        return self

    def stream(
        self,
        target: str,
        _req_t: Type[RQ],
        res_t: Type[RS],
    ) -> SyncWebsocketStream[RQ, RS]:
        socket: SyncWebsocketStream[RQ, RS] | None = None

        def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            nonlocal socket
            out_ctx = Context(target, "websocket", "client")
            try:
                ws = sync_connect(
                    self._endpoint.child(target).stringify(),
                    additional_headers=self.additional_headers(ctx.params),
                    max_size=self._max_message_size,
                    **self._kwargs,
                )
                socket = SyncWebsocketStream[RQ, RS](self._encoder, ws, res_t)
            except Exception as e:
                return out_ctx, e
            return out_ctx, None

        _, exc = self.exec(Context(target, "websocket", "client"), finalizer)
        if exc is not None:
            raise exc

        assert socket is not None
        return socket
