import numpy as np

from .. import channel, telem
from ..transport import Transport
from . import writer
from . import iterator


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

    def write(self, to: str, data: np.ndarray, start: telem.UnparsedTimeStamp):
        _writer = self.new_writer([to])
        try:
            _writer.write(to, data, start)
        finally:
            _writer.close()

    def new_iterator(self, keys: list[str], tr: telem.TimeRange) -> iterator.Core:
        _iter = iterator.Core(transport=self.transport.stream)
        _iter.open(keys, tr)
        return _iter
