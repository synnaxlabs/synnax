import asyncio
import copy

from websockets import WebSocketClientProtocol, connect
from websockets.exceptions import ConnectionClosedOK

from .endpoint import Endpoint
from .transport import RS, RQ, Payload
from typing import Generic, Callable
from dataclasses import dataclass
from .errors import StreamClosed
from .error_registry import ErrorPayload
from .encoder import EncoderDecoder
from . import stream, errors

_DATA_MESSAGE = "data"
_CLOSE_MESSAGE = "close"


@dataclass
class _Message(Payload):
    type: str
    payload: Payload | None = None
    error: ErrorPayload | None = None


def empty_message(payload: Payload) -> _Message:
    return _Message(type=_DATA_MESSAGE, payload=payload, error=ErrorPayload(None, None))


class AsyncWSClient(Generic[RS, RQ]):
    endpoint: Endpoint
    encoder: EncoderDecoder

    def __init__(self, encoder: EncoderDecoder, endpoint: Endpoint) -> None:
        self.encoder = encoder
        self.endpoint = copy.copy(endpoint)
        self.endpoint.protocol = "ws"

    async def stream(
            self, target: str, response_factory: Callable[[], RS]
    ) -> stream.AsyncStream[RS, RQ]:
        ws = await connect(
            self.endpoint.build(target),
            extra_headers={"Content-Type": self.encoder.content_type()},
        )
        return AsyncWSStream[RS, RQ](self.encoder, ws, response_factory)


class AsyncWSStream(Generic[RS, RQ]):
    encoder: EncoderDecoder
    wrapped: WebSocketClientProtocol
    server_closed: Exception | None
    send_closed: bool
    lock: asyncio.Lock
    response_factory: Callable[[], RS]

    def __init__(
            self,
            encoder: EncoderDecoder,
            ws: WebSocketClientProtocol,
            response_factory: Callable[[], RS],
    ):
        self.encoder = encoder
        self.wrapped = ws
        self.send_closed = False
        self.server_closed = None
        self.lock = asyncio.Lock()
        self.response_factory = response_factory

    async def receive(self) -> (RS, Exception | None):
        if self.server_closed is not None:
            return None, self.server_closed

        data = await self.wrapped.recv()
        msg = empty_message(self.response_factory())
        self.encoder.decode(data, msg)

        if msg.type == _CLOSE_MESSAGE:
            await self._close_server(errors.decode(msg.error))
            return None, self.server_closed

        return msg.payload, None

    async def send(self, payload: RQ) -> Exception | None:
        # If the server closed with an error, we return freighter.EOF to the
        # caller, and expect them to discover the close error by calling
        # receive().
        if self.server_closed is not None:
            return errors.EOF()

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
            return errors.EOF()

    async def close_send(self):
        await self.lock.acquire()
        if self.send_closed or self.server_closed is not None:
            return
        self.lock.release()

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
