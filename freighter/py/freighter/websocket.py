import asyncio
from typing import Generic, Type

from freighter.metadata import MetaData
from pydantic import BaseModel
from websockets.exceptions import ConnectionClosedOK
from websockets import connect, WebSocketClientProtocol

from . import AsyncStream
from .encoder import EncoderDecoder
from .exceptions import EOF, ExceptionPayload, StreamClosed, decode_exception
from .transport import RQ, RS, P, AsyncMiddlewareCollector
from .url import URL

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


class WebsocketStream(Generic[RQ, RS]):
    """An implementation of AsyncStream that is backed by a websocket."""

    encoder: EncoderDecoder
    wrapped: WebSocketClientProtocol
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
        self.wrapped = ws
        self.send_closed = False
        self.server_closed = None
        self.res_msg_t = _new_res_msg_t(res_t)

    async def receive(self) -> tuple[RS | None, Exception | None]:
        """Implements the AsyncStream protocol."""
        if self.server_closed is not None:
            return None, self.server_closed

        data = await self.wrapped.recv()
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
            await self.wrapped.send(encoded)
        except ConnectionClosedOK:
            return EOF()
        return None

    async def close_send(self):
        """Implements the AsyncStream protocol."""
        if self.send_closed or self.server_closed is not None:
            return

        msg = _Message(type=_CLOSE_MESSAGE, payload=None, error=None)
        try:
            await self.wrapped.send(self.encoder.encode(msg))
        finally:
            self.send_closed = True

    async def _close_server(self, server_err: Exception | None):
        if self.server_closed is not None:
            return

        if server_err is not None:
            self.server_closed = server_err

        await self.wrapped.close()


DEFAULT_MAX_SIZE = 2 ** 20


class WebsocketClient(AsyncMiddlewareCollector):
    """An implementation of AsyncStreamClient that is backed by a websocket

    :param encoder: The encoder to use for this client.
    :param base_url: A base url to use as a prefix for all requests.
    :param max_message_size: The maximum size of a message to receive. Defaults to
    DEFAULT_MAX_SIZE.
    """

    _endpoint: URL
    _encoder: EncoderDecoder
    _max_message_size: int
    _socket: WebsocketStream[RQ, RS] | None

    def __init__(
            self,
            encoder: EncoderDecoder,
            base_url: URL,
            max_message_size: int = DEFAULT_MAX_SIZE,
    ) -> None:
        super(WebsocketClient, self).__init__()
        self._encoder = encoder
        self._endpoint = base_url.replace(protocol="ws")
        self._max_message_size = max_message_size

    async def stream(
            self, target: str, req_type: Type[RQ], res_type: Type[RS]
    ) -> AsyncStream[RQ, RS]:
        """Implements the AsyncStreamClient protocol."""

        headers = {"Content-Type": self._encoder.content_type()}

        async def finalizer(md: MetaData) -> Exception | None:
            headers.update(md.params)
            try:
                ws = await connect(
                    self._endpoint.child(target).stringify(),
                    extra_headers=headers,
                    max_size=self._max_message_size,
                )
                self._socket = WebsocketStream[RQ, RS](self._encoder, ws, res_type)
            except Exception as e:
                return e
            return None

        exc = await self.exec(MetaData(target, "websocket"), finalizer)
        if exc is not None:
            print(exc.__dict__)
            raise exc

        try:
            return self._socket
        finally:
            self._socket = None

