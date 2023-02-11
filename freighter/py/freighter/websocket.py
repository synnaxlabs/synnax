#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Generic, Type, Any

from freighter.metadata import MetaData
from pydantic import BaseModel
from websockets.exceptions import ConnectionClosedOK
from websockets.client import connect, WebSocketClientProtocol

from freighter.stream import AsyncStream, AsyncStreamClient
from freighter.encoder import EncoderDecoder
from freighter.exceptions import EOF, ExceptionPayload, StreamClosed, decode_exception
from freighter.transport import RQ, RS, P, AsyncMiddlewareCollector
from freighter.url import URL

_DATA_MESSAGE = "data"
_CLOSE_MESSAGE = "close"


class _Message(Generic[P], BaseModel):
    type: str
    payload: P | None
    error: ExceptionPayload | None


def _new_res_msg_t(res_t: Type[RS]) -> Type[_Message[RS]]:
    class _ResMsg(_Message[RS]):
        payload: res_t | None

    return _ResMsg


class WebsocketStream(AsyncStream[RQ, RS]):
    """An implementation of AsyncStream that is backed by a websocket."""

    encoder: EncoderDecoder
    internal: WebSocketClientProtocol
    server_closed: Exception | None
    send_closed: bool
    res_msg_t: Type[_Message[RS]]

    def __init__(
        self,
        encoder: EncoderDecoder,
        ws: WebSocketClientProtocol,
        res_t: Type[RS],
    ):
        self.encoder = encoder
        self.internal = ws
        self.send_closed = False
        self.server_closed = None
        self.res_msg_t = _new_res_msg_t(res_t)

    async def receive(self) -> tuple[RS | None, Exception | None]:
        """Implements the AsyncStream protocol."""
        if self.server_closed is not None:
            return None, self.server_closed

        data = await self.internal.recv()
        assert isinstance(data, bytes)
        msg = self.encoder.decode(data, self.res_msg_t)

        if msg.type == _CLOSE_MESSAGE:
            assert msg.error is not None
            await self._close_server(decode_exception(msg.error))
            return None, self.server_closed

        return msg.payload, None

    async def send(self, payload: RQ) -> Exception | None:
        """Implements the AsyncStream protocol."""
        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        if self.server_closed is not None:
            return EOF()

        if self.send_closed:
            raise StreamClosed

        msg = _Message(type=_DATA_MESSAGE, payload=payload, error=None)
        encoded = self.encoder.encode(msg)

        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        try:
            await self.internal.send(encoded)
        except ConnectionClosedOK:
            return EOF()
        return None

    async def close_send(self):
        """Implements the AsyncStream protocol."""
        if self.send_closed or self.server_closed is not None:
            return

        msg = _Message(type=_CLOSE_MESSAGE, payload=None, error=None)
        try:
            await self.internal.send(self.encoder.encode(msg))
        finally:
            self.send_closed = True

    async def _close_server(self, server_err: Exception | None):
        if self.server_closed is not None:
            return

        if server_err is not None:
            self.server_closed = server_err

        await self.internal.close()


DEFAULT_MAX_SIZE = 2**20


class WebsocketClient(AsyncMiddlewareCollector):
    """An implementation of AsyncStreamClient that is backed by a websocket"""

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
        """
        :param encoder: The encoder to use for this client.
        :param base_url: A base url to use as a prefix for all requests.
        :param max_message_size: The maximum size of a message to receive. Defaults to
        DEFAULT_MAX_SIZE.
        :param secure: Whether to use TLS encryption on the connection or not.
        """
        super(WebsocketClient, self).__init__()
        self._encoder = encoder
        self._secure = secure
        self._endpoint = base_url.replace(protocol="ws" if not secure else "wss")
        self._max_message_size = max_message_size
        self._kwargs = kwargs

    def _(self) -> AsyncStreamClient:
        return self

    async def stream(
        self,
        target: str,
        req_t: Type[RQ],
        res_t: Type[RS],
    ) -> AsyncStream[RQ, RS]:
        """Implements the AsyncStreamClient protocol."""

        headers = {"Content-Type": self._encoder.content_type()}
        socket: WebsocketStream[RQ, RS] | None = None

        async def finalizer(md: MetaData) -> tuple[MetaData, Exception | None]:
            nonlocal socket
            out_meta_data = MetaData(target, "websocket")
            headers.update(md.params)
            try:
                ws = await connect(
                    self._endpoint.child(target).stringify(),
                    extra_headers=headers,
                    max_size=self._max_message_size,
                    **self._kwargs,
                )

                socket = WebsocketStream[RQ, RS](self._encoder, ws, res_t)
            except Exception as e:
                return out_meta_data, e
            return out_meta_data, None

        _, exc = await self.exec(MetaData(target, "websocket"), finalizer)
        if exc is not None:
            raise exc

        assert socket is not None
        return socket
