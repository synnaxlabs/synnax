#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl
from collections.abc import MutableMapping
from contextlib import ExitStack
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
from websockets.sync.client import reconnect as SyncConnector

from freighter.context import Context
from freighter.exceptions import EOF, StreamClosed
from freighter.stream import AsyncStream, AsyncStreamClient, Stream, StreamClient
from freighter.transport import RQ, RS, AsyncMiddlewareCollector, MiddlewareCollector, P
from freighter.url import URL
from x.codec import Codec
from x.exceptions import ExceptionPayload, Registry

_CONTEXT_CANCELLED_CLOSE_CODE = 1001


def _handle_close_err(e: ConnectionClosedOK) -> Exception:
    if (
        e.rcvd is not None
        and e.rcvd.code == _CONTEXT_CANCELLED_CLOSE_CODE
        and e.sent is not None
        and e.sent.code == _CONTEXT_CANCELLED_CLOSE_CODE
    ):
        return StreamClosed()
    return EOF()


def _decode_close_exception(
    registry: Registry, exc_pld: ExceptionPayload | None
) -> Exception:
    # A close without an error is a clean close.
    exc = registry.decode(exc_pld)
    return EOF() if exc is None else exc


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
            extra: Literal["allow", "ignore", "forbid"] | None = None,
            from_attributes: bool | None = None,
            context: Any | None = None,
            by_alias: bool | None = None,
            by_name: bool | None = None,
        ) -> Self:
            # Ensure the payload is validated as the correct type
            obj["payload"] = res_t.model_validate(
                obj["payload"],
                strict=strict,
                extra=extra,
                from_attributes=from_attributes,
                context=context,
                by_alias=by_alias,
                by_name=by_name,
            )
            return super().model_validate(
                obj,
                strict=strict,
                extra=extra,
                from_attributes=from_attributes,
                context=context,
                by_alias=by_alias,
                by_name=by_name,
            )

    return _ResMsg


class AsyncWebsocketStream(AsyncStream[RQ, RS]):
    """An implementation of AsyncStream that is backed by a websocket."""

    _encoder: Codec
    _registry: Registry
    _internal: AsyncClientConnection
    _server_closed: Exception | None
    _send_closed: bool
    _res_msg_t: type[Message[RS]]

    def __init__(
        self,
        *,
        encoder: Codec,
        registry: Registry,
        ws: AsyncClientConnection,
        res_t: type[RS],
    ):
        self._encoder = encoder
        self._registry = registry
        self._internal = ws
        self._send_closed = False
        self._server_closed = None
        self._res_msg_t = _new_res_msg_t(res_t)

    async def receive(self) -> RS:
        """Implements the AsyncStream protocol."""
        server_closed = self._server_closed
        if server_closed is not None:
            raise server_closed

        try:
            data = await self._internal.recv()
        except ConnectionClosedOK as e:
            raise _handle_close_err(e) from e
        assert isinstance(data, bytes)
        msg = self._encoder.decode(data, self._res_msg_t)

        if msg.type == "close":
            raise await self._close_server(msg.error)

        assert msg.payload is not None
        return msg.payload

    async def send(self, payload: RQ) -> None:
        """Implements the AsyncStream protocol."""
        # If the server closed with an error, we raise freighter.EOF to the caller, and
        # expect them to discover the close error by calling receive().
        if self._server_closed is not None:
            raise EOF()

        if self._send_closed:
            raise StreamClosed

        msg = Message[RQ](type="data", payload=payload, error=None)
        encoded = self._encoder.encode(msg)

        try:
            await self._internal.send(encoded)
        except ConnectionClosedOK as e:
            raise _handle_close_err(e) from e

    async def receive_open_ack(self) -> None:
        """Waits for the server to ack the open. Closes the connection on failure."""
        try:
            msg = await self._internal.recv()
            assert isinstance(msg, bytes)
            decoded_msg = self._encoder.decode(msg, Message)
            if decoded_msg.type == "open":
                return
            exc = self._registry.decode(decoded_msg.error)
            if exc is not None:
                raise exc
            raise RuntimeError(
                f"expected 'open' ack from server, got message type "
                f"{decoded_msg.type!r}"
            )
        except BaseException:
            await self._internal.close()
            raise

    async def close_send(self) -> None:
        """Implements the AsyncStream protocol."""
        if self._send_closed or self._server_closed is not None:
            return

        msg = Message[RQ](type="close", payload=None, error=None)
        try:
            await self._internal.send(self._encoder.encode(msg))
        except ConnectionClosedOK as e:
            raise _handle_close_err(e) from e
        finally:
            self._send_closed = True

    async def _close_server(self, exc_pld: ExceptionPayload | None) -> Exception:
        if self._server_closed is not None:
            return self._server_closed
        try:
            self._server_closed = _decode_close_exception(self._registry, exc_pld)
            return self._server_closed
        finally:
            await self._internal.close()


_DEFAULT_MAX_SIZE = 2**20


class SyncWebsocketStream(Stream[RQ, RS]):
    _encoder: Codec
    _registry: Registry
    _connection: ExitStack
    _internal: SyncClientProtocol
    _server_closed: Exception | None
    _send_closed: bool
    _res_msg_t: type[Message[RS]]

    def __init__(
        self,
        *,
        encoder: Codec,
        registry: Registry,
        connector: SyncConnector,
        res_t: type[RS],
    ):
        self._encoder = encoder
        self._registry = registry
        self._connection = ExitStack()
        self._internal = self._connection.enter_context(connector)
        self._send_closed = False
        self._server_closed = None
        self._res_msg_t = _new_res_msg_t(res_t)

    def receive(self, *, timeout: float | None = None) -> RS:
        server_closed = self._server_closed
        if server_closed is not None:
            raise server_closed

        try:
            data = self._internal.recv(timeout)
        except ConnectionClosedOK as e:
            raise _handle_close_err(e) from e
        assert isinstance(data, bytes)
        msg = self._encoder.decode(data, self._res_msg_t)

        if msg.type == "close":
            raise self._close_server(msg.error)

        assert msg.payload is not None
        return msg.payload

    def received(self) -> bool:
        return self._internal.recv_bufsize > 0

    def receive_open_ack(self) -> None:
        """Waits for the server to ack the open. Closes the connection on failure."""
        try:
            msg = self._internal.recv()
            assert isinstance(msg, bytes)
            decoded_msg = self._encoder.decode(msg, self._res_msg_t)
            if decoded_msg.type == "open":
                return
            exc = self._registry.decode(decoded_msg.error)
            if exc is not None:
                raise exc
            raise RuntimeError(
                f"expected 'open' ack from server, got message type "
                f"{decoded_msg.type!r}"
            )
        except BaseException:
            self._connection.close()
            raise

    def send(self, payload: RQ) -> None:
        if self._server_closed is not None:
            raise EOF()

        if self._send_closed:
            raise StreamClosed

        msg = Message[RQ](type="data", payload=payload, error=None)
        encoded = self._encoder.encode(msg)

        try:
            self._internal.send(encoded)
        except ConnectionClosedOK as e:
            raise _handle_close_err(e) from e

    def close_send(self) -> None:
        if self._send_closed or self._server_closed is not None:
            return

        msg = Message[RQ](type="close", payload=None, error=None)
        try:
            self._internal.send(self._encoder.encode(msg))
        except ConnectionClosedOK as e:
            raise _handle_close_err(e) from e
        finally:
            self._send_closed = True

    def _close_server(self, exc_pld: ExceptionPayload | None) -> Exception:
        if self._server_closed is not None:
            return self._server_closed
        try:
            self._server_closed = _decode_close_exception(self._registry, exc_pld)
            return self._server_closed
        finally:
            self._connection.close()


class _Base:
    _endpoint: URL
    _encoder: Codec
    _registry: Registry
    _max_message_size: int
    _ssl_context: ssl.SSLContext | None
    _kwargs: dict[str, Any]

    def __init__(
        self,
        *,
        encoder: Codec,
        registry: Registry,
        base_url: URL,
        max_message_size: int = _DEFAULT_MAX_SIZE,
        ssl_context: ssl.SSLContext | None = None,
        **kwargs: Any,
    ) -> None:
        self._encoder = encoder
        self._registry = registry
        self._ssl_context = ssl_context
        self._endpoint = base_url.replace(
            protocol="ws" if ssl_context is None else "wss"
        )
        self._max_message_size = max_message_size
        self._kwargs = kwargs

    def additional_headers(self, others: MutableMapping[str, str]) -> dict[str, str]:
        return {"Content-Type": self._encoder.content_type(), **others}


class AsyncWebsocketClient(_Base, AsyncMiddlewareCollector, AsyncStreamClient):
    """An implementation of AsyncStreamClient that is backed by a websocket"""

    def __init__(
        self,
        *,
        encoder: Codec,
        registry: Registry,
        base_url: URL,
        max_message_size: int = _DEFAULT_MAX_SIZE,
        ssl_context: ssl.SSLContext | None = None,
        **kwargs: Any,
    ) -> None:
        """
        :param encoder: The encoder to use for this client.
        :param registry: Decodes the errors that the server sends.
        :param base_url: A base url to use as a prefix for all requests.
        :param max_message_size: The maximum size of a message to receive.
        :param ssl_context: The TLS context that verifies the server. The client uses
            TLS when set and a plain connection when None.
        :param kwargs: Options for the websockets connect function.
        """
        AsyncMiddlewareCollector.__init__(self)
        _Base.__init__(
            self,
            encoder=encoder,
            registry=registry,
            base_url=base_url,
            max_message_size=max_message_size,
            ssl_context=ssl_context,
            **kwargs,
        )

    async def stream(
        self,
        target: str,
        _req_t: type[RQ],
        res_t: type[RS],
    ) -> AsyncStream[RQ, RS]:
        """Implements the AsyncStreamClient protocol."""
        socket_container: list[AsyncWebsocketStream[RQ, RS] | None] = [None]

        async def finalizer(ctx: Context) -> Context:
            out_ctx = Context(protocol="websocket", target=target, role="client")
            ws = await connect(
                self._endpoint.child(target).stringify(),
                additional_headers=self.additional_headers(ctx.params),
                max_size=self._max_message_size,
                ssl=self._ssl_context,
                **self._kwargs,
            )
            socket = AsyncWebsocketStream[RQ, RS](
                encoder=self._encoder, registry=self._registry, ws=ws, res_t=res_t
            )
            await socket.receive_open_ack()
            socket_container[0] = socket
            return out_ctx

        await self.exec(
            Context(protocol="websocket", target=target, role="client"),
            finalizer=finalizer,
        )

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
            registry=self._registry,
            base_url=self._endpoint,
            max_message_size=self._max_message_size,
            ssl_context=self._ssl_context,
            **self._kwargs,
        )
        # Copy middleware
        for middleware in self._middleware:
            client.use(middleware)
        return client


class WebsocketClient(_Base, MiddlewareCollector, StreamClient):
    def __init__(
        self,
        *,
        encoder: Codec,
        registry: Registry,
        base_url: URL,
        max_message_size: int = _DEFAULT_MAX_SIZE,
        ssl_context: ssl.SSLContext | None = None,
        **kwargs: Any,
    ) -> None:
        """
        :param encoder: The encoder to use for this client.
        :param registry: Decodes the errors that the server sends.
        :param base_url: A base url to use as a prefix for all requests.
        :param max_message_size: The maximum size of a message to receive.
        :param ssl_context: The TLS context that verifies the server. The client uses
            TLS when set and a plain connection when None.
        :param kwargs: Options for the websockets connect function.
        """
        MiddlewareCollector.__init__(self)
        _Base.__init__(
            self,
            encoder=encoder,
            registry=registry,
            base_url=base_url,
            max_message_size=max_message_size,
            ssl_context=ssl_context,
            **kwargs,
        )

    def stream(
        self,
        target: str,
        req_t: type[RQ],
        res_t: type[RS],
    ) -> SyncWebsocketStream[RQ, RS]:
        socket_container: list[SyncWebsocketStream[RQ, RS] | None] = [None]

        def finalizer(ctx: Context) -> Context:
            out_ctx = Context(protocol="websocket", target=target, role="client")
            connector = sync_connect(
                self._endpoint.child(target).stringify(),
                additional_headers=self.additional_headers(ctx.params),
                max_size=self._max_message_size,
                legacy=False,
                ssl=self._ssl_context,
                **self._kwargs,
            )
            socket = SyncWebsocketStream[RQ, RS](
                encoder=self._encoder,
                registry=self._registry,
                connector=connector,
                res_t=res_t,
            )
            socket.receive_open_ack()
            socket_container[0] = socket
            return out_ctx

        self.exec(
            Context(protocol="websocket", target=target, role="client"),
            finalizer=finalizer,
        )
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
            registry=self._registry,
            base_url=self._endpoint,
            max_message_size=self._max_message_size,
            ssl_context=self._ssl_context,
            **self._kwargs,
        )
        # Copy middleware
        for middleware in self._middleware:
            client.use(middleware)
        return client
