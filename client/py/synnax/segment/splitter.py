from .. import telem
from .sugared import SugaredBinarySegment


class Splitter:
    threshold: telem.Size

    def __init__(self, threshold: telem.Size) -> None:
        self.threshold = threshold

    def split(self, segment: SugaredBinarySegment) -> list[SugaredBinarySegment]:
        if segment.size <= self.threshold:
            return [segment]
        split_v = self.threshold - (self.threshold % segment.channel.density)
        truncated = SugaredBinarySegment(
            segment.channel, segment.start, segment.data[:split_v]
        )
        _next = SugaredBinarySegment(
            segment.channel, truncated.end, segment.data[split_v:]
        )
        return [truncated, *self.split(_next)]
