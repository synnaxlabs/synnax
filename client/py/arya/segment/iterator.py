from dataclasses import dataclass
from enum import Enum

import freighter

from arya import telem
from arya.segment import BinarySegment

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
    SEEK_FIRST = 8
    SEEK_LAST = 9
    SEEK_LT = 10
    SEEK_GE = 11
    VALID = 12
    ERROR = 13
    CLOSE = 14
    EXHAUST = 15


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
    sync: bool = False


@dataclass
class Response:
    variant: ResponseVariant
    ack: bool
    command: Command
    error: freighter.ErrorPayload
    segments: list[BinarySegment]


def _response_factory() -> Response:
    return Response(ResponseVariant.ACK, False, Command.OPEN,
                    freighter.ErrorPayload(None, None), [])


class Core:
    transport: freighter.StreamClient
    stream: freighter.Stream[Request, Response]
    values: list[BinarySegment]

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport

    def open(self, keys: list[str], tr: telem.TimeRange):
        self.stream = self.transport.stream(_ENDPOINT, Request, _response_factory)
        self.exec(command=Command.OPEN, range=tr, keys=keys)
        self.values = []

    def exec(self, **kwargs) -> bool:
        exc = self.stream.send(Request(**kwargs))
        if exc is not None:
            raise exc
        while True:
            r, exc = self.stream.receive()
            if exc is not None:
                raise exc
            if r.variant == ResponseVariant.ACK:
                return r.ack
            self.values = r.segments

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
        try:
            self.exec(command=Command.CLOSE)
        except freighter.errors.EOF:
            raise ValueError("segment iterator closed unexpectedly")
        except Exception as e:
            raise e

