from __future__ import annotations

from dataclasses import dataclass

from numpy import append, ndarray

from synnax.channel.payload import ChannelPayload
from synnax.exceptions import ContiguityError, ValidationError
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

    @classmethod
    def sugar(cls, ch: ChannelPayload, seg: SegmentPayload) -> SugaredBinarySegment:
        return cls(ch, seg.start, seg.data)

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
    data: ndarray = ndarray([])

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
        if self.channel.key != other.channel.key:
            raise ValidationError(
                f"""Cannot extend segment because channel keys mismatch.
                Segment Channel: {self.channel.key}
                Next Segment Channel: {other.channel.key}
                """
            )
        if self.end != other.start:
            raise ContiguityError(
                f"""Cannot extend segment because end and start times are not equal.
                Segment End: {self.end}
                Next Segment Start: {other.start}
                """
            )
        self.data = append(self.data, other.data)
