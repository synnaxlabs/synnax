#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl
from typing import Any, Generic, Literal, Type

from pydantic import BaseModel
from websockets.client import WebSocketClientProtocol, connect
from websockets.exceptions import ConnectionClosedOK

from freighter.context import Context
from freighter.encoder import EncoderDecoder
from freighter.exceptions import EOF, ExceptionPayload, StreamClosed, decode_exception
from freighter.stream import AsyncStream, AsyncStreamClient
from freighter.transport import RQ, RS, AsyncMiddlewareCollector, P
from freighter.url import URL


class _Message(Generic[P], BaseModel):
    type: Literal["data", "close"]
    payload: P | None
    error: ExceptionPayload | None


def _new_res_msg_t(res_t: Type[RS]) -> Type[_Message[RS]]:
    class _ResMsg(_Message[RS]):
        payload: res_t | None

    return _ResMsg


class WebsocketStream(AsyncStream[RQ, RS]):
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

    async def receive(self) -> tuple[RS | None, Exception | None]:
        """Implements the AsyncStream protocol."""
        if self.__server_closed is not None:
            return None, self.__server_closed

        data = await self.__internal.recv()
        assert isinstance(data, bytes)
        msg = self.__encoder.decode(data, self.__res_msg_t)

        if msg.type == "close":
            assert msg.error is not None
            await self.__close_server(decode_exception(msg.error))
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

    async def __close_server(self, server_err: Exception | None):
        if self.__server_closed is not None:
            return

        if server_err is not None:
            self.__server_closed = server_err

        await self.__internal.close()


DEFAULT_MAX_SIZE = 2**20


class WebsocketClient(AsyncMiddlewareCollector):
    """An implementation of AsyncStreamClient that is backed by a websocket"""

    __endpoint: URL
    __encoder: EncoderDecoder
    __max_message_size: int
    __secure: bool = False
    __kwargs: dict[str, Any]

    def __init__(
        self,
        encoder: EncoderDecoder,
        base_url: URL,
        max_message_size: int = DEFAULT_MAX_SIZE,
        secure: bool = False,
        **kwargs,
    ) -> None:
        """
        :param encoder: The encoder to use for this client.
        :param base_url: A base url to use as a prefix for all requests.
        :param max_message_size: The maximum size of a message to receive. Defaults to
        DEFAULT_MAX_SIZE.
        :param secure: Whether to use TLS encryption on the connection or not.
        """
        super(WebsocketClient, self).__init__()
        self.__encoder = encoder
        self.__secure = secure
        self.__endpoint = base_url.replace(protocol="ws" if not secure else "wss")
        self.__max_message_size = max_message_size
        self.__kwargs = kwargs

    def __(self) -> AsyncStreamClient:
        return self

    async def stream(
        self,
        target: str,
        _req_t: Type[RQ],
        res_t: Type[RS],
    ) -> AsyncStream[RQ, RS]:
        """Implements the AsyncStreamClient protocol."""

        headers = {"Content-Type": self.__encoder.content_type()}
        socket: WebsocketStream[RQ, RS] | None = None

        async def finalizer(ctx: Context) -> tuple[Context, Exception | None]:
            nonlocal socket
            out_ctx = Context(target, "websocket", "client")
            headers.update(ctx.params)
            try:
                if self.__secure and "ssl" not in self.__kwargs:
                    self.__kwargs["ssl"] = ssl._create_unverified_context()
                ws = await connect(
                    self.__endpoint.child(target).stringify(),
                    extra_headers=headers,
                    max_size=self.__max_message_size,
                    **self.__kwargs,
                )

                socket = WebsocketStream[RQ, RS](self.__encoder, ws, res_t)
            except Exception as e:
                return out_ctx, e
            return out_ctx, None

        _, exc = await self.exec(Context(target, "websocket", "client"), finalizer)
        if exc is not None:
            raise exc

        assert socket is not None
        return socket
