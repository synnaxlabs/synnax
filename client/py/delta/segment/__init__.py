from dataclasses import dataclass

import numpy

from delta import telem
from delta.channel import Channel
from delta.telem import TimeStamp, TimeSpan, TimeRange


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

    @property
    def size(self) -> telem.Size:
        return telem.Size(len(self.data))

    @property
    def span(self) -> TimeSpan:
        return self.channel.rate.span(self.size)

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
