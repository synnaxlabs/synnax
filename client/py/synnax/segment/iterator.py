from dataclasses import dataclass
from datetime import datetime
from enum import Enum

import freighter

from synnax import telem, channel
from synnax.channel import entity
from synnax.segment import BinarySegment, NumpySegment, encoder

_ENDPOINT = "/segment/iterate"


class Command(int, Enum):
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


class ResponseVariant(int, Enum):
    ACK = 1
    DATA = 2


@dataclass
class Request:
    command: Command
    span: telem.TimeSpan | None = None
    range: telem.TimeRange | None = None
    stamp: telem.TimeStamp | None = None
    keys: list[str] | None = None


@dataclass
class Response:
    variant: ResponseVariant
    ack: bool
    command: Command
    error: freighter.ErrorPayload
    segments: list[BinarySegment]

    def load(self, data: dict):
        if data["variant"] == 0:
            return
        self.variant = ResponseVariant(data["variant"])
        if self.variant == ResponseVariant.ACK:
            self.ack = data["ack"]
            self.command = Command(data["command"])
            if "error" in data:
                self.error = freighter.ErrorPayload(data["error"]["type"],
                                                    data["error"]["data"])
        elif self.variant == ResponseVariant.DATA:
            self.segments = [BinarySegment(**seg) for seg in data["segments"]]
        else:
            raise ValueError("Unexpected response variant")


def _response_factory() -> Response:
    return Response(ResponseVariant.ACK, False, Command.OPEN,
                    freighter.ErrorPayload(None, None), [])


class Core:
    transport: freighter.StreamClient
    stream: freighter.Stream[Request, Response]
    values: list[BinarySegment]
    aggregate: bool

    def __init__(self, transport: freighter.StreamClient, aggregate: bool = False) -> None:
        self.transport = transport
        self.aggregate = aggregate

    def open(self, keys: list[str], tr: telem.TimeRange):
        self.stream = self.transport.stream(_ENDPOINT, Request, _response_factory)
        self.exec(command=Command.OPEN, range=tr, keys=keys)
        self.values = []

    def exec(self, **kwargs) -> bool:
        exc = self.stream.send(Request(**kwargs))
        if exc is not None:
            raise exc
        if not self.aggregate:
            self.values = []
        while True:
            r, exc = self.stream.receive()
            if exc is not None:
                raise exc
            if r.variant == ResponseVariant.ACK:
                return r.ack
            self.values += r.segments

    def next(self) -> bool:
        return self.exec(command=Command.NEXT)

    def prev(self) -> bool:
        return self.exec(command=Command.PREV)

    def first(self) -> bool:
        return self.exec(command=Command.FIRST)

    def last(self) -> bool:
        return self.exec(command=Command.LAST)

    def next_span(self, span: telem.TimeSpan) -> bool:
        return self.exec(command=Command.NEXT_SPAN, span=span)

    def prev_span(self, span: telem.TimeSpan) -> bool:
        return self.exec(command=Command.PREV_SPAN, span=span)

    def next_range(self, rng: telem.TimeRange) -> bool:
        return self.exec(command=Command.NEXT_RANGE, range=rng)

    def seek_first(self) -> bool:
        return self.exec(command=Command.SEEK_FIRST)

    def seek_last(self) -> bool:
        return self.exec(command=Command.SEEK_LAST)

    def seek_lt(self, stamp: telem.TimeStamp) -> bool:
        return self.exec(command=Command.SEEK_LT, stamp=stamp)

    def seek_ge(self, stamp: telem.TimeStamp) -> bool:
        return self.exec(command=Command.SEEK_GE, stamp=stamp)

    def valid(self) -> bool:
        return self.exec(command=Command.VALID)

    def exhaust(self) -> bool:
        return self.exec(command=Command.EXHAUST)

    def close(self):
        exc = self.stream.close_send()
        if exc is not None:
            raise exc
        pld, exc = self.stream.receive()
        print(pld)
        if not isinstance(exc, freighter.EOF):
            print(exc)
            raise exc


class Numpy(Core):
    decoder: encoder.NumpyEncoderDecoder
    channel_client: channel.Client
    channels: channel.Registry

    def __init__(
            self,
            transport: freighter.StreamClient,
            channel_client: channel.Client,
            aggregate: bool = False
    ):
        super().__init__(transport, aggregate)
        self.decoder = encoder.NumpyEncoderDecoder()
        self.channel_client = channel_client

    def open(self, keys: list[str], tr: telem.TimeRange) -> None:
        self.channels = channel.Registry(self.channel_client.retrieve(keys))
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
