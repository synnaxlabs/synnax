from . import SugaredBinarySegment
from .. import telem


class Splitter:
    threshold: telem.Size

    def __init__(self, threshold: telem.Size) -> None:
        self.threshold = threshold

    def split(self, segment: SugaredBinarySegment) -> list[SugaredBinarySegment]:
        if segment.size <= self.threshold:
            return [segment]
        else:
            truncated = SugaredBinarySegment(segment.channel, segment.start,
                                             segment.data[:self.threshold])
            _next = SugaredBinarySegment(segment.channel, truncated.end,
                                         segment.data[self.threshold:])
            return [truncated, *self.split(_next)]
