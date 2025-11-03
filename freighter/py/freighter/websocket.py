#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl
from collections.abc import MutableMapping
from typing import Any, Generic, Literal, Self

from pydantic import BaseModel
from websockets.asyncio.client import ClientConnection as AsyncClientConnection
from websockets.asyncio.client import connect
from websockets.exceptions import ConnectionClosedError as ConnectionClosedError
from websockets.exceptions import (
    ConnectionClosedOK,
)
from websockets.sync.client import ClientConnection as SyncClientProtocol
from websockets.sync.client import connect as sync_connect

from freighter.codec import Codec
from freighter.context import Context
from freighter.exceptions import EOF, ExceptionPayload, StreamClosed, decode_exception
from freighter.stream import AsyncStream, AsyncStreamClient, Stream, StreamClient
from freighter.transport import RQ, RS, AsyncMiddlewareCollector, MiddlewareCollector, P
from freighter.url import URL

CONTEXT_CANCELLED_CLOSE_CODE = 1001


def handle_close_err(e: ConnectionClosedOK) -> Exception:
    if (
        e.rcvd is not None
        and e.rcvd.code == CONTEXT_CANCELLED_CLOSE_CODE
        and e.sent is not None
        and e.sent.code == CONTEXT_CANCELLED_CLOSE_CODE
    ):
        return StreamClosed()
    return EOF()


class Message(BaseModel, Generic[P]):
    type: Literal["data", "close", "open"]
    payload: P | None = None
    error: ExceptionPayload | None = None


def _new_res_msg_t(res_t: type[RS]) -> type[Message[RS]]:
    class _ResMsg(Message[RS]):
        payload: RS | None = None

        @classmethod
        def model_validate(
            cls,
            obj: Any,
            *,
            strict: bool | None = None,
            from_attributes: bool | None = None,
            context: Any | None = None,
            by_alias: bool | None = None,
            by_name: bool | None = None,
        ) -> Self:
            # Ensure the payload is validated as the correct type
            obj["payload"] = res_t.model_validate(
                obj["payload"],
                strict=strict,
                from_attributes=from_attributes,
                context=context,
                by_alias=by_alias,
                by_name=by_name,
            )
            return super().model_validate(
                obj,
                strict=strict,
                from_attributes=from_attributes,
                context=context,
                by_alias=by_alias,
                by_name=by_name,
            )

    return _ResMsg


class AsyncWebsocketStream(AsyncStream[RQ, RS]):
    """An implementation of AsyncStream that is backed by a websocket."""

    __encoder: Codec
    __internal: AsyncClientConnection
    __server_closed: Exception | None
    __send_closed: bool
    __res_msg_t: type[Message[RS]]

    def __init__(self, encoder: Codec, ws: AsyncClientConnection, res_t: type[RS]):
        self.__encoder = encoder
        self.__internal = ws
        self.__send_closed = False
        self.__server_closed = None
        self.__res_msg_t = _new_res_msg_t(res_t)

    async def receive(self) -> tuple[RS, None] | tuple[None, Exception]:
        """Implements the AsyncStream protocol."""
        server_closed = self.__server_closed
        if server_closed is not None:
            return None, server_closed

        data = await self.__internal.recv()
        assert isinstance(data, bytes)
        msg = self.__encoder.decode(data, self.__res_msg_t)

        if msg.type == "close":
            await self.__close_server(msg.error)
            assert self.__server_closed is not None
            return None, self.__server_closed

        assert msg.payload is not None
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

        msg = Message[RQ](type="data", payload=payload, error=None)
        encoded = self.__encoder.encode(msg)

        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        try:
            await self.__internal.send(encoded)
        except ConnectionClosedOK:
            return EOF()
        return None

    async def receive_open_ack(self) -> Exception | None:
        msg = await self.__internal.recv()
        assert isinstance(msg, bytes)
        decoded_msg = self.__encoder.decode(msg, Message)
        if decoded_msg.type == "open":
            return None
        return decode_exception(decoded_msg.error)

    async def close_send(self) -> Exception | None:
        """Implements the AsyncStream protocol."""
        if self.__send_closed or self.__server_closed is not None:
            return None

        msg = Message[RQ](type="close", payload=None, error=None)
        try:
            await self.__internal.send(self.__encoder.encode(msg))
        finally:
            self.__send_closed = True
        return None

    async def __close_server(self, exc_pld: ExceptionPayload | None) -> None:
        if self.__server_closed is not None:
            return
        try:
            assert exc_pld is not None
            self.__server_closed = decode_exception(exc_pld)
        finally:
            await self.__internal.close()


DEFAULT_MAX_SIZE = 2**20


class SyncWebsocketStream(Stream[RQ, RS]):
    __encoder: Codec
    __internal: SyncClientProtocol
    __server_closed: Exception | None
    __send_closed: bool
    __res_msg_t: type[Message[RS]]

    def __init__(
        self,
        encoder: Codec,
        ws: SyncClientProtocol,
        res_t: type[RS],
    ):
        self.__encoder = encoder
        self.__internal = ws
        self.__send_closed = False
        self.__server_closed = None
        self.__res_msg_t = _new_res_msg_t(res_t)

    def receive(
        self, timeout: float | None = None
    ) -> tuple[RS, None] | tuple[None, Exception]:
        server_closed = self.__server_closed
        if server_closed is not None:
            return None, server_closed

        try:
            data = self.__internal.recv(timeout)
        except ConnectionClosedOK as e:
            return None, handle_close_err(e)
        assert isinstance(data, bytes)
        msg = self.__encoder.decode(data, self.__res_msg_t)

        if msg.type == "close":
            self.__close_server(msg.error)
            assert self.__server_closed is not None
            return None, self.__server_closed

        assert msg.payload is not None
        return msg.payload, None

    def received(self) -> bool:
        return self.__internal.recv_bufsize > 0

    def receive_open_ack(self) -> Exception | None:
        msg = self.__internal.recv()
        assert isinstance(msg, bytes)
        decoded_msg = self.__encoder.decode(msg, self.__res_msg_t)
        if decoded_msg.type == "open":
            return None
        return decode_exception(decoded_msg.error)

    def send(self, payload: RQ) -> Exception | None:
        if self.__server_closed is not None:
            return EOF()

        if self.__send_closed:
            raise StreamClosed

        msg = Message[RQ](type="data", payload=payload, error=None)
        encoded = self.__encoder.encode(msg)

        try:
            self.__internal.send(encoded)
        except ConnectionClosedOK as e:
            return handle_close_err(e)
        return None

    def close_send(self) -> Exception | None:
        if self.__send_closed or self.__server_closed is not None:
            return None

        msg = Message[RQ](type="close", payload=None, error=None)
        try:
            self.__internal.send(self.__encoder.encode(msg))
        except ConnectionClosedOK as e:
            return handle_close_err(e)
        finally:
            self.__send_closed = True
        return None

    def __close_server(self, exc_pld: ExceptionPayload | None) -> None:
        if self.__server_closed is not None:
            return
        try:
            assert exc_pld is not None
            self.__server_closed = decode_exception(exc_pld)
        finally:
            self.__internal.close()


class _Base:
    _endpoint: URL
    _encoder: Codec
    _max_message_size: int
    _secure: bool = False
    _kwargs: dict[str, Any]

    def __init__(
        self,
        encoder: Codec,
        base_url: URL,
        max_message_size: int = DEFAULT_MAX_SIZE,
        secure: bool = False,
        **kwargs: Any,
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

    def __init__(self, **kwargs: Any) -> None:
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
        _req_t: type[RQ],
        res_t: type[RS],
    ) -> AsyncStream[RQ, RS]:
        """Implements the AsyncStreamClient protocol."""
        socket_container: list[AsyncWebsocketStream[RQ, RS] | None] = [None]

        async def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            out_ctx = Context(target, "websocket", "client")
            try:
                ws = await connect(
                    self._endpoint.child(target).stringify(),
                    additional_headers=self.additional_headers(ctx.params),
                    max_size=self._max_message_size,
                    **self._kwargs,
                )
                socket = AsyncWebsocketStream[RQ, RS](self._encoder, ws, res_t)
                e = await socket.receive_open_ack()
                socket_container[0] = socket
                return out_ctx, e
            except Exception as e:
                return out_ctx, e

        _, exc = await self.exec(Context(target, "websocket", "client"), finalizer)
        if exc is not None:
            raise exc

        socket = socket_container[0]
        assert socket is not None
        return socket

    def with_codec(self, codec: Codec) -> "AsyncWebsocketClient":
        """
        Create a new client with a different codec.

        Args:
            codec: The codec to use for the new client

        Returns:
            A new AsyncWebsocketClient with the specified codec
        """
        client = AsyncWebsocketClient(
            encoder=codec,
            base_url=self._endpoint,
            secure=self._secure,
            **self._kwargs,
        )
        # Copy middleware
        for middleware in self._middleware:
            client.use(middleware)
        return client


class WebsocketClient(_Base, MiddlewareCollector, StreamClient):
    def __init__(self, **kwargs: Any) -> None:
        MiddlewareCollector.__init__(self)
        _Base.__init__(self, **kwargs)

    def __(self) -> StreamClient:
        return self

    def stream(
        self,
        target: str,
        req_t: type[RQ],
        res_t: type[RS],
    ) -> SyncWebsocketStream[RQ, RS]:
        socket_container: list[SyncWebsocketStream[RQ, RS] | None] = [None]

        def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            out_ctx = Context(target, "websocket", "client")
            try:
                ws = sync_connect(
                    self._endpoint.child(target).stringify(),
                    additional_headers=self.additional_headers(ctx.params),
                    max_size=self._max_message_size,
                    **self._kwargs,
                )
                socket = SyncWebsocketStream[RQ, RS](self._encoder, ws, res_t)
                e = socket.receive_open_ack()
                socket_container[0] = socket
                return out_ctx, e
            except Exception as e:
                return out_ctx, e

        _, exc = self.exec(Context(target, "websocket", "client"), finalizer)
        if exc is not None:
            raise exc
        socket = socket_container[0]
        assert socket is not None
        return socket

    def with_codec(self, codec: Codec) -> "WebsocketClient":
        """
        Create a new client with a different codec.

        Args:
            codec: The codec to use for the new client

        Returns:
            A new WebsocketClient with the specified codec
        """
        client = WebsocketClient(
            encoder=codec,
            base_url=self._endpoint,
            max_message_size=self._max_message_size,
            secure=self._secure,
            **self._kwargs,
        )
        # Copy middleware
        for middleware in self._middleware:
            client.use(middleware)
        return client
