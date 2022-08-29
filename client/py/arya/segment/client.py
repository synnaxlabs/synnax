import numpy as np

from .. import channel, telem
from ..transport import Transport
from arya.segment.writer import Numpy, Core


class Client:
    transport: Transport
    channel_client: channel.Client

    def __init__(self, transport: Transport, channel_client: channel.Client):
        self.transport = transport
        self.channel_client = channel_client

    def new_writer(self, keys: list[str]) -> Numpy:
        core = Core(transport=self.transport.stream)
        npw = Numpy(core=core, channel_client=self.channel_client)
        npw.open(keys)
        return npw

    def write(self, to: str, data: np.ndarray, start: telem.UnparsedTimeStamp):
        writer = self.new_writer([to])
        try:
            writer.write(to, data, start)
        finally:
            writer.close()