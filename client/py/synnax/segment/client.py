import numpy as np

from synnax.channel.registry import ChannelRegistry
from synnax.segment.iterator import NumpyIterator as NumpySegmentIterator
from synnax.segment.writer import CoreWriter as CoreSegmentWriter
from synnax.segment.writer import NumpyWriter as NumpySegmentWriter
from synnax.telem import TimeRange, UnparsedTimeStamp

from ..transport import Transport
from . import iterator
from .sugared import NumpySegment


class SegmentClient:
    transport: Transport
    channels: ChannelRegistry

    def __init__(self, transport: Transport, registry: ChannelRegistry):
        self.transport = transport
        self.channels = registry

    def new_writer(self, keys: list[str]) -> NumpySegmentWriter:
        core = CoreSegmentWriter(client=self.transport.stream)
        npw = NumpySegmentWriter(core=core, channels=self.channels)
        npw.open(keys)
        return npw

    def new_iterator(
        self,
        keys: list[str],
        tr: TimeRange,
        aggregate: bool = False,
    ) -> NumpySegmentIterator:
        npi = iterator.NumpyIterator(
            transport=self.transport.stream,
            channels=self.channels,
            aggregate=aggregate,
        )
        npi.open(keys, tr)
        return npi

    def write(self, to: str, start: UnparsedTimeStamp, data: np.ndarray):
        _writer = self.new_writer([to])
        try:
            _writer.write(to, data, start)
        finally:
            _writer.close()

    def read(
        self, from_: str, start: UnparsedTimeStamp, end: UnparsedTimeStamp
    ) -> np.ndarray:
        seg = self.read_seg(from_, start, end)
        return seg.data

    def read_seg(
        self, from_: str, start: UnparsedTimeStamp, end: UnparsedTimeStamp
    ) -> NumpySegment:
        _iterator = self.new_iterator([from_], TimeRange(start, end), aggregate=True)
        seg = None
        try:
            _iterator.first()
            while _iterator.next():
                pass
            seg = _iterator.value[from_]
        finally:
            _iterator.close()
        return seg
