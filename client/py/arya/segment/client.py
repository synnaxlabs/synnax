from datetime import datetime

import numpy as np

from .. import channel, telem
from ..transport import Transport
from . import writer
from . import iterator
from .entity import NumpySegment


class Client:
    transport: Transport
    channel_client: channel.Client

    def __init__(self, transport: Transport, channel_client: channel.Client):
        self.transport = transport
        self.channel_client = channel_client

    def new_writer(self, keys: list[str]) -> writer.Numpy:
        core = writer.Core(transport=self.transport.stream)
        npw = writer.Numpy(core=core, channel_client=self.channel_client)
        npw.open(keys)
        return npw

    def new_iterator(self, keys: list[str], tr: telem.TimeRange, aggregate: bool = False) -> iterator.Numpy:
        npi = iterator.Numpy(transport=self.transport.stream,
                             channel_client=self.channel_client,
                             aggregate=aggregate)
        npi.open(keys, tr)
        return npi

    def write(self, to: str, data: np.ndarray, start: telem.UnparsedTimeStamp):
        _writer = self.new_writer([to])
        try:
            _writer.write(to, data, start)
        finally:
            _writer.close()

    def read(self, from_: str, tr: telem.TimeRange) -> np.ndarray:
        seg = self.read_seg(from_, tr)
        return seg.data

    def read_seg(self, from_: str, tr: telem.TimeRange) -> NumpySegment:
        _iterator = self.new_iterator([from_], tr, aggregate=True)
        seg = None
        try:
            _iterator.first()
            while _iterator.next():
                pass
            seg = _iterator.value[from_]
        finally:
            _iterator.close()
        return seg
