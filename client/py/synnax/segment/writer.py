import asyncio

from dataclasses import dataclass

import numpy as np

import synnax.errors
from . import BinarySegment, NumpySegment
import freighter

from . import validate
from .. import telem
from .. import channel
from .splitter import Splitter
from .encoder import NumpyEncoderDecoder

_ENDPOINT = "/segment/write"


@dataclass
class Request:
    open_keys: list[str]
    segments: list[BinarySegment]


@dataclass
class Response:
    ack: bool
    error: freighter.ErrorPayload


def _response_factory() -> Response:
    return Response(False, freighter.ErrorPayload(None, None))


class BaseCore:
    keys: list[str]

    def _ack_open(self, res: Response | None, exc: Exception | None):
        if exc is not None:
            raise exc
        assert res is not None
        if not res.ack:
            raise synnax.errors.UnexpectedError(
                "Writer failed to positively acknowledge open request. This is a bug"
                + "please report it."
            )


class AsyncCore(BaseCore):
    transport: freighter.AsyncStreamClient
    stream: freighter.AsyncStream[Request, Response]
    responses: asyncio.Task[Exception | None]

    def __init__(self, transport: freighter.AsyncStreamClient) -> None:
        self.transport = transport

    async def open(self, keys: list[str]):
        self.stream = await self.transport.stream(_ENDPOINT, Request, _response_factory)
        await self.stream.send(Request(keys, []))
        res, err = await self.stream.receive()
        self._ack_open(res, err)
        self.responses = asyncio.create_task(self.receive_errors())

    async def write(self, segments: list[BinarySegment]) -> bool:
        if self.responses.done():
            return False
        err = await self.stream.send(Request([], segments))
        if err is not None:
            raise err
        return True

    async def close(self):
        await self.stream.close_send()
        err = await self.responses
        assert err is not None
        if not isinstance(err, freighter.EOF):
            raise err

    async def receive_errors(self) -> Exception | None:
        res, exc = await self.stream.receive()
        if exc is None:
            assert res is not None
            exc = freighter.errors.decode(res.error)
        return exc


class Core(BaseCore):
    transport: freighter.StreamClient
    stream: freighter.Stream[Request, Response]

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport

    def open(self, keys: list[str]):
        self.stream = self.transport.stream(_ENDPOINT, Request, _response_factory)
        self.stream.send(Request(keys, []))
        res, err = self.stream.receive()
        self._ack_open(res, err)

    def write(self, segments: list[BinarySegment]) -> bool:
        if self.stream.received():
            return False
        err = self.stream.send(Request([], segments))
        if err is not None:
            raise err
        return True

    def close(self):
        self.stream.close_send()
        res, err = self.stream.receive()
        if err is None:
            err = freighter.errors.decode(res.error)
        if not isinstance(err, freighter.EOF):
            raise err


class Numpy:
    core: Core
    channel_client: channel.Client
    validators: list[validate.Validator]
    encoder: NumpyEncoderDecoder
    channels: channel.Registry
    splitter: Splitter

    def __init__(
        self,
        core: Core,
        channel_client: channel.Client,
    ) -> None:
        self.channel_client = channel_client
        self.core = core
        self.validators = [
            validate.ScalarType(),
            validate.Contiguity(dict(), allow_no_high_water_mark=True),
        ]
        self.encoder = NumpyEncoderDecoder()
        self.splitter = Splitter(threshold=telem.Size(4e6))

    def open(self, keys: list[str]):
        self.channels = channel.Registry(self.channel_client.retrieve(keys))
        self.core.open(keys)

    def write(self, to: str, data: np.ndarray, start: telem.UnparsedTimeStamp) -> bool:
        ch = self.channels.get(to)
        if ch is None:
            raise synnax.errors.QueryError(f"channel with key {to} not found")
        seg = NumpySegment(ch, telem.TimeStamp(start), data)
        for val in self.validators:
            val.validate(seg)
        encoded = self.encoder.encode(seg).sugar(ch)
        split = self.splitter.split(encoded)
        return self.core.write([seg.desugar() for seg in split])

    def close(self):
        self.core.close()
