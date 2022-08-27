import asyncio
from asyncio import Task

from dataclasses import dataclass

import numpy as np

import delta.errors
from . import BinarySegment, NumpySegment
import freighter

from .encoder import NumpyEncoderDecoder
from .validator import Validator, ScalarTypeValidator, ContiguityValidator
from .. import telem
from ..channel.client import ChannelClient
from ..channel.registry import ChannelRegistry
from ..util.notification import Notification
from .splitter import Splitter

_ENDPOINT = "/segment/write"


@dataclass
class WriterRequest:
    open_keys: list[str]
    segments: list[BinarySegment]


@dataclass
class WriterResponse:
    ack: bool
    error: freighter.ErrorPayload


def _response_factory() -> WriterResponse:
    return WriterResponse(False, freighter.ErrorPayload(None, None))


class BaseCore:
    keys: list[str]

    def _ack_open(self, res: WriterResponse | None, exc: Exception | None):
        if exc is not None:
            raise exc
        assert res is not None
        if res.ack:
            raise delta.errors.UnexpectedError(
                "Writer failed to positively acknowledge open request. This is a bug"
                + "please report it."
            )


class AsyncCore(BaseCore):
    transport: freighter.AsyncStreamClient[WriterRequest, WriterResponse]
    stream: freighter.AsyncStream[WriterRequest, WriterResponse]
    responses: Task[Exception | None]

    def __init__(self, transport: freighter.AsyncStreamClient) -> None:
        self.transport = transport

    async def open(self, keys: list[str]):
        self.stream = await self.transport.stream(_ENDPOINT, _response_factory)
        await self.stream.send(WriterRequest(keys, []))
        res, err = await self.stream.receive()
        self._ack_open(res, err)
        self.responses = asyncio.create_task(self.receive_errors())

    async def write(self, segments: list[BinarySegment]) -> bool:
        if self.responses.done():
            return False
        err = await self.stream.send(WriterRequest([], segments))
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
    stream: freighter.Stream[WriterRequest, WriterResponse]

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport

    def open(self, keys: list[str]):
        self.stream = self.transport.stream(_ENDPOINT, _response_factory)
        self.stream.send(WriterRequest(keys, []))
        res, err = self.stream.receive()
        self._ack_open(res, err)

    def write(self, segments: list[BinarySegment]) -> bool:
        if self.stream.received():
            return False
        err = self.stream.send(WriterRequest([], segments))
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


class NumpyWriter:
    core: Core
    channel_client: ChannelClient
    validators: list[Validator]
    encoder: NumpyEncoderDecoder
    channels: ChannelRegistry
    splitter: Splitter

    def __init__(
            self,
            keys: list[str],
            transport: freighter.StreamClient,
            channel_client: ChannelClient,
    ) -> None:
        self.channel_client = channel_client
        channels = self.channel_client.retrieve(keys)
        if len(channels) != len(keys):
            missing = set(keys) - set([c.key for c in channels])
            raise delta.errors.ValidationError(f"Channels not found: {missing}")
        self.channels = ChannelRegistry(channels)
        self.core = Core(transport)
        self.validators = [
            ScalarTypeValidator(),
            ContiguityValidator(dict(), allow_no_high_water_mark=True)
        ]
        self.encoder = NumpyEncoderDecoder()
        self.core.open(keys)

    def write(self, to: str, data: np.ndarray, start: telem.UnparsedTimeStamp) -> bool:
        ch = self.channels.get(to)
        if ch is None:
            raise delta.errors.QueryError(f"channel with key {to} not found")
        seg = NumpySegment(ch, telem.TimeStamp(start), data)
        for val in self.validators:
            val.validate(seg)
        encoded = self.encoder.encode(seg)
        return self.core.write(self.splitter.split(encoded))

    def close(self):
        self.core.close()
