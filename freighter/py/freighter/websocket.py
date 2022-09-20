import asyncio
from dataclasses import dataclass
from typing import Generic, Type

from websockets.exceptions import ConnectionClosedOK
from websockets.legacy.client import WebSocketClientProtocol, connect

from . import AsyncStream
from .encoder import EncoderDecoder
from .errors import EOF, ErrorPayload, StreamClosed
from .errors import decode as decode_error
from .transport import RQ, RS, P
from .url import URL

_DATA_MESSAGE = "data"
_CLOSE_MESSAGE = "close"


@dataclass
class _Message(Generic[P]):
    type: str
    payload: P
    error: ErrorPayload | None


def empty_message(payload: P) -> _Message[P]:
    return _Message(type=_DATA_MESSAGE, payload=payload, error=ErrorPayload(None, None))


class WebsocketStream(Generic[RQ, RS]):
    """An implementation of AsyncStream that is backed by a websocket."""

    encoder: EncoderDecoder
    wrapped: WebSocketClientProtocol
    server_closed: Exception | None
    send_closed: bool
    response_factory: Type[RS]

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
        self.lock = asyncio.Lock()
        self.response_factory = res_t

    async def receive(self) -> tuple[RS | None, Exception | None]:
        if self.server_closed is not None:
            return None, self.server_closed

        data = await self.wrapped.recv()
        msg: _Message[RS] = empty_message(self.response_factory.new())
        assert isinstance(data, bytes)
        self.encoder.decode(data, msg)

        if msg.type == _CLOSE_MESSAGE:
            assert msg.error is not None
            await self._close_server(decode_error(msg.error))
            return None, self.server_closed

        return msg.payload, None

    async def send(self, payload: RQ) -> Exception | None:
        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        if self.server_closed is not None:
            return EOF()

        if self.send_closed:
            raise StreamClosed

        msg = _Message(_DATA_MESSAGE, payload, None)
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
        if self.send_closed or self.server_closed is not None:
            return

        msg = _Message(_CLOSE_MESSAGE, None, None)
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


DEFAULT_MAX_SIZE = 2**20


class WebsocketClient:
    """An implementation of AsyncStreamClient that is backed by a websocket

    :param encoder: The encoder to use for this client.
    :param endpoint: A base url to use as a prefix for all requests.
    :param max_message_size: The maximum size of a message to receive. Defaults to
    DEFAULT_MAX_SIZE.
    """

    _endpoint: URL
    _encoder: EncoderDecoder
    _max_message_size: int

    def __init__(
        self,
        encoder: EncoderDecoder,
        endpoint: URL,
        max_message_size: int = DEFAULT_MAX_SIZE,
    ) -> None:
        self._encoder = encoder
        self._endpoint = endpoint.replace(protocol="ws")
        self._max_message_size = max_message_size

    async def stream(
        self, target: str, req_type: Type[RQ], res_type: Type[RS]
    ) -> AsyncStream[RQ, RS]:
        ws = await connect(
            self._endpoint.child(target).stringify(),
            extra_headers={"Content-Type": self._encoder.content_type()},
            max_size=self._max_message_size,
        )
        return WebsocketStream[RQ, RS](self._encoder, ws, res_type)
