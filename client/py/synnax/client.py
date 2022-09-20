from synnax.channel import ChannelClient
from synnax.channel.create import ChannelCreator
from synnax.channel.registry import ChannelRegistry
from synnax.channel.retrieve import ChannelRetriever
from freighter import URL
from synnax.segment import SegmentClient

from .transport import Transport


class Synnax:
    transport: Transport
    channel: ChannelClient
    data: SegmentClient

    def __init__(
        self,
        host: str,
        port: int,
    ):
        self.transport = Transport(URL(host=host, port=port))
        ch_retriever = ChannelRetriever(self.transport.http)
        ch_creator = ChannelCreator(self.transport.http)
        ch_registry = ChannelRegistry(ch_retriever)
        self.data = SegmentClient(self.transport, ch_registry)
        self.channel = ChannelClient(self.data, ch_retriever, ch_creator)
