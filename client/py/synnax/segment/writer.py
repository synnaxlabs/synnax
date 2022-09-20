import asyncio

from freighter import (
    EOF,
    AsyncStream,
    AsyncStreamClient,
    ExceptionPayload,
    Payload,
    Stream,
    StreamClient,
    decode_exception,
)
from numpy import ndarray

from synnax.channel.registry import ChannelRegistry
from synnax.exceptions import UnexpectedError, ValidationError, ValidationField
from .sugared import NumpySegment, SugaredBinarySegment
from synnax.telem import Size, TimeStamp, UnparsedTimeStamp

from .encoder import NumpyEncoderDecoder
from .payload import SegmentPayload
from .splitter import Splitter
from .validate import ContiguityValidator, ScalarTypeValidator, Validator

_ENDPOINT = "/segment/write"


class _Request(Payload):
    open_keys: list[str]
    segments: list[SegmentPayload]


class _Response(Payload):
    ack: bool
    error: ExceptionPayload


class BaseWriter:
    keys: list[str]

    def _ack_open(self, res: _Response | None, exc: Exception | None):
        if exc is not None:
            raise exc
        assert res is not None
        if not res.ack:
            raise UnexpectedError(
                "Writer failed to positively acknowledge open request. This is a bug"
                + "please report it."
            )

    def check_keys(self, segments: list[SegmentPayload]):
        for segment in segments:
            if segment.channel_key not in self.keys:
                raise ValidationError(
                    ValidationField(
                        "key",
                        f"key {segment.key} is not in the list of keys for this writer.",
                    )
                )


class AsyncCoreWriter(BaseWriter):
    client: AsyncStreamClient
    stream: AsyncStream[_Request, _Response]
    responses: asyncio.Task[Exception | None]

    def __init__(self, client: AsyncStreamClient) -> None:
        self.client = client

    async def open(self, keys: list[str]):
        self.keys = keys
        self.stream = await self.client.stream(_ENDPOINT, _Request, _Response)
        await self.stream.send(_Request(open_keys=keys, segments=[]))
        res, err = await self.stream.receive()
        self._ack_open(res, err)
        self.responses = asyncio.create_task(self.receive_errors())

    async def write(self, segments: list[SegmentPayload]) -> bool:
        if self.responses.done():
            return False

        self.check_keys(segments)
        err = await self.stream.send(_Request(open_keys=[], segments=segments))
        if err is not None:
            raise err
        return True

    async def close(self):
        await self.stream.close_send()
        err = await self.responses
        assert err is not None
        if not isinstance(err, EOF):
            raise err

    async def receive_errors(self) -> Exception | None:
        res, exc = await self.stream.receive()
        if exc is None:
            assert res is not None
            exc = decode_exception(res.error)
        return exc


class CoreWriter(BaseWriter):
    client: StreamClient
    stream: Stream[_Request, _Response]

    def __init__(self, client: StreamClient) -> None:
        self.client = client

    def open(self, keys: list[str]):
        self.keys = keys
        self.stream = self.client.stream(_ENDPOINT, _Request, _Response)
        self.stream.send(_Request(open_keys=keys, segments=[]))
        res, err = self.stream.receive()
        self._ack_open(res, err)

    def write(self, segments: list[SegmentPayload]) -> bool:
        if self.stream.received():
            return False

        self.check_keys(segments)
        err = self.stream.send(_Request(open_keys=[], segments=segments))
        if err is not None:
            raise err
        return True

    def close(self):
        self.stream.close_send()
        res, err = self.stream.receive()
        if err is None:
            err = decode_exception(res.error)
        if not isinstance(err, EOF):
            raise err


class NumpyWriter:
    core: CoreWriter
    validators: list[Validator]
    encoder: NumpyEncoderDecoder
    splitter: Splitter

    def __init__(
        self,
        core: CoreWriter,
        channels: ChannelRegistry,
    ) -> None:
        self.core = core
        self.validators = [
            ScalarTypeValidator(),
            ContiguityValidator(dict(), allow_no_high_water_mark=True),
        ]
        self.encoder = NumpyEncoderDecoder()
        self.splitter = Splitter(threshold=Size(4e6))
        self.channels = channels

    def open(self, keys: list[str]):
        self.core.open(keys)

    def write(self, to: str, data: ndarray, start: UnparsedTimeStamp) -> bool:
        ch = self.channels.get(to)
        seg = NumpySegment(ch, TimeStamp(start), data)
        for val in self.validators:
            val.validate(seg)
        encoded = SugaredBinarySegment.sugar(ch,self.encoder.encode(seg))
        split = self.splitter.split(encoded)
        return self.core.write([seg.payload() for seg in split])

    def close(self):
        self.core.close()
