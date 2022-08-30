from dataclasses import dataclass
from enum import Enum

import freighter

import arya.errors
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
    span: telem.TimeSpan
    range: telem.TimeRange
    stamp: telem.TimeStamp
    keys: list[str]


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

    def __init__(self, transport: freighter.StreamClient) -> None:
        self.transport = transport

    def open(self, keys: list[str]):
        self.stream = self.transport.stream(_ENDPOINT, Request, _response_factory)
        self.stream.send(Request(Command.OPEN, None, None, None, keys))
        _, exc = self.stream.receive()
        if exc is not None:
            raise exc


