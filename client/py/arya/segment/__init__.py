from __future__ import annotations
from dataclasses import dataclass

import numpy

from arya import telem
from arya.channel import Channel
from arya.telem import TimeStamp, TimeSpan, TimeRange


@dataclass
class SugaredHeader:
    channel: Channel
    start: TimeStamp = TimeStamp(0)

    def __init__(self, channel: Channel, start: telem.UnparsedTimeStamp):
        self.channel = channel
        self.start = TimeStamp(start)


@dataclass
class Header:
    channel_key: str
    start: TimeStamp = TimeStamp(0)

    def __init__(self, channel_key: str, start: telem.UnparsedTimeStamp):
        self.channel_key = channel_key
        self.start = TimeStamp(start)

    def sugar(self, channel: Channel) -> SugaredHeader:
        return SugaredHeader(channel, self.start)


@dataclass
class BinarySegment(Header):
    data: bytes = b""

    def __init__(self,
                 channel_key: str = "",
                 start: telem.UnparsedTimeStamp = telem.TimeStamp(0),
                 data: bytes = b"",
                 ):
        super().__init__(channel_key, start)
        self.data = data

    def sugar(self, channel: Channel) -> SugaredBinarySegment:
        return SugaredBinarySegment(channel, self.start, self.data)

    @property
    def size(self) -> telem.Size:
        return telem.Size(len(self.data))


@dataclass
class SugaredBinarySegment(SugaredHeader):
    data: bytes = b""

    def __init__(self,
                 channel: Channel,
                 start: telem.UnparsedTimeStamp,
                 data: bytes = b""):
        super().__init__(channel, start)
        self.data = data

    def desugar(self) -> BinarySegment:
        return BinarySegment(self.channel.key, self.start, self.data)

    @property
    def size(self) -> telem.Size:
        return telem.Size(len(self.data))

    @property
    def span(self) -> TimeSpan:
        return self.channel.rate.size_span(self.size, self.channel.data_type.density)

    @property
    def range(self) -> TimeRange:
        return self.start.span_range(self.span)

    @property
    def end(self) -> TimeStamp:
        return self.range.end


@dataclass
class NumpySegment(SugaredHeader):
    data: numpy.ndarray = numpy.array([])

    def __init__(self, channel: Channel, start: TimeStamp, data: numpy.ndarray):
        super().__init__(channel, start)
        self.data = data

    @property
    def span(self) -> TimeSpan:
        return self.channel.rate.span(self.data.size)

    @property
    def range(self) -> TimeRange:
        return self.start.span_range(self.span)

    @property
    def end(self) -> TimeStamp:
        return self.range.end
