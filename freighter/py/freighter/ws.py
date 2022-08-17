import asyncio

import websockets

from .transport import I, O, Payload
from typing import Generic
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
    return _Message(
        type=_DATA_MESSAGE,
        payload=payload,
        error=ErrorPayload(None, None)
    )


class StreamClient(Generic[I, O]):
    endpoint: str
    encoder: EncoderDecoder

    def __init__(self, encoder: EncoderDecoder, endpoint: str) -> None:
        self.encoder = encoder
        self.endpoint = endpoint

    async def stream(self, target: str) -> stream.Stream[I, O]:
        ws = await websockets.connect(
            "ws://" + self.endpoint + target,
            extra_headers={"Content-Type": self.encoder.content_type()}
        )
        return Stream(self.encoder, ws)


class Stream(Generic[I, O]):
    encoder: EncoderDecoder
    ws: websockets.WebSocketClientProtocol
    server_closed: Exception | None
    send_closed: bool
    close_lock: asyncio.Lock

    def __init__(
            self,
            encoder: EncoderDecoder,
            ws: websockets.WebSocketClientProtocol,
    ):
        self.encoder = encoder
        self.ws = ws
        self.send_closed = False
        self.server_closed = None
        self.close_lock = asyncio.Lock()

    async def receive(self, payload: I) -> Exception | None:
        if self.server_closed is not None:
            return self.server_closed

        data = await self.ws.recv()
        msg = empty_message(payload)
        self.encoder.decode(data, msg)

        if msg.type == _CLOSE_MESSAGE:
            err = errors.decode(msg.error)
            await self._close(err)
            return self.server_closed

        return None

    async def send(self, payload: O) -> Exception | None:
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
            await self.ws.send(encoded)
        except websockets.exceptions.ConnectionClosedOK:
            await self._close(None)
            return errors.EOF()

    async def close_send(self):
        if self.send_closed or self.server_closed is not None:
            return

        msg = _Message(_CLOSE_MESSAGE, None, None)
        try:
            await self.ws.send(self.encoder.encode(msg))
        except websockets.exceptions.ConnectionClosedOK:
            await self._close(None)
            return

    async def _close(self, server_err: Exception | None):
        await self.close_lock.acquire()

        if self.server_closed is not None:
            return

        if server_err is not None:
            self.server_closed = server_err

        await self.ws.close()

        self.close_lock.release()
