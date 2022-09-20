from enum import Enum

from freighter import EOF, ExceptionPayload, Payload, Stream, StreamClient

from synnax.channel.registry import ChannelRegistry
from synnax.telem import TimeRange, TimeSpan, TimeStamp

from .encoder import NumpyEncoderDecoder
from .payload import SegmentPayload
from .sugared import NumpySegment


class _Command(int, Enum):
    OPEN = 0
    NEXT = 1
    PREV = 2
    FIRST = 3
    LAST = 4
    NEXT_SPAN = 5
    PREV_SPAN = 6
    NEXT_RANGE = 7
    VALID = 8
    ERROR = 9
    SEEK_FIRST = 11
    SEEK_LAST = 12
    SEEK_LT = 13
    SEEK_GE = 14


class _ResponseVariant(int, Enum):
    ACK = 1
    DATA = 2


class _Request(Payload):
    command: _Command
    span: TimeSpan | None = None
    range: TimeRange | None = None
    stamp: TimeStamp | None = None
    keys: list[str] | None = None


class _Response(Payload):
    variant: _ResponseVariant
    ack: bool
    command: _Command
    error: ExceptionPayload
    segments: list[SegmentPayload]


class CoreIterator:
    _ENDPOINT = "/segment/iterate"

    transport: StreamClient
    stream: Stream[_Request, _Response]
    values: list[SegmentPayload]
    aggregate: bool

    def __init__(self, transport: StreamClient, aggregate: bool = False) -> None:
        self.transport = transport
        self.aggregate = aggregate

    def open(self, keys: list[str], tr: TimeRange):
        self.stream = self.transport.stream(self._ENDPOINT, _Request, _Response)
        self.exec(command=_Command.OPEN, range=tr, keys=keys)
        self.values = []

    def exec(self, **kwargs) -> bool:
        exc = self.stream.send(_Request(**kwargs))
        if exc is not None:
            raise exc
        if not self.aggregate:
            self.values = []
        while True:
            r, exc = self.stream.receive()
            if exc is not None:
                raise exc
            if r.variant == _ResponseVariant.ACK:
                return r.ack
            self.values += r.segments

    def next(self) -> bool:
        return self.exec(command=_Command.NEXT)

    def prev(self) -> bool:
        return self.exec(command=_Command.PREV)

    def first(self) -> bool:
        return self.exec(command=_Command.FIRST)

    def last(self) -> bool:
        return self.exec(command=_Command.LAST)

    def next_span(self, span: TimeSpan) -> bool:
        return self.exec(command=_Command.NEXT_SPAN, span=span)

    def prev_span(self, span: TimeSpan) -> bool:
        return self.exec(command=_Command.PREV_SPAN, span=span)

    def next_range(self, rng: TimeRange) -> bool:
        return self.exec(command=_Command.NEXT_RANGE, range=rng)

    def seek_first(self) -> bool:
        return self.exec(command=_Command.SEEK_FIRST)

    def seek_last(self) -> bool:
        return self.exec(command=_Command.SEEK_LAST)

    def seek_lt(self, stamp: TimeStamp) -> bool:
        return self.exec(command=_Command.SEEK_LT, stamp=stamp)

    def seek_ge(self, stamp: TimeStamp) -> bool:
        return self.exec(command=_Command.SEEK_GE, stamp=stamp)

    def valid(self) -> bool:
        return self.exec(command=_Command.VALID)

    def close(self):
        exc = self.stream.close_send()
        if exc is not None:
            raise exc
        pld, exc = self.stream.receive()
        if not isinstance(exc, EOF):
            raise exc


class NumpyIterator(CoreIterator):
    decoder: NumpyEncoderDecoder
    channels: ChannelRegistry

    def __init__(
        self,
        transport: StreamClient,
        channels: ChannelRegistry,
        aggregate: bool = False,
    ):
        super().__init__(transport, aggregate)
        self.decoder = NumpyEncoderDecoder()
        self.channel_client = channels

    def open(self, keys: list[str], tr: TimeRange) -> None:
        super().open(keys, tr)

    @property
    def value(self) -> dict[str, NumpySegment]:
        decoded = []
        self.values.sort(key=lambda v: v.start)
        res = dict()
        for i, seg in enumerate(self.values):
            decoded.append(self.decoder.decode(self.channels.get(seg.channel_key), seg))
        for i, dec in enumerate(decoded):
            if dec.channel.key in res:
                res[dec.channel.key].extend(dec)
            else:
                res[dec.channel.key] = dec
        return res
