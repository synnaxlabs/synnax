from __future__ import annotations

from array import array
from dataclasses import dataclass

from numpy import append, ndarray

from synnax.channel.payload import ChannelPayload
from synnax.telem import Size, TimeRange, TimeSpan, TimeStamp, UnparsedTimeStamp

from .payload import SegmentPayload


@dataclass
class SugaredHeader:
    channel: ChannelPayload
    start: TimeStamp = TimeStamp(0)

    def __init__(self, channel: ChannelPayload, start: UnparsedTimeStamp):
        self.channel = channel
        self.start = TimeStamp(start)


class SugaredBinarySegment(SugaredHeader):
    data: bytes = b""

    def __init__(
        self, channel: ChannelPayload, start: UnparsedTimeStamp, data: bytes = b""
    ):
        super().__init__(channel, start)
        self.data = data

    def payload(self) -> SegmentPayload:
        return SegmentPayload(
            channel_key=self.channel.key, start=self.start, data=self.data
        )

    @property
    def size(self) -> Size:
        return Size(len(self.data))

    @property
    def span(self) -> TimeSpan:
        return self.channel.rate.size_span(self.size, self.channel.density)

    @property
    def range(self) -> TimeRange:
        return self.start.span_range(self.span)

    @property
    def end(self) -> TimeStamp:
        return self.range.end


@dataclass
class NumpySegment(SugaredHeader):
    data: ndarray = array()

    def __init__(self, channel: ChannelPayload, start: TimeStamp, data: ndarray):
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

    def extend(self, other: NumpySegment):
        assert self.channel == other.channel
        assert self.end == other.start
        self.data = append(self.data, other.data)
